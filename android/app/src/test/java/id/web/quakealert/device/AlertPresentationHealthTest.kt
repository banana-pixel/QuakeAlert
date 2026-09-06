package id.web.quakealert.device

import android.app.NotificationManager
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Diagnostic for degraded in-use presentation (D-017, U-012).
 * [evaluatePresentationHealth] is pure: same inputs, same verdict, no Android
 * calls, no side effects — so this suite needs no device. The verdict never
 * gates alert handling; only user-facing warning text comes out of it.
 */
class AlertPresentationHealthTest {

    private fun channel(
        importance: Int = NotificationManager.IMPORTANCE_HIGH,
        soundNull: Boolean = false
    ) = EmergencyChannelSnapshot(importance = importance, soundNull = soundNull)

    @Test
    fun `heads_up off degrades even when the channel is perfect`() {
        val verdict = evaluatePresentationHealth(
            headsUpEnabledGlobally = false,
            channel = channel()
        )

        assertEquals(setOf(PresentationDegradation.HEADS_UP_DISABLED_GLOBALLY), verdict)
    }

    @Test
    fun `enabled switch plus healthy channel is healthy and silent`() {
        val verdict = evaluatePresentationHealth(
            headsUpEnabledGlobally = true,
            channel = channel()
        )

        assertTrue(verdict.isEmpty())
        assertNull(verdict.warningCopy())
    }

    @Test
    fun `lowered channel importance degrades`() {
        val verdict = evaluatePresentationHealth(
            headsUpEnabledGlobally = true,
            channel = channel(importance = NotificationManager.IMPORTANCE_DEFAULT)
        )

        assertEquals(setOf(PresentationDegradation.CHANNEL_IMPORTANCE_LOWERED), verdict)
    }

    @Test
    fun `explicitly silent channel degrades`() {
        val verdict = evaluatePresentationHealth(
            headsUpEnabledGlobally = true,
            channel = channel(soundNull = true)
        )

        assertEquals(setOf(PresentationDegradation.CHANNEL_SILENT), verdict)
    }

    @Test
    fun `missing channel is healthy because notify registers it first`() {
        val verdict = evaluatePresentationHealth(
            headsUpEnabledGlobally = true,
            channel = null
        )

        assertTrue(verdict.isEmpty())
    }

    @Test
    fun `independent causes accumulate instead of shadowing each other`() {
        val verdict = evaluatePresentationHealth(
            headsUpEnabledGlobally = false,
            channel = channel(
                importance = NotificationManager.IMPORTANCE_LOW,
                soundNull = true
            )
        )

        assertEquals(
            setOf(
                PresentationDegradation.HEADS_UP_DISABLED_GLOBALLY,
                PresentationDegradation.CHANNEL_IMPORTANCE_LOWERED,
                PresentationDegradation.CHANNEL_SILENT
            ),
            verdict
        )
    }

    @Test
    fun `warning names the cause and the remedy, and stays diagnostic`() {
        val copy = setOf(PresentationDegradation.HEADS_UP_DISABLED_GLOBALLY).warningCopy()

        requireNotNull(copy)
        assertTrue(copy.contains("heads-up", ignoreCase = true))
        assertTrue(copy.contains("system Settings", ignoreCase = true))
        assertTrue(copy.contains("locked-screen", ignoreCase = true))
    }

    @Test
    fun `channel warning names the channel without touching alert behavior`() {
        val copy = setOf(
            PresentationDegradation.CHANNEL_IMPORTANCE_LOWERED,
            PresentationDegradation.CHANNEL_SILENT
        ).warningCopy()

        requireNotNull(copy)
        assertTrue(copy.contains("Earthquake Emergency Alerts"))
    }
}
