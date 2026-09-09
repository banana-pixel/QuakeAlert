package id.web.quakealert.service

import android.util.Log
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import id.web.quakealert.data.AppSettingsRepository
import id.web.quakealert.data.network.QuakeNetwork
import id.web.quakealert.data.network.mapper.toOperatorUpdateOrNull
import id.web.quakealert.data.network.mapper.toWsAlertMessageOrNull
import id.web.quakealert.domain.resolveDisplayLanguage
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking

/**
 * Background delivery path for earthquake alerts.
 *
 * This is the part that makes the app an early-warning system rather than a
 * dashboard: the WebSocket only carries an alert while the app is open, and an
 * earthquake does not wait for the user to open an app. The payload is data-only by
 * contract (contracts/fcm/alert_payload.json) precisely so this service is invoked
 * even when the process was killed — a `notification` block would be handled by the
 * system tray instead and never reach this code.
 *
 * The sequence is fixed and every step matters:
 *  1. parse the all-string data map into the same [WsAlertMessage] the socket
 *     produces ([toWsAlertMessageOrNull]);
 *  2. drop it if the shared [id.web.quakealert.domain.AlertDedup] has already acted
 *     on it, so a socket frame and its push copy raise one alert, not two;
 *  3. **apply [AlertGate]** — mandatory, per .clinerules/20 rule 2. The server's FCM
 *     target is a nationwide topic, so without this every device in the country
 *     sounds a siren for every tremor;
 *  4. post the full-screen notification.
 *
 * Operator announcements (`ADMIN_BROADCAST`) arrive on this same service and are
 * sorted out before any of that, onto [UpdatesNotifier]'s low-importance channel.
 * They are not alerts and share nothing with them — not the dedup key, not the gate,
 * not the channel.
 *
 * Registered in the manifest but only ever instantiated when Firebase initialised,
 * which requires an `app/google-services.json`. Without one this class is dead code
 * and the app runs WebSocket-only (docs/FIREBASE_SETUP.md).
 */
class QuakeMessagingService : FirebaseMessagingService() {

    private val network by lazy { QuakeNetwork.from(applicationContext) }

    private val settings by lazy { AppSettingsRepository(applicationContext) }

    /**
     * A rotated token, delivered whenever Firebase issues a new one — including
     * while the app is not running, which is why registration cannot live only in
     * app start.
     */
    override fun onNewToken(token: String) {
        network.applicationScope.launch {
            network.pushRegistrar.uploadToken(token)
        }
    }

    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        // Announcements are sorted out first and never reach any of the alert
        // machinery below: no dedup key shared with an event, no AlertGate, no siren.
        // The server already decided who to tell, by region
        // (server/internal/dispatch/broadcast.go).
        val update = remoteMessage.data.toOperatorUpdateOrNull()
        if (update != null) {
            // onMessageReceived runs on an FCM background thread (never the main
            // thread), so a bounded first() read for the stored language is safe.
            val lang = resolveDisplayLanguage(
                runCatching { runBlocking { settings.language.first() } }.getOrNull()
            )
            UpdatesNotifier.notify(applicationContext, update, lang)
            return
        }

        val message = remoteMessage.data.toWsAlertMessageOrNull()
        if (message == null) {
            Log.w(TAG, "push payload had no recognisable type; dropped")
            return
        }

        // The raise itself lives in AlertRaiser, shared with the local drill
        // test: parsing is push-specific, everything after it is not.
        network.applicationScope.launch {
            AlertRaiser.raiseAlertFrame(applicationContext, message)
        }
    }

    private companion object {
        const val TAG = "QuakeMessaging"
    }
}
