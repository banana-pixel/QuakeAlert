package id.web.quakealert.domain

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Sender-declared validity (D-018, U-010): [WsAlertMessage.isActionable] honours
 * `validityMs` when the server stated one and falls back to the legacy
 * [WsAlertMessage.isRecent] window otherwise. Absence is legacy, never expired.
 *
 * Scenario coverage required by the decision: old/new mixed frames, process
 * death with redelivery, clock skew, expired-frame suppression, and the
 * absent-field legacy fallback. Stand-down (`EVENT_RESOLVED`) expiry bypass
 * lives at the three call sites (push, foreground socket, background bridge),
 * which return before consulting recency at all — asserted by inspection of
 * those call sites, since a unit test cannot observe a branch it never takes.
 */
class ValidityWindowTest {

    private fun alert(
        timestampMs: Long,
        validityMs: Long = 0L
    ) = WsAlertMessage(
        type = AlertType.EARTHQUAKE_ALERT,
        eventId = "evt-1",
        mmi = "V",
        intensityLabel = "strong",
        pgaGal = 48.5,
        centroidLat = -6.9175,
        centroidLon = 107.6191,
        locationName = "Bandung, West Java",
        timestampMs = timestampMs,
        nodeCount = 3,
        validityMs = validityMs
    )

    @Test
    fun `fresh frame with declared validity is actionable`() {
        val t0 = 1_755_000_000_000L
        assertTrue(alert(timestampMs = t0, validityMs = 90_000L).isActionable(nowMs = t0 + 30_000L))
    }

    @Test
    fun `frame older than its declared validity is never raised`() {
        val t0 = 1_755_000_000_000L
        val expired = alert(timestampMs = t0, validityMs = 90_000L)
        assertFalse(expired.isActionable(nowMs = t0 + 90_001L))
        // Boundary is inclusive: exactly at validity it is still the event's.
        assertTrue(expired.isActionable(nowMs = t0 + 90_000L))
    }

    @Test
    fun `absent validity falls back to the legacy recent window`() {
        val t0 = 1_755_000_000_000L
        val legacy = alert(timestampMs = t0)
        assertTrue(legacy.isActionable(nowMs = t0 + 5 * 60_000L))
        assertFalse(legacy.isActionable(nowMs = t0 + 60 * 60_000L))
    }

    @Test
    fun `mixed old and new frames share one rule without special cases`() {
        val t0 = 1_755_000_000_000L
        val nowMs = t0 + 120_000L
        // Old server, 2 minutes old: inside the legacy 15-minute window.
        assertTrue(alert(timestampMs = t0).isActionable(nowMs = nowMs))
        // New server, 90 s validity: expired at 2 minutes.
        assertFalse(alert(timestampMs = t0, validityMs = 90_000L).isActionable(nowMs = nowMs))
        // New server, drill-length 20 s validity: long expired.
        assertFalse(alert(timestampMs = t0, validityMs = 20_000L).isActionable(nowMs = nowMs))
    }

    @Test
    fun `device clock behind the server never discards a live alert`() {
        val t0 = 1_755_000_000_000L
        val skew = t0 - 30_000L
        assertTrue(alert(timestampMs = t0, validityMs = 90_000L).isActionable(nowMs = skew))
        assertTrue(alert(timestampMs = t0).isActionable(nowMs = skew))
    }

    @Test
    fun `redelivery after process death obeys validity, not dedup memory`() {
        // Fresh AlertDedup (process was killed): nothing remembered. An expired
        // frame must still be suppressed — by validity, which survives death
        // because it travels on the frame — while an absent-field frame of the
        // same age falls back to the legacy window.
        val t0 = 1_755_000_000_000L
        val nowMs = t0 + 120_000L
        val freshDedup = AlertDedup()
        val expired = alert(timestampMs = t0, validityMs = 90_000L)
        val legacy = alert(timestampMs = t0)

        // Simulate the raise-path order: validity first, dedup second.
        val expiredRaised = expired.isActionable(nowMs = nowMs) &&
            freshDedup.markIfNew(expired)
        val legacyRaised = legacy.isActionable(nowMs = nowMs) &&
            freshDedup.markIfNew(legacy)

        assertFalse(expiredRaised)
        assertTrue(legacyRaised)
    }
}
