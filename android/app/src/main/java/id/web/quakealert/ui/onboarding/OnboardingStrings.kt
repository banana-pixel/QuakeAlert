package id.web.quakealert.ui.onboarding

import id.web.quakealert.domain.DisplayLanguage

/**
 * Every literal in onboarding, in one place.
 *
 * Onboarding runs before any preference exists, so there is no stored override
 * to read: the language resolves from the system locale (Indonesian devices
 * read Indonesian). Pure and unit-testable.
 */
data class OnboardingStrings(
    val back: String,
    val next: String,
    val getStarted: String,
    val start: String,
    val toastEnableFirst: String,
    val enableAlertsFirst: String,
    val testNotification: String,
    val testNotificationDetail: String,
    val testSound: String,
    val testSoundDetail: String,
    val tapToAllow: String,
    val allowed: String,
    val disabled: String,
    val readyPara1: String,
    val readyReport: String,
    val readyHere: String,
    val readySuffix: String,
    val readyBy: String
)

private fun onboardingStringsEn(): OnboardingStrings = OnboardingStrings(
    back = "Back",
    next = "Next",
    getStarted = "Get Started",
    start = "Start",
    toastEnableFirst = "Enable notifications first to test alerts.",
    enableAlertsFirst = "Turn on earthquake alerts to run the drill test.",
    testNotification = "Test Earthquake Alert",
    testNotificationDetail = "Sounds the drill alert on this phone",
    testSound = "Test Alert Sound",
    testSoundDetail = "Plays the siren, to check it is loud enough to wake you",
    tapToAllow = "Tap to allow",
    allowed = "Allowed",
    disabled = "Disabled",
    readyPara1 = "You will receive earthquake warning depends on sensor " +
        "availability in your area. If theres no sensors ready in your " +
        "area, this app wont be working. You can check for sensors " +
        "availibility on Sensors page.",
    readyReport = "Report bugs here ",
    readyHere = "GitHub",
    readySuffix = " if you find one!",
    readyBy = "by "
)

private fun onboardingStringsId(): OnboardingStrings = OnboardingStrings(
    back = "Kembali",
    next = "Lanjut",
    getStarted = "Mulai",
    start = "Mulai",
    toastEnableFirst = "Nyalakan notifikasi dulu untuk menguji peringatan.",
    enableAlertsFirst = "Nyalakan peringatan gempa untuk menjalankan uji latihan.",
    testNotification = "Uji Peringatan Gempa",
    testNotificationDetail = "Membunyikan peringatan latihan di ponsel ini",
    testSound = "Uji Suara Peringatan",
    testSoundDetail = "Bunyikan sirene, untuk memastikan cukup keras membangunkan Anda",
    tapToAllow = "Ketuk untuk mengizinkan",
    allowed = "Diizinkan",
    disabled = "Dimatikan",
    readyPara1 = "Anda akan menerima peringatan gempa bumi tergantung ketersediaan " +
        "sensor di daerah Anda. Jika belum ada sensor yang siap di daerah Anda, " +
        "aplikasi ini belum bisa bekerja. Anda dapat memeriksa sensor di halaman Sensor.",
    readyReport = "Laporkan bug di sini ",
    readyHere = "GitHub",
    readySuffix = " jika Anda menemukannya!",
    readyBy = "oleh "
)

internal fun onboardingStrings(lang: DisplayLanguage): OnboardingStrings =
    if (lang == DisplayLanguage.ID) onboardingStringsId() else onboardingStringsEn()
