package id.web.quakealert.ui.warning

import id.web.quakealert.domain.DisplayLanguage

/**
 * Every literal in the "Emergency Steps & Contacts" overlay, in one place.
 *
 * Built once per composition from the selected language, like every other
 * screen holder. This overlay used to read its copy from string resources
 * (values and values-in), which follow the *system* locale — so switching
 * the app to Indonesian while the phone stayed English left exactly this
 * screen untranslated. Pure and unit-testable.
 *
 * Acuan ID: booklet BMKG-JICA ("Lindungi diri Anda!", "Lindungi Kepala") dan
 * Buku Saku BPBD. Nomor teleponnya sendiri tetap data di
 * [id.web.quakealert.domain.EmergencyContacts], bukan copy.
 */
data class EmergencyStrings(
    val title: String,
    val duringTitle: String,
    val dropTitle: String,
    val dropDetail: String,
    val coverTitle: String,
    val coverDetail: String,
    val holdTitle: String,
    val holdDetail: String,
    val afterTitle: String,
    val aftershocks: String,
    val gas: String,
    val exit: String,
    val injuries: String,
    val contactsTitle: String,
    val contactsNote: String,
    val dialAction: (number: String) -> String,
    val positionTitle: String,
    val positionNote: String,
    val positionUnknown: String,
    val offlineNote: String
)

private fun emergencyStringsEn(): EmergencyStrings = EmergencyStrings(
    title = "Emergency Steps & Contacts",
    duringTitle = "While the ground is shaking",
    dropTitle = "1. Drop",
    dropDetail = "Get down on your hands and knees before the shaking knocks you down. " +
        "Stay where you are. Most injuries happen to people moving between rooms or running outside.",
    coverTitle = "2. Cover",
    coverDetail = "Cover your head and neck with one arm. Get under a sturdy table if one is " +
        "within reach, or against an interior wall away from windows, mirrors and anything tall.",
    holdTitle = "3. Hold on",
    holdDetail = "Hold on to your shelter and stay put until the shaking stops. If you are in " +
        "bed, stay there and cover your head with a pillow.",
    afterTitle = "Once the shaking stops",
    aftershocks = "Expect aftershocks. The next one can arrive within seconds and can be strong " +
        "enough to finish what the first one started, so keep clear of anything already damaged.",
    gas = "Check for gas, water and electrical damage. If you smell gas, shut the valve, open " +
        "a window and do not touch light switches.",
    exit = "Leave only when it is safe to move, using stairs and never a lift. Check your exit " +
        "route before you commit to it.",
    injuries = "Help anyone near you who cannot move themselves before you leave the building.",
    contactsTitle = "Emergency numbers",
    contactsNote = "112 works on any GSM network, with or without a SIM card, and is routed to " +
        "the local emergency service. The numbers below it are the ones for the country this " +
        "phone is currently connected to.",
    dialAction = { number -> "Dial $number" },
    positionTitle = "Your last known position",
    positionNote = "Read these coordinates out to a dispatcher if you cannot describe where you are.",
    positionUnknown = "No position has synced yet, so there are no coordinates to read out. " +
        "Describe a landmark or a street instead.",
    offlineNote = "Everything on this screen is stored on the phone and needs no connection."
)

private fun emergencyStringsId(): EmergencyStrings = EmergencyStrings(
    title = "Langkah Darurat & Kontak",
    duringTitle = "Saat tanah berguncang",
    dropTitle = "1. Menunduk",
    dropDetail = "Segera menunduk ke lantai sebelum guncangan menjatuhkan Anda. Tetap di tempat. " +
        "Kebanyakan cedera terjadi pada orang yang berpindah ruangan atau lari ke luar.",
    coverTitle = "2. Lindungi Kepala",
    coverDetail = "Lindungi kepala dan leher dengan satu lengan. Berlindung di bawah meja yang " +
        "kokoh bila terjangkau, atau di dekat dinding dalam yang jauh dari jendela, cermin, " +
        "dan benda tinggi.",
    holdTitle = "3. Berpegangan",
    holdDetail = "Berpeganglah pada tempat berlindung dan tetap di sana sampai guncangan berhenti. " +
        "Jika Anda di tempat tidur, tetap di sana dan lindungi kepala dengan bantal.",
    afterTitle = "Setelah guncangan berhenti",
    aftershocks = "Waspadai gempa susulan. Gempa berikutnya bisa tiba dalam hitungan detik dan " +
        "cukup kuat untuk merobohkan yang sudah rusak, jadi jauhi apa pun yang sudah rusak.",
    gas = "Periksa kerusakan gas, air, dan listrik. Jika mencium bau gas, tutup katup, buka " +
        "jendela, dan jangan sentuh saklar lampu.",
    exit = "Keluar hanya bila sudah aman bergerak, lewat tangga dan jangan gunakan lift. " +
        "Periksa rute keluar sebelum Anda melewatinya.",
    injuries = "Bantu siapa pun di dekat Anda yang tidak bisa bergerak sendiri sebelum Anda " +
        "meninggalkan gedung.",
    contactsTitle = "Nomor darurat",
    contactsNote = "112 berfungsi di semua jaringan GSM, dengan atau tanpa kartu SIM, dan " +
        "diteruskan ke layanan darurat setempat. Nomor di bawahnya adalah nomor untuk negara " +
        "tempat ponsel ini terhubung.",
    dialAction = { number -> "Hubungi $number" },
    positionTitle = "Posisi terakhir Anda",
    positionNote = "Bacakan koordinat ini kepada petugas bila Anda tidak bisa menjelaskan di mana " +
        "Anda berada.",
    positionUnknown = "Belum ada posisi yang tersinkronkan, jadi tidak ada koordinat untuk dibacakan. " +
        "Sebutkan patokan atau nama jalan sebagai gantinya.",
    offlineNote = "Semua di layar ini tersimpan di ponsel dan tidak butuh koneksi."
)

internal fun emergencyStrings(lang: DisplayLanguage): EmergencyStrings =
    if (lang == DisplayLanguage.ID) emergencyStringsId() else emergencyStringsEn()
