package id.web.quakealert.data.network.mapper

import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale
import kotlin.math.abs

/**
 * Display formatting shared by the event and alert mappers, so a quake rendered
 * from the REST history and the same quake rendered from a live WebSocket frame
 * produce byte-identical strings.
 *
 * Every function takes the display [locale] (default US = the long-standing
 * behaviour): the UI copy is rendered in the resolved [id.web.quakealert.domain.DisplayLanguage],
 * and a device set to another locale must not produce a card that mixes languages
 * mid-line. The *zone*, by contract, is the device's — timestamps cross the wire
 * as UTC and are converted only at the display boundary (docs/CLIENT_SPEC.md §7).
 *
 * Numbers keep [Locale.US] in every language: canonical units (gal, coordinates)
 * use a decimal dot by contract, and an Indonesian decimal comma there would be a
 * data defect, not a translation.
 */
internal object QuakeFormat {

    /**
     * Placeholder for a value the server contract does not carry.
     *
     * A hyphen rather than an em dash: this is printed inside composed rows
     * ("RSSI : - dBm"), and the app's user-visible copy holds no em dashes at all,
     * so a lone one here would be the only place the character appeared on screen.
     */
    const val UNAVAILABLE: String = "-"

    private fun dateFormatter(locale: Locale): DateTimeFormatter =
        DateTimeFormatter.ofPattern("dd MMM yyyy", locale)

    private fun timeFormatter(locale: Locale): DateTimeFormatter =
        DateTimeFormatter.ofPattern("HH:mm:ss zzz", locale)

    /** e.g. "20 Jun 2026" (EN) / "20 Jun 2026" (ID — same digits, local month). */
    fun date(instant: Instant, zone: ZoneId, locale: Locale = Locale.US): String =
        dateFormatter(locale).format(instant.atZone(zone))

    /** e.g. "07:19:18 WIB" — `zzz` renders the device zone's short name. */
    fun time(instant: Instant, zone: ZoneId, locale: Locale = Locale.US): String =
        timeFormatter(locale).format(instant.atZone(zone))

    /**
     * Send time inside a chat bubble, e.g. "09:41".
     *
     * Minutes only, and no zone suffix: a chat bubble is read in the room it was
     * sent to, so the seconds and the zone name that a quake read-out needs would
     * only be noise here.
     */
    fun chatTime(instant: Instant, zone: ZoneId): String =
        DateTimeFormatter.ofPattern("HH:mm", Locale.US).format(instant.atZone(zone))

    /** PGA in the canonical unit, e.g. "61.5 gal". Never converted to `g` here. */
    fun pga(pgaGal: Double): String = String.format(Locale.US, "%.1f gal", pgaGal)

    /**
     * How many stations reported the shaking, e.g. "3 stations" / "3 stasiun".
     *
     * Replaces the shaking duration in the detail overlay's third metric cell. The
     * REST contract carries no duration: the firmware does send `dur_ms`, and the
     * server range-checks and HMAC-signs it, but `earthquake_events` has no column
     * for it, so it is discarded after verification. Printing a permanent "-" in a
     * cell taught the user nothing, while the node count is a fact the response
     * already carries and is the one that says how much to trust the reading.
     *
     * Zero is [UNAVAILABLE] rather than "0 stations": an event exists because
     * stations triggered, so a zero here is a missing field, not a real count.
     *
     * Indonesian has no plural inflection, so the two English shapes collapse.
     */
    fun reportingNodes(count: Int, locale: Locale = Locale.US): String {
        if (count <= 0) return UNAVAILABLE
        val noun = if (isIndonesian(locale)) "stasiun" else "station"
        if (count == 1) return "1 $noun"
        return if (isIndonesian(locale)) "$count $noun" else "$count ${noun}s"
    }

