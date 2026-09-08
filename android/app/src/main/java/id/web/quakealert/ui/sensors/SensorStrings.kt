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
    val loading: String
)

private fun sensorStringsEn(): SensorStrings = SensorStrings(
    appBar = "Sensors",
    loading = "Scanning the sensor network..."
)

private fun sensorStringsId(): SensorStrings = SensorStrings(
    appBar = "Sensor",
    loading = "Memindai jaringan sensor..."
)

internal fun sensorStrings(lang: DisplayLanguage): SensorStrings =
    if (lang == DisplayLanguage.ID) sensorStringsId() else sensorStringsEn()
