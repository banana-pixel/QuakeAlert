// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/banana-pixel/quakealert/server/internal/store"
)

// fakeTestAlertFanout mencatat drill yang diteruskan ke jalur WS/FCM.
type fakeTestAlertFanout struct {
	got []TestAlert
}

func (f *fakeTestAlertFanout) DispatchTestAlert(t TestAlert) { f.got = append(f.got, t) }

// newDrillServer membangun handler dengan kunci operator dan jalur drill
// terpasang. fanout nil sengaja dibiarkan mungkin: itu keadaan yang harus
// menjawab 503, bukan 200.
func newDrillServer(fanout TestAlertFanout, key string) http.Handler {
	srv := NewServer(&fakeRepo{}, fakeDecryptCipher{}, NewMemoryRateLimiter(),
		MQTTPublic{Broker: "b", Port: 8883, TLS: true},
		AuthConfig{JWTSecret: []byte(testSecret), TokenTTL: testTokenTTL},
		testLogger())
	srv.SetAdminAPIKey(key)
	if fanout != nil {
		srv.SetTestAlertFanout(fanout)
	}
	return srv.Router(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusSwitchingProtocols)
	}, testLogger())
}

func drillRequest(body, key string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/test-alert", strings.NewReader(body))
	if key != "" {
		req.Header.Set(AdminKeyHeader, key)
	}
	return req
}

// Yang diuji di sini bukan hanya bahwa drill terkirim, tetapi bahwa ia TIDAK
// menyentuh basis data: sebuah baris earthquake_events dari latihan akan muncul
// di kartu aktivitas pengguna dan menggeser hitungan 30 harinya.
func TestCreateTestAlert_DispatchesWithoutPersisting(t *testing.T) {
	fanout := &fakeTestAlertFanout{}
	h := newDrillServer(fanout, adminTestKey)

	body := `{"pga_gal":200,"latitude":-6.9175,"longitude":107.6191,
	          "location_name":"Bandung,  Jawa Barat"}`
	rec := do(h, drillRequest(body, adminTestKey))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, mau 202: %s", rec.Code, rec.Body.String())
	}
	if len(fanout.got) != 1 {
		t.Fatalf("fanout = %+v", fanout.got)
	}
	got := fanout.got[0]
	// MMI dan label diturunkan dari PGA oleh fungsi jalur konsensus, bukan
	// diterima dari pemanggil.
	if got.MMI != "VII" || got.IntensityLabel != "strong" {
		t.Fatalf("mmi/label = %q/%q, mau VII/strong", got.MMI, got.IntensityLabel)
	}
	// Id mengaku dirinya latihan, dan tidak mungkin bertabrakan dengan
	// gen_random_uuid() milik basis data.
	if !strings.HasPrefix(got.EventID, "test-") {
		t.Fatalf("event_id = %q, mau berawalan test-", got.EventID)
	}
	// Spasi ganda dari here-doc shell diratakan, seperti pada siaran.
	if got.LocationName != "Bandung, Jawa Barat" {
		t.Fatalf("location_name = %q", got.LocationName)
	}
	// Tanpa node_count, drill memakai ambang CONFIRMED — bentuk peringatan yang
	// sedang dilatih.
	if got.NodeCount != defaultTestNodeCount {
		t.Fatalf("node_count = %d, mau %d", got.NodeCount, defaultTestNodeCount)
	}

	var resp testAlertResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.IsTest || resp.Topic != "test_alerts" {
		t.Fatalf("resp = %+v", resp)
	}
}

// Tanpa location_name teksnya sendiri mengaku latihan: kalaupun sebuah build
// lolos kedua pagar dan menampilkannya, isinya masih mengatakan apa itu.
func TestCreateTestAlert_NamesItselfWhenNoLocationGiven(t *testing.T) {
	fanout := &fakeTestAlertFanout{}
	h := newDrillServer(fanout, adminTestKey)

	rec := do(h, drillRequest(`{"pga_gal":20,"latitude":0,"longitude":0}`, adminTestKey))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, mau 202: %s", rec.Code, rec.Body.String())
	}
	if fanout.got[0].LocationName != defaultTestLocationName {
		t.Fatalf("location_name = %q", fanout.got[0].LocationName)
	}
	// 0,0 dikirim eksplisit, jadi harus diterima sebagai koordinat — bukan
	// dianggap field yang absen.
	if fanout.got[0].Latitude != 0 || fanout.got[0].Longitude != 0 {
		t.Fatalf("koordinat 0,0 eksplisit harus lolos: %+v", fanout.got[0])
	}
}

