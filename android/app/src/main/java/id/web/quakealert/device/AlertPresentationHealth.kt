package id.web.quakealert.device

import android.app.NotificationManager
import android.content.Context
import android.os.Build
import android.provider.Settings
import androidx.core.content.getSystemService
import id.web.quakealert.domain.DisplayLanguage

/**
 * Diagnostic for the approved in-use warning behaviour (D-017, U-012):
 * UNLOCKED + screen-on CONFIRMED alerts use heads-up + siren/vibration and
 * never seize the foreground.
 *
 * This file only *detects* degraded presentation capability and describes it.
 * It changes no channel, no importance, no full-screen intent, no contract and
 * no alert behaviour: a degraded result only produces user-facing warning
 * text. Alert handling (raise / suppress / clear) never consults this file.
 *
 * Two independent degradations, matching the two causes separated on device
 * (POCO F1, API 36):
 *
 * 1. The device-wide `heads_up_notifications_enabled` switch is off. This is a
 *    `Settings.Global` condition evaluated by SystemUI *before* any
 *    per-notification filter, so it suppresses heads-up for every app — the
 *    2026-08-31 device-wide suppression record. Only an explicit `0` counts:
 *    an absent key is treated as enabled (stock platform default), because
 *    warning on an unknown would nag users about a condition that may not
 *    exist on their OS build.
 * 2. The emergency channel itself is degraded: importance below HIGH (heads-up
 *    requires HIGH), or an explicitly silent sound. Observed on-device
 *    (dumpsys): a fresh channel resolves `sound` to the system default URI,
 *    while the 2026-08-31 `setSound(null, null)` channel showed null — so
 *    null sound here means explicitly silenced, not "default". Only the
 *    nullness is recorded, never the URI (no PII in diagnostics).
 */

/** Why in-use presentation is degraded. Empty means healthy. */
enum class PresentationDegradation {
    /** Device-wide heads-up switch is explicitly off: no app can peek. */
    HEADS_UP_DISABLED_GLOBALLY,

    /** Emergency channel importance dropped below HIGH: no heads-up possible. */
    CHANNEL_IMPORTANCE_LOWERED,

    /** Emergency channel sound explicitly silenced. */
    CHANNEL_SILENT
}

/**
 * Snapshot of the emergency channel facts the diagnostic needs — nullness of
 * the sound only, never the URI — so the evaluator stays a pure function of
 * plain values and is unit-testable without Robolectric.
 *
 * Null (no channel object on API 26+) means healthy: [WarningNotifier]
 * registers the channel on every notify, so a missing channel is created
 * before it can matter.
 */
data class EmergencyChannelSnapshot(
    val importance: Int,
    val soundNull: Boolean
)

/**
 * Pure evaluator: the same inputs always yield the same verdict, with no
 * Android calls and no side effects. Diagnostic only — the result must never
 * gate raising, suppressing, or clearing an alert.
 */
fun evaluatePresentationHealth(
    headsUpEnabledGlobally: Boolean,
    channel: EmergencyChannelSnapshot?
): Set<PresentationDegradation> = buildSet {
    if (!headsUpEnabledGlobally) add(PresentationDegradation.HEADS_UP_DISABLED_GLOBALLY)
    if (channel != null) {
        if (channel.importance < NotificationManager.IMPORTANCE_HIGH) {
            add(PresentationDegradation.CHANNEL_IMPORTANCE_LOWERED)
        }
        if (channel.soundNull) add(PresentationDegradation.CHANNEL_SILENT)
    }
}

/**
 * Reads the device-wide heads-up switch. Absent key counts as enabled (stock
 * platform default; see file KDoc) — only an explicit `0` degrades.
 */
fun Context.isHeadsUpEnabledGlobally(): Boolean {
    if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return true
    return try {
        Settings.Global.getInt(contentResolver, "heads_up_notifications_enabled", -1) != 0
    } catch (_: SecurityException) {
        true
    }
}

/** Reads the emergency channel facts the diagnostic needs, or null when absent. */
fun Context.readEmergencyChannel(): EmergencyChannelSnapshot? {
    if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return null
    val manager = getSystemService<NotificationManager>() ?: return null
    val channel = manager.getNotificationChannel(
        id.web.quakealert.service.WarningNotifier.CHANNEL_ID
    ) ?: return null
    return EmergencyChannelSnapshot(
        importance = channel.importance,
        soundNull = channel.sound == null
    )
}

/** Full diagnostic for this device right now. Diagnostic only, never a gate. */
fun Context.alertPresentationHealth(): Set<PresentationDegradation> =
    evaluatePresentationHealth(
        headsUpEnabledGlobally = isHeadsUpEnabledGlobally(),
        channel = readEmergencyChannel()
    )

/**
 * User-facing warning copy for a degraded result, null when healthy (so callers
 * render nothing). Names what is affected (in-use/unlocked alerts), what to do
 * about each cause, and what is NOT affected (locked-screen alarm path).
 */
fun Set<PresentationDegradation>.warningCopy(lang: DisplayLanguage = DisplayLanguage.EN): String? {
    if (isEmpty()) return null
    if (lang == DisplayLanguage.ID) return warningCopyId()
    val causes = map {
        when (it) {
            PresentationDegradation.HEADS_UP_DISABLED_GLOBALLY ->
                "heads-up notifications are turned off device-wide in system settings"
            PresentationDegradation.CHANNEL_IMPORTANCE_LOWERED ->
                "the Earthquake Emergency Alerts channel importance was lowered below High"
            PresentationDegradation.CHANNEL_SILENT ->
                "the Earthquake Emergency Alerts channel sound was turned off"
        }
    }
    return "In-use alerts may not appear: " + causes.joinToString("; ") + ". " +
        "While unlocked, earthquake warnings show as a heads-up banner with siren - " +
        "that banner cannot appear in this state. " +
        "Fix it in system Settings (Notifications), then re-check here. " +
        "The locked-screen alarm path is unaffected."
}

// Indonesian branch (B2). Acuan string sistem: "Pengaturan" (Settings >
// Notifikasi).
private fun Set<PresentationDegradation>.warningCopyId(): String? {
    if (isEmpty()) return null
    val causes = map {
        when (it) {
            PresentationDegradation.HEADS_UP_DISABLED_GLOBALLY ->
                "notifikasi heads-up dimatikan di seluruh perangkat pada pengaturan sistem"
            PresentationDegradation.CHANNEL_IMPORTANCE_LOWERED ->
                "prioritas kanal Peringatan Darurat Gempa Bumi diturunkan di bawah Tinggi"
            PresentationDegradation.CHANNEL_SILENT ->
                "suara kanal Peringatan Darurat Gempa Bumi dimatikan"
        }
    }
    return "Peringatan saat dipakai mungkin tidak muncul: " + causes.joinToString("; ") + ". " +
        "Saat tidak dikunci, peringatan gempa tampil sebagai banner heads-up dengan sirene - " +
        "banner itu tidak dapat muncul dalam kondisi ini. " +
        "Perbaiki di Pengaturan sistem (Notifikasi), lalu periksa lagi di sini. " +
        "Jalur alarm layar terkunci tidak terpengaruh."
}
