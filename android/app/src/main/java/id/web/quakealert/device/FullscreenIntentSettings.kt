package id.web.quakealert.device

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.provider.Settings

/**
 * Opens the system page where the full-screen-intent permission is granted.
 *
 * Per the platform docs (source.android.com/docs/core/permissions/fsi-limits)
 * that page is the dedicated "Manage full screen intents" screen
 * ([Settings.ACTION_MANAGE_APP_USE_FULL_SCREEN_INTENT]), NOT the app's
 * general notification settings: the toggle does not live there, and sending
 * the user to the wrong page is how a denied permission stays denied. Used by
 * the onboarding fullscreen page and the Settings delivery-checklist row —
 * both ask for the same grant, so both must land on the same page.
 *
 * Pre-API 34 there is no such screen (no such permission either — every app
 * may use full-screen intents there), so the app's notification settings is
 * the honest fallback. Returns false when nothing could be opened, so the
 * caller can say so instead of silently doing nothing.
 *
 * The [launch] overload (D-026) routes the dedicated page through the caller's
 * settings launcher so its return callback fires and the onboarding pill
 * re-reads the grant on return, like the battery page already does. The
 * no-arg overload keeps the direct-`startActivity` behaviour for callers with
 * no launcher (e.g. the Settings checklist row, which refreshes on resume).
 */
fun Context.openFullscreenIntentSettings(): Boolean =
    openFullscreenIntentSettings({ startActivity(it) })

fun Context.openFullscreenIntentSettings(launch: (Intent) -> Unit): Boolean {
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
        val manage = Intent(Settings.ACTION_MANAGE_APP_USE_FULL_SCREEN_INTENT).apply {
            data = Uri.fromParts("package", packageName, null)
        }
        if (manage.resolveActivity(packageManager) != null) {
            if (runCatching { launch(manage) }.isSuccess) return true
            // else fall through to the fallback below, as before
        }
    }
    val fallback = Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS)
        .putExtra(Settings.EXTRA_APP_PACKAGE, packageName)
    return runCatching { launch(fallback) }.isSuccess
}
