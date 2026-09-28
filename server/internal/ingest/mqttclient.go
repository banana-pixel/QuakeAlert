package ingest

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gclient "github.com/256dpi/gomqtt/client"
	"github.com/256dpi/gomqtt/packet"
	"github.com/256dpi/gomqtt/transport"
)

// Batas semantik MQTT tingkat-broker, menggantikan tipe paho di batas paket
// ini (AGPLv3: paho EPL-2.0 diganti gomqtt Apache-2.0 — lihat
// docs/planning/licensing/final-paho-v151-licensing-decision-audit.md).
// Topik, QoS, dan bentuk payload kontrak TIDAK berubah.

// Message adalah satu pesan MQTT masuk: topik + payload mentah.
type Message interface {
	Topic() string
	Payload() []byte
}

// MessageHandler dipanggil untuk setiap pesan pada langganan yang cocok.
type MessageHandler func(msg Message)

// Client adalah operasi MQTT yang dipakai Subscriber: langganan QoS,
// pemeriksaan koneksi untuk health check, dan putus-gracioso saat shutdown.
type Client interface {
	Subscribe(topic string, qos byte, cb MessageHandler) error
	IsConnected() bool
	Disconnect(quiesceMs uint)
}

// DialConfig membawa opsi koneksi yang sebelumnya dipegang ClientOptions
// paho di main: nilai-nilainya dipertahankan satu per satu.
type DialConfig struct {
	BrokerURL      string // mis. tls://host:8883 (tanpa kredensial)
	ClientID       string
	Username       string
	Password       string
	CleanSession   bool          // produksi: false (langganan QoS 1 lestari)
	KeepAlive      time.Duration // produksi: 30s
	ConnectTimeout time.Duration // produksi: IOTimeout; dipakai untuk connect+subscribe
	TLSConfig      *tls.Config   // nil = plaintext (pengembangan lokal saja)
	Logger         *slog.Logger
	OnDrop         func(err error) // log putus-koneksi (pengganti ConnectionLostHandler)
}

// Dial menghubungkan ke broker dan mengembalikan Client yang siap
// dilanggan. Kredensial disematkan sebagai userinfo URL (cara gomqtt
// membaca username/password), ekuivalen dengan SetUsername/SetPassword.
func Dial(cfg DialConfig) (*gomqttClient, error) {
	if cfg.Logger == nil {
		return nil, fmt.Errorf("ingest: Dial membutuhkan Logger")
	}
	brokerURL, err := withCredentials(cfg.BrokerURL, cfg.Username, cfg.Password)
	if err != nil {
		return nil, err
	}
	c := &gomqttClient{
		log:       cfg.Logger,
		dial:      cfg,
		brokerURL: brokerURL,
		subs:      make(map[string]subscription),
		stopCh:    make(chan struct{}),
	}
	if err := c.connectOnce(); err != nil {
		return nil, err
	}
	return c, nil
}

// withCredentials menyematkan username/password ke URL broker.
// URL tanpa kredensial dikembalikan apa adanya (simulator lokal).
func withCredentials(rawURL, username, password string) (string, error) {
	if username == "" {
		return rawURL, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("ingest: URL broker invalid: %w", err)
	}
	u.User = url.UserPassword(username, password)
	return u.String(), nil
}

type subscription struct {
	qos byte
	cb  MessageHandler
}

// gomqttClient membungkus satu koneksi gomqtt di belakang Client.
// Objek wrapper TIDAK PERNAH diganti: supervisor reconnect menukar handle
// gomqtt di dalamnya, sehingga Subscriber yang memegang wrapper tidak perlu
// tahu reconnect terjadi (ekuivalen AutoReconnect paho).
type gomqttClient struct {
	log       *slog.Logger
	dial      DialConfig
	brokerURL string

	mu   sync.Mutex // melindungi cli + subs
	cli  *gclient.Client
	subs map[string]subscription

	connected atomic.Bool
	closed    atomic.Bool
	stopCh    chan struct{}
	wg        sync.WaitGroup

	supervising atomic.Bool // tepat-satu supervisor reconnect
}

