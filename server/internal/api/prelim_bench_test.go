// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- Bench pra-wujud uji: endpoint admin meniru drill, tanpa DB sama sekali ---

// fakeBench captures input dan mengembalikan hasil yang diprogram.
type fakeBench struct {
	got   []PrelimTestBenchInput
	out   string
	reas  string
	calls int
}

func (f *fakeBench) bench(_ context.Context, in PrelimTestBenchInput) (string, string) {
	f.calls++
	f.got = append(f.got, in)
	return f.out, f.reas
}

// newBenchServer membangun handler dengan kunci operator dan jembatan bench
// terpasang. bench nil = mode off (harus 503, sistemnya sesuai drill).
func newBenchServer(bench *fakeBench, key string) http.Handler {
	srv := NewServer(&fakeRepo{}, fakeDecryptCipher{}, NewMemoryRateLimiter(),
		MQTTPublic{Broker: "b", Port: 8883, TLS: true},
		AuthConfig{JWTSecret: []byte(testSecret), TokenTTL: testTokenTTL},
		testLogger())
	srv.SetAdminAPIKey(key)
	if bench != nil {
		srv.SetPrelimTestBench(bench.bench)
	}
	return srv.Router(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusSwitchingProtocols)
	}, testLogger())
}

func benchRequest(body, key string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/prelim-test", strings.NewReader(body))
	if key != "" {
		req.Header.Set(AdminKeyHeader, key)
	}
	return req
}

// TestPrelimBench_HappyPath: raised → 200 dengan payload is_test "true" +
// event_id berawalan "test-" + input terpasang tanpa penyulapan field
// siklus hidup (revision/state dibangun bridge, bukan handler Athena).
func TestPrelimBench_HappyPath(t *testing.T) {
	b := &fakeBench{out: prelimBenchOutcomeRaised, reas: "ADMIN_NOT_FINAL"}
	h := newBenchServer(b, adminTestKey)

	body := `{"pga_gal":300,"latitude":-6.9175,"longitude":107.6191,
	          "location_name":"Lantai Lab, Bandung"}`
	rec := do(h, benchRequest(body, adminTestKey))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200 (%s)", rec.Code, rec.Body.String())
	}
	var resp PrelimTestBenchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body bukan JSON response: %v", err)
	}
	if !resp.IsTest {
		t.Fatal("is_test false — label latihan hilang")
	}
	if resp.Type != "EARTHQUAKE_ALERT" || resp.Outcome != "raised" {
		t.Fatalf("type/outcome = %q/%q, mau EARTHQUAKE_ALERT/raised", resp.Type, resp.Outcome)
	}
	if !strings.HasPrefix(resp.EventID, "test-") {
		t.Fatalf("event_id = %q, mau berawalan test-", resp.EventID)
	}
	if b.calls != 1 {
		t.Fatalf("panggilan bench = %d, mau tepat 1", b.calls)
	}
	in := b.got[0]
	if in.EventID != resp.EventID || in.PGAGal != 300 {
		t.Fatalf("masukan bench meleset: id=%q pga=%g", in.EventID, in.PGAGal)
	}
	if in.CentroidLat != -6.9175 || in.CentroidLon != 107.6191 {
		t.Fatalf("koordinat = %g/%g, mau memperlewat pultas", in.CentroidLat, in.CentroidLon)
	}
	if in.Timestamp == 0 {
		t.Fatal("timestamp bench kosong")
	}
	if in.LocationName != "Lantai Lab, Bandung" {
		t.Fatalf("location_name = %q tanpa normalisasi yang diinginkan", in.LocationName)
	}
}

// TestPrelimBench_GateRefusedSillOK: refusal gate adalah pengamatan benar
// (200), bukan galat server — log dan respons tetap bercerita sama.
func TestPrelimBench_GateRefusedStillOK(t *testing.T) {
	b := &fakeBench{out: prelimBenchOutcomeRefused, reas: "BELOW_FLOOR"}
	h := newBenchServer(b, adminTestKey)

	rec := do(h, benchRequest(`{"pga_gal":40,"latitude":-6.9175,"longitude":107.6191}`, adminTestKey))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200", rec.Code)
	}
	var resp PrelimTestBenchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Outcome != "refused" || resp.Reason != "BELOW_FLOOR" || !resp.IsTest {
		t.Fatalf("refusal salah: %+v", resp)
	}
}

// TestPrelimBench_Inactive503: bridge menolak (mode OFF/fail-closed) → 503,
// tanpa memaksa 200 yang menipu operator.
func TestPrelimBench_Inactive503(t *testing.T) {
	b := &fakeBench{out: prelimBenchOutcomeInactive, reas: "penadahan tidak aktif"}
	h := newBenchServer(b, adminTestKey)

	rec := do(h, benchRequest(`{"pga_gal":300,"latitude":0,"longitude":0}`, adminTestKey))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, mau 503", rec.Code)
	}
	var resp PrelimTestBenchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Outcome != "inactive" || !resp.IsTest {
		t.Fatalf("outcome inactive salah: %+v", resp)
	}
}

// TestPrelimBench_BridgeNotInstalled: setter tidak pernah dipanggil (wire
// mode OFF) → 503 dan jembatan tidak pernah jalan (calls == 0).
func TestPrelimBench_BridgeNotInstalled(t *testing.T) {
	h := newBenchServer(nil, adminTestKey)

	rec := do(h, benchRequest(`{"pga_gal":300,"latitude":0,"longitude":0}`, adminTestKey))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, mau 503 (fail-closed tanpa jembatan)", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "event_id") || strings.Contains(rec.Body.String(), "raised") {
		t.Fatalf("body 503 membawa penaikan: %s", rec.Body.String())
	}
}

// TestPrelimBench_InputValidation: rentang pga/koordinat dan kunci admin
// menahan permintaan — input tidak valid TIDAK menyentuh jembatan.
func TestPrelimBench_InputValidation(t *testing.T) {
	b := &fakeBench{out: prelimBenchOutcomeRaised, reas: "ADMIN_NOT_FINAL"}
	h := newBenchServer(b, adminTestKey)

	cases := []struct {
		name string
		body string
		key  string
		want int
	}{
		{"pga nol", `{"pga_gal":0,"latitude":0,"longitude":0}`, adminTestKey, http.StatusBadRequest},
		{"pga di atas 2000", `{"pga_gal":2001,"latitude":0,"longitude":0}`, adminTestKey, http.StatusBadRequest},
		{"tanpa koordinat", `{"pga_gal":300}`, adminTestKey, http.StatusBadRequest},
		{"latitude di luar rentang", `{"pga_gal":300,"latitude":95,"longitude":0}`, adminTestKey, http.StatusBadRequest},
		{"longitude di luar rentang", `{"pga_gal":300,"latitude":0,"longitude":200}`, adminTestKey, http.StatusBadRequest},
		{"kunci salah", `{"pga_gal":300,"latitude":0,"longitude":0}`, "salah", http.StatusUnauthorized},
		{"tanpa kunci", `{"pga_gal":300,"latitude":0,"longitude":0}`, "", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(h, benchRequest(tc.body, tc.key))
			if rec.Code != tc.want {
				t.Fatalf("status = %d, mau %d (body=%s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
	if b.calls != 0 {
		t.Fatalf("permintaan tidak valid sampai ke jembatan: %d panggilan", b.calls)
	}
}
