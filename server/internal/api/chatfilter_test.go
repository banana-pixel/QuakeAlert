package api

import (
	"testing"
	"time"
)

func TestContainsBlockedWord(t *testing.T) {
	clean := []string{
		"gempa di mana?",
		"tolong, butuh bantuan di sini",
		"Scunthorpe adalah kota di Inggris",
		"menggoblokkan lawan tidak baik",
		"",
	}
	for _, body := range clean {
		if containsBlockedWord(body) {
			t.Errorf("bersih diblokir: %q", body)
		}
	}
	dirty := []string{
		"dasar TOLOL!",
		"Anjing, kencang sekali",
		"...goblok...",
		"kamu fuck banget",
		"what the SHIT is this",
	}
	for _, body := range dirty {
		if !containsBlockedWord(body) {
			t.Errorf("kotor lolos: %q", body)
		}
	}
}

func TestIsDuplicateSpam(t *testing.T) {
	now := time.Now()
	if !isDuplicateSpam("halo", "halo", now.Add(-time.Minute), now) {
		t.Error("sama dalam jendela harus spam")
	}
	if isDuplicateSpam("halo", "halo", now.Add(-10*time.Minute), now) {
		t.Error("sama di luar jendela bukan spam")
	}
	if isDuplicateSpam("halo", "halo?", now.Add(-time.Minute), now) {
		t.Error("beda isi bukan spam")
	}
	if isDuplicateSpam("HALO", "halo", now.Add(-time.Minute), now) {
		t.Error("beda kapitalisasi bukan spam (penekanan, bukan banjir)")
	}
}
