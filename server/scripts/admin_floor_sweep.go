//go:build ignore

// admin_floor_sweep.go — alat RISET OFFLINE (bukan produksi): menyapu lantai
// kandidat peringatan lokal Admin Node atas satu jendela ledger, HANYA-BACA,
// dan menjawab satu pertanyaan saja — DETEKSI: "andai lantainya X gal, frame
// UNCONFIRMED mana yang akan lolos gerbang kelayakan?".
//
// Empat sifat yang menentukan bentuk berkas ini:
//
//  1. LANTAI PRODUKSI TIDAK DISENTUH. AdminNodeMinPGAGal (140.0) tetap
//     konstanta compile-time (D-007). Alat ini membawa SALINAN sengaja dari
//     EvaluateAdminNodeEligibility (lihat eligibleAtFloor) yang berbeda pada
//     SATU baris saja: lantai adalah parameter, bukan konstanta. Itu persis
//     yang diizinkan 05-experiment-plan §3 dan 08-safety-analysis §4
//     ("harness parametrizes a COPY only"). Tidak ada baris produksi berubah.
//
//  2. SALINAN DIBUKTIKAN SETIA SEBELUM MELAPOR. Sebuah salinan yang sudah
//     melenceng dari produksi akan melaporkan angka yang bukan perilaku
//     sistem. Maka sebelum menyapu, alat menjalankan salinannya pada lantai
//     140 dan membandingkannya frame-demi-frame dengan res.AdminOutcomes
//     (evaluator PRODUKSI yang sama yang dipakai TrustedLocalFrameFor). Bila
//     ada satu pun beda, alat BERHENTI — bukan melaporkan sapuan yang tak
//     dapat dipercaya.
//
//  3. HANYA-BACA, seperti replay_window.go: dua SELECT lewat store, Replay
//     tanpa persister, Tracker instans-fungsi yang tidak menulis apa pun.
//     Jalankan di bawah peran/replica read-only (default_transaction_read_only
//     =on) agar dijamin, bukan sekadar diklaim.
//
//  4. HANYA MENJAWAB DETEKSI. Alat ini TIDAK mengukur:
//       - LATENSI (seberapa lebih awal lantai lebih rendah memicu): kurva
//         pertumbuhan PGA intra-event (saat 100->140 dilewati) TIDAK terekam —
//         firmware hanya mencatat PRELIM peak-so-far dan FINAL peak (F-04).
//       - LAJU PERINGATAN PALSU (metric D, 05-experiment): butuh baterai
//         gangguan fisik di bench, bukan pemutaran ulang riwayat. Baris
//         100-140 gal di 08-safety §2 KOSONG sampai baterai itu dijalankan.
//     Sapuan deteksi adalah SATU dari enam metrik, bukan keputusan.
//
// Pemakaian (mode DB, jendela nyata — dijalankan PEMILIK atas basis read-only):
//
//	FROM_TS=1788255600000 TO_TS=1788255710000 \
//	ADMIN_NODE_ID=NODE-52960B47 ADMIN_NODE_VERIFIED=true \
//	DATABASE_URL="postgres://readonly@host/quakealert?sslmode=disable" \
//	go run scripts/admin_floor_sweep.go
//
// Pemakaian (mode SELFTEST, tanpa DB — membuktikan salinan==produksi @140 dan
// mendemokan sapuan atas frame sintetis; dapat dijalankan siapa saja, kapan saja):
//
//	SELFTEST=1 go run scripts/admin_floor_sweep.go
//
// Env sama dengan replay_window.go (FROM_TS, TO_TS, ADMIN_NODE_*, profil
// parameter opsional). ADMIN_NODE_ID WAJIB di mode DB: tanpa designasi tidak
// ada satu pun frame yang dinilai (fitur mati), jadi tidak ada yang disapu.
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/banana-pixel/quakealert/server/internal/event"
	"github.com/banana-pixel/quakealert/server/internal/store"
)

// floors adalah kandidat yang disapu (06-replay-plan §"floors X"). 140 SELALU
// disertakan sebagai kolom kontrol: ia harus mereproduksi keputusan produksi.
var floors = []float64{60, 80, 100, 120, 140, 160}

