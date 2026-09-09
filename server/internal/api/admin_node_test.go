package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/banana-pixel/quakealert/server/internal/store"
)

// --- Designasi Admin Node (migrasi 000010, D-036 PROPOSED) ---
//
// Uji handler dengan fakeRepo (tanpa Postgres): kontrak HTTP, pemetaan galat
// store ke status/kode, idempotensi, gerbang kunci, dan kebenaran baris audit.
// Perilaku SQL-nya sendiri (indeks unik, atomisitas) dibuktikan berkas
// integrasi internal/store/admin_node_test.go.

func designateRequest(target, key string) *http.Request {
	return adminNodeRequest(http.MethodPost, target, "", key)
}

func decodeAdminNode(t *testing.T, recBody []byte) adminNodeResponse {
	t.Helper()
	var resp adminNodeResponse
	if err := json.Unmarshal(recBody, &resp); err != nil {
		t.Fatalf("decode respons admin node: %v", err)
	}
	return resp
}

func decodeAPIError(t *testing.T, recBody []byte) string {
	t.Helper()
	var e struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(recBody, &e); err != nil {
		t.Fatalf("decode apiError: %v", err)
	}
	return e.Code
}

func TestDesignateAdminNode_Success(t *testing.T) {
	repo := &fakeRepo{}
	h := newAdminServer(repo, nil, adminTestKey)

	rec := do(h, designateRequest("/api/v1/admin/nodes/NODE-163A149F/admin-designate", adminTestKey))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200: %s", rec.Code, rec.Body.String())
	}
	resp := decodeAdminNode(t, rec.Body.Bytes())
	if resp.StationID != "NODE-163A149F" || !resp.IsAdminNode {
		t.Fatalf("respons = %+v, mau {NODE-163A149F true}", resp)
	}
	if repo.adminDesignatedID != "NODE-163A149F" {
		t.Fatalf("store dipanggil id=%q", repo.adminDesignatedID)
	}
}

func TestDesignateAdminNode_Rejections(t *testing.T) {
	for name, tc := range map[string]struct {
		err    error
		status int
		code   string
	}{
		"belum terverifikasi": {fmt.Errorf("simulasi: %w", store.ErrNodeNotVerified), http.StatusConflict, "NODE_NOT_VERIFIED"},
		"sudah ada admin":     {fmt.Errorf("simulasi: %w", store.ErrAdminNodeExists), http.StatusConflict, "ADMIN_NODE_EXISTS"},
		"id tidak ada":        {fmt.Errorf("simulasi: %w", store.ErrNodeNotFound), http.StatusNotFound, "NODE_NOT_FOUND"},
		"store galat":         {errors.New("boom"), http.StatusInternalServerError, "INTERNAL"},
	} {
		t.Run(name, func(t *testing.T) {
			repo := &fakeRepo{adminDesignateErr: tc.err}
			h := newAdminServer(repo, nil, adminTestKey)

			rec := do(h, designateRequest("/api/v1/admin/nodes/NODE-163A149F/admin-designate", adminTestKey))
			if rec.Code != tc.status {
				t.Fatalf("status = %d, mau %d: %s", rec.Code, tc.status, rec.Body.String())
			}
			if got := decodeAPIError(t, rec.Body.Bytes()); got != tc.code {
				t.Fatalf("code = %q, mau %q", got, tc.code)
			}
		})
	}
}

func TestRevokeAdminNode_Success(t *testing.T) {
	// Revoke pemegang maupun non-pemegang sama-sama 200 di handler — tidak ada
	// keadaan "sudah dicabut" yang perlu dibedakan (idempotensi lintas state
	// dibuktikan di tingkat store).
	repo := &fakeRepo{}
	h := newAdminServer(repo, nil, adminTestKey)

	rec := do(h, designateRequest("/api/v1/admin/nodes/NODE-163A149F/admin-revoke", adminTestKey))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, mau 200: %s", rec.Code, rec.Body.String())
	}
	resp := decodeAdminNode(t, rec.Body.Bytes())
	if resp.StationID != "NODE-163A149F" || resp.IsAdminNode {
		t.Fatalf("respons = %+v, mau {NODE-163A149F false}", resp)
	}
	if repo.adminRevokedID != "NODE-163A149F" {
		t.Fatalf("store dipanggil id=%q", repo.adminRevokedID)
	}
}

func TestRevokeAdminNode_Rejections(t *testing.T) {
	for name, tc := range map[string]struct {
		repo   *fakeRepo
		status int
		code   string
	}{
		"id tidak ada": {&fakeRepo{adminRevokeMissing: true}, http.StatusNotFound, "NODE_NOT_FOUND"},
		"store galat":  {&fakeRepo{adminRevokeErr: errors.New("boom")}, http.StatusInternalServerError, "INTERNAL"},
	} {
		t.Run(name, func(t *testing.T) {
			h := newAdminServer(tc.repo, nil, adminTestKey)

			rec := do(h, designateRequest("/api/v1/admin/nodes/NODE-163A149F/admin-revoke", adminTestKey))
			if rec.Code != tc.status {
				t.Fatalf("status = %d, mau %d: %s", rec.Code, tc.status, rec.Body.String())
			}
			if got := decodeAPIError(t, rec.Body.Bytes()); got != tc.code {
				t.Fatalf("code = %q, mau %q", got, tc.code)
			}
		})
	}
}

