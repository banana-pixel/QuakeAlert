package id.web.quakealert.device

import android.content.Context
import android.os.PowerManager
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.getSystemService

/**
 * The two OS-owned conditions an alert has to pass through, beside
 * [hasLocationPermission].
 *
 * Shared here rather than kept private to a screen because three surfaces now read
 * them — the Settings delivery checklist, the status notification, and the alert
 * toggle's "blocked by system settings" pill — and a second copy of the check is how
 * they come to disagree.
 *
 * Neither is observable: both are changed in system Settings while the app is in the
 * background, with no callback, so every caller re-reads them on resume.
 */

/**
 * Whether the OS currently allows this app to post notifications.
 *
 * The runtime `POST_NOTIFICATIONS` grant plus the app-level toggle
 * ([NotificationManagerCompat.areNotificationsEnabled]): an in-app grant dialog
 * only pauses the activity, and a toggle flipped in system Settings never
 * callbacks the app, so callers re-read this on every resume. Channel-level
 * degradation stays diagnostic-only per `AlertPresentationHealth` and never
 * gates (D-021).
 *
 * True below API 33, where the runtime grant did not exist, via
 * `areNotificationsEnabled`. Distinct from the user's own alert switch: this
 * says the app *may* post, that one says they *want* it.
 */
fun Context.canPostNotifications(): Boolean =
    NotificationManagerCompat.from(this).areNotificationsEnabled()

/**
 * Whether the app is exempt from battery optimisation.
 *
 * Relevant to alert delivery, not battery life: under Doze a data-only FCM message
 * can be held back until the next maintenance window, which for an earthquake
 * warning is indistinguishable from never arriving.
 */
fun Context.isBatteryUnrestricted(): Boolean =
    getSystemService<PowerManager>()?.isIgnoringBatteryOptimizations(packageName) ?: false
