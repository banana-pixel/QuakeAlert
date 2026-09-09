package id.web.quakealert.domain

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The local-warning gate (D-036): distance-only against
 * [SafetyPolicy.LOCAL_WARNING_RADIUS_KM], with deliberately no intensity
 * override, reached through the single choke point [AlertGate.decideFor].
 *
 * What is pinned here:
 *  - a trusted-local frame inside 20 km alarms with a local reason;
 *  - the 20 km boundary (19.9 alarms, 20.1 does not — 100 m of margin either
 *    side, so no float-equality flake);
 *  - severe values do NOT escalate a local frame (PGA 300 / MMI VIII at
 *    100 km stays silent — single-station authority must never borrow the
 *    confirmed/severe semantics);
 *  - an unknown position fails open, like the confirmed gate;
 *  - the confirmed path is untouched: severe still overrides there, and an
 *    ordinary frame never takes the local gate.
 */
class LocalWarningGateTest {

    private val user = UserLocation(latitude = 0.0, longitude = 100.0)

    /** Point due east of [user] by [km], on the equator for exact arithmetic. */
    private fun eastOf(km: Double): Pair<Double, Double> =
        0.0 to 100.0 + km / 111.195

    private fun localMessage(
        centroidLat: Double = 0.0,
        centroidLon: Double = 100.0,
        mmi: String = "V",
        pgaGal: Double = 150.0
    ): WsAlertMessage = WsAlertMessage(
        type = AlertType.EARTHQUAKE_ALERT,
        eventId = "evt-local-1",
        mmi = mmi,
        intensityLabel = "moderate",
        pgaGal = pgaGal,
        centroidLat = centroidLat,
        centroidLon = centroidLon,
        locationName = "Test Area",
        timestampMs = 1_755_000_000_000L,
        nodeCount = 1,
        trustedLocal = true
    )

    @Test
    fun `local radius is 20 km and stays distinct from the confirmed radius`() {
        assertEquals(20, SafetyPolicy.LOCAL_WARNING_RADIUS_KM)
        assertEquals(200, SafetyPolicy.ALERT_RADIUS_KM)
    }

    @Test
    fun `alarms inside 20 km with a local reason`() {
        val (lat, lon) = eastOf(10.0)
        val decision = AlertGate.decideLocalWarning(
            userLocation = user,
            centroidLat = lat,
            centroidLon = lon
        )

        assertTrue(decision.shouldAlarm)
        assertEquals(AlertGateReason.LOCAL_WITHIN_RADIUS, decision.reason)
    }

    @Test
    fun `alarms on the same point`() {
        val decision = AlertGate.decideLocalWarning(
            userLocation = user,
            centroidLat = 0.0,
            centroidLon = 100.0
        )

        assertTrue(decision.shouldAlarm)
        assertEquals(AlertGateReason.LOCAL_WITHIN_RADIUS, decision.reason)
    }

    @Test
    fun `boundary pins within two hundred metres`() {
        val (nearLat, nearLon) = eastOf(19.9)
        assertTrue(
            AlertGate.decideLocalWarning(user, nearLat, nearLon).shouldAlarm
        )

        val (farLat, farLon) = eastOf(20.1)
        val far = AlertGate.decideLocalWarning(user, farLat, farLon)
        assertFalse(far.shouldAlarm)
        assertEquals(AlertGateReason.LOCAL_OUTSIDE_RADIUS, far.reason)
    }

    @Test
    fun `severe values never escalate a local frame`() {
        val (lat, lon) = eastOf(100.0)
        for (m in listOf(
            localMessage(centroidLat = lat, centroidLon = lon, mmi = "VIII", pgaGal = 60.0),
            localMessage(centroidLat = lat, centroidLon = lon, mmi = "V", pgaGal = 300.0),
            localMessage(centroidLat = lat, centroidLon = lon, mmi = "VIII", pgaGal = 500.0)
        )) {
            val decision = AlertGate.decideFor(message = m, userLocation = user)
            assertFalse("severe local frame at 100 km must stay silent", decision.shouldAlarm)
            assertEquals(AlertGateReason.LOCAL_OUTSIDE_RADIUS, decision.reason)
        }
    }

    @Test
    fun `fails open with an unknown position`() {
        val decision = AlertGate.decideLocalWarning(
            userLocation = null,
            centroidLat = 0.0,
            centroidLon = 100.0
        )

        assertTrue(decision.shouldAlarm)
        assertEquals(AlertGateReason.LOCATION_UNKNOWN, decision.reason)
        assertNull(decision.distanceKm)
    }

    @Test
    fun `decideFor routes a trusted frame to the local gate`() {
        val (lat, lon) = eastOf(100.0)
        val decision = AlertGate.decideFor(
            message = localMessage(centroidLat = lat, centroidLon = lon),
            userLocation = user
        )

        assertFalse(decision.shouldAlarm)
        // The confirmed gate would have failed open on severity or distance;
        // the local reason proves the local gate decided.
        assertEquals(AlertGateReason.LOCAL_OUTSIDE_RADIUS, decision.reason)
    }

    @Test
    fun `decideFor keeps an ordinary frame on the confirmed path`() {
        val (lat, lon) = eastOf(100.0)
        val ordinary = localMessage(centroidLat = lat, centroidLon = lon).copy(trustedLocal = false)

        // 150 gal is below the severe override and 100 km is inside 200 km:
        // the confirmed gate alarms with its own reason, never a local one.
        val decision = AlertGate.decideFor(message = ordinary, userLocation = user)
        assertTrue(decision.shouldAlarm)
        assertEquals(AlertGateReason.WITHIN_RADIUS, decision.reason)
    }

    @Test
    fun `confirmed severe regression - intensity still overrides distance`() {
        val (lat, lon) = eastOf(500.0)
        val severe = localMessage(centroidLat = lat, centroidLon = lon, mmi = "VIII", pgaGal = 60.0)
            .copy(trustedLocal = false)

        val decision = AlertGate.decideFor(message = severe, userLocation = user)
        assertTrue(decision.shouldAlarm)
        assertEquals(AlertGateReason.SEVERE_OVERRIDE, decision.reason)
    }
}
