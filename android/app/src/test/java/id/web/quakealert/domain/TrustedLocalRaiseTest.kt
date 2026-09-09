package id.web.quakealert.domain

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Raise-path behaviour specific to trusted-local frames, at the layers a JVM
 * test can reach (the three raise entry points need Android framework classes;
 * what they share — expiry, logging shape, drill fencing — is pure and pinned
 * here):
 *
 *  - D-018 sender validity applies to local frames through the shared
 *    [WsAlertMessage.isActionable]: an expired local frame stays down, a fresh
 *    one stays raisable;
 *  - D-019 log lines for the local gate reasons carry the event id and the
 *    reason name only — no coordinate, no distance, structurally (the factory
 *    takes only an id and an enum);
 *  - the local drill is untouched by D-036: it stays `isTest` and never
 *    `trustedLocal`.
 */
class TrustedLocalRaiseTest {

    private fun localMessage(
        ageMs: Long = 1_000L,
        validityMs: Long = 0L,
        nowMs: Long = 1_755_000_000_000L
    ): WsAlertMessage = WsAlertMessage(
        type = AlertType.EARTHQUAKE_ALERT,
        eventId = "evt-local-9",
        mmi = "V",
        intensityLabel = "moderate",
        pgaGal = 150.0,
        centroidLat = -6.9175,
        centroidLon = 107.6191,
        locationName = "Bandung, West Java",
        timestampMs = nowMs - ageMs,
        nodeCount = 1,
        trustedLocal = true,
        validityMs = validityMs
    )

    // --- D-018 validity on local frames -------------------------------------

    @Test
    fun `an expired local frame is not actionable`() {
        // Declared 90 s, arrived 120 s late: down, like any frame.
        assertFalse(localMessage(ageMs = 120_000L, validityMs = 90_000L).isActionable(1_755_000_000_000L))
    }

    @Test
    fun `a fresh local frame is actionable`() {
        assertTrue(localMessage(ageMs = 1_000L, validityMs = 90_000L).isActionable(1_755_000_000_000L))
    }

    @Test
    fun `a local frame without validity uses the legacy window`() {
        assertTrue(localMessage(ageMs = 60_000L).isActionable(1_755_000_000_000L))
        assertFalse(localMessage(ageMs = 16 * 60_000L).isActionable(1_755_000_000_000L))
    }

    // --- D-019 logging for the local gate ------------------------------------

    @Test
    fun `gated-out lines name the local reason and nothing else`() {
        for (reason in listOf(AlertGateReason.LOCAL_WITHIN_RADIUS, AlertGateReason.LOCAL_OUTSIDE_RADIUS)) {
            val line = RaiseOutcomeLog.gatedOut("evt-local-9", reason)
            assertTrue(line.contains("evt-local-9"))
            assertTrue(line.contains(reason.name))
        }
    }

    @Test
    fun `log lines cannot carry a position by construction`() {
        // The factory signature is (eventId, reason): there is no parameter a
        // coordinate could travel through. The assertion that matters is on the
        // shape — a fixed template around an opaque id and an enum name.
        val line = RaiseOutcomeLog.gatedOut("evt-local-9", AlertGateReason.LOCAL_OUTSIDE_RADIUS)
        assertEquals("raise evt-local-9 gated-out (LOCAL_OUTSIDE_RADIUS); not raising", line)
        assertFalse(line.contains("-6.9"))
        assertFalse(line.contains("107.6"))
    }

    // --- drill fence regression ----------------------------------------------

    @Test
    fun `the local drill stays a drill and never a local warning`() {
        val drill = LocalTestEvent.build(nowMs = 1_000_000L, userLocation = null)

        assertTrue(drill.isTest)
        assertFalse(drill.trustedLocal)
    }
}
