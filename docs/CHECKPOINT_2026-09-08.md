# Checkpoint 2026-09-08 — i18n (5b) + drill POCO

Tanggal: 2026-09-08. Cabang: `development` @ `b03dd3d` (dipush).

## Selesai
- **B1 plumbing bahasa**: `DisplayLanguage` + `resolveDisplayLanguage`
  (override > sistem > EN), param `lang` default-EN di semua fungsi copy,
  `QuakeFormat`, mapper, notifier, ViewModel. Teks EN byte-identik.
- **B2 isi Indonesia**: seluruh cabang ID beracuan BMKG/BPBD/string-sistem +
  glosarium; holder per layar; `values-in/strings.xml`; label channel;
  kata severity; 1 em dash user-visible dihapus.
- **B3 guard**: `IndonesianCopyTest` (glossary pins + sweep anti-em-dash),
  cek em-dash XML di CI, `Locale.isIndonesian()` terima ejaan `in`/`id`
  (bug nyata: `Locale("in").language == "id"` di JDK modern).
- **Kompilasi hijau + 327/327 test pass lokal** (`assembleDebug`,
  `testDebugUnitTest`, APK 86MB). 30+ error kompilasi + 2 ekspektasi test
  salah diperbaiki di perjalanan.
- **Lingkungan ARM64**: AGP 9.1.1 membundel aapt2 x86_64-only → build lokal
  wajib `-Pandroid.aapt2FromMavenOverride=/opt/android-sdk/build-tools/36.0.0/aapt2`
  (aapt2 SDK bersifat ARM64). Tanpa ini `processDebugResources` gagal
  menyesatkan ("Daemon startup failed"). `gradle.properties` dibatasi
  workers=1, Xmx2g (VPS 1vCPU anti-crash).
- **Fase 4 produksi**: skema 8→9 di VPS, `CORRELATION_WINDOW_MS=20000`,
  endpoint near-confirmed live honest-empty, bukti G4 diarsip
  (`docs/evidence/p4-deploy-000009/2026-09-07-production/`).

## Temuan perangkat (POCO F1, API 36, `id.web.quakealert.debug`)
- APK fresh terpasang (uninstall dulu: beda debug key). UI langsung
  **Indonesia penuh** — DataStore me-restore `language="id"` (pilihan pil
  ID lama yang dulu inert, kini hidup). Pipeline bahasa terbukti end-to-end.
- ADB via Tailscale `:5555` stabil setelah `adb tcpip 5555`; pairing port ≠
  connect port; toggle wireless debugging menghapus pairing.

## Blocker aktif
1. **Drill server tak sampai**: build tanpa `google-services.json` → FCM
   off ("WebSocket only"). Drill `test-alert` (topik FCM) takkan tiba.
2. **Badge Offline**: REST/WS gagal di `AuthInterceptor` (pesan error asli
   belum ketangkap). Perlu diagnosis sebelum drill WebSocket.
3. Tanpa toolchain rilis: keystore, `google-services.json` rilis, kredensial
   FCM server belum ada → AAB tertunda (Fase 7).

## Langkah selanjutnya
- **Agen**: (a) tangkap pesan error jaringan (logcat `-B`), (b) drill
  notifikasi lokal "Uji Notifikasi" (render heads-up + suara + log D-019,
  tanpa server), (c) tur screenshot tab ID, (d) arsip bukti B4.
- **Owner**: (a) sediakan `google-services.json` (debug + rilis) +
  kredensial FCM server untuk drill penuh, (b) siapkan keystore
  (`QUAKE_KEYSTORE_*`) untuk AAB, (c) nyatakan PASS drill (G-validasi),
  (d) putuskan nasib restore backup lama (data lama ikut pulih sekali
  meski `allowBackup=false`).
- Setelah itu: Fase 6 firmware (host test HMAC) + Fase 7 runbook/release
  notes → track tertutup → publik terbatas. Phase F tetap BLOCKED (1 node).
