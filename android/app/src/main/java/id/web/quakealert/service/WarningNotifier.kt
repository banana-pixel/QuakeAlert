package id.web.quakealert.service

import android.annotation.SuppressLint
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import id.web.quakealert.R
import id.web.quakealert.data.AppSettingsRepository
import id.web.quakealert.data.network.QuakeNetwork
import id.web.quakealert.data.network.mapper.intensityValueLabel
import id.web.quakealert.device.canPostNotifications
import id.web.quakealert.domain.ActiveAlertBoard
import id.web.quakealert.domain.AlertDecision
import id.web.quakealert.domain.DisplayLanguage
import id.web.quakealert.domain.RaiseOutcomeLog
import id.web.quakealert.domain.WsAlertMessage
import id.web.quakealert.ui.warning.WarningActivity
import kotlin.math.roundToInt

/**
 * Posts the emergency notification that wakes the device, and hands it the
 * full-screen intent to [WarningActivity].
 *
 * Separate from the siren demo dialog (which only plays the tone in-app):
 * the local drill test raises through this very notifier as a marked
 * `is_test` event, and this one is insistent where a demo is dismissible.
 *
 * The caller applies [id.web.quakealert.domain.AlertGate] before reaching here —
 * this class only reports the [AlertDecision] it was given.
 */
object WarningNotifier {

    const val CHANNEL_ID = "quakealert_emergency_alerts"
    private const val TAG = "WarningNotifier"

    /**
     * Per-event live state lives on the shared coexistence board
     * ([QuakeNetwork.activeAlerts]), not here: one object-level event_id cannot
     * represent concurrent quakes. This object stays a stateless poster —
     * notification ids are allocated from the board's deterministic pool.
     */

    /**
     * The event_id selected for display, or blank when nothing is live.
     *
     * Backed by the shared coexistence board ([QuakeNetwork.activeAlerts]):
     * with several events live this is the newest sounding one, not the only
     * one. Used by tests and stand-down observers that only need to know
     * *whether* something is displayed without enumerating the board.
     */
    fun activeNotificationEventId(context: Context): String =
        QuakeNetwork.from(context).activeAlerts.selectedId() ?: ""

    /** Registers the emergency channel. Safe to call repeatedly. */
    fun ensureChannel(context: Context, lang: DisplayLanguage = DisplayLanguage.EN) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = context.getSystemService(NotificationManager::class.java) ?: return
        if (manager.getNotificationChannel(CHANNEL_ID) != null) return

