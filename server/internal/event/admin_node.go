package event

import (
	"context"
	"time"

	"github.com/banana-pixel/quakealert/server/internal/dispatch"
)

// --- Kelayakan peringatan lokal Admin Node (D-036 PROPOSED) ---
//
// Lapisan kecil, murni, dan deterministik: dari status operator (designation +
// verifikasi + umur heartbeat, dimuat SEKALI oleh pemanggil) dan satu Snapshot,
// ia menjawab layak/tidak beserta alasannya. Tidak menyentuh basis data,
// tidak menyentuh jam, tidak mengubah state apa pun — sehingga seluruh
// matriks positif/negatif/batas dapat diuji tanpa Postgres.
//
// Ambangnya konstanta compile-time (D-007): ia menyertai biner yang membuat
// keputusan, dan operator yang dapat mengubahnya saat runtime dapat mengubah
// arti peringatan tanpa keputusan tercatat.
//
// Yang SENGAJA tidak ada di sini: perubahan klasifikasi (classify utuh),
// perubahan kuorum/independensi, state keenam, dan pembacaan heartbeat sebagai
// bukti kegempaan — kesegaran denyut murni gerbang kelayakan dispatch.
const (
	// AdminNodeMinPGAGal adalah PGA puncak (gal) minimum pada KONTRIBUSI node
	// admin sendiri agar memenuhi syarat: yang membenarkan peringatan lokal
	// adalah guncangan yang diukur node itu, bukan puncak event dari node
	// lain.
	AdminNodeMinPGAGal = 140.0

	// AdminNodeHeartbeatMaxAge adalah umur last_heartbeat maksimum agar
	// memenuhi syarat. Basi menahan peringatan lokal, tetapi TIDAK PERNAH
	// membuatnya — dan tidak mempengaruhi ingest/UNCONFIRMED normal sama
	// sekali.
	AdminNodeHeartbeatMaxAge = 5 * time.Minute
)

// Kosakata tertutup alasan keputusan: satu nilai per gerbang, dipakai log dan
// uji. Bukan teks bebas — penolakan yang tidak bernama tidak dapat diagregasi.
const (
	ReasonAdminNodeEligible       = "ADMIN_ELIGIBLE"
	ReasonAdminNodeNoneDesignated = "NO_ADMIN_DESIGNATED"
	ReasonAdminNodeUnverified     = "ADMIN_UNVERIFIED"
	ReasonAdminNodeNotUnconfirmed = "NOT_UNCONFIRMED"
	ReasonAdminNodeNoContribution = "NO_ADMIN_CONTRIBUTION"
	ReasonAdminNodeNotFinal       = "ADMIN_NOT_FINAL"
	ReasonAdminNodeBelowFloor     = "ADMIN_PGA_BELOW_FLOOR"
	ReasonAdminNodeHeartbeatStale = "ADMIN_HEARTBEAT_STALE"
)

// AdminNodeState adalah separuh operator dari kelayakan: designation,
// verifikasi, dan umur heartbeat pemegang is_admin_node, dibaca dalam satu
// query (store.GetAdminNodeStatus) agar tidak tercampur dua waktu baca.
// Zero value berarti tidak layak (tidak ditunjuk).
type AdminNodeState struct {
	Designated bool
	StationID  string
	Verified   bool
	// HeartbeatAge adalah NOW() - last_heartbeat saat dibaca; negatif sudah
	// dijepit pemuat menjadi nol.
	HeartbeatAge time.Duration
}

// AdminNodeSource memuat AdminNodeState dari luar paket event — produksi dari
// store lewat adaptor tipis di main, uji dari fake. Satu metode agar jahitannya
// sempit; ok=false berarti "tidak ada status tepercaya" (gagal-tertutup:
// tanpa itu tidak ada peringatan lokal).
type AdminNodeSource interface {
	GetAdminNodeState(ctx context.Context) (AdminNodeState, bool)
}

// AdminNodeSourceFunc mengadaptasi fungsi biasa menjadi AdminNodeSource,
// seperti http.HandlerFunc.
type AdminNodeSourceFunc func(ctx context.Context) (AdminNodeState, bool)

// GetAdminNodeState memenuhi AdminNodeSource.
func (f AdminNodeSourceFunc) GetAdminNodeState(ctx context.Context) (AdminNodeState, bool) {
	return f(ctx)
}

// AdminNodeDecision adalah hasil murni evaluasi: layak atau tidak, dan gerbang
// mana yang menentukan.
type AdminNodeDecision struct {
	Eligible bool
	Reason   string
}

