package id.web.quakealert.ui.common

import id.web.quakealert.domain.DisplayLanguage

/**
 * Every literal in the shared filter row + sheet, in one place.
 *
 * Built once per composition from the selected language. Enum option labels
 * live beside their entries ([QuakeIntensity.label], [QuakeTimeWindow.label],
 * [QuakeStationStatus.label]); this holder owns the chrome around them.
 */
data class FilterStrings(
    val title: String,
    val intensityGroup: String,
    val radiusGroup: String,
    val radiusNote: (alertKm: Int) -> String,
    val sensorsWithin: (maxLabel: String) -> String,
    val timeGroup: String,
    val stationGroup: String,
    val stationNote: String,
    val reset: String,
    val apply: String,
    val filter: String,
    val all: String,
    val near: String,
    val addSensor: String
)

private fun filterStringsEn(): FilterStrings = FilterStrings(
    title = "Filter",
    intensityGroup = "Shaking Intensity",
    radiusGroup = "Search Radius",
    radiusNote = { alertKm ->
        "Applies to the \"Near\" pill only. Emergency alerts always use " +
            "a fixed $alertKm km radius and cannot be changed."
    },
    sensorsWithin = { maxLabel ->
        "Sensors are listed within $maxLabel. " +
            "That is the furthest that tab can search."
    },
    timeGroup = "Time Range",
    stationGroup = "Station Status",
    stationNote = "Offline stations stay in the list unless you narrow it here.",
    reset = "Reset",
    apply = "Apply",
    filter = "Filter",
    all = "All",
    near = "Near",
    addSensor = "+ Add Sensor"
)

private fun filterStringsId(): FilterStrings = FilterStrings(
    title = "Filter",
    intensityGroup = "Intensitas Guncangan",
    radiusGroup = "Radius Pencarian",
    radiusNote = { alertKm ->
        "Hanya berlaku untuk filter \"Dekat\". Peringatan darurat selalu memakai " +
            "radius tetap $alertKm km dan tidak bisa diubah."
    },
    sensorsWithin = { maxLabel ->
        "Sensor ditampilkan dalam $maxLabel. " +
            "Itu jarak terjauh yang bisa dicari."
    },
    timeGroup = "Rentang Waktu",
    stationGroup = "Status Stasiun",
    stationNote = "Stasiun offline tetap di daftar kecuali Anda memfilternya di sini.",
    reset = "Atur Ulang",
    apply = "Terapkan",
    filter = "Filter",
    all = "Semua",
    near = "Dekat",
    addSensor = "+ Tambah Sensor"
)

internal fun filterStrings(lang: DisplayLanguage): FilterStrings =
    if (lang == DisplayLanguage.ID) filterStringsId() else filterStringsEn()
