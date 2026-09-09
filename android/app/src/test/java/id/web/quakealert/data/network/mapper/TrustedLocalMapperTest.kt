package id.web.quakealert.data.network.mapper

import id.web.quakealert.data.network.model.WsAlertMessageDto
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * `trusted_local` parsing on both transports (D-036).
 *
 * The contract carries the flag as the string `"true"`/`"false"` on FCM
 * (every data value is a string there) and as a native boolean on the socket.
 * Both converge on [WsAlertMessageDto.trustedLocal], defaulting to false: a
 * frame that says nothing is an ordinary warning, never a local one.
 */
class TrustedLocalMapperTest {

    private fun alertPayload(): Map<String, String> = mapOf(
        "type" to "EARTHQUAKE_ALERT",
        "event_id" to "evt-local-1",
        "mmi" to "V",
        "intensity_label" to "moderate",
        "pga_gal" to "150.0",
        "centroid_lat" to "-6.9175",
        "centroid_lon" to "107.6191",
        "location_name" to "Bandung, West Java",
        "timestamp" to "1755000000000"
    )

    @Test
    fun `absent trusted_local parses as an ordinary warning`() {
        val message = alertPayload().toWsAlertMessageOrNull(nowMs = 1_755_000_000_000L)

        requireNotNull(message)
        assertFalse(message.trustedLocal)
    }

    @Test
    fun `explicit false parses as an ordinary warning`() {
        val message = alertPayload()
            .plus("trusted_local" to "false")
            .toWsAlertMessageOrNull(nowMs = 1_755_000_000_000L)

        requireNotNull(message)
        assertFalse(message.trustedLocal)
    }

    @Test
    fun `exact true string marks a local warning`() {
        val message = alertPayload()
            .plus("trusted_local" to "true")
            .toWsAlertMessageOrNull(nowMs = 1_755_000_000_000L)

        requireNotNull(message)
        assertTrue(message.trustedLocal)
        // Still an alert of the frozen type enum — the flag refines, it does
        // not retype.
        assertEquals("EARTHQUAKE_ALERT", message.type.name)
    }

    @Test
    fun `near-miss spellings never claim local trust`() {
        // Surrounding whitespace is trimmed first — the same rule as is_test —
        // so only the token itself is judged here.
        for (spelling in listOf("TRUE", "True", "1", "yes", "trust")) {
            val message = alertPayload()
                .plus("trusted_local" to spelling)
                .toWsAlertMessageOrNull(nowMs = 1_755_000_000_000L)

            requireNotNull(message)
            assertFalse("spelling \"$spelling\" must read as ordinary", message.trustedLocal)
        }
    }

    @Test
    fun `socket DTO defaults to false and carries an explicit true`() {
        assertFalse(WsAlertMessageDto(type = "EARTHQUAKE_ALERT").trustedLocal)

        val kept = WsAlertMessageDto(type = "EARTHQUAKE_ALERT", trustedLocal = true)
            .toDomainOrNull(allowTestAlerts = true)

        requireNotNull(kept)
        assertTrue(kept.trustedLocal)
    }

    @Test
    fun `a trusted flag on an advisory does not change its type`() {
        // The server never sends this combination; if it ever arrives, the
        // advisory drop upstream still sees an advisory. The flag refines the
        // alert kind, it never promotes one.
        val message = alertPayload()
            .plus("type" to "EARTHQUAKE_ADVISORY")
            .plus("event_id" to "")
            .plus("trusted_local" to "true")
            .toWsAlertMessageOrNull(nowMs = 1_755_000_000_000L)

        requireNotNull(message)
        assertEquals("EARTHQUAKE_ADVISORY", message.type.name)
        assertTrue(message.trustedLocal)
    }
}
