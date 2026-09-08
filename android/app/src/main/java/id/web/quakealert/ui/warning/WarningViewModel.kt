package id.web.quakealert.ui.warning

import android.app.Application
import android.util.Log
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import id.web.quakealert.data.AppSettingsRepository
import id.web.quakealert.data.UnitSystem
import id.web.quakealert.data.network.QuakeNetwork
import id.web.quakealert.data.network.mapper.QuakeFormat
import id.web.quakealert.data.network.mapper.intensityBannerLabel
import id.web.quakealert.data.network.mapper.intensityValueLabel
import id.web.quakealert.data.network.mapper.toHistoryItem
import id.web.quakealert.device.AlertSiren
import id.web.quakealert.device.DeviceCountry
import id.web.quakealert.device.TorchController
import id.web.quakealert.device.canPostNotifications
import id.web.quakealert.domain.ActiveAlertBoard
import id.web.quakealert.domain.AlertGate
import id.web.quakealert.domain.AlertType
import id.web.quakealert.domain.DisplayLanguage
import id.web.quakealert.domain.EarthquakeEvent
import id.web.quakealert.domain.EmergencyContacts
import id.web.quakealert.domain.EventState
import id.web.quakealert.domain.EventStatus
import id.web.quakealert.domain.RaiseOutcomeLog
import id.web.quakealert.domain.SafetyPolicy
import id.web.quakealert.domain.UserLocation
import id.web.quakealert.domain.WsAlertMessage
import id.web.quakealert.domain.distanceKmTo
import id.web.quakealert.domain.resolveDisplayLanguage
import id.web.quakealert.domain.standDownCopyFor
import id.web.quakealert.domain.unconfirmedActivityLabel
import id.web.quakealert.service.WarningNotifier
import id.web.quakealert.ui.common.errorCopy
import id.web.quakealert.ui.history.QuakeHistoryItem
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.flow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.temporal.ChronoUnit
import java.util.Locale
import kotlin.math.roundToInt

/**
 * The two delivery facts the Protection Status overlay reports: the user's alert
 * switch and the OS notification grant. A plain pair rather than part of the sealed
 * [WarningUiState] — see [WarningViewModel.protectionFacts].
 */
data class ProtectionFacts(
    val alertsEnabled: Boolean = true,
    val notificationsPermitted: Boolean = true
)

/**
 * Hosts the [WarningUiState] for the Warning screen and exposes it as a
 * [StateFlow] following unidirectional data flow.
 *
 * Two sources feed it, and the split matters:
 *  - **REST**, once per load: the newest event from `GET /api/v1/events` decides
 *    which state the screen opens in, so a cold start *during* a quake opens on the
 *    emergency screen instead of a calm banner that waits for the next push.
 *  - **WebSocket / FCM**, continuously: `EARTHQUAKE_ALERT` raises
 *    [WarningUiState.ActiveAlert], `EVENT_RESOLVED` stands it down,
 *    `EARTHQUAKE_ADVISORY` only nudges the idle banner's possibility read.
 *
 * Both realtime channels share [onAlertReceived], because the FCM data payload is
 * the same shape as the socket frame (contracts/fcm/alert_payload.json) — one entry
 * point is what keeps a push-delivered alert and a socket-delivered alert from
 * producing two different screens.
 *
 * The ViewModel also owns the alert's two hardware effects, [AlertSiren] and
 * [TorchController], and that ownership is deliberate: both must outlive
 * recomposition and both must be torn down exactly once, on stand-down or in
 * [onCleared]. Driving them from the composable instead would tie a burning torch
 * to the lifetime of a composition.
 *
 * The persisted [UnitSystem] from [AppSettingsRepository] is folded into every
 * emission so the emergency card's proximity read, the detail overlay's "Distance
 * from you" row and the share text all render the unit the user picked in Settings.
 *
 * Sharing is deliberately absent here: firing `Intent.ACTION_SEND` needs a
 * `Context`, not app state, so it lives in [WarningRoute] alongside the other
 * composition-local work.
 */
class WarningViewModel(application: Application) : AndroidViewModel(application) {

    private val repository = AppSettingsRepository(application)