func main() {
	if os.Getenv("SELFTEST") != "" {
		selftest()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fromTS := mustInt("FROM_TS")
	toTS := mustInt("TO_TS")
	if fromTS > toTS {
		die("FROM_TS (%d) > TO_TS (%d)", fromTS, toTS)
	}

	prof := profileFromEnv()
	if prof.AdminNode == nil {
		die("ADMIN_NODE_ID wajib: tanpa designasi tidak ada frame yang dinilai, jadi tidak ada yang disapu")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://quakealert:devpassword@localhost:5432/quakealert?sslmode=disable"
	}
	st, err := store.New(ctx, dbURL)
	if err != nil {
		die("koneksi basis data: %v", err)
	}
	defer st.Close()

	obs, err := st.ListObservationsForReplay(ctx, fromTS, toTS)
	if err != nil {
		die("baca sensor_observations: %v", err)
	}

	banner(fromTS, toTS, prof, len(obs))
	if len(obs) == 0 {
		fmt.Println("TIDAK ADA OBSERVASI di jendela ini. Tidak ada yang dapat disapu.")
		os.Exit(2)
	}

	res, err := event.Replay(ctx, obs, prof)
	if err != nil {
		die("replay: %v", err)
	}

	// (2) Buktikan salinan setia SEBELUM menyapu. Berhenti bila menyimpang.
	assertCopyFaithful(res)

	sweep(res)
}

// ---------------------------------------------------------------------------
// Gerbang kelayakan yang DIPARAMETERKAN — SALINAN SENGAJA dari produksi
// ---------------------------------------------------------------------------

// eligibleAtFloor adalah salinan baris-demi-baris dari
// event.EvaluateAdminNodeEligibility, berbeda pada SATU tempat: `floor`
// menggantikan konstanta AdminNodeMinPGAGal. Urutan gerbang, kosakata alasan,
// dan pembanding "<" identik — sengaja, agar kolom 140 mereproduksi produksi
// dan assertCopyFaithful dapat membuktikannya. Jangan "rapikan": kesamaan
// itulah yang membuat sapuan ini bermakna.
func eligibleAtFloor(state event.AdminNodeState, s event.Snapshot, floor float64) (bool, string) {
	switch {
	case !state.Designated:
		return false, event.ReasonAdminNodeNoneDesignated
	case !state.Verified:
		return false, event.ReasonAdminNodeUnverified
	case s.To != event.StateUnconfirmed:
		return false, event.ReasonAdminNodeNotUnconfirmed
	}
	contrib := adminContribution(state.StationID, s.Evidence.Contributors)
	switch {
	case contrib == nil:
		return false, event.ReasonAdminNodeNoContribution
	case contrib.Phase != event.PhaseFinal:
		return false, event.ReasonAdminNodeNotFinal
	case contrib.PeakPGA < floor:
		return false, event.ReasonAdminNodeBelowFloor
	case state.HeartbeatAge > event.AdminNodeHeartbeatMaxAge:
		return false, event.ReasonAdminNodeHeartbeatStale
	default:
		return true, event.ReasonAdminNodeEligible
	}
}

// adminContribution meniru helper unexported bernama sama di admin_node.go:
// kecocokan pertama pada node_id sudah cukup (satu node = satu kontributor).
func adminContribution(stationID string, cs []event.ContributorEvidence) *event.ContributorEvidence {
	for i := range cs {
		if cs[i].NodeID == stationID {
			return &cs[i]
		}
	}
	return nil
}

// adminStateFromProfile membangun AdminNodeState persis seperti evaluateReplayAdmin
// di replay.go: designated=true bila ada designasi, umur negatif dijepit ke nol.
func adminStateFromProfile(a *event.ReplayAdminNode) event.AdminNodeState {
	age := a.HeartbeatAge
	if age < 0 {
		age = 0
	}
	return event.AdminNodeState{Designated: true, StationID: a.StationID, Verified: a.Verified, HeartbeatAge: age}
}

// ---------------------------------------------------------------------------
// Penjaga kesetiaan + sapuan
// ---------------------------------------------------------------------------

// assertCopyFaithful membuktikan salinan == produksi pada lantai 140. res.Frames
// dan res.AdminOutcomes sejajar 1:1 (replay.go), dan res.AdminOutcomes berasal
// dari EvaluateAdminNodeEligibility PRODUKSI. Jadi salinan pada 140 harus
// memberi (eligible, reason) yang sama untuk SETIAP frame. Bila tidak, salinan
// telah menyimpang dari produksi dan tak satu pun angka sapuan boleh dipercaya.
func assertCopyFaithful(res *event.ReplayResult) {
	if res.Profile.AdminNode == nil {
		die("assertCopyFaithful: profil tanpa AdminNode; res.AdminOutcomes kosong")
	}
	if len(res.Frames) != len(res.AdminOutcomes) {
		die("assertCopyFaithful: frame=%d outcome=%d; harus sejajar", len(res.Frames), len(res.AdminOutcomes))
	}
	state := adminStateFromProfile(res.Profile.AdminNode)
	for i, f := range res.Frames {
		wantElig, wantReason := res.AdminOutcomes[i].Eligible, res.AdminOutcomes[i].Reason
		gotElig, gotReason := eligibleAtFloor(state, f, event.AdminNodeMinPGAGal)
		if gotElig != wantElig || gotReason != wantReason {
			die("SALINAN MENYIMPANG DARI PRODUKSI @140 pada %s rev%d: produksi=(%v,%s) salinan=(%v,%s). "+
				"Sapuan DIBATALKAN — perbaiki eligibleAtFloor agar identik dengan EvaluateAdminNodeEligibility.",
				f.EventID, f.Revision, wantElig, wantReason, gotElig, gotReason)
		}
	}
	fmt.Printf("penjaga kesetiaan : OK — salinan == EvaluateAdminNodeEligibility pada 140 gal untuk %d frame\n\n",
		len(res.Frames))
}

// sweep mencetak, untuk tiap frame UNCONFIRMED (satu-satunya yang bisa layak),
// puncak kontribusi admin lalu kelayakan pada tiap lantai kandidat, dan menyorot
// frame yang lantai lebih rendah AKAN loloskan padahal 140 menolaknya.
func sweep(res *event.ReplayResult) {
	state := adminStateFromProfile(res.Profile.AdminNode)
	fmt.Println("--- sapuan lantai (DETEKSI saja; bukan latensi, bukan laju palsu) ---------")
	fmt.Printf("%-12s %-4s %-10s", "event", "rev", "admin_pga")
	for _, fl := range floors {
		fmt.Printf(" %6.0f", fl)
	}
	fmt.Println("   lantai-terendah-yang-loloskan")

	var newlyAdmitted int
	for i, f := range res.Frames {
		if f.To != event.StateUnconfirmed {
			continue
		}
		contrib := adminContribution(state.StationID, f.Evidence.Contributors)
		pga := -1.0
		if contrib != nil {
			pga = contrib.PeakPGA
		}
		fmt.Printf("%-12s %-4d %-10.4f", f.EventID, f.Revision, pga)
		lowest := ""
		for _, fl := range floors {
			elig, _ := eligibleAtFloor(state, f, fl)
			mark := "·"
			if elig {
				mark = "✓"
				if lowest == "" {
					lowest = fmt.Sprintf("%.0f", fl)
				}
			}
			fmt.Printf(" %6s", mark)
		}
		// Alasan pada 140 (produksi): mengapa frame ini tidak/di ambang lolos.
		_, r140 := eligibleAtFloor(state, f, 140.0)
		fmt.Printf("   %-4s (@140: %s)\n", lowest, r140)
		e140, _ := eligibleAtFloor(state, f, 140.0)
		e60, _ := eligibleAtFloor(state, f, 60.0)
		if !e140 && e60 && res.AdminOutcomes[i].Reason == event.ReasonAdminNodeBelowFloor {
			newlyAdmitted++
		}
	}
	fmt.Printf("\nframe yang DITOLAK 140 semata karena lantai, tetapi lolos di >=60: %d\n", newlyAdmitted)
	fmt.Println("Ini DETEKSI, bukan keputusan. Latensi (F-04, kurva intra-event tak terekam)")
	fmt.Println("dan laju peringatan palsu (metric D, baterai gangguan) TIDAK terjawab di sini.")
}

// ---------------------------------------------------------------------------
// SELFTEST — tanpa DB: buktikan kesetiaan salinan + demokan sapuan
// ---------------------------------------------------------------------------

const adminID = "NODE-ADM-SELFTEST"

// synthFrame membangun Snapshot sintetis dengan satu kontribusi milik admin.
func synthFrame(id string, to event.State, node, phase string, pga float64) event.Snapshot {
	return event.Snapshot{
		EventID:  id,
		To:       to,
		Revision: 1,
		Evidence: event.EvidenceSummary{
			Contributors: []event.ContributorEvidence{
				{NodeID: node, PeakPGA: pga, Phase: phase},
			},
		},
	}
}

func selftest() {
	fresh := event.AdminNodeState{Designated: true, StationID: adminID, Verified: true, HeartbeatAge: time.Minute}
	stale := event.AdminNodeState{Designated: true, StationID: adminID, Verified: true, HeartbeatAge: time.Hour}

	// Kasus lintas-batas + setiap cabang negatif, agar kesetiaan diuji di
	// SELURUH gerbang, bukan cuma perbandingan lantai.
	type tc struct {
		name  string
		state event.AdminNodeState
		snap  event.Snapshot
	}
	cases := []tc{
		{"pga 50 (< semua)", fresh, synthFrame("e-050", event.StateUnconfirmed, adminID, event.PhaseFinal, 50)},
		{"pga 77.89 (4fcc3374-kelas)", fresh, synthFrame("e-078", event.StateUnconfirmed, adminID, event.PhaseFinal, 77.89)},
		{"pga 100 (tepat)", fresh, synthFrame("e-100", event.StateUnconfirmed, adminID, event.PhaseFinal, 100)},
		{"pga 120", fresh, synthFrame("e-120", event.StateUnconfirmed, adminID, event.PhaseFinal, 120)},
		{"pga 139.999 (di bawah 140)", fresh, synthFrame("e-139", event.StateUnconfirmed, adminID, event.PhaseFinal, 139.999)},
		{"pga 140 (tepat lantai)", fresh, synthFrame("e-140", event.StateUnconfirmed, adminID, event.PhaseFinal, 140)},
		{"pga 288.36 (004c5f65-kelas)", fresh, synthFrame("e-288", event.StateUnconfirmed, adminID, event.PhaseFinal, 288.36)},
		{"PRELIM bukan FINAL", fresh, synthFrame("e-pre", event.StateUnconfirmed, adminID, "PRELIM", 300)},
		{"CONFIRMED bukan UNCONFIRMED", fresh, synthFrame("e-con", "CONFIRMED", adminID, event.PhaseFinal, 300)},
		{"kontributor node lain", fresh, synthFrame("e-oth", event.StateUnconfirmed, "NODE-LAIN", event.PhaseFinal, 300)},
		{"heartbeat basi 1 jam", stale, synthFrame("e-hbt", event.StateUnconfirmed, adminID, event.PhaseFinal, 300)},
	}

	fmt.Println("=== SELFTEST: salinan eligibleAtFloor(·,140) vs EvaluateAdminNodeEligibility ===")
	bad := 0
	for _, c := range cases {
		wantElig := event.EvaluateAdminNodeEligibility(c.state, c.snap)
		gotElig, gotReason := eligibleAtFloor(c.state, c.snap, event.AdminNodeMinPGAGal)
		ok := gotElig == wantElig.Eligible && gotReason == wantElig.Reason
		status := "OK"
		if !ok {
			status = "MENYIMPANG"
			bad++
		}
		fmt.Printf("  %-28s produksi=(%v,%s) salinan=(%v,%s)  %s\n",
			c.name, wantElig.Eligible, wantElig.Reason, gotElig, gotReason, status)
	}
	if bad > 0 {
		die("SELFTEST GAGAL: %d kasus menyimpang — salinan tidak setia, jangan dipakai menyapu", bad)
	}
	fmt.Printf("kesetiaan: OK untuk %d kasus di seluruh gerbang.\n\n", len(cases))

	// Demokan sapuan atas frame UNCONFIRMED/FINAL milik admin.
	fmt.Println("=== SELFTEST: demo sapuan (frame UNCONFIRMED/FINAL, heartbeat segar) ===")
	fmt.Printf("%-10s", "admin_pga")
	for _, fl := range floors {
		fmt.Printf(" %6.0f", fl)
	}
	fmt.Println()
	for _, pga := range []float64{50, 77.89, 100, 120, 139.999, 140, 160, 288.36} {
		snap := synthFrame("demo", event.StateUnconfirmed, adminID, event.PhaseFinal, pga)
		fmt.Printf("%-10.3f", pga)
		for _, fl := range floors {
			elig, _ := eligibleAtFloor(fresh, snap, fl)
			mark := "·"
			if elig {
				mark = "✓"
			}
			fmt.Printf(" %6s", mark)
		}
		fmt.Println()
	}
	fmt.Println("\nSELFTEST selesai. Ini menguji LOGIKA sapuan, bukan data produksi mana pun.")
}

// ---------------------------------------------------------------------------
// Profil, banner, pembaca env (sejajar dengan replay_window.go)
// ---------------------------------------------------------------------------

func profileFromEnv() event.ReplayProfile {
	p := event.DefaultReplayProfile()
	p.Options.CorrelationWindowMs = envInt("CORRELATION_WINDOW_MS", p.Options.CorrelationWindowMs)
	p.Options.AttachRadiusKm = envFloat("ATTACH_RADIUS_KM", p.Options.AttachRadiusKm)
	p.Options.IndependenceCellKm = envFloat("INDEPENDENCE_CELL_KM", p.Options.IndependenceCellKm)
	p.Options.MinIndependentCells = int(envInt("MIN_INDEPENDENT_CELLS", int64(p.Options.MinIndependentCells)))
	p.Options.MaxEventDiameterKm = envFloat("MAX_EVENT_DIAMETER_KM", p.Options.MaxEventDiameterKm)
	p.Options.ResolveAfterMs = envInt("EVENT_RESOLVE_AFTER_MS", p.Options.ResolveAfterMs)
	p.Options.SweepIntervalMs = envInt("EVENT_SWEEP_INTERVAL_MS", p.Options.SweepIntervalMs)
	p.DecidedAtToleranceMs = envInt("DECIDED_AT_TOLERANCE_MS", 0)
	if id := os.Getenv("ADMIN_NODE_ID"); id != "" {
		verifiedRaw := os.Getenv("ADMIN_NODE_VERIFIED")
		if verifiedRaw != "true" && verifiedRaw != "false" {
			die("ADMIN_NODE_ID diisi tetapi ADMIN_NODE_VERIFIED=%q: wajib \"true\"/\"false\" eksplisit", verifiedRaw)
		}
		p.AdminNode = &event.ReplayAdminNode{
			StationID:    id,
			Verified:     verifiedRaw == "true",
			HeartbeatAge: time.Duration(envInt("ADMIN_NODE_HEARTBEAT_AGE_MS", 0)) * time.Millisecond,
		}
	}
	return p
}

func banner(fromTS, toTS int64, p event.ReplayProfile, nObs int) {
	fmt.Println("===========================================================================")
	fmt.Println("QuakeAlert — SAPUAN LANTAI ADMIN NODE (RISET, HANYA-BACA, DETEKSI-saja)")
	fmt.Println("===========================================================================")
	fmt.Println("Lantai produksi (AdminNodeMinPGAGal=140.0) TIDAK disentuh: alat ini membawa")
	fmt.Println("SALINAN gerbang dengan lantai sebagai parameter (D-007; 05-experiment §3).")
	fmt.Println("Salinan DIBUKTIKAN identik dengan produksi pada 140 sebelum sapuan dilaporkan.")
	fmt.Println("Menjawab DETEKSI saja — bukan latensi (F-04), bukan laju palsu (metric D).")
	fmt.Println("Parameter keputusan DIASSERSI OPERATOR, tidak dipulihkan dari baris (F2/F3).")
	fmt.Println("---------------------------------------------------------------------------")
	fmt.Printf("jendela   : %d .. %d  (%s .. %s UTC)\n", fromTS, toTS,
		time.UnixMilli(fromTS).UTC().Format(time.RFC3339), time.UnixMilli(toTS).UTC().Format(time.RFC3339))
	fmt.Printf("basis algo: %s\n", event.AlgoVerBase())
	fmt.Printf("observasi : %d baris\n", nObs)
	fmt.Printf("admin     : %s verified=%v heartbeat_age=%s (DIASSERSI OPERATOR)\n",
		p.AdminNode.StationID, p.AdminNode.Verified, p.AdminNode.HeartbeatAge)
	fmt.Printf("lantai    : %v gal\n", floors)
	fmt.Println("===========================================================================")
}

func mustInt(key string) int64 {
	v := os.Getenv(key)
	if v == "" {
		die("%s wajib diisi (ms epoch UTC)", key)
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		die("%s=%q bukan bilangan bulat", key, v)
	}
	return n
}

func envInt(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		die("%s=%q bukan bilangan bulat", key, v)
	}
	return n
}

func envFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		die("%s=%q bukan bilangan pecahan", key, v)
	}
	return f
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "admin_floor_sweep: "+format+"\n", args...)
	os.Exit(1)
}