// dialer membangun transport dengan TLS + timeout yang sama seperti perilaku
// paho: CA sistem bila TLSConfig diberikan (produksi: MinVersion TLS1.2),
// plaintext bila nil (pengembangan lokal, sudah diperingatkan di main).
func (c *gomqttClient) dialer() *transport.Dialer {
	return transport.NewDialer(transport.DialConfig{
		TLSConfig: c.dial.TLSConfig,
		Timeout:   c.dial.ConnectTimeout,
	})
}

// connectOnce membuka SATU koneksi + melanggan ulang semua langganan
// terdaftar. Dipakai saat Dial awal dan setiap putus-koneksi oleh supervisor.
func (c *gomqttClient) connectOnce() error {
	cli := gclient.New()
	cli.Callback = c.callback

	gcfg := gclient.NewConfig(c.brokerURL)
	gcfg.ClientID = c.dial.ClientID
	gcfg.CleanSession = c.dial.CleanSession
	gcfg.KeepAlive = c.dial.KeepAlive.String()
	gcfg.Dialer = c.dialer()
	// ValidateSubs=false: paho tidak pernah menggagalkan Subscribe atas
	// penolakan suback; perilaku itu dipertahankan (broker tak pernah menolak
	// topik kontrak kita).
	gcfg.ValidateSubs = false

	fut, err := cli.Connect(gcfg)
	if err != nil {
		return fmt.Errorf("ingest: mqtt connect: %w", err)
	}
	if err := fut.Wait(c.dial.ConnectTimeout); err != nil {
		return fmt.Errorf("ingest: mqtt connect menunggu connack: %w", err)
	}

	c.mu.Lock()
	c.cli = cli
	subs := make(map[string]subscription, len(c.subs))
	for k, v := range c.subs {
		subs[k] = v
	}
	c.mu.Unlock()

	// Langganan ulang semuanya: idempoten di broker; menjamin pulih penuh
	// walau sesi broker tidak lestari (mencerminkan perilaku paho
	// CleanSession=false + kirim-ulang langganan).
	for topic, sub := range subs {
		if err := c.subscribeOnce(topic, sub.qos); err != nil {
			return fmt.Errorf("ingest: mqtt subscribe %s: %w", topic, err)
		}
	}
	c.connected.Store(true)
	return nil
}

// subscribeOnce melanggan satu topik pada koneksi AKTIF dan menunggu suback
// dalam batas ConnectTimeout. (Paho menunggu tanpa batas; batas di sini
// disengaja agar Start() tak pernah menggantung — disiplin IO server.)
func (c *gomqttClient) subscribeOnce(topic string, qos byte) error {
	c.mu.Lock()
	cli := c.cli
	c.mu.Unlock()
	if cli == nil {
		return fmt.Errorf("ingest: mqtt belum terhubung")
	}
	fut, err := cli.Subscribe(topic, packet.QOS(qos))
	if err != nil {
		return err
	}
	return fut.Wait(c.dial.ConnectTimeout)
}

// Subscribe mendaftarkan langganan (diingat untuk kirim-ulang reconnect)
// lalu melanggannya pada koneksi aktif. QoS 1 untuk jalur life-safety,
// sama seperti sebelumnya.
func (c *gomqttClient) Subscribe(topic string, qos byte, cb MessageHandler) error {
	if c.closed.Load() {
		return fmt.Errorf("ingest: mqtt client sudah ditutup")
	}
	if !c.connected.Load() {
		return fmt.Errorf("ingest: mqtt belum terhubung")
	}
	c.mu.Lock()
	c.subs[topic] = subscription{qos: qos, cb: cb}
	c.mu.Unlock()
	if err := c.subscribeOnce(topic, qos); err != nil {
		c.mu.Lock()
		delete(c.subs, topic)
		c.mu.Unlock()
		return err
	}
	c.log.Info("subscribed", "topic", topic, "qos", int(qos))
	return nil
}

