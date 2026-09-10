package dispatch

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/banana-pixel/quakealert/server/internal/ledger"
)

// --- Jalur dispatch lokal Admin Node (D-036 PROPOSED) ---
//
// DispatchTrustedLocalEventFrame sinkron (lookup + kirim + catat), jadi uji di
// bawah menegaskan tanpa polling — tidak seperti dispatchFCM yang async.

// localFrame membangun frame lokal satu-node; severe=true membuatnya
// melampaui SeverePGAGal + MMI VIII — tepat kasus yang TIDAK BOLEH menyentuh
// GeoTopic pada jalur ini.
func localFrame(severe bool) *AlertMessage {
	m := &AlertMessage{
		Type:           TypeAlert,
		EventID:        "evt-local-1",
		MMI:            "V",
		IntensityLabel: "moderate",
		PGAGal:         150,
		CentroidLat:    -6.9,
		CentroidLon:    107.6,
		LocationName:   "Bandung, West Java, ID",
		Timestamp:      time.Now().UnixMilli(),
		NodeCount:      1,
		EventState:     "UNCONFIRMED",
		EventRevision:  1,
		TrustedLocal:   true,
	}
	if severe {
		m.PGAGal = SeverePGAGal + 100
		m.MMI = "VIII"
	}
	return m
}

func trustedLocalFixture(t *testing.T, tokens []string) (*Dispatcher, *fakeFCM, *fakeEmissionWriter, *client, *fakeTargetedSaver) {
	t.Helper()
	saver := &fakeTargetedSaver{tokens: tokens}
	fcm := &fakeFCM{}
	h := NewHub(testLogger(), func(*http.Request) bool { return true })
	d := NewDispatcher(saver, h, fcm, time.Hour, testLogger())
	w := &fakeEmissionWriter{}
	d.SetLedger(w)
	return d, fcm, w, registerClient(h), saver
}

// Radius token HARUS 20 km (bukan 200 km normal): fakeTargetedSaver mencatat
// setiap rangeKm yang diminta.
func TestTrustedLocalUses20KmRadius(t *testing.T) {
	d, fcm, w, c, saver := trustedLocalFixture(t, []string{"tok-a", "tok-b"})

	d.DispatchTrustedLocalEventFrame(context.Background(), localFrame(false))

	tokens, topics := fcm.targets()
	if len(tokens) != 2 || len(topics) != 0 {
		t.Fatalf("tokens=%v topics=%v, mau 2 token tanpa topic", tokens, topics)
	}
	saver.mu.Lock()
	calls := append([]float64(nil), saver.calls...)
	saver.mu.Unlock()
	if len(calls) != 1 || calls[0] != TrustedLocalRadiusKm {
		t.Fatalf("radius diminta = %v, mau tepat [%d]", calls, TrustedLocalRadiusKm)
	}
	if TrustedLocalRadiusKm != 20 {
		t.Fatalf("TrustedLocalRadiusKm = %d, mau 20", TrustedLocalRadiusKm)
	}

	var ws AlertMessage
	if err := json.Unmarshal(readMessage(t, c.send), &ws); err != nil {
		t.Fatalf("unmarshal WS: %v", err)
	}
	if !ws.TrustedLocal || ws.Type != TypeAlert {
		t.Fatalf("frame WS = (%q, trusted=%v)", ws.Type, ws.TrustedLocal)
	}

	rows := w.snapshot()
	if len(rows) != 1 || rows[0].Audience != ledger.AudienceTokensRadiusLocal {
		t.Fatalf("audience = %v, mau [TOKENS_RADIUS_20KM]", w.audiences())
	}
}

// INVARIAN: single-node SEVERE pada jalur lokal TIDAK PERNAH menyentuh
// GeoTopic — override severe hanya hidup di dispatchFCM normal, dan jalur ini
// tidak memanggilnya. Ini regresi langsung larangan GeoTopic nasional.
func TestTrustedLocalSevereSingleNodeNeverGeoTopic(t *testing.T) {
	d, fcm, w, _, _ := trustedLocalFixture(t, []string{"tok-a"})

	d.DispatchTrustedLocalEventFrame(context.Background(), localFrame(true))

	_, topics := fcm.targets()
	if len(topics) != 0 {
		t.Fatalf("severe satu-node menyentuh topic %v pada jalur lokal", topics)
	}
	if got := w.audiences(); len(got) != 1 || got[0] != ledger.AudienceTokensRadiusLocal {
		t.Fatalf("audience = %v, mau [TOKENS_RADIUS_20KM]", got)
	}
}

