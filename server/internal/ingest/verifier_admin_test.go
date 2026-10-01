// Copyright (c) 2026 QuakeAlert contributors.
// SPDX-License-Identifier: AGPL-3.0-or-later

package ingest

import (
	"context"
	"testing"

	"github.com/banana-pixel/quakealert/server/internal/store"
)

// Lencana admin menempel pada trigger yang lolos verifikasi (satu kolom dari
// baris node yang sama — tanpa I/O baru): tracker memakainya untuk tepi-PRELIM
// Admin (keputusan D-043).
func TestVerifyTrigger_AttachesAdminFlag(t *testing.T) {
	admin := &fakeNodeSource{node: &store.NodeSecret{
		StationID: "NODE-AD000001", IsActive: true, Verified: true, IsAdminNode: true,
	}}
	v := newTestVerifier(t, admin)
	tr, err := v.Verify(context.Background(),
		signedV2([]byte("test-key"), PhasePrelim, 196609, 1, 0.4215, v2DurPr, v2OnsetPr, 0, v2Now))
	if err != nil {
		t.Fatalf("PRELIM admin yang sah ditolak: %v", err)
	}
	if !tr.IsAdminNode {
		t.Error("lencana admin hilang pada trigger terverifikasi")
	}

	plain := &fakeNodeSource{node: &store.NodeSecret{
		StationID: "NODE-0A1B2C3D", IsActive: true, Verified: true,
	}}
	v2 := newTestVerifier(t, plain)
	tr2, err := v2.Verify(context.Background(),
		signedV2([]byte("test-key"), PhasePrelim, 196609, 1, 0.4215, v2DurPr, v2OnsetPr, 0, v2Now))
	if err != nil {
		t.Fatalf("PRELIM biasa yang sah ditolak: %v", err)
	}
	if tr2.IsAdminNode {
		t.Error("lencana admin dikarang untuk node biasa")
	}
}
