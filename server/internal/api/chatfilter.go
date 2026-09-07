package api

import (
	"strings"
	"time"
	"unicode"
)

// chatDuplicateWindow adalah jeda di mana pesan yang SAMA dari user yang SAMA
// ditolak sebagai spam. Beda dari chatSendWindow (2s, laju) — ini soal isi:
// menekan "kirim" dua kali atau menahan tombol enter. Lima menit cukup lama
// untuk menghentikan banjir, cukup singkat untuk tidak menghukum pengulangan
// yang sah (mis. menanyakan hal yang sama sejam kemudian).
const chatDuplicateWindow = 5 * time.Minute

// blockedWords adalah daftar kata kasar yang ditolak server (Indonesia +
// Inggris, dikurasi minimal — bukan sensor moral, melainkan rem ruang publik
// darurat). Cocok per kata utuh setelah lowercase, jadi "menggoblokkan" atau
// "Scunthorpe" tidak kena. Daftar ini closed-world by design: kata baru
// ditambah lewat review, bukan lewat laporan otomatis.
var blockedWords = map[string]struct{}{
	"anjing": {}, "bangsat": {}, "bajingan": {}, "tolol": {},
	"goblok": {}, "bego": {}, "dungu": {}, "babi": {},
	"monyet": {}, "kontol": {}, "memek": {}, "ngentot": {},
	"asu": {}, "jancuk": {}, "jancok": {}, "kampret": {},
	"sialan": {}, "idiot": {},
	"fuck": {}, "fucking": {}, "shit": {}, "bitch": {},
	"asshole": {}, "bastard": {}, "dick": {}, "pussy": {},
	"whore": {}, "slut": {},
}

// containsBlockedWord melaporkan apakah body memuat kata daftar-blokir sebagai
// kata utuh (case-insensitive). Pemisah kata = semua yang bukan huruf,
// sehingga tanda baca atau angka di sekitar kata tidak menyembunyikannya,
// tetapi imbuhan (awalan/akhiran) tidak memicu.
func containsBlockedWord(body string) bool {
	for _, w := range splitWords(strings.ToLower(body)) {
		if _, blocked := blockedWords[w]; blocked {
			return true
		}
	}
	return false
}

func splitWords(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
}

// isDuplicateSpam melaporkan apakah body sama persis (setelah trim) dengan
// pesan terakhir pengirim dalam chatDuplicateWindow. Perbandingan case-sensitive
// yang disengaja: "TOLONG" dan "tolong" adalah penekanan berbeda, bukan spam.
func isDuplicateSpam(body string, lastBody string, lastAt time.Time, now time.Time) bool {
	if body != lastBody {
		return false
	}
	return now.Sub(lastAt) < chatDuplicateWindow
}