func TestCreateTestAlert_RejectsUnusableInput(t *testing.T) {
	cases := map[string]string{
		"pga absen":            `{"latitude":-6.9,"longitude":107.6}`,
		"pga di bawah rentang": `{"pga_gal":0.5,"latitude":-6.9,"longitude":107.6}`,
		"pga di atas rentang":  `{"pga_gal":5000,"latitude":-6.9,"longitude":107.6}`,
		"koordinat absen":      `{"pga_gal":200}`,
		"latitude absen":       `{"pga_gal":200,"longitude":107.6}`,
		"latitude di luar":     `{"pga_gal":200,"latitude":91,"longitude":107.6}`,
		"longitude di luar":    `{"pga_gal":200,"latitude":-6.9,"longitude":181}`,
		"node_count berlebih":  `{"pga_gal":200,"latitude":-6.9,"longitude":107.6,"node_count":100}`,
		"nama terlalu panjang": `{"pga_gal":200,"latitude":-6.9,"longitude":107.6,` +
			`"location_name":"` + strings.Repeat("a", maxLocationNameLen+1) + `"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			fanout := &fakeTestAlertFanout{}
			h := newDrillServer(fanout, adminTestKey)

			rec := do(h, drillRequest(body, adminTestKey))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, mau 400: %s", rec.Code, rec.Body.String())
			}
			if len(fanout.got) != 0 {
				t.Fatal("permintaan tidak valid tidak boleh menyiarkan apa pun")
			}
		})
	}
}

// Kunci yang sama dengan siaran, jadi jalur otentikasinya juga sama — termasuk
// keadaan tanpa ADMIN_API_KEY, di mana rutenya tidak ada sama sekali.
func TestCreateTestAlert_NeedsTheOperatorKey(t *testing.T) {
	fanout := &fakeTestAlertFanout{}
	h := newDrillServer(fanout, adminTestKey)

	rec := do(h, drillRequest(`{"pga_gal":200,"latitude":-6.9,"longitude":107.6}`, "kunci-salah"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}

	absent := newDrillServer(fanout, "")
	rec = do(absent, drillRequest(`{"pga_gal":200,"latitude":-6.9,"longitude":107.6}`, adminTestKey))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404", rec.Code)
	}
	if len(fanout.got) != 0 {
		t.Fatal("tidak ada drill yang boleh lolos tanpa kunci")
	}
}

// Drill yang diterima tetapi tidak dikirim ke mana pun adalah cara terburuk
// untuk mengetahui jalur push mati, jadi keadaan itu 503 dan bukan 202.
func TestCreateTestAlert_SaysSoWhenThereIsNoFanout(t *testing.T) {
	h := newDrillServer(nil, adminTestKey)

	rec := do(h, drillRequest(`{"pga_gal":200,"latitude":-6.9,"longitude":107.6}`, adminTestKey))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, mau 503: %s", rec.Code, rec.Body.String())
	}
}

// fakePseudonymTokens pemuat semu FcmTokenByPseudonym: mengembalikan token
// tetap, ErrUserNotFound (pseudonim tak dikenal / tanpa token), atau error
// lain (kanal DB) — tiga keadaan yang masing-masing berbeda perilaku drill.
// token=="", err==nil sengaja berarti "profil tanpa token": sama-sama tidak
// ada target yang bisa dikirimi.
type fakePseudonymTokens struct {
	token  string
	err    error
	called int
	pseudo string
}

func (f *fakePseudonymTokens) FcmTokenByPseudonym(_ context.Context, pseudo string) (string, error) {
	f.called++
	f.pseudo = pseudo
	if f.err != nil {
		return "", f.err
	}
	if f.token == "" {
		return "", store.ErrUserNotFound
	}
	return f.token, nil
}

// newDrillServerWithFinder membangun handler drill dengan pembaca token
// terpasang (atau nil — untuk keadaan 503 target).
func newDrillServerWithFinder(fanout TestAlertFanout, finder PseudonymTokenFinder) http.Handler {
	srv := NewServer(&fakeRepo{}, fakeDecryptCipher{}, NewMemoryRateLimiter(),
		MQTTPublic{Broker: "b", Port: 8883, TLS: true},
		AuthConfig{JWTSecret: []byte(testSecret), TokenTTL: testTokenTTL},
		testLogger())
	srv.SetAdminAPIKey(adminTestKey)
	if fanout != nil {
		srv.SetTestAlertFanout(fanout)
	}
	if finder != nil {
		srv.SetPseudonymTokenFinder(finder)
	}
	return srv.Router(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusSwitchingProtocols)
	}, testLogger())
}

const drillBody = `{"pga_gal":300,"latitude":-6.9175,"longitude":107.6191,
	"target_pseudonym":"Quakezen-6EB53CF4"}`

// Target token TIDAK BOLEH menyentuh topic sama sekali: pengiriman tidak
// berbagi apapun dengan test_alerts — fanout menerima Token, channelnya
// eksklusif; respons menamai target via pseudonim dan TIDAK pernah token.
func TestCreateTestAlert_TargetTokenIsExclusiveNoTopicFallback(t *testing.T) {
	fanout := &fakeTestAlertFanout{}
	// sintetis, bukan token asli: cukup sebagai pengenal kebocoran dalam uji.
	tok := "uji-sintetis-drill-token-ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	finder := &fakePseudonymTokens{token: tok}
	h := newDrillServerWithFinder(fanout, finder)

	rec := do(h, drillRequest(drillBody, adminTestKey))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, mau 202: %s", rec.Code, rec.Body.String())
	}
	if len(fanout.got) != 1 {
		t.Fatalf("fanout = %+v", fanout.got)
	}
	got := fanout.got[0]
	if got.TargetToken != tok {
		t.Fatal("token harus diteruskan tegas ke fanout (bukan topic)")
	}
	if got.TargetPseudonym != "Quakezen-6EB53CF4" {
		t.Fatalf("pseudonim = %q", got.TargetPseudonym)
	}

	var resp testAlertResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	// Respons menamai target dengan PSEUDONIM, bukan token; topic tetap kosong
	if resp.Target != "Quakezen-6EB53CF4" || resp.Topic != "" {
		t.Fatalf("resp = %+v", resp)
	}
	// tidak ada token yang muncul dalam body respons
	body := rec.Body.String()
	if strings.Contains(body, tok) {
		t.Fatal("token tidak boleh muncul di respons")
	}
	// event id latihan tetap berawalan test-, is_test tetap true
	if !strings.HasPrefix(got.EventID, "test-") || !resp.IsTest {
		t.Fatalf("payload latihan berubah: %+v", got)
	}
}

// Tanpa pemembaca token terpasang, drill TARGET harus 503 (bukan diam) —
// drill biasa TANPA target tetap berjalan normal.
func TestCreateTestAlert_TargetNeedsFinder(t *testing.T) {
	fanout := &fakeTestAlertFanout{}
	h := newDrillServerWithFinder(fanout, nil)

	rec := do(h, drillRequest(drillBody, adminTestKey))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, mau 503: %s", rec.Code, rec.Body.String())
	}
	if len(fanout.got) != 0 {
		t.Fatal("target tanpa finder tidak boleh menyiarkan apa pun")
	}

	// regression: drill biasa tanpa target masih 202 topic.
	rec = do(h, drillRequest(`{"pga_gal":300,"latitude":-6.9175,"longitude":107.6191}`, adminTestKey))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("drill biasa: status = %d, mau 202", rec.Code)
	}
	var resp testAlertResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Topic != "test_alerts" || resp.Target != "" {
		t.Fatalf("drill biasa berubah: %+v", resp)
	}
}

// Pseudonim tak dikenal = 404, tanpa meng-echo informasi target, fanout
// tidak diberi apa-apa, dan TIDAK ADA SEND yang terjadi.
func TestCreateTestAlert_TargetUnknownIs404NoDispatch(t *testing.T) {
	fanout := &fakeTestAlertFanout{}
	finder := &fakePseudonymTokens{} // token kosong => ErrUserNotFound
	h := newDrillServerWithFinder(fanout, finder)

	rec := do(h, drillRequest(drillBody, adminTestKey))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404: %s", rec.Code, rec.Body.String())
	}
	if finder.called != 1 {
		t.Fatalf("finder called = %d", finder.called)
	}
	if len(fanout.got) != 0 {
		t.Fatal("target tidak dikenal tidak boleh menyiarkan apa pun")
	}
}

// Kegagalan DB pada pencarian token harus 500 dengan body generic, tanpa
// melepaskan token apa pun di dalamnya, dan tanpa pengiriman.
func TestCreateTestAlert_TargetLookupFailureIs500NoTokenLeak(t *testing.T) {
	fanout := &fakeTestAlertFanout{}
	finder := &fakePseudonymTokens{err: errors.New("pgx: server ditutup koneksi")}
	h := newDrillServerWithFinder(fanout, finder)

	rec := do(h, drillRequest(drillBody, adminTestKey))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, mau 500: %s", rec.Code, rec.Body.String())
	}
	if len(fanout.got) != 0 {
		t.Fatal("gagal baca token tidak boleh menyiarkan apa pun")
	}
	// body tidak berisi pesan error internal
	if strings.Contains(rec.Body.String(), "pgx") {
		t.Fatalf("error internal tidak boleh bocor: %s", rec.Body.String())
	}
}

// Autentikasi endpoint: dengoldan kunci salah — group admin yang sama,
// target pun tidak menembus, dan pencarian token bahkan tidak dipanggil.
func TestCreateTestAlert_TargetStillNeedsOperatorKey(t *testing.T) {
	fanout := &fakeTestAlertFanout{}
	finder := &fakePseudonymTokens{token: "tok"}
	h := newDrillServerWithFinder(fanout, finder)

	rec := do(h, drillRequest(drillBody, "kunci-salah"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, mau 401", rec.Code)
	}
	if finder.called != 0 {
		t.Fatal("pencarian token tidak boleh terjadi tanpa autentikasi")
	}
	if len(fanout.got) != 0 {
		t.Fatal("drill bertarget tidak boleh lolos tanpa kunci")
	}
}
