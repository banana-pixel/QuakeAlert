// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package ingest

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func mqttTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestTopicMatch(t *testing.T) {
	cases := []struct {
		filter, topic string
		want          bool
	}{
		{"sensor/+/trigger", "sensor/NODE-1234ABCD/trigger", true},
		{"sensor/+/trigger", "sensor/NODE-1234ABCD/heartbeat", false},
		{"sensor/+/trigger", "sensor/a/b/trigger", false},
		{"sensor/+/heartbeat", "sensor/NODE-1234ABCD/heartbeat", true},
		{"#", "sensor/anything/at/all", true},
		{"sensor/#", "sensor/a/b", true},
		{"sensor/+/trigger", "other/NODE-1234ABCD/trigger", false},
		{"sensor/NODE-1234ABCD/trigger", "sensor/NODE-1234ABCD/trigger", true},
	}
	for _, c := range cases {
		if got := topicMatch(c.filter, c.topic); got != c.want {
			t.Errorf("topicMatch(%q, %q) = %v, mau %v", c.filter, c.topic, got, c.want)
		}
	}
}

func TestWithCredentials(t *testing.T) {
	// Tanpa kredensial URL tidak berubah (simulator lokal plaintext).
	u, err := withCredentials("tcp://localhost:1883", "", "")
	if err != nil || u != "tcp://localhost:1883" {
		t.Fatalf("tanpa kredensial = %q, %v; mau URL asli", u, err)
	}
	// Kredensial disematkan sebagai userinfo (cara gomqtt membaca auth).
	u, err = withCredentials("tls://broker.example.id:8883", "quakealert-server", "s3cr3t!")
	if err != nil {
		t.Fatalf("dengan kredensial err = %v", err)
	}
	if u != "tls://quakealert-server:s3cr3t%21@broker.example.id:8883" {
		t.Fatalf("userinfo salah: %q", u)
	}
	// URL rusak ditolak SEBELUM dial (tanpa sentuh jaringan).
	if _, err := withCredentials("://bad url", "u", "p"); err == nil {
		t.Fatal("URL rusak harus ditolak")
	}
}

func TestDialRequiresLogger(t *testing.T) {
	_, err := Dial(DialConfig{BrokerURL: "tcp://localhost:1883", ClientID: "x"})
	if err == nil {
		t.Fatal("Dial tanpa Logger harus gagal")
	}
}

func TestDialRejectsBadURL(t *testing.T) {
	_, err := Dial(DialConfig{
		BrokerURL:      "://bad url",
		ClientID:       "x",
		ConnectTimeout: time.Second,
		Logger:         mqttTestLogger(),
	})
	if err == nil {
		t.Fatal("Dial URL rusak harus gagal tanpa dial")
	}
}

func TestSubscribeRequiresConnection(t *testing.T) {
	c := &gomqttClient{subs: make(map[string]subscription), stopCh: make(chan struct{})}
	called := false
	if err := c.Subscribe(TriggerTopic, 1, func(Message) { called = true }); err == nil {
		t.Fatal("Subscribe tanpa koneksi harus gagal")
	}
	if called {
		t.Fatal("callback tak boleh dipanggil saat subscribe gagal")
	}
}

func TestDisconnectIdempotent(t *testing.T) {
	c := &gomqttClient{subs: make(map[string]subscription), stopCh: make(chan struct{})}
	c.Disconnect(0)
	c.Disconnect(0) // kedua kali tidak boleh panic/block
	if c.IsConnected() {
		t.Fatal("client yang tak pernah terhubung harus reported disconnected")
	}
}

func TestMessageAdapter(t *testing.T) {
	var m Message = message{topic: "sensor/NODE-1/trigger", payload: []byte{1, 2, 3}}
	if m.Topic() != "sensor/NODE-1/trigger" {
		t.Fatalf("Topic() = %q", m.Topic())
	}
	if len(m.Payload()) != 3 {
		t.Fatalf("Payload() len = %d", len(m.Payload()))
	}
}