        // Name/description follow the language at creation time only: Android
        // freezes them afterwards, so switching language later does not rename
        // an existing channel (documented; reinstall picks the new label up).
        val name = if (lang == DisplayLanguage.ID) channelNameId() else "Earthquake Emergency Alerts"
        val desc = if (lang == DisplayLanguage.ID) channelDescId() else "Life-safety warnings for earthquakes near you."
        val channel = NotificationChannel(
            CHANNEL_ID,
            name,
            // IMPORTANCE_HIGH is the minimum a full-screen intent is honoured at.
            NotificationManager.IMPORTANCE_HIGH
        ).apply {
            description = desc
            enableVibration(true)
            enableLights(true)
            // Audible HIGH channel: makes a sound and appears as a heads-up
            // notification when unlocked. The sustained 90s siren is handled
            // separately by AlertSiren (USAGE_ALARM) in WarningActivity for the
            // locked FSI path; the notification sound is the short HUN chime
            // that makes the unlocked heads-up visible.
            lockscreenVisibility = NotificationCompat.VISIBILITY_PUBLIC
        }
        manager.createNotificationChannel(channel)
    }

    /**
     * Posts the alert.
     *
     * The full-screen intent is what turns a notification into a warning: it launches
     * [WarningActivity] over the lock screen without the user touching anything.
     * From API 34 that needs `canUseFullScreenIntent()` to be true — the permission is
     * granted by default only to calling and alarm apps — so the heads-up notification
     * is the documented fallback rather than an afterthought.
     *
     * Returns false when nothing could be posted (no `POST_NOTIFICATIONS` grant),
     * so the caller can log a delivery that the user will never see.
     */
    // canPost() below is exactly the checkSelfPermission call lint asks for; it cannot
    // see through the helper, and letting the post throw instead would drop an alert.
    @SuppressLint("MissingPermission")
    fun notify(
        context: Context,
        message: WsAlertMessage,
        decision: AlertDecision,
        lang: DisplayLanguage = DisplayLanguage.EN
    ): Boolean {
        ensureChannel(context, lang)
        if (!canPost(context)) {
            Log.w(TAG, "POST_NOTIFICATIONS not granted; alert cannot be shown")
            return false
        }

        val board = QuakeNetwork.from(context).activeAlerts
        // Coexistence (D-020): register first so the slot exists even if the
        // post below throws; a posted-then-unknown id would be un-clearable.
        val slot = board.upsert(message.eventId, sounded = false)
        for (evictedId in slot.cancelNotificationIds) {
            NotificationManagerCompat.from(context).cancel(evictedId)
        }
        if (slot.collapsedId != null) {
            android.util.Log.i(TAG, RaiseOutcomeLog.collapsedIntoCount(slot.collapsedId))
        }
        if (slot.supersededIds.isNotEmpty()) {
            android.util.Log.i(
                TAG,
                RaiseOutcomeLog.superseded(message.eventId, slot.supersededIds)
            )
        }

        val distanceKm = decision.distanceKm?.roundToInt()
        val fullScreen = PendingIntent.getActivity(
            context,
            slot.notificationId,
            WarningActivity.intent(
                context = context,
                eventId = message.eventId,
                intensityValue = message.intensityValueLabel(lang.locale()),
                locationName = message.locationName,
                distanceKm = distanceKm,
                isTest = message.isTest,
                activeCount = board.extraActiveCount()
            ),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val builder = NotificationCompat.Builder(context, CHANNEL_ID)
            // Official logo, transparent variant whole (D-023).
            .setSmallIcon(R.drawable.ic_quake_logo)
            // A drill says so in the shade as well as on the screen. Only ever
            // reachable on a debug build (the mapper drops an is_test frame
            // otherwise), so this branch cannot change what a real user is told.
            .setContentTitle(alertTitle(message, lang))
            .setContentText(bodyText(message, distanceKm, lang))
            .setPriority(NotificationCompat.PRIORITY_MAX)
            .setCategory(NotificationCompat.CATEGORY_ALARM)
            .setVisibility(NotificationCompat.VISIBILITY_PUBLIC)
            // Ongoing and non-cancelling: the user stands the alert down by acting on
            // it, not by swiping it away.
            .setOngoing(true)
            .setAutoCancel(false)
            .setContentIntent(fullScreen)

        if (canUseFullScreen(context)) {
            builder.setFullScreenIntent(fullScreen, true)
        } else {
            Log.w(TAG, "full-screen intents not permitted; falling back to heads-up")
        }

        NotificationManagerCompat.from(context).notify(slot.notificationId, builder.build())
        // Success posts a line too (D-019, U-013): the failure branch below is
        // not the only observable outcome. event_id and outcome only.
        android.util.Log.i(TAG, RaiseOutcomeLog.notificationPosted(message.eventId))
        // Remember what was shown, for the status notification's "Last alert" line. Here
        // and not on arrival: the claim it feeds is that the app has alerted this user,
        // and an alert filtered out by the distance gate or dropped by the OS never did.
        AppSettingsRepository(context).setLastAlert(
            summary = summaryText(message, distanceKm, lang),
            epochMs = System.currentTimeMillis()
        )
        return true
    }

    /**
     * Clears the emergency notification(s) on `EVENT_RESOLVED`, strictly by
     * event identity (D-020): the resolved event's own notification id is
     * cancelled and its board slot released, so one event's all-clear can
     * never remove another's.
     *
     * Returns what was removed, so renderers (the ViewModel screen, the
     * Activity) can promote survivors without re-sounding them. Callers that
     * only fire and forget (push service, bridge) ignore the result.
     *
     * @param standDownEventId the event_id carried by the resolved/cancelled frame.
     *   Blank (pre-Phase-3 frames, or callers that do not have an id) clears
     *   only when exactly one event is live; with zero or several live it is
     *   ignored and logged, since an unscoped clear cannot know what to remove.
     */
    fun clear(
        context: Context,
        standDownEventId: String = ""
    ): ActiveAlertBoard.StandDownResult {
        val board = QuakeNetwork.from(context).activeAlerts
        if (standDownEventId.isBlank()) {
            return when (val single = board.standDownBlank()) {
                is ActiveAlertBoard.BlankStandDown.NoneActive ->
                    ActiveAlertBoard.StandDownResult(emptyList(), board.selectedId(), false)
                is ActiveAlertBoard.BlankStandDown.ClearedSingle -> {
                    single.notificationId?.let {
                        NotificationManagerCompat.from(context).cancel(it)
                    }
                    ActiveAlertBoard.StandDownResult(
                        listOfNotNull(single.notificationId),
                        board.selectedId(),
                        removedAny = true
                    )
                }
                is ActiveAlertBoard.BlankStandDown.IgnoredAmbiguous -> {
                    Log.d(TAG, "blank stand-down ignored; several events live")
                    ActiveAlertBoard.StandDownResult(
                        emptyList(),
                        board.selectedId(),
                        removedAny = false
                    )
                }
            }
        }
        val result = board.standDown(standDownEventId)
        for (nid in result.cancelledNotificationIds) {
            NotificationManagerCompat.from(context).cancel(nid)
        }
        if (result.promotedId != null && result.removedAny) {
            Log.i(TAG, RaiseOutcomeLog.promotedSilent(result.promotedId))
        }
        return result
    }

    /**
     * The one-line record kept for the status notification: what and where, no advice and
     * no distance-unknown caveat. It is read weeks later in a shade, not during shaking.
     */
    private fun summaryText(message: WsAlertMessage, distanceKm: Int?, lang: DisplayLanguage): String {
        if (lang == DisplayLanguage.ID) return summaryTextId(message, distanceKm)
        val where = message.locationName.takeIf { it.isNotBlank() } ?: "your area"
        // Kept distinguishable weeks later: a "Last alert" line that cannot be told
        // apart from a real one would misrepresent what the app has warned about.
        if (message.isTest) return "Drill (test alert) near $where"
        val proximity = distanceKm?.let { ", $it km away" }.orEmpty()
        return "Intensity ${message.mmi} near $where$proximity"
    }

    private fun bodyText(message: WsAlertMessage, distanceKm: Int?, lang: DisplayLanguage): String {
        if (lang == DisplayLanguage.ID) return bodyTextId(message, distanceKm)
        val where = message.locationName.takeIf { it.isNotBlank() } ?: "your area"
        // "Distance unknown" rather than a fabricated number — the gate fails open on
        // an unknown position, so this is a real case and not a defensive branch.
        val proximity = distanceKm?.let { "$it km away" } ?: "distance unknown"
        return "Intensity ${message.mmi} near $where ($proximity). Drop, cover, hold on."
    }

    private fun alertTitle(message: WsAlertMessage, lang: DisplayLanguage): String {
        if (lang == DisplayLanguage.ID) return alertTitleId(message)
        return if (message.isTest) "TEST - earthquake drill" else "Earthquake detected"
    }

    // Indonesian branches (B2). Acuan BMKG: "Lindungi diri Anda!".
    private fun channelNameId(): String = "Peringatan Darurat Gempa Bumi"
    private fun channelDescId(): String = "Peringatan keselamatan jiwa untuk gempa bumi di dekat Anda."
    private fun summaryTextId(message: WsAlertMessage, distanceKm: Int?): String {
        val where = message.locationName.takeIf { it.isNotBlank() } ?: "daerah Anda"
        if (message.isTest) return "Latihan (peringatan uji) di dekat $where"
        val proximity = distanceKm?.let { ", $it km dari Anda" }.orEmpty()
        return "Intensitas ${message.mmi} di dekat $where$proximity"
    }
    private fun bodyTextId(message: WsAlertMessage, distanceKm: Int?): String {
        val where = message.locationName.takeIf { it.isNotBlank() } ?: "daerah Anda"
        val proximity = distanceKm?.let { "$it km dari Anda" } ?: "jarak tidak diketahui"
        return "Intensitas ${message.mmi} di dekat $where ($proximity). Menunduk, lindungi kepala, berpegangan."
    }
    private fun alertTitleId(message: WsAlertMessage): String =
        if (message.isTest) "UJI - latihan gempa bumi" else "Gempa bumi terdeteksi"

    /** Single source is [canPostNotifications] (D-021): runtime grant + app-level toggle. */
    private fun canPost(context: Context): Boolean = context.canPostNotifications()

    private fun canUseFullScreen(context: Context): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.UPSIDE_DOWN_CAKE) return true
        val manager = context.getSystemService(NotificationManager::class.java) ?: return false
        return runCatching { manager.canUseFullScreenIntent() }.getOrDefault(false)
    }
}
