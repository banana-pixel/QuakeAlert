package id.web.quakealert.ui.warning

import id.web.quakealert.domain.DisplayLanguage

/**
 * Every literal on the Warning screen and its overlays, in one place.
 *
 * Built once per composition from the selected language. Pure and
 * unit-testable. Safety-critical wording (drop-cover-hold, all-clear) lives in
 * the tested copy functions, not here — this holder owns chrome copy only.
 */
data class WarningStrings(
    val lang: DisplayLanguage,
    val appBar: String,
    val seeDetails: String,
    val retryButton: String,
    val emergencyCta: String,
    val offlineMessage: String,
    val cardTitle: String,
    val within: (radius: String, days: Int) -> String,
    val confirmedEvents: String,
    val mostRecent: String,
    val strongest: String,
    val disclaimer: String,
    val drillBadge: String,
    val localWarningTitle: String,
    val localWarningBadge: String,
    val localWarningSource: String,
    val alertTitle: String,
    val estimatedIntensity: String,
    val suggestedActions: String,
    val moreActive: (count: Int) -> String,
    val soundOn: String,
    val muteAlert: String,
    val endTest: String,
    val sosOn: String,
    val sosLight: String,
    val noLight: String,
    val statusTitle: String,
    val automatic: String,
    val badgeBlocked: String,
    val badgeOff: String,
    val badgeActive: String,
    val ruleRadiusTitle: (radius: String) -> String,
    val ruleRadiusDetail: String,
    val ruleSevereTitle: String,
    val ruleSevereDetail: (pgaGal: Int) -> String,
    val cannotDeliver: String,
    val reEnable: String,
    val     cannotDeliverDetail: String,
    val reEnableDetail: String,
    val detailTitle: String,
    val checkingNetwork: String,
    val noGuidance: String,
    val noGuidanceSub: String,
    val resetFilters: String
)

private fun warningStringsEn(): WarningStrings = WarningStrings(
    lang = DisplayLanguage.EN,
    appBar = "Warning",
    seeDetails = "SEE DETAILS",
    retryButton = "RETRY",
    emergencyCta = "EMERGENCY STEPS & CONTACTS",
    offlineMessage = "Offline: alerts are paused. The guidance below works without a connection.",
    cardTitle = "Recent Seismic Activity",
    within = { radius, days -> "Within $radius, past $days days" },
    confirmedEvents = "Confirmed Events",
    mostRecent = "Most Recent",
    strongest = "Strongest Shaking",
    disclaimer = "Counts come from QuakeAlert's own stations and depend on how many are near you. " +
        "They describe shaking already recorded. They are not a forecast of what comes next.",
    drillBadge = "TEST - DRILL, NOT A REAL EARTHQUAKE",
    localWarningTitle = "Local Warning",
    localWarningBadge = "LOCAL WARNING - SINGLE STATION REPORT",
    localWarningSource = "Admin Node",
    alertTitle = "Earthquake Alert",
    estimatedIntensity = "Estimated Intensity :",
    suggestedActions = "Suggested Actions :",
    moreActive = { count -> "+$count more active" },
    soundOn = "SOUND ON",
    muteAlert = "MUTE ALERT",
    endTest = "END TEST",
    sosOn = "SOS ON",
    sosLight = "SOS LIGHT",
    noLight = "NO LIGHT",
    statusTitle = "Protection Status",
    automatic = "Automatic",
    badgeBlocked = "Blocked",
    badgeOff = "Turned off",
    badgeActive = "Active",
    ruleRadiusTitle = { radius -> "Alerts within $radius" },
    ruleRadiusDetail = "Any earthquake whose estimated centroid falls inside this " +
        "distance sounds the alarm. The radius is set by the system, the " +
        "same value the server uses to choose who to notify.",
    ruleSevereTitle = "Severe quakes ignore distance",
    ruleSevereDetail = { pgaGal ->
        "MMI VII and above, or peak ground acceleration of " +
            "$pgaGal gal or more, alarms " +
            "wherever you are. At that size there is no distance at which you " +
            "did not need to know."
    },
    cannotDeliver = "Warnings cannot be delivered",
    reEnable = "Re-enable earthquake warnings in Settings.",
    cannotDeliverDetail = "Notifications are blocked in system settings. Allow them for " +
        "QuakeAlert so warnings can reach your screen.",
    reEnableDetail = "The earthquake-warnings switch is on the Settings screen. " +
        "Turning it back on restores protection immediately.",
    detailTitle = "Recent Earthquake",
    checkingNetwork = "Checking the alert network...",
    noGuidance = "No Guidance Available",
    noGuidanceSub = "Preparedness guidance for your area will appear here.",
    resetFilters = "Reset Filters"
)

