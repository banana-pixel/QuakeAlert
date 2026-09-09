package id.web.quakealert.domain

/**
 * Builder for the local drill-test event behind "Uji Peringatan Gempa".
 *
 * The test must meet the *identical* production checks as a server drill —
 * validity (D-018), dedup, the user switch and [AlertGate] — so this builds a
 * plain [WsAlertMessage] and nothing else: no bypass flags, no test-only
 * branches in the raise path. What makes it a test is only its content:
 *
 *  - `isTest = true`, plumbed end-to-end into the TEST title, TEST badge and
 *    the end-test affordance. A server frame can never carry this on release
 *    (the mapper drops `is_test` there); locally it is the only constructor.
 *  - a `local-test-*` event id, namespaced so dedup, board and stand-down can
 *    never collide with a server event id.
 *  - deliberately NON-severe values (MMI V, 60 gal, below
 *    `SafetyPolicy.OVERRIDE_PGA_GAL`): a severe event would pass the gate by
 *    override and the test would prove nothing about distance. With moderate
 *    values the distance gate genuinely decides.
 *  - centroid at the user's own position, so a synced device reads
 *    WITHIN_RADIUS honestly; with no position the gate fails open
 *    (LOCATION_UNKNOWN) exactly as it would for a real drill.
 *
 * Pure and unit-testable. The message never leaves the device: no upload path
 * accepts a [WsAlertMessage] (all references are inbound/render), so a
 * synthetic event cannot become a real earthquake report.
 */
object LocalTestEvent {

    /** Prefix reserving the synthetic id namespace; see the class docs. */
    const val ID_PREFIX = "local-test-"

    /** Freshness window for the synthetic frame (D-018 runs unmodified). */
    const val VALIDITY_MS = 120_000L

    /** Moderate intensity: strong enough to render, never a severe override. */
    const val MMI = "V"

    /** Moderate PGA in gal, well below the severe override. */
    const val PGA_GAL = 60.0

    /** Confirmed-sized node count (≥ 3), matching the ALERT type. */
    const val NODE_COUNT = 4

    fun build(nowMs: Long, userLocation: UserLocation?): WsAlertMessage =
        WsAlertMessage(
            type = AlertType.EARTHQUAKE_ALERT,
            eventId = "$ID_PREFIX$nowMs",
            mmi = MMI,
            intensityLabel = "",
            pgaGal = PGA_GAL,
            centroidLat = userLocation?.latitude ?: 0.0,
            centroidLon = userLocation?.longitude ?: 0.0,
            locationName = userLocation?.locationName.orEmpty(),
            timestampMs = nowMs,
            nodeCount = NODE_COUNT,
            isTest = true,
            eventState = EventState.CONFIRMED,
            validityMs = VALIDITY_MS
        )
}
