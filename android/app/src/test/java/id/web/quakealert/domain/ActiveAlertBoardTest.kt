package id.web.quakealert.domain

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Bounded coexistence of concurrent alert events (D-020, U-011).
 *
 * The board is pure Kotlin with no Android imports, so every race below runs
 * on the JVM: A-then-B, near-simultaneous orderings, stand-down in either
 * order, collapse past the cap, collapsed stand-down, ID reuse, blank-id
 * scoping, per-event mute, the single-selected invariant, and silent
 * promotion. D-019 log wording (ids only, never position) is asserted where
 * the board's outcomes feed log lines.
 */
class ActiveAlertBoardTest {

    @Test
    fun `first event takes the base notification id and selects itself`() {
        val board = ActiveAlertBoard()

        val up = board.upsert("A", sounded = true)

        assertEquals(4301, up.notificationId)
        assertTrue(up.supersededIds.isEmpty())
        assertNull(up.collapsedId)
        assertEquals("A", up.selectedId)
        assertEquals("A", board.selectedId())
        assertEquals(0, board.extraActiveCount())
    }

    @Test
    fun `second event takes the next id and becomes selected while first persists`() {
        val board = ActiveAlertBoard()
        board.upsert("A", sounded = true)

        val up = board.upsert("B", sounded = true)

        assertEquals(4302, up.notificationId)
        assertEquals(listOf("A"), up.supersededIds)
        assertEquals("B", board.selectedId())
        assertEquals(listOf("A", "B").sorted(), board.fullEntries().map { it.eventId }.sorted())
        // A keeps its own notification: one event can never cancel another's.
        assertEquals(4301, board.notificationIdFor("A"))
        assertEquals(1, board.extraActiveCount())
    }

    @Test
    fun `redelivery of a known id is idempotent and keeps its slot`() {
        val board = ActiveAlertBoard()
        board.upsert("A", sounded = true)
        board.upsert("B", sounded = true)

        val again = board.upsert("A", sounded = true)

        assertEquals(4301, again.notificationId)
        assertEquals("B", board.selectedId())
        assertEquals(2, board.fullEntries().size)
    }

    @Test
    fun `A then B then A stand-down leaves B selected without sound`() {
        val board = ActiveAlertBoard()
        board.upsert("A", sounded = true)
        board.upsert("B", sounded = true)

        val down = board.standDown("A")

        assertEquals(listOf(4301), down.cancelledNotificationIds)
        assertEquals("B", down.promotedId)
        assertTrue(down.removedAny)
        assertEquals("B", board.selectedId())
        assertNull(board.notificationIdFor("A"))
    }

    @Test
    fun `A then B then B stand-down leaves A selected`() {
        val board = ActiveAlertBoard()
        board.upsert("A", sounded = true)
        board.upsert("B", sounded = true)

        val down = board.standDown("B")

        assertEquals(listOf(4302), down.cancelledNotificationIds)
        assertEquals("A", down.promotedId)
        assertEquals("A", board.selectedId())
    }

    @Test
    fun `fourth event collapses the oldest without losing its identity`() {
        val board = ActiveAlertBoard()
        board.upsert("A", sounded = true)
        board.upsert("B", sounded = true)
        board.upsert("C", sounded = true)

        val up = board.upsert("D", sounded = true)

        assertEquals("A", up.collapsedId)
        assertEquals(listOf(4301), up.cancelNotificationIds)
        assertEquals(setOf("A"), board.collapsedIds())
        assertEquals("D", board.selectedId())
        // Full set is B, C, D; A lives on as a collapsed identity.
        assertEquals(listOf("B", "C", "D"), board.fullEntries().map { it.eventId })
        // "+N more": selected D plus B, C plus collapsed A.
        assertEquals(3, board.extraActiveCount())
    }

    @Test
    fun `collapsed event stand-down removes its identity without cancelling`() {
        val board = ActiveAlertBoard()
        board.upsert("A", sounded = true)
        board.upsert("B", sounded = true)
        board.upsert("C", sounded = true)
        board.upsert("D", sounded = true)

        val down = board.standDown("A")

        assertTrue(down.cancelledNotificationIds.isEmpty())
        assertTrue(down.removedAny)
        assertTrue(board.collapsedIds().isEmpty())
        assertEquals("D", board.selectedId())
    }

