package id.web.quakealert.ui.warning

import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.Bundle
import android.view.WindowManager
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.lifecycle.lifecycleScope
import id.web.quakealert.device.AlertSiren
import id.web.quakealert.device.TorchController
import id.web.quakealert.domain.DisplayLanguage
import id.web.quakealert.domain.RaiseOutcomeLog
import id.web.quakealert.data.network.QuakeNetwork
import id.web.quakealert.domain.AlertType
import id.web.quakealert.service.WarningNotifier
import id.web.quakealert.ui.theme.Dimens
import id.web.quakealert.ui.theme.QuakeAlertTheme
import id.web.quakealert.ui.theme.OnboardingBackgroundBrush
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * The full-screen earthquake alert, raised from a push notification's full-screen
 * intent.
 *
 * Deliberately an Activity and not a nav route inside [id.web.quakealert.MainActivity]:
 * it has to open from a process that was not running, over the lock screen, with no
 * user interaction — and a Compose destination cannot be reached from a dead process.
 *
 * **The gate has already run before this Activity exists.** [WarningNotifier] applies
 * [id.web.quakealert.domain.AlertGate] before it builds the notification, so
 * reaching `onCreate` already means "this quake is inside the user's radius". The
 * siren therefore starts here unconditionally.
 *
 * It owns its own [AlertSiren] and [TorchController] rather than sharing
 * [WarningViewModel]'s: that ViewModel belongs to MainActivity's screen, which may
 * not exist. Both effects are torn down in [onDestroy], so a torch cannot outlive
 * the alert that turned it on.
 */
class WarningActivity : ComponentActivity() {

    private val siren by lazy { AlertSiren(this) }

    private val torch by lazy { TorchController(this) }

    private var state by mutableStateOf(WarningUiState.ActiveAlert(
        eventId = "",
        intensityValue = "",
        distanceKm = null,
        locationName = ""
    ))

    /**
     * Chrome language for the card (D-030). Frozen from the raising intent's
     * `EXTRA_LANG`, like the intensity label and location it sits beside — the
     * screen is one coherent snapshot of what the raiser decided to show, so a
     * mid-alert language switch does not re-render it. Absent or unrecognised
     * (a pre-change PendingIntent, or anything forged) degrades to EN, the same
     * default every warning component already carries.
     */
    private var lang by mutableStateOf(DisplayLanguage.EN)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        showOverLockScreen()
        enableEdgeToEdge()

        state = intent.toActiveAlert()
        lang = intent.alertLang()
        siren.start()
        observeStandDown()
        if (state.isTest) armTestAutoEnd()