// ID salah bentuk tidak boleh menyentuh store pada kedua rute — mengikuti
// konvensi HandleVerifyNode (404 sebelum basis data).
func TestAdminNode_MalformedIDNeverTouchesStore(t *testing.T) {
	for _, target := range []string{
		"/api/v1/admin/nodes/hello/admin-designate",
		"/api/v1/admin/nodes/node-163a149f/admin-designate",
		"/api/v1/admin/nodes/hello/admin-revoke",
		"/api/v1/admin/nodes/node-163a149f/admin-revoke",
	} {
		repo := &fakeRepo{}
		h := newAdminServer(repo, nil, adminTestKey)

		rec := do(h, designateRequest(target, adminTestKey))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s: status = %d, mau 404", target, rec.Code)
		}
		if repo.adminDesignatedID != "" || repo.adminRevokedID != "" {
			t.Fatalf("%s: station_id salah bentuk menyentuh store", target)
		}
	}
}

// Gerbang kunci operator berlaku sama seperti rute admin lain: kunci yang
// salah tidak boleh sampai mengubah designasi.
func TestAdminNode_WrongKeyNeverTouchesStore(t *testing.T) {
	for _, target := range []string{
		"/api/v1/admin/nodes/NODE-163A149F/admin-designate",
		"/api/v1/admin/nodes/NODE-163A149F/admin-revoke",
	} {
		repo := &fakeRepo{}
		h := newAdminServer(repo, nil, adminTestKey)

		rec := do(h, designateRequest(target, "kunci-salah-yang-sama-panjangnya!"))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: status = %d, mau 401", target, rec.Code)
		}
		if repo.adminDesignatedID != "" || repo.adminRevokedID != "" {
			t.Fatalf("%s: kunci salah menyentuh store", target)
		}
	}
}

// --- Kebenaran baris audit ---
//
// Konvensi repo: audit designate/revoke adalah baris log terstruktur berisi
// station_id + hasil, tanpa koordinat (lihat admin_nodes.go). Uji ini
// menangkap log handler ke buffer dan membuktikan ketiganya: ID tercatat,
// hasil tercatat, dan tidak ada posisi yang bocor.

func newAdminServerWithLogBuffer(repo Repo, buf *bytes.Buffer) http.Handler {
	srv := NewServer(repo, fakeDecryptCipher{}, NewMemoryRateLimiter(),
		MQTTPublic{Broker: "b", Port: 8883, TLS: true},
		AuthConfig{JWTSecret: []byte(testSecret), TokenTTL: testTokenTTL},
		slog.New(slog.NewTextHandler(buf, nil)))
	srv.SetAdminAPIKey(adminTestKey)
	return srv.Router(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusSwitchingProtocols)
	}, testLogger())
}

func TestAdminNode_AuditLines(t *testing.T) {
	for name, tc := range map[string]struct {
		target string
		repo   *fakeRepo
		status int
		words  []string
	}{
		"designate sukses": {
			"/api/v1/admin/nodes/NODE-163A149F/admin-designate",
			&fakeRepo{}, http.StatusOK,
			[]string{"admin node ditunjuk", "NODE-163A149F"},
		},
		"designate ditolak belum terverifikasi": {
			"/api/v1/admin/nodes/NODE-163A149F/admin-designate",
			&fakeRepo{adminDesignateErr: store.ErrNodeNotVerified}, http.StatusConflict,
			[]string{"designate admin ditolak", "NODE-163A149F"},
		},
		"revoke sukses": {
			"/api/v1/admin/nodes/NODE-163A149F/admin-revoke",
			&fakeRepo{}, http.StatusOK,
			[]string{"admin node dicabut", "NODE-163A149F"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			h := newAdminServerWithLogBuffer(tc.repo, &buf)

			rec := do(h, designateRequest(tc.target, adminTestKey))
			if rec.Code != tc.status {
				t.Fatalf("status = %d, mau %d: %s", rec.Code, tc.status, rec.Body.String())
			}
			logs := buf.String()
			for _, w := range tc.words {
				if !strings.Contains(logs, w) {
					t.Fatalf("log tidak memuat %q:\n%s", w, logs)
				}
			}
			// Tidak ada posisi di baris audit: tidak ada koordinat, tidak ada
			// nama lokasi — hanya station_id sebagai identitas.
			for _, leak := range []string{"107.54", "-6.87", "location", "centroid", "lat=", "lon="} {
				if strings.Contains(strings.ToLower(logs), strings.ToLower(leak)) {
					t.Fatalf("log membocorkan posisi (%q):\n%s", leak, logs)
				}
			}
		})
	}
}