// Tanpa token = tanpa audiens FCM (nol teramati), tanpa fallback topik —
// bahkan saat severe. Guard normal tidak disentuh: hasil yang sama tercapai
// karena jalur ini memang tidak punya cabang GeoTopic.
func TestTrustedLocalNoTokensMeansNoAudience(t *testing.T) {
	for _, severe := range []bool{false, true} {
		d, fcm, w, c, _ := trustedLocalFixture(t, nil)

		d.DispatchTrustedLocalEventFrame(context.Background(), localFrame(severe))

		if fcm.count() != 0 {
			_, topics := fcm.targets()
			t.Fatalf("severe=%v: FCM terkirim %d (topics=%v), mau 0", severe, fcm.count(), topics)
		}
		if got := w.audiences(); len(got) != 1 || got[0] != ledger.AudienceNone {
			t.Fatalf("severe=%v: audience = %v, mau [NONE]", severe, got)
		}
		readMessage(t, c.send) // WS tetap disiarkan seperti setiap frame
	}
}

// Tanpa sender FCM tidak panik dan tercatat NONE tanpa flag configured —
// mengikuti konvensi dispatchFCM untuk instalasi tanpa kredensial.
func TestTrustedLocalWithoutFCMSender(t *testing.T) {
	saver := &fakeTargetedSaver{tokens: []string{"tok-a"}}
	h := NewHub(testLogger(), func(*http.Request) bool { return true })
	d := NewDispatcher(saver, h, nil, time.Hour, testLogger())
	w := &fakeEmissionWriter{}
	d.SetLedger(w)

	d.DispatchTrustedLocalEventFrame(context.Background(), localFrame(false))

	rows := w.snapshot()
	if len(rows) != 1 || rows[0].Audience != ledger.AudienceNone {
		t.Fatalf("audience = %v, mau [NONE]", w.audiences())
	}
	if rows[0].FCMAttempted != nil {
		t.Fatal("fcm_attempted terisi padahal FCM tak dikonfigurasi (mau NULL)")
	}
}

// D-019 untuk emisi lokal (D-037): tepat satu baris outcome per pemanggilan
// dispatch — event_id + enum saja. Tanpa koordinat, jarak, atau data turunan
// lokasi: satu-satunya angka pada baris ini adalah ID event.
func TestTrustedLocalD019OutcomeExactlyOnce(t *testing.T) {
	var buf bytes.Buffer
	saver := &fakeTargetedSaver{tokens: []string{"tok-a"}}
	fcm := &fakeFCM{}
	h := NewHub(testLogger(), func(*http.Request) bool { return true })
	d := NewDispatcher(saver, h, fcm, time.Hour,
		slog.New(slog.NewTextHandler(&buf, nil)))
	d.SetLedger(&fakeEmissionWriter{})

	d.DispatchTrustedLocalEventFrame(context.Background(), localFrame(false))

	// Tepat satu baris outcome untuk satu emisi (jalur token di bawah juga
	// mencatat baris kirimnya sendiri — yang dihitung di sini hanya outcome).
	if got := strings.Count(buf.String(), "trusted-local emitted"); got != 1 {
		t.Fatalf("baris outcome = %d, mau tepat 1:\n%s", got, buf.String())
	}
	var outcome string
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if strings.Contains(line, "trusted-local emitted") {
			outcome = line
		}
	}
	for _, want := range []string{"evt-local-1", "ADMIN_ELIGIBLE"} {
		if !strings.Contains(outcome, want) {
			t.Fatalf("baris outcome tidak memuat %q:\n%s", want, outcome)
		}
	}
	// Privasi D-019: centroid frame ini (-6.9, 107.6) tidak boleh bocor ke
	// log mana pun pada jalur ini.
	for _, leak := range []string{"-6.9", "107.6", "centroid", "distance"} {
		if strings.Contains(buf.String(), leak) {
			t.Fatalf("log membocorkan posisi (%q):\n%s", leak, buf.String())
		}
	}
}

// REGRESI CONFIRMED: jalur normal tidak berubah oleh adanya jalur lokal —
// CONFIRMED 3-node non-severe tetap memakai radius 200 km + fallback topik,
// dan frame-nya tidak membawa trusted_local.
func TestNormalDispatchUnchangedByLocalPath(t *testing.T) {
	d, fcm, w, _, saver := trustedLocalFixture(t, nil)

	d.Dispatch(context.Background(), nonSevereEvent(3))

	if !waitFor(func() bool { return len(w.snapshot()) == 1 }) {
		t.Fatal("timeout menunggu baris emisi normal")
	}
	if got := w.audiences(); len(got) != 1 || got[0] != ledger.AudienceGeoTopicAll {
		t.Fatalf("audience normal = %v, mau [GEO_TOPIC_ALL] (fallback 200 km utuh)", got)
	}
	saver.mu.Lock()
	calls := append([]float64(nil), saver.calls...)
	saver.mu.Unlock()
	for _, c := range calls {
		if c != AlertRadiusKm {
			t.Fatalf("jalur normal meminta radius %v, mau 200", calls)
		}
	}
	tokens, topics := fcm.targets()
	if len(topics) != 1 || len(tokens) != 0 {
		t.Fatalf("tokens=%v topics=%v, mau fallback 1 topic", tokens, topics)
	}
}
