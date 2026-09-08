package id.web.quakealert.ui.common

import id.web.quakealert.domain.DisplayLanguage

/**
 * Copy for the shared loading/error/empty state cards, in one place.
 *
 * Built once per composition from the selected language. Pure and
 * unit-testable.
 */
data class StateStrings(
    val retry: String,
    val resetFilters: String,
    val noHistory: String,
    val noHistorySub: String,
    val noDataAvailable: String,
    val noDataFiltered: (summary: String) -> String,
    val noCoverage: String,
    val noCoverageSub: String,
    val widenRadius: String,
    val orAddSensor: String,
    val noPosition: String,
    val noPositionSub: String,
    val syncLocation: String,
    val noStationsMatch: String
)

private fun stateStringsEn(): StateStrings = StateStrings(
    retry = "Retry",
    resetFilters = "Reset Filters",
    noHistory = "No Earthquake History",
    noHistorySub = "Events detected by the sensor network will appear here.",
    noDataAvailable = "No Data Available",
    noDataFiltered = { summary -> "No events $summary. Try a wider filter." },
    noCoverage = "No Sensors In This Area",
    noCoverageSub = "QuakeAlert's sensor network does not cover this area yet.",
    widenRadius = "Widen Search Radius",
    orAddSensor = "Or Add Your Own Sensor",
    noPosition = "Location Not Synced",
    noPositionSub = "QuakeAlert needs your location before it can show what is near you.",
    syncLocation = "Sync Location",
    noStationsMatch = "No Stations Match"
)

private fun stateStringsId(): StateStrings = StateStrings(
    retry = "Coba Lagi",
    resetFilters = "Atur Ulang Filter",
    noHistory = "Belum Ada Riwayat Gempa Bumi",
    noHistorySub = "Kejadian yang terdeteksi jaringan sensor akan muncul di sini.",
    noDataAvailable = "Tidak Ada Data",
    noDataFiltered = { summary -> "Tidak ada kejadian $summary. Coba filter yang lebih luas." },
    noCoverage = "Tidak Ada Sensor Di Daerah Ini",
    noCoverageSub = "Jaringan sensor QuakeAlert belum mencakup daerah ini.",
    widenRadius = "Perluas Radius Pencarian",
    orAddSensor = "Atau Tambah Sensor Anda Sendiri",
    noPosition = "Lokasi Belum Tersinkron",
    noPositionSub = "QuakeAlert membutuhkan lokasi Anda sebelum dapat menunjukkan apa yang dekat.",
    syncLocation = "Sinkronkan Lokasi",
    noStationsMatch = "Tidak Ada Stasiun yang Cocok"
)

internal fun stateStrings(lang: DisplayLanguage): StateStrings =
    if (lang == DisplayLanguage.ID) stateStringsId() else stateStringsEn()
