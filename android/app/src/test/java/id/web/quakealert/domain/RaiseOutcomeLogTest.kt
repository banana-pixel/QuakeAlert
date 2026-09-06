package id.web.quakealert.domain

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Raise-path log wording (D-019, U-013): every outcome carries the opaque
 * `event_id` and nothing position-derived. These tests pin both halves —
 * presence of the id and outcome, and absence of anything shaped like a
 * coordinate or a precise distance — because the privacy property is the
 * point of the decision, not a comment beside it.
 *
 * What is NOT tested here: that call sites actually call these functions.
 * That is verified by code inspection of the four call sites
 * (`WarningViewModel.raiseAlert` gated-out branch, `raise`,
 * `onAlertReceived` expiry branch, `WarningNotifier.notify` success) plus the
 * drill rig, since logcat output is not assertable from a JVM test.
 */
class RaiseOutcomeLogTest {

    private val id = "8f804561-1558-45ad-8982-1ab9193be589"

    /** No decimal-degree coordinate pattern may appear in any line. */
    private fun assertNoPosition(line: String) {
        assertFalse(
            "log line must not carry coordinates or precise distance: $line",
            Regex("-?\\d+\\.\\d+").containsMatchIn(line)
        )
    }

    @Test
    fun `shown carries id and siren outcome`() {
        val started = RaiseOutcomeLog.shown(id, sirenStarted = true)
        val muted = RaiseOutcomeLog.shown(id, sirenStarted = false)

        assertTrue(started.contains(id))
        assertTrue(muted.contains(id))
        assertNoPosition(started)
        assertNoPosition(muted)
    }

    @Test
    fun `gated-out carries id and gate reason`() {
        val line = RaiseOutcomeLog.gatedOut(id, AlertGateReason.OUTSIDE_RADIUS)

        assertTrue(line.contains(id))
        assertTrue(line.contains("OUTSIDE_RADIUS"))
        assertNoPosition(line)
    }

    @Test
    fun `duplicate-suppressed carries id`() {
        val line = RaiseOutcomeLog.duplicateSuppressed(id)

        assertTrue(line.contains(id))
        assertNoPosition(line)
    }

    @Test
    fun `notification posted carries id`() {
        val line = RaiseOutcomeLog.notificationPosted(id)

        assertTrue(line.contains(id))
        assertNoPosition(line)
    }

    @Test
    fun `expired distinguishes declared validity from legacy window`() {
        val declared = RaiseOutcomeLog.expired(id, validityDeclared = true)
        val legacy = RaiseOutcomeLog.expired(id, validityDeclared = false)

        assertTrue(declared.contains(id))
        assertTrue(legacy.contains(id))
        assertTrue(declared != legacy)
        assertNoPosition(declared)
        assertNoPosition(legacy)
    }
}
