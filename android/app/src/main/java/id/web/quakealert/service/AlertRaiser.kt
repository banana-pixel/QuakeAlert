package id.web.quakealert.service

import android.content.Context
import android.util.Log
import id.web.quakealert.data.AppSettingsRepository
import id.web.quakealert.data.network.QuakeNetwork
import id.web.quakealert.device.canPostNotifications
import id.web.quakealert.domain.AlertGate
import id.web.quakealert.domain.AlertType
import id.web.quakealert.domain.LocalTestEvent
import id.web.quakealert.domain.RaiseOutcomeLog
import id.web.quakealert.domain.WsAlertMessage
import id.web.quakealert.domain.resolveDisplayLanguage
import kotlinx.coroutines.flow.first

/**
 * The single shared raise path for a parsed [WsAlertMessage], whatever delivered
 * it.
 *
 * Extracted verbatim from `QuakeMessagingService.onMessageReceived` so there is
 * exactly one implementation of the sequence every raise must follow, in order:
 * stand-down short-circuit → advisory drop → validity (D-018) → dedup →
 * user-switch check → distance gate → [WarningNotifier.notify], with a D-019
 * log line on every outcome. A caller that reimplemented any step would drift;
 * a caller that skipped one would not be raising alerts, only noise.
 *
 * Entry points converge here rather than each owning a copy:
 *  - FCM push (`QuakeMessagingService`), the original owner of this block;
 *  - the local drill test, which builds a synthetic `is_test=true` event and
 *    must meet the identical gate, dedup, validity and logging behaviour —
 *    only the event source differs.
 *
 * Suspend so callers run it on their own scope (`applicationScope` for push,
 * `viewModelScope` for UI-initiated tests). All dependencies resolve from
 * [context]: no caller-held state, nothing to keep in sync.
 */
object AlertRaiser {

    /**
     * Applies one parsed alert frame: clears on all-clear, ignores advisories,
     * expires stale frames, suppresses duplicates and gated-out events, and
     * posts the emergency notification otherwise.
     */
    suspend fun raiseAlertFrame(context: Context, message: WsAlertMessage) {
        val appContext = context.applicationContext
        val network = QuakeNetwork.from(appContext)
        val settings = AppSettingsRepository(appContext)

        // An all-clear takes the notification down and needs no gate: the user is
        // being told something ended, and being told that too far away is harmless.
        if (message.type == AlertType.EVENT_RESOLVED) {
            network.alertDedup.markIfNew(message)
            WarningNotifier.clear(appContext, message.eventId)
            // CANCELLED and RESOLVED share this wire type and differ only in
            // event_state (the type enum is frozen so an un-updated install still
            // clears its alarm); both take the notification down, and only the wording
            // the app shows on return differs.
            Log.i(TAG, "push stand-down ${message.eventId}: ${message.eventState}")
            return
        }

        // Advisories are 1–2 unconfirmed nodes. They never wake the device — the app
        // shows them as a banner when it is open. Escalating them here would train
        // users to dismiss the real thing.
        //
        // Since server Phase 3 an advisory is not published to FCM at all, so this is
        // now a safety net rather than a live branch. It stays: what makes an
        // advisory banner-only is this check, not the server's send list, and a
        // configuration change on one deployment must not be able to turn an
        // unconfirmed tremor into a full-screen alarm.
        if (message.type == AlertType.EARTHQUAKE_ADVISORY) return

        if (!message.isActionable()) {
            Log.i(TAG, RaiseOutcomeLog.expired(message.eventId, message.validityMs > 0))
            return
        }

        if (!network.alertDedup.markIfNew(message)) {
            Log.i(TAG, "push alert ${message.eventId} already handled; dropped")
            return
        }

        if (!settings.notificationsEnabledOrDefault()) {
            Log.i(TAG, "user disabled alert notifications; not raising")
            return
        }

        val userLocation = network.sessionStore.readUserLocation()
        // One choke point for both gates (AlertGate.decideFor): a trusted-local
        // frame takes the 20 km no-override gate, everything else the confirmed
        // path. The advisory drop above already ran, so a local flag on an
        // advisory can never reach either gate.
        val decision = AlertGate.decideFor(
            message = message,
            userLocation = userLocation
        )

        if (!decision.shouldAlarm) {
            // D-019: event_id + gate reason only. The previous wording logged
            // the rounded distance, which is location-derived and now
            // forbidden on the raise path; the reason enum already says why.
            Log.i(TAG, RaiseOutcomeLog.gatedOut(message.eventId, decision.reason))
            return
        }

        WarningNotifier.notify(
            appContext,
            message,
            decision,
            resolveDisplayLanguage(runCatching { settings.language.first() }.getOrNull())
        )
    }

    private suspend fun AppSettingsRepository.notificationsEnabledOrDefault(): Boolean =
        runCatching { notificationsEnabled.first() }.getOrDefault(true)

    /** Why [runLocalTest] did not raise, for the caller's user-facing message. */
    enum class LocalTestOutcome { RAN, SWITCH_OFF, NO_PERMISSION }

    /**
     * Runs the user-facing drill test: builds a synthetic `is_test=true` event
     * and feeds it into [raiseAlertFrame], the same pipeline a server drill
     * travels. Only the event source differs.
     *
     * Honors the user's own alert switch exactly like push delivery does: with
     * alerts off the test refuses rather than demonstrating an alarm the user's
     * settings forbid. Clears the previous test's event first, so synthetic
     * notifications can never accumulate — no server resolve will ever come
     * for a `local-test-*` id.
     */
    suspend fun runLocalTest(context: Context): LocalTestOutcome {
        val appContext = context.applicationContext
        val settings = AppSettingsRepository(appContext)
        if (!settings.notificationsEnabledOrDefault()) return LocalTestOutcome.SWITCH_OFF
        if (!appContext.canPostNotifications()) return LocalTestOutcome.NO_PERMISSION
        runCatching { settings.lastLocalTestEventId.first() }.getOrNull()
            ?.takeIf { it.isNotBlank() }
            ?.let { WarningNotifier.clear(appContext, it) }
        val network = QuakeNetwork.from(appContext)
        val userLocation = runCatching { network.sessionStore.readUserLocation() }.getOrNull()
        val message = LocalTestEvent.build(
            nowMs = System.currentTimeMillis(),
            userLocation = userLocation
        )
        settings.setLastLocalTestEventId(message.eventId)
        raiseAlertFrame(appContext, message)
        return LocalTestOutcome.RAN
    }

    private const val TAG = "AlertRaiser"
}