private fun warningStringsId(): WarningStrings = WarningStrings(
    lang = DisplayLanguage.ID,
    appBar = "Peringatan",
    seeDetails = "LIHAT DETAIL",
    retryButton = "COBA LAGI",
    emergencyCta = "LANGKAH DARURAT & KONTAK",
    offlineMessage = "Offline: peringatan dijeda. Panduan di bawah tetap berfungsi tanpa koneksi.",
    cardTitle = "Aktivitas Seismik Terkini",
    within = { radius, days -> "Dalam $radius, $days hari terakhir" },
    confirmedEvents = "Kejadian Terkonfirmasi",
    mostRecent = "Terbaru",
    strongest = "Guncangan Terkuat",
    disclaimer = "Data berasal dari stasiun QuakeAlert sendiri dan bergantung pada berapa banyak yang dekat dengan Anda. " +
        "Data tersebut menggambarkan guncangan yang sudah tercatat. Bukan prediksi kejadian berikutnya.",
    drillBadge = "UJI - LATIHAN, BUKAN GEMPA SEBENARNYA",
    localWarningTitle = "Peringatan Lokal",
    localWarningBadge = "PERINGATAN LOKAL - LAPORAN SATU STASIUN",
    localWarningSource = "Admin Node",
    alertTitle = "Peringatan Gempa Bumi",
    estimatedIntensity = "Estimasi Intensitas :",
    suggestedActions = "Tindakan yang Disarankan :",
    moreActive = { count -> "+$count aktif lainnya" },
    soundOn = "SUARA AKTIF",
    muteAlert = "MATIKAN SUARA",
    endTest = "AKHIRI TES",
    sosOn = "SOS AKTIF",
    sosLight = "LAMPU SOS",
    noLight = "TANPA LAMPU",
    statusTitle = "Status Perlindungan",
    automatic = "Otomatis",
    badgeBlocked = "Diblokir",
    badgeOff = "Dimatikan",
    badgeActive = "Aktif",
    ruleRadiusTitle = { radius -> "Peringatan dalam $radius" },
    ruleRadiusDetail = "Setiap gempa yang centroid estimasinya jatuh di dalam jarak " +
        "ini akan membunyikan alarm. Radius diatur oleh sistem, nilai yang sama dipakai " +
        "server untuk memilih siapa yang diberi tahu.",
    ruleSevereTitle = "Gempa kuat mengabaikan jarak",
    ruleSevereDetail = { pgaGal ->
        "MMI VII ke atas, atau percepatan tanah puncak sebesar " +
            "$pgaGal gal atau lebih, alarm akan berbunyi di " +
            "mana pun Anda berada. Pada kekuatan sebesar itu, Anda perlu tahu di mana pun Anda berada."
    },
    cannotDeliver = "Peringatan tidak bisa ditampilkan",
    reEnable = "Nyalakan lagi peringatan gempa di Pengaturan.",
    cannotDeliverDetail = "Notifikasi diblokir di pengaturan sistem. Izinkan untuk " +
        "QuakeAlert agar peringatan bisa masuk ke layar Anda.",
    reEnableDetail = "Saklar peringatan gempa ada di layar Pengaturan. " +
        "Menyalakannya lagi segera memulihkan perlindungan.",
    detailTitle = "Gempa Terkini",
    checkingNetwork = "Memeriksa jaringan peringatan...",
    noGuidance = "Belum Ada Panduan",
    noGuidanceSub = "Panduan kesiapsiagaan untuk daerah Anda akan muncul di sini.",
    resetFilters = "Atur Ulang Filter"
)

internal fun warningStrings(lang: DisplayLanguage): WarningStrings =
    if (lang == DisplayLanguage.ID) warningStringsId() else warningStringsEn()
