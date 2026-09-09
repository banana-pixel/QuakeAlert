package id.web.quakealert.domain

import id.web.quakealert.ui.warning.WarningActivity
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Stand-down isolation for the local drill test.
 *
 * Ending a test must clear exactly the synthetic event — never a live server
 * event — and ending a non-existent test must be a no-op. The board is pure
 * JVM, so this is asserted directly; the notifier's cancel call takes the ids
 * returned here.
 */
class LocalTestStandDownTest {

    @Test
    fun `stand-down clears exactly the test event`() {
        val board = ActiveAlertBoard()
        val testId = "${LocalTestEvent.ID_PREFIX}123"
        board.upsert(testId, sounded = true)
        board.upsert("server-evt-1", sounded = true)

        val result = board.standDown(testId)

        assertTrue(result.removedAny)
        assertTrue(result.cancelledNotificationIds.isNotEmpty())
        assertEquals("server-evt-1", board.selectedId())
    }

    @Test
    fun `stand-down for an unknown test id removes nothing`() {
        val board = ActiveAlertBoard()
        board.upsert("server-evt-1", sounded = true)

        val result = board.standDown("${LocalTestEvent.ID_PREFIX}999")

        assertFalse(result.removedAny)
        assertTrue(result.cancelledNotificationIds.isEmpty())
        assertEquals("server-evt-1", board.selectedId())
    }

    @Test
    fun `safety timeout is pinned to one minute`() {
        // The timer itself lives in the Activity (JVM-untestable); what is
        // pinned here is the contract the manual drill verifies: an un-ended
        // test cleans itself up after 60 s through the same clear path above.
        assertEquals(60_000L, WarningActivity.TEST_AUTO_END_MS)
    }
}