// EvaluateAdminNodeEligibility menilai satu Snapshot terhadap status operator.
//
// Urutan gerbang (tercantum agar log forensik stabil): designation, verifikasi,
// state UNCONFIRMED, kontribusi milik admin, FINAL, lantai PGA, kesegaran
// heartbeat. HANYA transisi UNCONFIRMED yang dipertimbangkan — CONFIRMED,
// RESOLVED, CANCELLED, dan DETECTED tidak pernah menghasilkan frame lokal
// (CONFIRMED memakai jalur normalnya yang tidak berubah).
func EvaluateAdminNodeEligibility(state AdminNodeState, s Snapshot) AdminNodeDecision {
	switch {
	case !state.Designated:
		return AdminNodeDecision{Reason: ReasonAdminNodeNoneDesignated}
	case !state.Verified:
		return AdminNodeDecision{Reason: ReasonAdminNodeUnverified}
	case s.To != StateUnconfirmed:
		return AdminNodeDecision{Reason: ReasonAdminNodeNotUnconfirmed}
	}

	contrib := adminContribution(state.StationID, s.Evidence.Contributors)
	switch {
	case contrib == nil:
		return AdminNodeDecision{Reason: ReasonAdminNodeNoContribution}
	case contrib.Phase != PhaseFinal:
		return AdminNodeDecision{Reason: ReasonAdminNodeNotFinal}
	case contrib.PeakPGA < AdminNodeMinPGAGal:
		return AdminNodeDecision{Reason: ReasonAdminNodeBelowFloor}
	case state.HeartbeatAge > AdminNodeHeartbeatMaxAge:
		return AdminNodeDecision{Reason: ReasonAdminNodeHeartbeatStale}
	default:
		return AdminNodeDecision{Eligible: true, Reason: ReasonAdminNodeEligible}
	}
}

// adminContribution menemukan kontribusi milik node admin dalam potret bukti.
// Satu node menyumbang tepat satu entri (PRELIM dan FINAL satu node adalah
// satu kontributor), jadi kecocokan pertama sudah cukup.
func adminContribution(stationID string, cs []ContributorEvidence) *ContributorEvidence {
	for i := range cs {
		if cs[i].NodeID == stationID {
			return &cs[i]
		}
	}
	return nil
}

// TrustedLocalFrameFor membangun frame peringatan lokal dari satu Snapshot
// bila dan hanya bila memenuhi syarat: tipe tetap EARTHQUAKE_ALERT dengan
// trusted_local=true, event_id/revisi/timestamp seperti frame normal — tanpa
// state baru, tanpa enum FCM baru. ok=false berarti "jangan emisikan".
//
// Satu-satunya choke point konstruksi frame lokal: pemanggil tidak menilai
// sendiri, sehingga frame lokal tidak dapat lahir tanpa lolos evaluator.
func TrustedLocalFrameFor(state AdminNodeState, s Snapshot) (*dispatch.AlertMessage, bool) {
	if d := EvaluateAdminNodeEligibility(state, s); !d.Eligible {
		return nil, false
	}
	return &dispatch.AlertMessage{
		Type:                 dispatch.TypeAlert,
		EventID:              s.EventID,
		MMI:                  s.MMIScale,
		IntensityLabel:       s.IntensityLabel,
		PGAGal:               s.PeakPGA,
		CentroidLat:          s.CentroidLat,
		CentroidLon:          s.CentroidLon,
		LocationName:         s.LocationName,
		Timestamp:            s.DecidedAt,
		NodeCount:            s.NodeCount,
		EventState:           string(s.To),
		EventRevision:        s.Revision,
		OriginTS:             s.OriginTS,
		OriginTSSource:       s.OriginTSSource,
		IndependentCellCount: s.IndependentCells,
		TrustedLocal:         true,
	}, true
}

// adminNodeLoadTimeout membatasi pembacaan status operator pada jalur emisi
// lokal. Deadline sendiri (bukan ctx pemanggil) agar goroutine yatim tidak
// menumpuk saat store macet — dan orang tua yang dibatalkan tidak ikut
// membatalkan peringatan yang sedang dievaluasi.
const adminNodeLoadTimeout = 2 * time.Second

// emitTrustedLocal mengevaluasi kelayakan Admin Node untuk satu Snapshot dan
// mendispatch frame lokal bila layak. Aditif penuh terhadap emisi normal di
// EmitTransition: filter To==UNCONFIRMED dan ketiadaan hook tidak menyentuh
// I/O sama sekali, dan seluruh I/O kelayakan berjalan di goroutine sendiri —
// basis data yang lambat hanya menunda frame lokal, tidak pernah peringatan
// normal (S1).
func (b *Bridge) emitTrustedLocal(ctx context.Context, s Snapshot) {
	if s.To != StateUnconfirmed {
		return
	}
	src, sink := b.adminSource, b.adminSink
	if src == nil || sink == nil {
		return
	}
	go func() {
		tctx, cancel := context.WithTimeout(context.Background(), adminNodeLoadTimeout)
		defer cancel()
		state, ok := src.GetAdminNodeState(tctx)
		if !ok {
			return // gagal-tertutup: tanpa status tepercaya, tanpa frame lokal
		}
		msg, ok := TrustedLocalFrameFor(state, s)
		if !ok {
			return
		}
		sink.DispatchTrustedLocalEventFrame(tctx, msg)
	}()
}