        setContent {
            QuakeAlertTheme {
                Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .background(OnboardingBackgroundBrush)
                        .padding(Dimens.ScreenHorizontalPadding)
                ) {
                    ActiveAlertCard(
                        state = state,
                        onMuteClick = ::onMuteClick,
                        onSosLightClick = ::onSosLightClick,
                        onEndTestClick = ::onEndTestClicked.takeIf { state.isTest },
                        lang = lang,
                        modifier = Modifier.fillMaxSize()
                    )
                }
            }
        }
    }

    /**
     * A second alert while this screen is up (`launchMode="singleTop"`).
     *
     * Coexistence (D-020, U-011): the new event registers on the shared board
     * and takes focus with its count; older live events are untouched. Mute
     * stays per event on the shared map, so a duplicate of the *same* event
     * keeps the user's mute — silencing a siren must not be undone by a
     * redelivery of the alert that was silenced — while a different event id
     * is a new quake and starts audible again. No second Activity is created.
     */
    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        val board = QuakeNetwork.from(applicationContext).activeAlerts
        val next = intent.toActiveAlert()
        lang = intent.alertLang()
        if (next.eventId.isNotBlank()) {
            val slot = board.upsert(next.eventId, sounded = true)
            if (slot.collapsedId != null) {
                android.util.Log.i(TAG, RaiseOutcomeLog.collapsedIntoCount(slot.collapsedId))
            }
            if (slot.supersededIds.isNotEmpty()) {
                android.util.Log.i(TAG, RaiseOutcomeLog.superseded(next.eventId, slot.supersededIds))
            }
        }
        val sameEvent = next.eventId.isNotBlank() && next.eventId == state.eventId
        state = next.copy(
            // Mute is per event on the shared map: a mute set on the in-app
            // card and a duplicate arriving here never disagree. A new quake
            // reads the map (false unless muted elsewhere first).
            isMuted = if (sameEvent) state.isMuted || board.isMuted(next.eventId)
                else board.isMuted(next.eventId),
            isSosLightOn = state.isSosLightOn,
            isSosLightUnavailable = state.isSosLightUnavailable,
            extraActiveCount = board.extraActiveCount()
        )
        if (!state.isMuted) siren.start()
    }

    /**
     * Closes the screen when the server sends the all-clear for the event it
     * shows — and only then (D-020).
     *
     * Collecting the socket here also connects it, which is the point: a device woken
     * by push has no live connection, and without one the red screen would have no
     * way to ever learn the shaking is over except the user dismissing it.
     *
     * Event scoping: a resolve for another live event must not close this
     * screen, and a blank-id legacy resolve closes it only when it is the
     * single live event — otherwise an unscoped all-clear could take down the
     * wrong quake's alarm.
     */
    private fun observeStandDown() {
        lifecycleScope.launch {
            QuakeNetwork.from(applicationContext).webSocketClient.alerts.collect { message ->
                if (message.type != AlertType.EVENT_RESOLVED) return@collect
                val board = QuakeNetwork.from(applicationContext).activeAlerts
                val resolvesThis = message.eventId.isNotBlank() && message.eventId == state.eventId
                val resolvesSingle = message.eventId.isBlank() && board.totalActive() <= 1 &&
                    (board.selectedId() == null || board.selectedId() == state.eventId)
                if (resolvesThis || resolvesSingle) finish()
                else if (message.eventId.isBlank()) {
                    android.util.Log.d(
                        TAG,
                        "blank stand-down ignored; several events live"
                    )
                }
            }
        }
    }

    private fun onMuteClick() {
        val muted = !state.isMuted
        if (muted) siren.mute() else siren.unmute()
        if (state.eventId.isNotBlank()) {
            QuakeNetwork.from(applicationContext).activeAlerts.setMuted(state.eventId, muted)
        }
        state = state.copy(isMuted = muted)
    }

    /**
     * Ends a drill test from the "AKHIRI TES" / "END TEST" control: clears the
     * exact test event's notification and board slot, then closes this screen.
     * Only reachable when the shown event is a test (the control renders
     * exclusively for `isTest`), so a real alert can never take this path.
     */
    private fun onEndTestClicked() {
        WarningNotifier.clear(this, state.eventId)
        finish()
    }

    /**
     * Safety timeout for a drill test: the same cleanup as the manual control,
     * fired once after [TEST_AUTO_END_MS]. A server resolve will never arrive
     * for a synthetic event, so an un-ended test must not linger past its
     * exercise. Scoped to the Activity lifecycle: dying with the screen is the
     * correct semantic, and the manual control cancels nothing — whichever
     * fires first wins, the second is a no-op clear plus a finish of an
     * already-finishing screen.
     */
    private fun armTestAutoEnd() {
        lifecycleScope.launch {
            delay(TEST_AUTO_END_MS)
            if (state.isTest) onEndTestClicked()
        }
    }

    private fun onSosLightClick() {        if (state.isSosLightOn) {
            torch.stop()
            state = state.copy(isSosLightOn = false, isSosLightUnavailable = false)
            return
        }
        val started = torch.start(lifecycleScope)
        state = state.copy(isSosLightOn = started, isSosLightUnavailable = !started)
    }

    /**
     * Turns the screen on and shows over the keyguard.
     *
     * `setShowWhenLocked` / `setTurnScreenOn` are the API 27+ replacements for the
     * deprecated window flags, and `FLAG_KEEP_SCREEN_ON` is separate from both — it
     * keeps the display awake for the duration of the alert rather than letting the
     * usual timeout black it out mid-quake.
     */
    private fun showOverLockScreen() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O_MR1) {
            setShowWhenLocked(true)
            setTurnScreenOn(true)
        }
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
    }

    override fun onDestroy() {
        siren.release()
        torch.stop()
        super.onDestroy()
    }

    private fun Intent.toActiveAlert() = WarningUiState.ActiveAlert(
        eventId = getStringExtra(EXTRA_EVENT_ID).orEmpty(),
        intensityValue = getStringExtra(EXTRA_INTENSITY).orEmpty(),
        // -1 is the "unknown" sentinel because Intent has no nullable Int; the UI
        // renders null as "Distance unknown" rather than inventing a number.
        distanceKm = getIntExtra(EXTRA_DISTANCE_KM, UNKNOWN_DISTANCE).takeIf {
            it != UNKNOWN_DISTANCE
        },
        locationName = getStringExtra(EXTRA_LOCATION_NAME).orEmpty(),
        // Defaults to false, so an intent built before this extra existed — or one
        // forged by anything else on the device — raises an ordinary alert rather
        // than a screen that tells the user to ignore it.
        isTest = getBooleanExtra(EXTRA_IS_TEST, false),
        // Same safe default as isTest: a pre-change PendingIntent, or anything
        // forged, raises an ordinary alert rather than claiming local trust.
        isLocalWarning = getBooleanExtra(EXTRA_IS_LOCAL_WARNING, false),
        // Coexisting live events beyond the one shown (D-020); 0 preserves the
        // single-event card exactly.
        extraActiveCount = getIntExtra(EXTRA_ACTIVE_COUNT, 0).coerceAtLeast(0)
    )

    /**
     * Chrome language frozen from the raising intent (D-030). Absent or
     * unrecognised degrades to EN — the pre-change behaviour — never to a mix.
     */
    private fun Intent.alertLang(): DisplayLanguage =
        DisplayLanguage.fromTagOrNull(getStringExtra(EXTRA_LANG)) ?: DisplayLanguage.EN

    companion object {
        private const val TAG = "WarningActivity"
        /**
         * Safety timeout for a drill test (one minute): long enough to verify
         * the lock-screen wake, siren and controls, short enough that a
         * forgotten test cleans itself up. Manual "AKHIRI TES" takes the same
         * path sooner.
         */
        const val TEST_AUTO_END_MS = 60_000L
        private const val EXTRA_EVENT_ID = "event_id"
        private const val EXTRA_INTENSITY = "intensity_value"
        private const val EXTRA_DISTANCE_KM = "distance_km"
        private const val EXTRA_LOCATION_NAME = "location_name"
        private const val EXTRA_IS_TEST = "is_test"
        private const val EXTRA_IS_LOCAL_WARNING = "is_local_warning"
        private const val EXTRA_ACTIVE_COUNT = "active_count"
        private const val EXTRA_LANG = "lang_tag"
        private const val UNKNOWN_DISTANCE = -1

        /**
         * Intent for the alert screen.
         *
         * `NEW_TASK` and `CLEAR_TOP` are both required: the notification launches this
         * from outside any task of ours, and a stale copy left behind by an earlier
         * quake must not sit under the new one.
         *
         * @param isTest marks a drill, which adds the "TEST" badge to the card. Only
         *   ever true on a debug build; see [WarningUiState.ActiveAlert.isTest].
         * @param isLocalWarning marks an Admin Node local warning (D-036), which
         *   renders the local-warning title and badge. See
         *   [WarningUiState.ActiveAlert.isLocalWarning].
         * @param langTag the chrome language tag frozen at raise time (D-030);
         *   null degrades to EN in [alertLang].
         */
        fun intent(
            context: Context,
            eventId: String,
            intensityValue: String,
            locationName: String,
            distanceKm: Int?,
            isTest: Boolean = false,
            isLocalWarning: Boolean = false,
            activeCount: Int = 0,
            langTag: String? = null
        ): Intent = Intent(context, WarningActivity::class.java).apply {
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP)
            putExtra(EXTRA_EVENT_ID, eventId)
            putExtra(EXTRA_INTENSITY, intensityValue)
            putExtra(EXTRA_LOCATION_NAME, locationName)
            putExtra(EXTRA_DISTANCE_KM, distanceKm ?: UNKNOWN_DISTANCE)
            putExtra(EXTRA_IS_TEST, isTest)
            putExtra(EXTRA_IS_LOCAL_WARNING, isLocalWarning)
            putExtra(EXTRA_ACTIVE_COUNT, activeCount.coerceAtLeast(0))
            putExtra(EXTRA_LANG, langTag)
        }
    }
}
