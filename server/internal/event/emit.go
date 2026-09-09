package event

import (
	"context"

	"github.com/banana-pixel/quakealert/server/internal/dispatch"
)

// frameSink adalah satu-satunya hal yang dibutuhkan paket ini dari dispatch:
// kirimkan frame ini, dorong FCM atau tidak. Implementasi: *dispatch.Dispatcher.
type frameSink interface {
	DispatchEventFrame(ctx context.Context, msg *dispatch.AlertMessage, push bool)
}

// Bridge adalah emitter yang menerjemahkan Snapshot menjadi frame klien (§8.3).
//
// Terpisah dari Tracker supaya Tracker tetap dapat diuji tanpa dispatch sama
// sekali, dan terpisah dari dispatch supaya dispatch tidak perlu tahu apa pun
// tentang state machine — arah impornya hanya satu, event -> dispatch.
type Bridge struct {
	sink frameSink
	// adminSource + adminSink adalah kait peringatan lokal Admin Node (D-036
	// PROPOSED), dipasang lewat SetAdminNodeHook. Nilai nol berarti
	// nonaktif: EmitTransition berperilaku persis seperti sebelum kait ada.
	adminSource AdminNodeSource
	adminSink   trustedLocalSink
}

// trustedLocalSink adalah satu-satunya yang dibutuhkan kait admin dari
// dispatch: kirimkan frame lokal ini lewat jalur token-only. Implementasi:
// *dispatch.Dispatcher.
type trustedLocalSink interface {
	DispatchTrustedLocalEventFrame(ctx context.Context, msg *dispatch.AlertMessage)
}

// SetAdminNodeHook memasang jalur peringatan lokal: sumber status operator
// dan sink dispatch lokal. Dipisahkan dari konstruktor seperti SetEmitter —
// dispatcher dan sumber dibangun pada titik yang berbeda di main.go — dan
// keduanya harus dipasang bersama: setengah kait tidak pernah berguna.
func (b *Bridge) SetAdminNodeHook(src AdminNodeSource, sink trustedLocalSink) {
	if b == nil {
		return
	}
	b.adminSource = src
	b.adminSink = sink
}

// NewBridge membuat emitter di atas sink. sink nil menghasilkan Bridge yang
// membuang setiap frame; dipakai bila FCM maupun Hub belum terpasang.
func NewBridge(sink frameSink) *Bridge { return &Bridge{sink: sink} }

// EmitTransition memenuhi antarmuka emitter.
func (b *Bridge) EmitTransition(ctx context.Context, s Snapshot) {
	if b == nil || b.sink == nil {
		return
	}
	msg, push, ok := FrameFor(s)
	if !ok {
		return
	}
	b.sink.DispatchEventFrame(ctx, msg, push)
	// Kait Admin Node (D-036): aditif setelah emisi normal — tidak mengubah
	// frame di atas, tidak menahannya, dan tidak membaca I/O apa pun secara
	// sinkron (lihat emitTrustedLocal di admin_node.go).
	b.emitTrustedLocal(ctx, s)
}

// FrameFor memetakan satu transisi ke frame yang diumumkannya, mengikuti tabel
// §8.1 baris per baris. ok bernilai false berarti transisi ini tidak diumumkan
// sama sekali.
//
// Nilai type TIDAK bertambah (D11): UNCONFIRMED memakai EARTHQUAKE_ADVISORY yang
// sudah dipahami setiap klien terpasang, dan CANCELLED memakai EVENT_RESOLVED
// yang setiap klien terpasang sudah menangani sebagai all-clear. Klien yang belum
// diperbarui karenanya MEMBERSIHKAN alarmnya pada sebuah penarikan walau ia tidak
// dapat mengatakan MENGAPA — degradasi yang aman, dan alasan D11 berjalan seperti
// ini.
func FrameFor(s Snapshot) (msg *dispatch.AlertMessage, push, ok bool) {
	var alertType string
	switch s.To {
	case StateUnconfirmed:
		// WebSocket saja (D10). Pengguna latar TIDAK dibangunkan untuk dugaan
		// berkeyakinan rendah: kanal alarm FCM hanya membawa bukti multi-stasiun
		// yang terkonfirmasi dan penarikannya.
		alertType, push = dispatch.TypeAdvisory, false
	case StateConfirmed:
		alertType, push = dispatch.TypeAlert, true
	case StateResolved, StateCancelled:
		// All-clear diutangkan TEPAT kepada audiens yang menerima alarmnya. Sebuah
		// event yang tidak pernah CONFIRMED tidak pernah membangunkan siapa pun,
		// jadi tidak ada siapa pun untuk ditenangkan (§8.1).
		alertType, push = dispatch.TypeResolved, s.EverConfirmed
	default:
		// -> DETECTED tidak mengumumkan apa pun: ia di bawah lantai PGA dan bukan
		// klaim tentang apa pun. Dalam praktiknya transisi ini tidak pernah ada
		// (D -> D bukan transisi), dan cabang ini memastikan tidak ada jalan bagi
		// sebuah state di bawah lantai untuk sampai ke klien walau nanti ada.
		return nil, false, false
	}

	return &dispatch.AlertMessage{
		Type:           alertType,
		EventID:        s.EventID,
		MMI:            s.MMIScale,
		IntensityLabel: s.IntensityLabel,
		PGAGal:         s.PeakPGA,
		CentroidLat:    s.CentroidLat,
		CentroidLon:    s.CentroidLon,
		LocationName:   s.LocationName,
		// timestamp adalah waktu KEPUTUSAN, bukan waktu onset: ia yang dipakai
		// klien untuk menilai kebaruan sebuah frame, dan origin_ts dibawa
		// terpisah justru supaya keduanya tidak lagi dikonflasikan.
		Timestamp:            s.DecidedAt,
		NodeCount:            s.NodeCount,
		EventState:           string(s.To),
		EventRevision:        s.Revision,
		OriginTS:             s.OriginTS,
		OriginTSSource:       s.OriginTSSource,
		IndependentCellCount: s.IndependentCells,
	}, push, true
}