    /**
     * Centroid as "-6.91750, 107.61910" — five decimals, matching the precision the
     * detail overlay was designed around (~1 m, well below the centroid's own error).
     */
    fun coordinates(latitude: Double, longitude: Double): String =
        String.format(Locale.US, "%.5f, %.5f", latitude, longitude)

    /**
     * Warning-banner intensity line, e.g. "Intensity : IV (moderate)" — the wording
     * the design uses on the banner (Figma 124:1297).
     *
     * Shared by the stored-event and realtime paths so the banner does not reword
     * itself when the WebSocket takes over from the REST seed. [fallbackWord] covers
     * a server that sent no `intensity_label`, so the line never trails an empty
     * bracket.
     */
    fun intensityBanner(
        mmi: String,
        label: String,
        fallbackWord: String,
        locale: Locale = Locale.US
    ): String =
        "${if (isIndonesian(locale)) "Intensitas" else "Intensity"} : ${intensityValue(mmi, label, fallbackWord, locale)}"

    /**
     * The bare intensity read, e.g. "IV (moderate)" / "IV (sedang)", for the active
     * alert card (Figma node 1:1067) — which renders "Estimated Intensity :" as its
     * own label line above the value and so must not carry the prefix.
     *
     * [intensityBanner] delegates here so the two screens can never drift into
     * spelling the same intensity differently. Server labels and the local
     * severity fallback share one fixed English vocabulary (light/moderate/strong),
     * mapped here so both read alike in Indonesian.
     */
    fun intensityValue(
        mmi: String,
        label: String,
        fallbackWord: String,
        locale: Locale = Locale.US
    ): String {
        val word = (label.takeIf { it.isNotBlank() } ?: fallbackWord)
            .let { if (isIndonesian(locale)) severityWordId(it) else it }
        val roman = mmi.takeIf { it.isNotBlank() } ?: UNAVAILABLE
        return "$roman (${word.lowercase(Locale.US)})"
    }

    /**
     * Fixed severity vocabulary (server labels and local bucket names) in
     * Indonesian. Unknown words pass through untouched rather than blanking a
     * reading the card needs.
     */
    private fun severityWordId(word: String): String = when (word.lowercase(Locale.US)) {
        "light" -> "ringan"
        "moderate" -> "sedang"
        "strong" -> "kuat"
        else -> word
    }

    /**
     * Coarse age of an event, e.g. "just now" / "baru saja", "20 minutes ago" /
     * "20 menit yang lalu".
     *
     * Deliberately coarse — the exact timestamp is one line away on the same card,
     * and rounding "89 seconds" to "a minute ago" is what makes the list scannable.
     * Future timestamps (device clock behind the server) collapse to "just now"
     * rather than rendering a negative age.
     */
    fun relativeTime(instant: Instant, now: Instant, locale: Locale = Locale.US): String {
        val seconds = now.epochSecond - instant.epochSecond
        if (seconds < MINUTE) return if (isIndonesian(locale)) "baru saja" else "just now"

        val (amount, unitEn, unitId) = when {
            seconds < HOUR -> Triple(seconds / MINUTE, "minute", "menit")
            seconds < DAY -> Triple(seconds / HOUR, "hour", "jam")
            seconds < MONTH -> Triple(seconds / DAY, "day", "hari")
            seconds < YEAR -> Triple(seconds / MONTH, "month", "bulan")
            else -> Triple(seconds / YEAR, "year", "tahun")
        }
        if (isIndonesian(locale)) return "$amount $unitId yang lalu"
        val plural = if (abs(amount) == 1L) "" else "s"
        return "$amount $unitEn$plural ago"
    }

    private fun isIndonesian(locale: Locale): Boolean = locale.language == "in"

    private const val MINUTE = 60L
    private const val HOUR = 60 * MINUTE
    private const val DAY = 24 * HOUR

    /** Calendar-average month/year lengths: this is a coarse label, not arithmetic. */
    private const val MONTH = 30 * DAY
    private const val YEAR = 365 * DAY
}
