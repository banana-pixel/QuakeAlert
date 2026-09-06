package id.web.quakealert.domain

/**
 * One-line, privacy-safe log wording for every raise-path outcome (D-019,
 * U-013).
 *
 * A single factory so the three raise paths (push service, foreground socket
 * ViewModel, background socket bridge) describe the same outcome the same way,
 * and so the wording itself is unit-testable: logcat output cannot be asserted
 * from a JVM test, but these pure strings can — including the privacy property
 * that matters, that no line can carry coordinates or a precise distance.
 *
 * Inputs are deliberately narrow: an opaque `event_id`, enum/boolean outcomes.
 * There is no parameter that could smuggle a position in, which is what makes
 * "never raw coordinates" structural rather than a review-time promise. The
 * distance-bearing [AlertDecision] stays at the call site; only its [reason]
 * enum name crosses into a log line.
 */
object RaiseOutcomeLog {

    /** An alert was raised on screen; whether its siren started. */
    fun shown(eventId: String, sirenStarted: Boolean): String =
        "raise $eventId shown (siren=${if (sirenStarted) "started" else "not-started"})"

    /** The distance gate kept an alert off the emergency screen. */
    fun gatedOut(eventId: String, reason: AlertGateReason): String =
        "raise $eventId gated-out (${reason.name}); not raising"

    /** A duplicate delivery was suppressed; the original raise stands. */
    fun duplicateSuppressed(eventId: String): String =
        "raise $eventId duplicate-suppressed; original stands"

    /** The emergency notification was posted to the shade. */
    fun notificationPosted(eventId: String): String =
        "notification posted for $eventId"

    /**
     * A frame arrived too late to act on. [validityDeclared] distinguishes the
     * D-018 sender-declared expiry from the legacy recent-window expiry, so a
     * reader can tell which rule fired without any timestamp arithmetic.
     */
    fun expired(eventId: String, validityDeclared: Boolean): String =
        if (validityDeclared) "raise $eventId expired sender-validity; not raising"
        else "raise $eventId outside recent window; not raising"
}