// IsConnected melaporkan koneksi HIDUP saat ini. Beda satu baris dari paho:
// paho true pula saat "akan reconnect"; di sini false selama putus (lebih
// jujur untuk health check — downtime reconnect memang downtime).
func (c *gomqttClient) IsConnected() bool {
	return c.connected.Load()
}

// Disconnect mengirim paket Disconnect lalu menunggu quiesceMs agar futures
// antre selesai (ekuivalen paho Disconnect(250)), menutup supervisor, dan
// menandai client agar reconnect berhenti permanen.
func (c *gomqttClient) Disconnect(quiesceMs uint) {
	if c.closed.Swap(true) {
		return
	}
	close(c.stopCh)
	c.mu.Lock()
	cli := c.cli
	c.mu.Unlock()
	if cli != nil {
		_ = cli.Disconnect(time.Duration(quiesceMs) * time.Millisecond)
	}
	c.connected.Store(false)
	c.wg.Wait()
}

// callback dirutekan per filter langganan. err != nil berarti koneksi mati:
// tandai putus, log via OnDrop, dan pastikan supervisor berjalan.
// TIDAK PERNAH menunggu future di sini (deadlock kata dokumentasi gomqtt);
// handler verifikasi sinkron seperti pada paho.
func (c *gomqttClient) callback(msg *packet.Message, err error) error {
	if err != nil {
		c.connected.Store(false)
		if c.dial.OnDrop != nil {
			c.dial.OnDrop(err)
		}
		c.ensureSupervisor()
		return nil
	}
	if msg == nil {
		return nil
	}
	c.mu.Lock()
	subs := make([]subscription, 0, len(c.subs))
	for filter, sub := range c.subs {
		if topicMatch(filter, msg.Topic) {
			subs = append(subs, sub)
		}
	}
	c.mu.Unlock()
	for _, sub := range subs {
		sub.cb(message{topic: msg.Topic, payload: msg.Payload})
	}
	return nil
}

// ensureSupervisor menjalankan TEPAT SATU loop reconnect (pengganti
// SetAutoReconnect paho): backoff 1s→30s, connect + langganan-ulang,
// berhenti hanya saat Disconnect.
func (c *gomqttClient) ensureSupervisor() {
	if c.closed.Load() {
		return
	}
	if !c.supervising.CompareAndSwap(false, true) {
		return
	}
	c.wg.Add(1)
	go c.supervise()
}

func (c *gomqttClient) supervise() {
	defer c.wg.Done()
	defer c.supervising.Store(false)
	backoff := time.Second
	const maxBackoff = 30 * time.Second
	for {
		if c.closed.Load() {
			return
		}
		select {
		case <-c.stopCh:
			return
		case <-time.After(backoff):
		}
		if c.closed.Load() {
			return
		}
		if err := c.connectOnce(); err != nil {
			c.log.Warn("mqtt reconnect gagal, mencoba lagi", "err", err, "backoff", backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		c.log.Info("mqtt terhubung kembali")
		return
	}
}

// message mengadaptasi packet.Message ke antarmuka Message paket ini.
type message struct {
	topic   string
	payload []byte
}

func (m message) Topic() string   { return m.topic }
func (m message) Payload() []byte { return m.payload }

// topicMatch mencocokkan filter langganan (mendukung "+" satu-level dan "#"
// multi-level ala MQTT) terhadap topik konkret. Dipakai karena callback
// gomqtt menerima topik mentah, bukan per-langganan seperti paho.
func topicMatch(filter, topic string) bool {
	if filter == "#" {
		return true
	}
	fParts := strings.Split(filter, "/")
	tParts := strings.Split(topic, "/")
	for i, fp := range fParts {
		if fp == "#" {
			return true
		}
		if i >= len(tParts) {
			return false
		}
		if fp == "+" {
			continue
		}
		if fp != tParts[i] {
			return false
		}
	}
	return len(fParts) == len(tParts)
}
