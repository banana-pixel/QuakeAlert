package id.web.quakealert.domain

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Pins the synthetic drill-test event behind "Uji Peringatan Gempa".
 *
 * The properties that matter are all about honesty: the event must be fresh
 * enough to pass validity, namespaced so it can never collide with a server
 * event, flagged test end-to-end, and moderate enough that the distance gate
 * genuinely decides instead of being overridden by severity.
 */
class LocalTestEventTest {

    @Test
    fun `synthetic event is test flagged and confirmed`() {
        val message = LocalTestEvent.build(nowMs = 1_000_000L, userLocation = null)

        assertTrue(message.isTest)
        assertEquals(AlertType.EARTHQUAKE_ALERT, message.type)
        assertEquals(EventState.CONFIRMED, message.eventState)
    }

    @Test
    fun `event ids are unique per tap and namespaced`() {
        val first = LocalTestEvent.build(nowMs = 1_000_000L, userLocation = null)
        val second = LocalTestEvent.build(nowMs = 1_000_001L, userLocation = null)

        assertTrue(first.eventId.startsWith(LocalTestEvent.ID_PREFIX))
        assertNotEquals(first.eventId, second.eventId)
    }

    @Test
    fun `synthetic event is fresh under its own validity`() {
        val nowMs = 1_000_000L
        val message = LocalTestEvent.build(nowMs = nowMs, userLocation = null)

        assertTrue(message.isActionable(nowMs))
    }

    @Test
    fun `synthetic values never trigger the severe override`() {
        assertTrue(
            "test values must stay below the severe override or the gate proves nothing",
            !SafetyPolicy.isSevere(LocalTestEvent.MMI, LocalTestEvent.PGA_GAL)
        )
    }

    @Test
    fun `gate genuinely decides at the user position`() {
        val userLocation = UserLocation(latitude = -6.9, longitude = 107.6)
        val message = LocalTestEvent.build(nowMs = 1_000_000L, userLocation = userLocation)

        // Centroid == user position: an honest WITHIN_RADIUS, decided by
        // distance rather than by override or by missing position.
        val decision = AlertGate.decide(
            userLocation = userLocation,
            centroidLat = message.centroidLat,
            centroidLon = message.centroidLon,
            mmi = message.mmi,
            pgaGal = message.pgaGal
        )
        assertTrue(decision.shouldAlarm)
        assertEquals(AlertGateReason.WITHIN_RADIUS, decision.reason)
    }

    @Test
    fun `no position fails open like a real drill`() {
        val message = LocalTestEvent.build(nowMs = 1_000_000L, userLocation = null)

        val decision = AlertGate.decide(
            userLocation = null,
            centroidLat = message.centroidLat,
            centroidLon = message.centroidLon,
            mmi = message.mmi,
            pgaGal = message.pgaGal
        )
        assertTrue(decision.shouldAlarm)
        assertEquals(AlertGateReason.LOCATION_UNKNOWN, decision.reason)
    }
}
