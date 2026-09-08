package id.web.quakealert.ui.onboarding

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import android.Manifest
import android.content.pm.PackageManager
import id.web.quakealert.R
import id.web.quakealert.domain.DisplayLanguage

/**
 * Small helper that owns the "test alert" notification channel and fires a
 * local notification so the user can confirm alerts reach them during
 * onboarding (Figma node 1:426). The test is a plain auto-cancelling alert;
 * real emergency alerts decide their own ongoing/insistent behaviour.
 */
object TestAlertNotifier {

    const val CHANNEL_ID = "quakealert_test_alerts"
    private const val NOTIFICATION_ID = 4201

    /** Registers the alert channel. Safe to call repeatedly. */
    fun ensureChannel(context: Context, lang: DisplayLanguage = DisplayLanguage.EN) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = context.getSystemService(NotificationManager::class.java)
        if (manager.getNotificationChannel(CHANNEL_ID) != null) return

        val channel = NotificationChannel(
            CHANNEL_ID,
            if (lang == DisplayLanguage.ID) channelNameId() else "Earthquake Test Alerts",
            NotificationManager.IMPORTANCE_HIGH
        ).apply {
            description = if (lang == DisplayLanguage.ID) {
                channelDescId()
            } else {
                "Test notifications used to verify the alert service."
            }
            enableVibration(true)
            enableLights(true)
        }
        manager.createNotificationChannel(channel)
    }

    // Indonesian branches land in B2.
    private fun channelNameId(): String = "Earthquake Test Alerts"
    private fun channelDescId(): String = "Test notifications used to verify the alert service."
    private fun testTitle(lang: DisplayLanguage): String =
        if (lang == DisplayLanguage.ID) testTitleId() else "QuakeAlert Test"
    private fun testBody(lang: DisplayLanguage): String =
        if (lang == DisplayLanguage.ID) testBodyId() else "This is a test alert. The notification service is working!"
    private fun testTitleId(): String = "QuakeAlert Test"
    private fun testBodyId(): String = "This is a test alert. The notification service is working!"

    /**
     * Displays the test notification. Returns false when the runtime
     * POST_NOTIFICATIONS permission has not been granted (API 33+), letting the
     * caller prompt the user instead of silently failing.
     */
    fun showTestAlert(context: Context, lang: DisplayLanguage = DisplayLanguage.EN): Boolean {
        ensureChannel(context, lang)

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            ContextCompat.checkSelfPermission(
                context,
                Manifest.permission.POST_NOTIFICATIONS
            ) != PackageManager.PERMISSION_GRANTED
        ) {
            return false
        }

        val notification = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_alert_test)
            .setContentTitle(testTitle(lang))
            .setContentText(testBody(lang))
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setCategory(NotificationCompat.CATEGORY_ALARM)
            .setAutoCancel(true)
            .build()

        NotificationManagerCompat.from(context).notify(NOTIFICATION_ID, notification)
        return true
    }
}