    /**
     * The two facts the Protection Status overlay states about delivery: the user's
     * own switch and the OS notification grant. Collected into plain state rather
     * than folded into [uiState] because they are read only when the overlay opens,
     * and threading them through the sealed hierarchy would touch every variant for
     * a value nothing else on the screen renders.
     */
    val protectionFacts: StateFlow<ProtectionFacts> = combine(
        repository.notificationsEnabled,
        flow { emit(getApplication<Application>().canPostNotifications()) }
    ) { enabled, permitted -> ProtectionFacts(enabled, permitted) }
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), ProtectionFacts())

    private val network = QuakeNetwork.from(application)

    private val apiClient = network.apiClient

    private val siren = AlertSiren(application)

    private val torch = TorchController(application)

    private val _uiState = MutableStateFlow<WarningUiState>(
        WarningUiState.Idle(isLoading = true)
    )

    val uiState: StateFlow<WarningUiState> = combine(
        repository.unitSystem,
        _uiState
    ) { unit, state -> state.withUnitSystem(unit) }.stateIn(
        scope = viewModelScope,
        started = SharingStarted.WhileSubscribed(5_000),
        initialValue = WarningUiState.Idle(isLoading = true)
    )

    /**
     * Language user strings are rendered in, following the stored override then
     * the system locale ([resolveDisplayLanguage]). Screens collect this to pass
     * down to components that compute copy from state; the state itself keeps
     * holding strings the ViewModel already rendered with the same language.
     */
    val displayLang: StateFlow<DisplayLanguage> = repository.language
        .map { resolveDisplayLanguage(it) }
        .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), DisplayLanguage.EN)

    /** Suspend twin of [displayLang] for builders that run off-collection. */
    private suspend fun currentLang(): DisplayLanguage =
        resolveDisplayLanguage(runCatching { repository.language.first() }.getOrNull())

    /**
     * Detail payload behind the idle banner's "SEE DETAILS" capsule, built from
     * whichever source last described a quake. Held outside [WarningUiState] because
     * it is not rendered until the overlay opens — putting it in the state would
     * recompose the screen for a value nothing is showing.
     */
    private var activeAlertDetails: QuakeHistoryItem? = null

    /**
     * Rendered cards for live coexistence entries (D-020), keyed by event_id
     * and pruned to board membership on every mutation. The board owns policy
     * (cap, selection, mute); this map owns presentation built from the frame
     * that raised each event.
     */
    private val liveCards = LinkedHashMap<String, WarningUiState.ActiveAlert>()

    /**
     * Last device position seen by either load path.
     *
     * Cached because [onSeeDetailsClicked] is called from a tap, not a coroutine,
     * and the Earthquake Possibility card it raises needs a coordinate to centre its
     * basemap on. Reading the store is suspending, so the alternative is launching a
     * coroutine on every tap to fetch something that changes far more slowly than
     * the user opens the card.
     */
    private var lastKnownLocation: UserLocation? = null

    /**
     * Last computed "Recent Seismic Activity" read, cached for the same reason
     * [lastKnownLocation] is: [onSeeDetailsClicked] runs on a tap, and the card it
     * raises must open with real numbers rather than a spinner. Recomputed by every
     * load, so it is never older than the screen around it.
     */
    private var recentActivity: RecentSeismicActivity = RecentSeismicActivity()

    init {
        load()
        observeAlerts()
    }

    /**
     * Re-runs the alert-feed load after a failure, from
     * [id.web.quakealert.ui.common.QuakeErrorState]'s "Retry" action.
     */
    fun onRetry() {
        load()
    }

    /**
     * Single entry point into the loading → content / error state machine, used by
     * both the initial load and [onRetry].
     *
     * Every write back into [_uiState] is guarded on the state still being
     * [WarningUiState.Idle]. A socket frame can raise a real emergency while this
     * request is in flight, and neither a stale "no recent earthquake" snapshot nor
     * a network error may take that screen down.
     */
    private fun load() {
        viewModelScope.launch {
            _uiState.update { state ->
                if (state is WarningUiState.Idle) {
                    state.copy(isLoading = true, isError = false, errorCopy = null)
                } else {
                    state
                }
            }
            try {
                when (val outcome = fetchWarning()) {
                    is LoadOutcome.Emergency -> raise(outcome.alert)
                    is LoadOutcome.DistantEmergency -> _uiState.update { state ->
                        if (state is WarningUiState.Idle) {
                            state.copy(
                                banner = outcome.snapshot.banner,
                                sectionTitle = outcome.snapshot.sectionTitle,
                                tips = outcome.snapshot.tips,
                                isLoading = false
                            )
                        } else {
                            state
                        }
                    }
                    is LoadOutcome.Resting -> _uiState.update { state ->
                        if (state is WarningUiState.Idle) {
                            state.copy(
                                banner = outcome.snapshot.banner,
                                sectionTitle = outcome.snapshot.sectionTitle,
                                tips = outcome.snapshot.tips,
                                isLoading = false
                            )
                        } else {
                            state
                        }
                    }
                }
            } catch (cancellation: CancellationException) {
                // Never treat scope cancellation as a load failure — rethrow so the
                // coroutine machinery sees it and the screen keeps its last state.
                throw cancellation
            } catch (throwable: Throwable) {
                // Logged rather than shown: the raw cause never reaches the screen,
                // so this is the only place it survives for a bug report.
                Log.w(TAG, "could not load the alert feed", throwable)
                val lang = currentLang()
                _uiState.update { state ->
                    if (state is WarningUiState.Idle) {
                        state.copy(
                            isLoading = false,
                            isError = true,
                            // No filter reaches this feed, so a rejected request has
                            // nothing here for the user to relax.
                            errorCopy = errorCopy(throwable, lang = lang)
                        )
                    } else {
                        state
                    }
                }
            }
        }
    }

    /**
     * Resolves the opening state from the newest stored event.
     *
     * A single event is enough: the feed is sorted `created_at DESC`, and only an
     * unresolved quake inside the recent window can justify opening on the emergency
     * screen. An unresolved one that has aged out of that window still gets the
     * recent-quake banner; a resolved one, or none at all, means the resting state.
     */
    private suspend fun fetchWarning(): LoadOutcome {
        val lang = currentLang()
        val locale = lang.locale()
        val latest = apiClient.fetchEvents(limit = 1).getOrThrow().firstOrNull()
        // Read before the resting early-return: the resting state is precisely the
        // one that offers the Earthquake Possibility card, so the position must be
        // cached on the quiet path too, not only when there is a quake to gate.
        val userLocation = apiClient.currentUserLocation()
        lastKnownLocation = userLocation
        if (latest == null || latest.status != EventStatus.HAPPENING) {
            activeAlertDetails = null
            recentActivity = fetchRecentActivity(userLocation, lang)
            return LoadOutcome.Resting(restingSnapshot(recentActivity, lang))
        }

        activeAlertDetails = latest.toHistoryItem(userLocation, locale = locale)

        // Unresolved, but older than the window the siren answers for. It is still an
        // open quake, so it gets the recent-quake banner — which is exactly what that
        // banner is for — rather than being represented by nothing but a number in the
        // activity count, which is what the resting branch used to do with it.
        if (!latest.isOngoing()) {
            return LoadOutcome.DistantEmergency(
                activeSnapshot(
                    intensityLabel = latest.intensityBannerLabel(locale),
                    timeAgo = QuakeFormat.relativeTime(latest.createdAt, Instant.now(), locale),
                    lang = lang
                )
            )
        }

        // The same gate the realtime path uses. A cold start during a quake on the
        // other side of the country must open on the banner, not the siren.
        val decision = AlertGate.decide(
            userLocation = userLocation,
            centroidLat = latest.latitude,
            centroidLon = latest.longitude,
            mmi = latest.mmi,
            pgaGal = latest.pgaGal
        )
        if (!decision.shouldAlarm) {
            return LoadOutcome.DistantEmergency(
                activeSnapshot(
                    intensityLabel = latest.intensityBannerLabel(locale),
                    timeAgo = QuakeFormat.relativeTime(latest.createdAt, Instant.now(), locale),
                    lang = lang
                )
            )
        }
        return LoadOutcome.Emergency(latest.toActiveAlert(userLocation, locale))
    }

    /**
     * Counts what the network has actually recorded near the user in the last
     * [ACTIVITY_WINDOW_DAYS] days, for the resting banner's read and the "Recent
     * Seismic Activity" card behind it.
     *
     * Only run on the resting path: that is the one state whose banner offers this
     * card, so a screen opening on a live quake never pays for a second request.
     *
     * The radius is the fixed [SafetyPolicy.ALERT_RADIUS_KM] rather than the History
     * filter's browse radius. "Near you" here has to mean the area alerts are issued
     * for; a number the user could widen by changing an unrelated filter would not be
     * comparable to anything.
     *
     * Two failure modes, kept distinct because they need different sentences:
     *  - **no position**: `fetchEvents` drops `range_km` when it has no centre, so the
     *    request would silently answer for the whole country. Refused rather than sent
     *    — a national count labelled "nearby" is worse than admitting the gap.
     *  - **request failed**: reported as unavailable, never as zero. "No quakes near
     *    you" is exactly the reading a life-safety app must not invent.
     */
    private suspend fun fetchRecentActivity(center: UserLocation?, lang: DisplayLanguage = DisplayLanguage.EN): RecentSeismicActivity {
        if (center == null) return RecentSeismicActivity()

        val label = QuakeFormat.coordinates(center.latitude, center.longitude)
        val since = Instant.now().minus(ACTIVITY_WINDOW_DAYS.toLong(), ChronoUnit.DAYS)
        val events = apiClient.fetchEvents(
            limit = ACTIVITY_PAGE_LIMIT,
            rangeKm = SafetyPolicy.ALERT_RADIUS_KM,
            center = center,
            since = since
        ).getOrElse {
            return RecentSeismicActivity(
                locationLabel = label,
                availability = ActivityAvailability.UNAVAILABLE,
                latitude = center.latitude,
                longitude = center.longitude
            )
        }

        val now = Instant.now()
        val locale = lang.locale()
        // The feed is sorted created_at DESC, so the newest is the head — but the
        // strongest has to be searched for: intensity and recency are unrelated.
        val newest = events.firstOrNull()
        val strongest = events.maxByOrNull { it.pgaGal }

        return RecentSeismicActivity(
            locationLabel = label,
            availability = ActivityAvailability.MEASURED,
            eventCount = events.size,
            // A full page is a floor, not a tally: there may be more behind it, and
            // printing the page size as the count would understate a busy month.
            isCountCapped = events.size >= ACTIVITY_PAGE_LIMIT,
            mostRecent = newest?.let {
                "${it.intensityValueLabel(locale)}, ${QuakeFormat.relativeTime(it.createdAt, now, locale)}"
            },
            strongest = strongest?.let {
                "${it.intensityValueLabel(locale)}, ${QuakeFormat.pga(it.pgaGal)}"
            },
            latitude = center.latitude,
            longitude = center.longitude
        )
    }

    /**
     * Collects the realtime stream for the ViewModel's lifetime.
     *
     * The flow reconnects internally, so a dropped socket is not an error state
     * here — a genuinely unreachable alert network already shows through the
     * [load] failure path.
     */
    private fun observeAlerts() {
        viewModelScope.launch {
            network.webSocketClient.alerts.collect { message -> onAlertReceived(message) }
        }
    }

    /**
     * Applies one realtime frame, from either the WebSocket or an FCM data payload.
     *
     * Public because push delivery arrives outside this ViewModel's collection: a
     * `FirebaseMessagingService` parses the payload into the same [WsAlertMessage]
     * and hands it here, so background push and foreground socket converge on one
     * state machine instead of two.
     *
     * [WsAlertMessage.isActionable] gates the alert path because the stream replays its
     * last frame to a new subscriber: without the guard, re-entering the screen
     * hours later would resurrect a finished quake as an active emergency.
     * Sender-declared validity (D-018) applies when present, the legacy recent
     * window otherwise.
     */
    fun onAlertReceived(message: WsAlertMessage) {
        viewModelScope.launch {
            when (message.type) {
                AlertType.EARTHQUAKE_ALERT -> {
                    if (message.isActionable()) raiseAlert(message)
                    else Log.i(
                        TAG,
                        RaiseOutcomeLog.expired(message.eventId, message.validityMs > 0)
                    )
                }

                // Deliberately *not* the emergency screen: an advisory is 1–2 nodes
                // and unconfirmed, and escalating it would train users to ignore the
                // real thing. It also must never *downgrade* a live alert, hence the
                // Idle guard.
                AlertType.EARTHQUAKE_ADVISORY -> {
                    val lang = currentLang()
                    _uiState.update { state ->
                        if (state is WarningUiState.Idle) {
                            state.copy(
                                banner = advisoryBanner(message, recentActivity.bannerLabel(lang), lang),
                                isLoading = false
                            )
                        } else {
                            state
                        }
                    }
                }

                // Both RESOLVED and CANCELLED arrive as this one type; the state is
                // what tells an all-clear from a withdrawn report, and it changes
                // nothing about the stand-down itself — only what the user is told.
                AlertType.EVENT_RESOLVED -> standDown(message.eventId, message.eventState)
            }
        }
    }

    /**
     * Raises the emergency screen for a confirmed alert — but only after the
     * distance gate agrees.
     *
     * The gate is not a nicety: `/ws` and the FCM topic are broadcast channels, so
     * this device receives every event in the country. Without [AlertGate] a tremor
     * 800 km away sounds the same siren as one under the user's building, which is
     * how a life-safety app teaches its users to ignore it.
     *
     * A gated-out alert is not discarded — it becomes the idle "Recent Earthquake"
     * banner, with the details still available behind "SEE DETAILS". The user is
     * informed, just not woken.
     */
    private suspend fun raiseAlert(message: WsAlertMessage) {
        val lang = currentLang()
        val locale = lang.locale()
        val userLocation: UserLocation? = apiClient.currentUserLocation()
        lastKnownLocation = userLocation
        activeAlertDetails = message.toHistoryItem(userLocation, locale = locale)

        val decision = AlertGate.decide(
            userLocation = userLocation,
            centroidLat = message.centroidLat,
            centroidLon = message.centroidLon,
            mmi = message.mmi,
            pgaGal = message.pgaGal
        )

        if (!decision.shouldAlarm) {
            Log.i(TAG, RaiseOutcomeLog.gatedOut(message.eventId, decision.reason))
            _uiState.update { state ->
                if (state is WarningUiState.Idle) {
                    val snapshot = distantSnapshot(message, lang)
                    state.copy(
                        banner = snapshot.banner,
                        sectionTitle = snapshot.sectionTitle,
                        tips = snapshot.tips,
                        isLoading = false,
                        isError = false,
                        errorCopy = null
                    )
                } else {
                    state
                }
            }
            return
        }

        // Cross-channel de-duplication. A false result means this exact event was
        // already acted on — by the push handler (so WarningActivity is up with its
        // own siren) or by an earlier instance of this ViewModel (so the socket is
        // replaying its last frame to a re-entered screen). Either way the visual
        // alert is still correct and still raised; what must not happen is a second
        // siren starting over the first, or an old alert becoming audible again.
        val alreadyRaised = !network.alertDedup.markIfNew(message)

        // Coexistence (D-020, U-011): register on the shared board before
        // rendering. A newer event takes focus (and the siren, unless this exact
        // event was already acted on); older live events persist silently and are
        // counted on the card rather than overwritten without a trace.
        val board = network.activeAlerts
        val slot = board.upsert(message.eventId, sounded = !alreadyRaised)
        liveCards[message.eventId] = message.toActiveAlert(userLocation, locale)
        if (slot.collapsedId != null) {
            Log.i(TAG, RaiseOutcomeLog.collapsedIntoCount(slot.collapsedId))
        }
        if (slot.supersededIds.isNotEmpty()) {
            Log.i(TAG, RaiseOutcomeLog.superseded(message.eventId, slot.supersededIds))
        }

        // ActiveAlert.proximityLabel already renders "Distance unknown" when the gate
        // failed open on a missing position, so nothing extra is needed for that case.
        renderSelected(board)
        if (!alreadyRaised) {
            raiseSound(board)
        }
    }

    /**
     * Renders the board's selected event. Mute comes from the shared per-event
     * map (D-020), so muting here, in the Activity, or across a recreation of
     * this ViewModel never disagrees about one event. Torch state stays local
     * to this screen: it describes hardware, not the quake.
     */
    private fun renderSelected(board: ActiveAlertBoard) {
        val selectedId = board.selectedId() ?: return
        if (!liveCards.containsKey(selectedId)) return
        val current = _uiState.value as? WarningUiState.ActiveAlert
        _uiState.update { state ->
            liveCards.getValue(selectedId).copy(
                isMuted = board.isMuted(selectedId),
                isSosLightOn = current?.isSosLightOn ?: torch.isOn,
                isSosLightUnavailable = current?.isSosLightUnavailable ?: false,
                extraActiveCount = board.extraActiveCount(),
                unitSystem = state.unitSystem
            )
        }
    }

    /**
     * Starts the siren for the selected event unless its per-event mute says
     * otherwise, and logs the outcome (D-019): event_id and outcome only.
     */
    private fun raiseSound(board: ActiveAlertBoard) {
        val selectedId = board.selectedId() ?: return
        // Idempotent, and a no-op while a carried-over mute is in effect.
        val sirenStarted = !board.isMuted(selectedId)
        Log.i(TAG, RaiseOutcomeLog.shown(selectedId, sirenStarted))
        if (sirenStarted) {
            siren.start()
        }
    }

    /**
     * Switches the screen to [WarningUiState.ActiveAlert] and starts the siren.
     *
     * Cold-start entry point (an unresolved quake found in the history feed on
     * launch). Like the live path it registers on the shared board first, so a
     * cold-started emergency and a socket-raised one coexist under one policy
     * instead of each believing it is the only quake.
     *
     * Re-raising the *same* event preserves the user's mute choice — silencing
     * a siren must not be undone by a duplicate of the alert that was silenced.
     * A genuinely new `event_id` starts audible again, because the previous
     * quake's mute says nothing about this one.
     */
    private fun raise(alert: WarningUiState.ActiveAlert, startSiren: Boolean = true) {
        if (alert.eventId.isBlank()) return
        val board = network.activeAlerts
        val slot = board.upsert(alert.eventId, sounded = startSiren)
        liveCards[alert.eventId] = alert
        if (slot.collapsedId != null) {
            Log.i(TAG, RaiseOutcomeLog.collapsedIntoCount(slot.collapsedId))
        }
        if (slot.supersededIds.isNotEmpty()) {
            Log.i(TAG, RaiseOutcomeLog.superseded(alert.eventId, slot.supersededIds))
        }
        // Mute carries per event (D-020): a stored mute survives re-raise of the
        // same event, while a new event starts audible.
        val muted = board.isMuted(alert.eventId)
        renderSelected(board)
        // Logged so a raised alarm and a silently gated-out one are never
        // indistinguishable in logcat again (D-019, U-013): event_id and outcome
        // only, never position.
        val sirenStarted = startSiren && !muted
        Log.i(TAG, RaiseOutcomeLog.shown(alert.eventId, sirenStarted))
        if (sirenStarted) {
            siren.start()
        }
    }

    /**
     * Stands down one event (D-020): routes once through [WarningNotifier.clear],
     * which owns board mutation and notification cancellation, then renders what
     * remains. A promoted event renders SILENTLY — never auto-sounded (approved
     * D-020 edge). Siren and torch stop only when something actually ended; an
     * unknown id while others live changes nothing.
     */
    private suspend fun standDown(eventId: String = "", eventState: EventState? = null) {
        val lang = currentLang()
        val board = network.activeAlerts
        val result = WarningNotifier.clear(getApplication(), eventId)
        if (!result.removedAny) return
        liveCards.keys.retainAll(board.fullEntries().map { it.eventId }.toSet())
        siren.release()
        torch.stop()
        if (board.selectedId() == null) {
            activeAlertDetails = null
            val snapshot = restingSnapshot(recentActivity, lang)
            // A withdrawn report is not an ended earthquake, and the banner is the one
            // surface that can say which happened. A build that does not recognise the
            // state falls back to all-clear wording, exactly as before.
            val copy = standDownCopyFor(eventState, lang)
            _uiState.update { state ->
                WarningUiState.Idle(
                    banner = SeismicActivityBanner(
                        title = copy.title,
                        activityLabel = copy.detail
                    ),
                    sectionTitle = snapshot.sectionTitle,
                    tips = snapshot.tips,
                    unitSystem = state.unitSystem
                )
            }
            return
        }
        // Still live: the promoted entry renders without sound by contract.
        renderSelected(board)
    }

    /**
     * Silences or re-enables the siren from the card's "MUTE ALERT" control (Figma
     * node 1:1073).
     *
     * A toggle rather than a one-way mute, and the visual alert is untouched either
     * way: the user asked for quiet, not to stop being warned.
     */
    fun onMuteClick() {
        val current = _uiState.value as? WarningUiState.ActiveAlert ?: return
        // Per-event mute (D-020) on the shared board: muting here agrees with
        // the Activity and survives this ViewModel's recreation.
        val muted = !network.activeAlerts.isMuted(current.eventId)
        network.activeAlerts.setMuted(current.eventId, muted)
        if (muted) siren.mute() else siren.unmute()
        _uiState.update { state ->
            if (state is WarningUiState.ActiveAlert) state.copy(isMuted = muted) else state
        }
    }

    /**
     * Toggles the SOS torch strobe from the card's "SOS LIGHT" control (Figma node
     * 1:1076).
     *
     * No runtime permission is requested because none exists for this:
     * `CameraManager.setTorchMode` is permission-free, and the app deliberately does
     * not declare `CAMERA` (see [TorchController]). What can fail is availability —
     * no flash unit, or the camera held by another app — and that failure is written
     * into the state so the control can say the light did not come on rather than
     * looking engaged over a dark LED.
     */
    fun onSosLightClick() {
        val current = _uiState.value as? WarningUiState.ActiveAlert ?: return

        if (current.isSosLightOn) {
            torch.stop()
            _uiState.update { state ->
                if (state is WarningUiState.ActiveAlert) {
                    state.copy(isSosLightOn = false, isSosLightUnavailable = false)
                } else {
                    state
                }
            }
            return
        }

        val started = torch.start(viewModelScope)
        _uiState.update { state ->
            if (state is WarningUiState.ActiveAlert) {
                state.copy(isSosLightOn = started, isSosLightUnavailable = !started)
            } else {
                state
            }
        }
    }

    /**
     * Raises the overlay behind the idle banner's "SEE DETAILS" capsule, dispatched
     * by the current variant: [ActiveQuakeBanner] opens the "Recent Earthquake"
     * event detail (Figma 124:1192), [SeismicActivityBanner] opens the "Recent
     * Seismic Activity" card (Figma 124:1605). Keeping the dispatch here means the screen
     * never needs to know which overlay a tap raises.
     */
    fun onSeeDetailsClicked() {
        val idle = _uiState.value as? WarningUiState.Idle ?: return
        when (idle.banner) {
            is ActiveQuakeBanner -> activeAlertDetails?.let { details ->
                _uiState.update { state ->
                    if (state is WarningUiState.Idle) {
                        state.copy(selectedEventDetails = details)
                    } else {
                        state
                    }
                }
            }
            // Opens the cached read rather than firing a request behind the tap: the
            // card must appear with its numbers already in it, and every load has
            // refreshed this. A user with no fix sees the NO_POSITION copy, which is the
            // honest answer to "what is near me?" when we do not know where "me" is.
            is SeismicActivityBanner -> _uiState.update { state ->
                if (state is WarningUiState.Idle) {
                    state.copy(selectedActivity = recentActivity)
                } else {
                    state
                }
            }
        }
    }

    /**
     * Closes the "Recent Earthquake" detail overlay. Called for every dismissal
     * path — the close (X) button, a back press and a tap outside the card.
     */
    fun onDetailDismissed() {
        _uiState.update { state ->
            if (state is WarningUiState.Idle) state.copy(selectedEventDetails = null) else state
        }
    }

    /**
     * Closes the "Recent Seismic Activity" overlay. Called for every dismissal
     * path — the close (X) button, a back press and a tap outside the card.
     */
    fun onActivityDismissed() {
        _uiState.update { state ->
            if (state is WarningUiState.Idle) state.copy(selectedActivity = null) else state
        }
    }

    /**
     * Raises the "Protection Status" overlay from the banner's info affordance.
     *
     * Only from the idle state, which is the only state that has a banner: during an
     * alert the screen is the alert, and an overlay explaining the alerting rules
     * would be covering the thing it describes.
     */
    fun onProtectionStatusClicked() {
        _uiState.update { state ->
            if (state is WarningUiState.Idle) state.copy(isProtectionStatusOpen = true) else state
        }
    }

    /**
     * Closes the "Protection Status" overlay. Called for every dismissal path — the
     * close (X) button, a back press and a tap outside the card.
     */
    fun onProtectionStatusDismissed() {
        _uiState.update { state ->
            if (state is WarningUiState.Idle) state.copy(isProtectionStatusOpen = false) else state
        }
    }

    /**
     * Opens the "Emergency Steps & Contacts" overlay, resolving its two variable
     * parts now rather than holding them all the time.
     *
     * The numbers come from the country the phone is *currently* attached to, so a
     * list resolved at startup would be the wrong country's for anyone who has since
     * crossed a border or landed. The position line comes from the last sync, which
     * is the same fact the maps show; it is null on a first launch and the overlay
     * says so rather than printing a placeholder coordinate to read to a dispatcher.
     */
    fun onEmergencyClicked() {
        val info = EmergencyInfoState(
            numbers = EmergencyContacts.forCountry(DeviceCountry.resolve(getApplication())),
            coordinatesLabel = lastKnownLocation?.let {
                QuakeFormat.coordinates(it.latitude, it.longitude)
            }
        )
        _uiState.update { state ->
            if (state is WarningUiState.Idle) state.copy(emergencyInfo = info) else state
        }
    }

    /**
     * Closes the emergency overlay. Called for every dismissal path — the close (X)
     * button, a back press and a tap outside the card.
     */
    fun onEmergencyInfoDismissed() {
        _uiState.update { state ->
            if (state is WarningUiState.Idle) state.copy(emergencyInfo = null) else state
        }
    }

    /**
     * Releases both hardware effects. `viewModelScope` is cancelled around this
     * point, which would cancel the strobe loop anyway — but the torch is switched
     * off explicitly rather than left to that ordering, because a leaked LED is the
     * one failure here the user cannot recover from inside the app.
     */
    override fun onCleared() {
        siren.release()
        torch.stop()
        super.onCleared()
    }

    /** What a REST load resolved to: an emergency, or an idle snapshot. */
    private sealed interface LoadOutcome {
        data class Emergency(val alert: WarningUiState.ActiveAlert) : LoadOutcome

        /**
         * An ongoing quake that failed the distance gate: shown as a recent-quake
         * banner rather than the emergency screen.
         */
        data class DistantEmergency(val snapshot: WarningSnapshot) : LoadOutcome
        data class Resting(val snapshot: WarningSnapshot) : LoadOutcome
    }

    /**
     * The three fields an idle load resolves together. Bundled so the banner
     * variant, its section title and its tip set can never be applied piecemeal
     * and leave the screen showing aftershock tips under a resting banner.
     */
    private data class WarningSnapshot(
        val banner: WarningBanner,
        val sectionTitle: String,
        val tips: List<PreparednessTip>
    )

    private companion object {
        const val TAG = "WarningViewModel"

        /** Copy for the idle banner variants, matching the design (Figma 124:1297 / 124:1426). */
        fun titleActive(lang: DisplayLanguage = DisplayLanguage.EN): String =
            if (lang == DisplayLanguage.ID) titleActiveId() else "Recent Earthquake Alert"
        fun sectionActive(lang: DisplayLanguage = DisplayLanguage.EN): String =
            if (lang == DisplayLanguage.ID) sectionActiveId() else "Stay alert for aftershocks"
        fun sectionResting(lang: DisplayLanguage = DisplayLanguage.EN): String =
            if (lang == DisplayLanguage.ID) sectionRestingId() else "Stay prepared for an earthquake"

        // Indonesian branches land in B2.
        private fun titleActiveId(): String = "Peringatan Gempa Terkini"
        private fun sectionActiveId(): String = "Tetap waspada terhadap gempa susulan"
        private fun sectionRestingId(): String = "Tetap siap menghadapi gempa bumi"

        /**
         * How many events one activity query may return. A page rather than a true
         * count: the endpoint pages, and 100 recorded events inside 200 km in a month
         * is far past the point where an exact number tells the user anything the
         * "100+" reading does not.
         */
        const val ACTIVITY_PAGE_LIMIT = 100

        fun activeSnapshot(
            intensityLabel: String,
            timeAgo: String,
            lang: DisplayLanguage = DisplayLanguage.EN
        ) = WarningSnapshot(
            banner = ActiveQuakeBanner(
                title = titleActive(lang),
                intensityLabel = intensityLabel,
                timeAgo = timeAgo
            ),
            sectionTitle = sectionActive(lang),
            tips = activeQuakeTips(lang)
        )

        /** A confirmed alert that the distance gate kept off the emergency screen. */
        fun distantSnapshot(message: WsAlertMessage, lang: DisplayLanguage = DisplayLanguage.EN) = activeSnapshot(
            intensityLabel = message.intensityBannerLabel(lang.locale()),
            timeAgo = QuakeFormat.relativeTime(
                Instant.ofEpochMilli(message.timestampMs),
                Instant.now(),
                lang.locale()
            ),
            lang = lang
        )

        fun restingSnapshot(activity: RecentSeismicActivity, lang: DisplayLanguage = DisplayLanguage.EN) = WarningSnapshot(
            // Both halves come from the activity: the headline has to agree with the
            // line under it, and only the activity knows whether "No Recent
            // Earthquake" is a measured fact or an unmeasured guess.
            banner = SeismicActivityBanner(
                title = activity.bannerTitle(lang),
                activityLabel = activity.bannerLabel(lang)
            ),
            sectionTitle = sectionResting(lang),
            tips = noActiveQuakeTips(lang)
        )

        /**
         * Idle banner shown while an unconfirmed tremor is being evaluated. Same
         * variant as the resting banner, so the layout is untouched — only the
         * read-out changes.
         *
         * When the frame says UNCONFIRMED, the read-out describes THIS tremor instead
         * of the 30-day activity count: what the user needs to know is that one or two
         * stations are shaking and that separated stations have not confirmed it. It
         * must not read as a confirmation, and it may not imply magnitude, epicentre
         * or an arrival time — none of which this network estimates (server §7.4,
         * §13.3). Without a recognised state the banner keeps the previous behaviour,
         * so a pre-Phase-3 server changes nothing here.
         */
        fun advisoryBanner(
            message: WsAlertMessage,
            activityLabel: String,
            lang: DisplayLanguage = DisplayLanguage.EN
        ) =
            SeismicActivityBanner(
                title = if (lang == DisplayLanguage.ID) advisoryTitleId() else "Possible Tremor Detected",
                activityLabel = if (message.eventState == EventState.UNCONFIRMED) {
                    unconfirmedActivityLabel(message.nodeCount, lang)
                } else {
                    activityLabel
                }
            )

        // Indonesian branch lands in B2.
        private fun advisoryTitleId(): String = "Kemungkinan Getaran Terdeteksi"

        /** Unresolved and inside the same window the realtime path uses. */
        fun EarthquakeEvent.isOngoing(nowMs: Long = System.currentTimeMillis()): Boolean =
            status == EventStatus.HAPPENING &&
                nowMs - createdAt.toEpochMilli() <= WsAlertMessage.RECENT_WINDOW_MS

        /**
         * Stored event → emergency screen state. Distance stays null when the device
         * position is unknown, rather than the 0 the History card falls back to: "0 km
         * away" on a full-screen alert reads as *at the epicentre*.
         */
        fun EarthquakeEvent.toActiveAlert(
            userLocation: UserLocation?,
            locale: Locale = Locale.US
        ) =
            WarningUiState.ActiveAlert(
                eventId = eventId,
                intensityValue = intensityValueLabel(locale),
                distanceKm = userLocation.distanceKmTo(latitude, longitude)?.roundToInt(),
                locationName = locationName
            )

        /** Realtime frame → emergency screen state. See the stored-event twin above. */
        fun WsAlertMessage.toActiveAlert(
            userLocation: UserLocation?,
            locale: Locale = Locale.US
        ) =
            WarningUiState.ActiveAlert(
                eventId = eventId,
                intensityValue = intensityValueLabel(locale),
                distanceKm = userLocation.distanceKmTo(centroidLat, centroidLon)?.roundToInt(),
                locationName = locationName,
                isTest = isTest
            )
    }
}
