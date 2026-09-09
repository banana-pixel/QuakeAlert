package store

import (
	"context"
	"testing"
)

// --- Batas radius token FCM pada lapisan data (D-036) ---
//
// FCMTokensWithin adalah satu-satunya penegak radius di sisi server: dispatch
// jalur normal memanggilnya dengan 200, jalur lokal Admin Node dengan 20
// (dispatch.TrustedLocalRadiusKm). Uji ini membuktikan parameter radius-lah
// yang memutuskan — bukan perubahan query — pada PostGIS nyata, bukan fake.
//
// Butuh Postgres NYATA (ST_DWithin geografi). Tanpa TEST_DATABASE_URL seluruh
// test di berkas ini skip. ID user, pseudonym, dan token berprefix unik agar
// tidak bertabrakan dengan berkas lain pada basis data uji bersama.

const (
	tokCentroidLat = -6.900
	tokCentroidLon = 107.600
	// 1 derajat bujur pada lintang ini ~= 110.39 km; margin 100 m di tiap sisi
	// batas 20 km sehingga galat float geografi tidak dapat mengaburkan verdict.
	tokNearLon = 107.600 + 19.9/110.39
	tokFarLon  = 107.600 + 20.1/110.39
)

func seedTokenUser(t *testing.T, st *Store, userID, pseudo, token string, lat, lon float64) {
	t.Helper()
	ctx := context.Background()
	if _, err := st.CreateUserProfile(ctx, userID, pseudo); err != nil {
		t.Fatalf("buat user %s: %v", userID, err)
	}
	t.Cleanup(func() {
		_, _ = st.pool.Exec(ctx, `DELETE FROM user_profiles WHERE user_id = $1`, userID)
	})
	if _, err := st.UpdateUserLocation(ctx, userID, lat, lon, "uji"); err != nil {
		t.Fatalf("lokasi user %s: %v", userID, err)
	}
	if _, err := st.UpdateUserFCMToken(ctx, userID, token); err != nil {
		t.Fatalf("token user %s: %v", userID, err)
	}
}

func tokenSet(tokens []string) map[string]bool {
	out := make(map[string]bool, len(tokens))
	for _, tok := range tokens {
		out[tok] = true
	}
	return out
}

func TestFCMTokensWithin_RadiusBoundary(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	seedTokenUser(t, st, "dddddddd-0000-4000-8000-000000000031", "tok-near-31", "tok-adm-near", tokCentroidLat, tokNearLon)
	seedTokenUser(t, st, "dddddddd-0000-4000-8000-000000000032", "tok-far-32", "tok-adm-far", tokCentroidLat, tokFarLon)

	// Radius lokal 20 km: yang 19.9 km masuk, yang 20.1 km tidak.
	got20, err := st.FCMTokensWithin(ctx, tokCentroidLat, tokCentroidLon, 20)
	if err != nil {
		t.Fatalf("radius 20: %v", err)
	}
	set20 := tokenSet(got20)
	if !set20["tok-adm-near"] {
		t.Errorf("radius 20 tidak memuat token 19.9 km: %v", got20)
	}
	if set20["tok-adm-far"] {
		t.Errorf("radius 20 memuat token 20.1 km: %v", got20)
	}

	// Radius normal 200 km atas query yang SAMA memuat keduanya: yang
	// mempersempit adalah parameter, bukan perubahan query.
	got200, err := st.FCMTokensWithin(ctx, tokCentroidLat, tokCentroidLon, 200)
	if err != nil {
		t.Fatalf("radius 200: %v", err)
	}
	set200 := tokenSet(got200)
	if !set200["tok-adm-near"] || !set200["tok-adm-far"] {
		t.Errorf("radius 200 = %v; mau memuat keduanya", got200)
	}
}

// Tanpa token = irisan kosong, bukan galat: pemanggil dispatch mencatat
// AudienceNone (jalur lokal) — tidak ada fallback yang dapat dipicu dari
// himpunan kosong.
func TestFCMTokensWithin_NoTokensMeansEmpty(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()

	got, err := st.FCMTokensWithin(ctx, 0.0, -140.0, 20)
	if err != nil {
		t.Fatalf("query daerah kosong: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("daerah kosong mengembalikan %d token; mau 0", len(got))
	}
}
