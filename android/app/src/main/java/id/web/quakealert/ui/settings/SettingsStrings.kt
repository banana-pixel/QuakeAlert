package id.web.quakealert.ui.settings

import id.web.quakealert.domain.DisplayLanguage

/**
 * Every literal on the Settings screen, in one place.
 *
 * Built once per composition from the selected language so the screen body
 * stays layout: `strings.syncNow` reads the same in either language position.
 * Pure and unit-testable; Indonesian lives beside its English twin rather than
 * scattered through the layout. Acuan: string sistem Android ("Pengaturan",
 * "Notifikasi").
 */
data class SettingsStrings(
    val appBar: String,
    val sectionLocation: String,
    val syncNow: String,
    val autoSync: String,
    val sectionAlert: String,
    val alerts: String,
    val blockedPill: String,
    val testNotification: String,
    val testNotificationDetail: String,
    val testSound: String,
    val testSoundDetail: String,
    val showStatus: String,
    val showStatusDetail: String,
    val deliveryChecklist: String,
    val allSet: String,
    val readyCount: (ready: Int, total: Int) -> String,
    val sectionAccount: String,
    val anonymousProfile: String,
    val pseudonym: String,
    val userId: String,
    val rerolling: String,
    val reroll: String,
    val resetting: String,
    val resetProfile: String,
    val sectionAppearance: String,
    val lightMode: String,
    val comingSoon: String,
    val units: String,
    val language: String,
    val sectionAbout: String,
    val     disableTitle: String,
    val disableBody: String,
    val turnOff: String,
    val cancel: String,
    val resetTitle: String,
    val resetBody: String,
    val reset: String,
    val syncingLocation: String,
    val syncLocationNow: String,
    val permNotifications: String,
    val permLocation: String,
    val permBackground: String,
    val permFullscreen: String,
    val allowed: String,
    val unrestricted: String,
    val tapToAllow: String,
    val notSignedIn: String,
    val moreAboutUs: String
)

private fun settingsStringsEn(): SettingsStrings = SettingsStrings(
    appBar = "Settings",
    sectionLocation = "Location & Coverage",
    syncNow = "Sync Location Now",
    autoSync = "Auto Sync Location",
    sectionAlert = "Alert & Notification",
    alerts = "Earthquake Alerts",
    blockedPill = "Blocked by system settings",
    testNotification = "Test Earthquake Alert",
    testNotificationDetail = "Sounds the drill alert on this phone",
    testSound = "Test Alert Sound",
    testSoundDetail = "Plays the siren, to check it is loud enough to wake you",
    showStatus = "Show Status in Notification Shade",
    showStatusDetail = "A silent, ongoing summary of whether alerts can reach you",
    deliveryChecklist = "Delivery Checklist",
    allSet = "All set: alerts can reach you",
    readyCount = { ready, total -> "$ready of $total ready" },
    sectionAccount = "Account & Privacy",
    anonymousProfile = "Anonymous Profile",
    pseudonym = "Pseudonym",
    userId = "User ID",
    rerolling = "Rerolling…",
    reroll = "Reroll Pseudonym",
    resetting = "Resetting…",
    resetProfile = "Reset Profile",
    sectionAppearance = "Appearance & Look",
    lightMode = "Light Mode (Beta)",
    comingSoon = "Coming Soon",
    units = "Units",
    language = "Language",
    sectionAbout = "About",
    disableTitle = "Turn off earthquake warnings?",
    disableBody = "You won't receive earthquake warnings while this setting " +
        "is turned off.",
    turnOff = "Turn off",
    cancel = "Cancel",
    resetTitle = "Reset profile?",
    resetBody = "This creates a brand-new anonymous identity. Your current " +
        "pseudonym and user ID are discarded and cannot be restored. " +
        "Your location and alert registration are sent again under the " +
        "new identity.",
    reset = "Reset",
    syncingLocation = "Syncing location",
    syncLocationNow = "Sync location now",
    permNotifications = "Notifications",
    permLocation = "Precise Location",
    permBackground = "Background Delivery",
    permFullscreen = "Full-Screen Alerts",
    allowed = "Allowed",
    unrestricted = "Unrestricted",
    tapToAllow = "Tap to allow",
    notSignedIn = "Not signed in yet",
    moreAboutUs = "More About Us"
)

// Indonesian (B2). Acuan: string sistem Android + pedoman BPBD.
private fun settingsStringsId(): SettingsStrings = SettingsStrings(
    appBar = "Pengaturan",
    sectionLocation = "Lokasi & Cakupan",
    syncNow = "Sinkronkan Lokasi Sekarang",
    autoSync = "Sinkron Lokasi Otomatis",
    sectionAlert = "Peringatan & Notifikasi",
    alerts = "Peringatan Gempa Bumi",
    blockedPill = "Diblokir oleh pengaturan sistem",
    testNotification = "Uji Peringatan Gempa",
    testNotificationDetail = "Membunyikan peringatan latihan di ponsel ini",
    testSound = "Uji Suara Peringatan",
    testSoundDetail = "Bunyikan sirene, untuk memastikan cukup keras membangunkan Anda",
    showStatus = "Tampilkan Status di Panel Notifikasi",
    showStatusDetail = "Ringkasan senyap yang menunjukkan apakah peringatan bisa masuk ke ponsel Anda",
    deliveryChecklist = "Status Pengiriman",
    allSet = "Siap: peringatan bisa Anda terima",
    readyCount = { ready, total -> "$ready dari $total siap" },
    sectionAccount = "Akun & Privasi",
    anonymousProfile = "Profil Anonim",
    pseudonym = "Nama samaran",
    userId = "ID Pengguna",
    rerolling = "Mengacak…",
    reroll = "Acak Nama Samaran",
    resetting = "Menyetel ulang…",
    resetProfile = "Setel Ulang Profil",
    sectionAppearance = "Tampilan & Gaya",
    lightMode = "Mode Terang (Beta)",
    comingSoon = "Segera Hadir",
    units = "Satuan",
    language = "Bahasa",
    sectionAbout = "Tentang",
    disableTitle = "Matikan peringatan gempa bumi?",
    disableBody = "Anda tidak akan menerima peringatan gempa bumi selama " +
        "pengaturan ini dimatikan.",
    turnOff = "Matikan",
    cancel = "Batal",
    resetTitle = "Setel ulang profil?",
    resetBody = "Ini membuat identitas anonim yang benar-benar baru. Nama " +
        "samaran dan ID pengguna Anda saat ini dibuang dan tidak bisa " +
        "dipulihkan. Lokasi dan pendaftaran peringatan Anda akan dikirim ulang " +
        "dengan identitas baru.",
    reset = "Setel Ulang",
    syncingLocation = "Menyinkronkan lokasi",
    syncLocationNow = "Sinkronkan lokasi sekarang",
    permNotifications = "Notifikasi",
    permLocation = "Lokasi Presisi",
    permBackground = "Pengiriman Latar Belakang",
    permFullscreen = "Peringatan Layar Penuh",
    allowed = "Diizinkan",
    unrestricted = "Tanpa batas",
    tapToAllow = "Ketuk untuk mengizinkan",
    notSignedIn = "Belum masuk",
    moreAboutUs = "Selengkapnya Tentang Kami"
)

internal fun settingsStrings(lang: DisplayLanguage): SettingsStrings =
    if (lang == DisplayLanguage.ID) settingsStringsId() else settingsStringsEn()
