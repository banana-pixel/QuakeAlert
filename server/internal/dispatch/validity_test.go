package dispatch

import (
	"encoding/json"
	"testing"
	"time"
)

// --- Masa berlaku yang dinyatakan pengirim (D-018, U-010) ---
//
// Aturan yang diuji di sini: setiap frame BARU membawa validity_ms (durasi
// resolve-after jalurnya), frame lama tanpa field ini tetap valid dan klien
// membaca ketiadaannya sebagai jendela legacy — tidak pernah kedaluwarsa.

// BuildAlertData hanya menambahkan validity_ms bila BERISI: payload lama
// tanpa field ini tetap byte-identik, dan klien lama yang tidak mengenal
// field ini mengabaikannya (ignoreUnknownKeys / map lookup).
func TestBuildAlertDataValidityAdditive(t *testing.T) {
	legacy := BuildAlertData(&AlertMessage{Type: TypeAlert, EventID: "event-1"})
	if _, ok := legacy["validity_ms"]; ok {
		t.Fatalf("validity_ms hadir tanpa dinyatakan: %v", legacy)
	}
	with := BuildAlertData(&AlertMessage{Type: TypeAlert, EventID: "event-1", ValidityMs: 90000})
	if with["validity_ms"] != "90000" {
		t.Fatalf("validity_ms = %q, mau \"90000\"", with["validity_ms"])
	}
}

// Marshal JSON WS menghilangkan validity_ms yang nol: frame lama (misalnya
// dari instalasi yang belum diupgrade) tetap byte-identik di kawat.
func TestAlertMessageValidityOmitsWhenUnset(t *testing.T) {
	raw, err := json.Marshal(&AlertMessage{Type: TypeAlert, EventID: "E1"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := got["validity_ms"]; ok {
		t.Fatalf("validity_ms hadir saat nol: %s", raw)
	}
	raw, err = json.Marshal(&AlertMessage{Type: TypeAlert, EventID: "E1", ValidityMs: 90000})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["validity_ms"] != float64(90000) {
		t.Fatalf("validity_ms = %v, mau 90000", got["validity_ms"])
	}
}

// Jalur emisi Fase 3 mengisi resolve-after dispatcher bila pemanggil (Tracker)
// belum menyatakan masa berlaku — dan tidak pernah menimpa nilai eksplisit
// (misalnya drill tidak boleh mendapat 90 detik).
func TestDispatchEventFrameStampsResolveAfterWhenUnset(t *testing.T) {
	fcm := &fakeFCM{}
	d, _, c := eventFrameFixture(t, []string{"token-a"}, fcm)

	d.DispatchEventFrame(t.Context(), frame(TypeAlert, "CONFIRMED", 2), true)

	raw := readMessage(t, c.send)
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("frame WS bukan JSON: %v", err)
	}
	// Fixture memakai resolveAfter satu jam: yang diuji adalah penyalurannya,
	// bukan angkanya (angka produksi 90 detik diuji di bawah).
	if got["validity_ms"] != float64(time.Hour.Milliseconds()) {
		t.Fatalf("validity_ms = %v, mau %d", got["validity_ms"], time.Hour.Milliseconds())
	}

	explicit := frame(TypeAlert, "CONFIRMED", 2)
	explicit.ValidityMs = 20000
	d.DispatchEventFrame(t.Context(), explicit, true)
	raw = readMessage(t, c.send)
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("frame WS bukan JSON: %v", err)
	}
	if got["validity_ms"] != float64(20000) {
		t.Fatalf("nilai eksplisit tertimpa: validity_ms = %v, mau 20000", got["validity_ms"])
	}
}

// Jalur Dispatch (legacy) dan resolve all-clear mencap resolve-after yang
// dikonfigurasi — bukan salinan angka kedua.
func TestDispatchStampsConfiguredResolveAfter(t *testing.T) {
	fcm := &fakeFCM{}
	h := NewHub(testLogger(), nil)
	d := NewDispatcher(&fakeTargetedSaver{tokens: []string{"token-a"}}, h, fcm, 90*time.Second, testLogger())
	c := registerClient(h)

	d.DispatchEventFrame(t.Context(), &AlertMessage{Type: TypeAlert, EventID: "E9"}, true)
	raw := readMessage(t, c.send)
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("frame WS bukan JSON: %v", err)
	}
	if got["validity_ms"] != float64(90000) {
		t.Fatalf("validity_ms = %v, mau 90000 (resolve-after terkonfigurasi)", got["validity_ms"])
	}
}

// Drill mencap janji all-clear-nya sendiri (20 detik), bukan 90 detik jalur
// nyata: kalau tidak, klien menahan layar latihan 70 detik setelah mati.
func TestDispatchTestAlertStampsDrillResolveAfter(t *testing.T) {
	fcm := &fakeFCM{}
	d := NewDispatcher(&fakeTargetedSaver{tokens: []string{"tok-a"}}, testHub(), fcm, 0, testLogger())

	d.DispatchTestAlert(testDrill())

	waitForSends(t, fcm, 1)
	if fcm.sends[0].Data["validity_ms"] != "20000" {
		t.Fatalf("validity_ms drill = %q, mau \"20000\"", fcm.sends[0].Data["validity_ms"])
	}
}
