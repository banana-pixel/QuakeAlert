package id.web.quakealert.ui.sensors

import id.web.quakealert.domain.DisplayLanguage

/**
 * Every literal on the Sensors screen, in one place.
 *
 * Built once per composition from the selected language. Pure and
 * unit-testable.
 */
data class SensorStrings(
    val appBar: String,
    val loading: String,
    /**
     * Operator-designation badge for the Admin Node (D-036). A proper noun,
     * identical in both languages — like "GitHub" in the About links — so no
     * translation is owned here, only the single source.
     */
    val adminNodeBadge: String
)

private fun sensorStringsEn(): SensorStrings = SensorStrings(
    appBar = "Sensors",
    loading = "Scanning the sensor network...",
    adminNodeBadge = "Admin Node"
)

private fun sensorStringsId(): SensorStrings = SensorStrings(
    appBar = "Sensor",
    loading = "Memindai jaringan sensor...",
    adminNodeBadge = "Admin Node"
)

internal fun sensorStrings(lang: DisplayLanguage): SensorStrings =
    if (lang == DisplayLanguage.ID) sensorStringsId() else sensorStringsEn()
