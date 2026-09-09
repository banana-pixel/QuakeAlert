# Referensi Logo QuakeAlert

Sumber: file dari owner via Telegram/SFTP, 2026-09-09. Disimpan verbatim (tidak diubah).

## quakealerticon.svg — DITERIMA (varian merah, background opaque)
- Inkscape 512x512, `docname` asli `quakealertlogored.svg`.
- Visual: persegi full-bleed sudut tajam, gradien diagonal marun gelap (`#2c0c23`) ke merah (`#82283a`); motif putih di tengah: kotak rounded-outline (chip) dengan 8 pin konektor di keempat sisi + garis waveform diagonal dengan ekor kaca pembesar; semua stroke putih 30px round cap.

## quakealerticontr.svg — DITERIMA ("tr" = background transparan)
- Inkscape 512x512, `docname` asli `quakealertlogored.svg`; tanpa rect background (`fill:none`) — transparan terkonfirmasi.
- Motif yang sama tanpa background: untuk About, onboarding, dan foreground launcher.

## Catatan konversi ke VectorDrawable (berlaku dua file)
- Sudut tajam aman (mask launcher + `clip(RoundedCornerShape)` in-app yang membulatkan).
- Stroke/round-cap/gradien linear didukung; grup `rotate`/`translate` perlu dicek ulang saat impor via Asset Studio.
- Varian transparan = kandidat foreground launcher + About/onboarding; varian merah = referensi warna + fallback background.

## Kompatibilitas (dicek 2026-09-09 thd spek Play + adaptive icon)
- Play listing (`quakealerticon.svg` → ekspor PNG): KOMPATIBEL. Kanvas 512x512 full-bleed sudut tajam (Play yang me-masking 30% + bayangan); motif tengah 88-423px (~65%) jauh dari zona potong sudut. Tinggal ekspor 32-bit PNG sRGB ≤1024KB.
- Adaptive launcher (foreground = varian transparan): motif ~65% lebar ≈ 70dp dari 108dp → pas di safe zone 72dp, margin ±18dp. KOMPATIBEL.
- Small icon notifikasi (varian transparan): BISA secara teknis (putih-di-transparan = alpha mask yang tepat) tetapi TIDAK DISARANKAN langsung — 8 pin + waveform terlalu detail untuk 24dp (stroke ~1.4dp). Wajib turunan siluet sederhana (mis. waveform saja, stroke tebal). Large icon boleh pakai varian transparan.

## Ekspor referensi (`exports/`, dibuat 2026-09-09 — bukan asset aplikasi)
- `play-listing-512.png` (512 RGB 26KB): siap upload Play (≤1024KB).
- `adaptive-foreground-432.png` (RGBA): foreground launcher referensi.
- Keputusan owner 2026-09-09: logo dipakai sesuai SVG — bentuk/motif tidak boleh diubah menjadi bentuk lain. Adjustment minor (sizing, stroke size) diizinkan. Draft siluet A/B dibuang. Small icon notifikasi = varian transparan utuh.

## Rencana pakai (menunggu nomor D)
1. Launcher: foreground (varian transparan) + background + monochrome (`mipmap-anydpi`), ganti placeholder robot/grid hijau.
2. About: ganti `AboutLogoBadge` (`AboutModal.kt`) dengan logo + `clip(RoundedCornerShape)`.
3. Onboarding: ganti/kombinasi `ic_puzzle_piece` (`OnboardingScreen.kt:491`).
