package id.web.quakealert.ui.addsensor

import id.web.quakealert.domain.DisplayLanguage

/**
 * Every literal in the add-a-sensor wizard dialog, in one place.
 *
 * Built once per composition from the selected language; bodies take the
 * holder instead of raw strings so layout code never names copy. Pure and
 * unit-testable. Indonesian (B2) follows string-sistem Android ("Pengaturan")
 * and pedoman BPBD untuk instruksi.
 */
data class AddSensorStrings(
    val title: String,
    val start: String,
    val back: String,
    val next: String,
    val finish: String,
    val exit: String,
    val stepBadge: (AddSensorWizardStep) -> String?,
    val processing: String,
    val configuredPending: String,
    val online: String,
    val     stationId: String,
    val secretsTitle: String,
    val checkNow: String,
    val rateLimitedBody: String,
    val welcomeTitle: String,
    val welcomeBody: String,
    val detectedCity: String,
    val tapToEnter: String,
    val yourCoords: String,
    val notAvailable: String,
    val showSecret: String,
    val copy: String,
    val copied: String,
    val networksDetected: String,
    val noneFound: String,
    val rescan: String,
    val wlanPassword: String,
    val enterPassword: String,
    val chosen: String,
    val choose: String,
    val     discardTitle: String,
    val discardRevoke: String,
    val discardPlain: String,
    val discard: String,
    val keepGoing: String,
    val findingLocation: String,
    val tapSyncHint: String,
    val syncingLocation: String,
    val syncLocationNow: String
)

private fun addSensorStringsEn(): AddSensorStrings = AddSensorStrings(
    title = "Add a Sensor",
    start = "Start",
    back = "Back",
    next = "Next",
    finish = "Finish",
    exit = "Exit",
    stepBadge = { step ->
        when (step) {
            AddSensorWizardStep.WELCOME -> null
            AddSensorWizardStep.LOCATION -> "Step 1"
            AddSensorWizardStep.CREDENTIALS -> "Step 2"
            AddSensorWizardStep.WLAN -> "Step 3"
            AddSensorWizardStep.FINISHING -> "Step 4"
            AddSensorWizardStep.RATE_LIMIT -> "Error"
        }
    },
    processing = "Processing, please hang tight...",
    configuredPending = "Configured. Your sensor is awaiting verification.",
    online = "Your sensor is online.",
    stationId = "Station ID",
    secretsTitle = "Provisioning Secrets",
    checkNow = "Check Now",
    rateLimitedBody = "You have added sensors as often as the network allows for now. " +
        "Try again in a few hours.",
    welcomeTitle = "Welcome to QuakeAlert Sensor Wizard!",
    welcomeBody = "You are going to add a new device to the QuakeAlert Network, for " +
        "further info you can visit Sensor Guide.\n\nWhen you are ready, start " +
        "the sensor addition process with the button below.",
    detectedCity = "Detected City Name :",
    tapToEnter = "Tap to enter a place name",
    yourCoords = "Your Current Coordinates :",
    notAvailable = "Not available",
    showSecret = "Show secret",
    copy = "Copy",
    copied = "Copied",
    networksDetected = "Networks Detected by Sensor :",
    noneFound = "None found yet. Rescan once the sensor has finished starting up.",
    rescan = "Rescan",
    wlanPassword = "WLAN Password (empty if open network) :",
    enterPassword = "Enter password...",
    chosen = "Chosen",
    choose = "Choose",
    discardTitle = "Discard sensor setup?",
    discardRevoke = "Leaving now cancels this sensor's registration and discards the " +
        "credentials on screen, which can never be shown again.",
    discardPlain = "This setup is not finished. Leaving now discards it, and the " +
        "credentials on screen cannot be shown again.",
    discard = "Discard",
    keepGoing = "Keep going",
    findingLocation = "Finding your location...",
    tapSyncHint = "Tap sync to put your location on the map.",
    syncingLocation = "Syncing location",
    syncLocationNow = "Sync location now"
)

private fun addSensorStringsId(): AddSensorStrings = AddSensorStrings(
    title = "Tambah Sensor",
    start = "Mulai",
    back = "Kembali",
    next = "Lanjut",
    finish = "Selesai",
    exit = "Keluar",
    stepBadge = { step ->
        when (step) {
            AddSensorWizardStep.WELCOME -> null
            AddSensorWizardStep.LOCATION -> "Langkah 1"
            AddSensorWizardStep.CREDENTIALS -> "Langkah 2"
            AddSensorWizardStep.WLAN -> "Langkah 3"
            AddSensorWizardStep.FINISHING -> "Langkah 4"
            AddSensorWizardStep.RATE_LIMIT -> "Error"
        }
    },
    processing = "Memproses, mohon tunggu...",
    configuredPending = "Terkonfigurasi. Sensor Anda menunggu verifikasi.",
    online = "Sensor Anda online.",
    stationId = "ID stasiun",
    secretsTitle = "Kredensial sensor",
    checkNow = "Periksa Sekarang",
    rateLimitedBody = "Anda sudah menambah sensor sesering yang diizinkan jaringan untuk saat ini. " +
        "Coba lagi dalam beberapa jam.",
    welcomeTitle = "Selamat datang di Penyiapan Sensor QuakeAlert!",
    welcomeBody = "Anda akan menambahkan perangkat baru ke jaringan QuakeAlert, untuk " +
        "info lebih lanjut Anda dapat mengunjungi Panduan Sensor.\n\nSetelah siap, " +
        "mulai proses penambahan sensor dengan tombol di bawah.",
    detectedCity = "Nama Kota Terdeteksi :",
    tapToEnter = "Ketuk untuk memasukkan nama tempat",
    yourCoords = "Koordinat Anda Saat Ini :",
    notAvailable = "Tidak tersedia",
    showSecret = "Tampilkan secret",
    copy = "Salin",
    copied = "Tersalin",
    networksDetected = "Jaringan Terdeteksi oleh Sensor :",
    noneFound = "Belum ada yang ditemukan. Pindai lagi setelah sensor selesai menyala.",
    rescan = "Pindai Ulang",
    wlanPassword = "Kata Sandi WLAN (kosongkan bila jaringan terbuka) :",
    enterPassword = "Masukkan kata sandi...",
    chosen = "Dipilih",
    choose = "Pilih",
    discardTitle = "Buang penyiapan sensor?",
    discardRevoke = "Pergi sekarang membatalkan pendaftaran sensor ini dan membuang " +
        "kredensial di layar, yang tidak pernah dapat ditampilkan lagi.",
    discardPlain = "Penyiapan ini belum selesai. Pergi sekarang membuangnya, dan " +
        "kredensial di layar tidak dapat ditampilkan lagi.",
    discard = "Buang",
    keepGoing = "Lanjutkan",
    findingLocation = "Mencari lokasi Anda...",
    tapSyncHint = "Ketuk sinkron untuk menaruh lokasi Anda di peta.",
    syncingLocation = "Menyinkronkan lokasi",
    syncLocationNow = "Sinkronkan lokasi sekarang"
)

internal fun addSensorStrings(lang: DisplayLanguage): AddSensorStrings =
    if (lang == DisplayLanguage.ID) addSensorStringsId() else addSensorStringsEn()