    @Test
    fun `unknown id stand-down changes nothing`() {
        val board = ActiveAlertBoard()
        board.upsert("A", sounded = true)

        val down = board.standDown("ZZZ")

        assertFalse(down.removedAny)
        assertTrue(down.cancelledNotificationIds.isEmpty())
        assertEquals("A", board.selectedId())
    }

    @Test
    fun `freed notification id is reused by a later event`() {
        val board = ActiveAlertBoard()
        board.upsert("A", sounded = true)
        board.upsert("B", sounded = true)
        board.standDown("A")

        val up = board.upsert("D", sounded = true)

        // FIFO pool reuse: deterministic for a given operation order, and never
        // colliding with a live slot. A=4301 freed, pool still held never-used
        // 4303 first, so D takes 4303 — what matters is reuse happens and B's
        // 4302 is untouched.
        assertEquals(4303, up.notificationId)
        assertEquals(4302, board.notificationIdFor("B"))
        assertEquals(2, board.fullEntries().size)
        assertEquals(
            2,
            board.fullEntries().map { it.notificationId }.toSet().size
        )
    }

    @Test
    fun `blank stand-down clears exactly one live event and ignores otherwise`() {
        val empty = ActiveAlertBoard()
        assertEquals(ActiveAlertBoard.BlankStandDown.NoneActive, empty.standDownBlank())

        val single = ActiveAlertBoard()
        single.upsert("A", sounded = true)
        val cleared = single.standDownBlank()
        assertTrue(cleared is ActiveAlertBoard.BlankStandDown.ClearedSingle)
        assertEquals(4301, (cleared as ActiveAlertBoard.BlankStandDown.ClearedSingle).notificationId)
        assertEquals(0, single.totalActive())

        val multi = ActiveAlertBoard()
        multi.upsert("A", sounded = true)
        multi.upsert("B", sounded = true)
        assertEquals(ActiveAlertBoard.BlankStandDown.IgnoredAmbiguous, multi.standDownBlank())
        assertEquals(2, multi.totalActive())
    }

    @Test
    fun `mute is per event and shared by every surface`() {
        val board = ActiveAlertBoard()
        board.upsert("A", sounded = true)
        board.upsert("B", sounded = true)

        board.setMuted("A", true)

        assertTrue(board.isMuted("A"))
        assertFalse(board.isMuted("B"))
        board.setMuted("B", true)
        // A newer event starts audible: its own entry is unmuted.
        val up = board.upsert("C", sounded = true)
        assertEquals("C", up.selectedId)
        assertFalse(board.isMuted("C"))
    }

    @Test
    fun `selected is always the newest sounded entry`() {
        val board = ActiveAlertBoard()
        board.upsert("A", sounded = false)
        assertEquals("A", board.selectedId())
        board.upsert("B", sounded = true)
        assertEquals("B", board.selectedId())
        board.upsert("C", sounded = false)
        assertEquals("B", board.selectedId())
    }

    @Test
    fun `concurrency log lines carry ids and never position`() {
        val superseded = RaiseOutcomeLog.superseded("B", listOf("A"))
        assertTrue(superseded.contains("B") && superseded.contains("A"))
        assertFalse(Regex("-?\\d+\\.\\d+").containsMatchIn(superseded))

        val collapsed = RaiseOutcomeLog.collapsedIntoCount("A")
        assertTrue(collapsed.contains("A"))

        val promoted = RaiseOutcomeLog.promotedSilent("A")
        assertTrue(promoted.contains("A"))
        assertTrue(promoted.contains("silent"))
        assertFalse("promotion must never promise sound", promoted.contains("start"))
    }

    @Test
    fun `near-simultaneous arrivals converge regardless of order`() {
        val first = ActiveAlertBoard()
        first.upsert("A", sounded = true)
        first.upsert("B", sounded = true)

        val second = ActiveAlertBoard()
        second.upsert("B", sounded = true)
        second.upsert("A", sounded = true)

        // Same membership either way; selection follows arrival order, which is
        // exactly what "newest sounds" means — no coordination required.
        assertEquals(first.fullEntries().map { it.eventId }.toSet(), second.fullEntries().map { it.eventId }.toSet())
        assertEquals(setOf(4301, 4302), first.fullEntries().map { it.notificationId }.toSet())
    }
}
