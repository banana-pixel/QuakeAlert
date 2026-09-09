package id.web.quakealert.ui.onboarding

import android.Manifest
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.PowerManager
import android.provider.Settings
import android.widget.Toast
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.systemBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.content.ContextCompat
import id.web.quakealert.R
import id.web.quakealert.data.network.QuakeNetwork
import id.web.quakealert.device.canUseFullScreenIntentCompat
import id.web.quakealert.device.openFullscreenIntentSettings
import id.web.quakealert.domain.DisplayLanguage
import id.web.quakealert.service.AlertRaiser
import id.web.quakealert.ui.common.TestAlertSoundDialog
import id.web.quakealert.ui.common.QuakePageIndicator
import id.web.quakealert.ui.common.QuakePrimaryButton
import id.web.quakealert.ui.common.QuakeSecondaryButton
import id.web.quakealert.ui.theme.AccentBlueTranslucent
import id.web.quakealert.ui.theme.BorderLight
import id.web.quakealert.ui.theme.NunitoFontFamily
import id.web.quakealert.ui.theme.OnboardingBackgroundBrush
import id.web.quakealert.ui.theme.OverlayLight
import id.web.quakealert.ui.theme.QuakeAlertTheme
import id.web.quakealert.ui.theme.TextPrimary
import id.web.quakealert.ui.theme.TextSecondary
import kotlinx.coroutines.launch

/**
 * Horizontal inset applied to page content and the bottom controls. Kept out
 * of the pager itself so [HorizontalPager.pageSpacing] shows as a clean gap
 * between pages while swiping, without clipping the resting content.
 */
private val ScreenHorizontalPadding = 28.dp

/** Whether POST_NOTIFICATIONS is granted (always true below API 33). */
private fun isNotificationPermissionGranted(context: Context): Boolean {
    if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return true
    return ContextCompat.checkSelfPermission(
        context,
        Manifest.permission.POST_NOTIFICATIONS
    ) == PackageManager.PERMISSION_GRANTED
}

/** Whether at least coarse location is granted. */
private fun isLocationPermissionGranted(context: Context): Boolean {
    val fine = ContextCompat.checkSelfPermission(
        context, Manifest.permission.ACCESS_FINE_LOCATION
    ) == PackageManager.PERMISSION_GRANTED
    val coarse = ContextCompat.checkSelfPermission(
        context, Manifest.permission.ACCESS_COARSE_LOCATION
    ) == PackageManager.PERMISSION_GRANTED
    return fine || coarse
}

/** Whether the app is exempt from battery optimizations. */
private fun isIgnoringBatteryOptimizations(context: Context): Boolean {
    val pm = context.getSystemService(Context.POWER_SERVICE) as PowerManager
    return pm.isIgnoringBatteryOptimizations(context.packageName)
}

/**
 * Full onboarding flow — data-driven across seven Figma pages (nodes 1:470,
 * 1:337, 1:354, 1:378, 1:402, 1:426, 1:453). Pages are described by a list of
 * [OnboardingPage] and rendered through a [HorizontalPager]. The indicator and
 * bottom action row react to the pager's current page. Interactive pages
 * (notification, battery, location, test-alert) are wired to runtime
 * permission launchers and system intents.
 */
@Composable
fun OnboardingScreen(
    modifier: Modifier = Modifier,
    onFinish: () -> Unit = {},
    lang: DisplayLanguage = DisplayLanguage.EN
) {
    val strings = remember(lang) { onboardingStrings(lang) }
    val pages = rememberOnboardingPages(strings, lang)
    val pagerState = rememberPagerState(pageCount = { pages.size })
    val coroutineScope = rememberCoroutineScope()
    val context = LocalContext.current

    // --- Permission / requirement state -----------------------------------
    var notificationGranted by remember {
        mutableStateOf(isNotificationPermissionGranted(context))
    }
    var locationGranted by remember {
        mutableStateOf(isLocationPermissionGranted(context))
    }
    var batteryUnrestricted by remember {
        mutableStateOf(isIgnoringBatteryOptimizations(context))
    }
    var fullscreenGranted by remember {
        mutableStateOf(context.canUseFullScreenIntentCompat())
    }

    val notificationLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestPermission()
    ) { granted -> notificationGranted = granted }

    val locationLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestMultiplePermissions()
    ) { result ->
        val granted = result.values.any { it }
        locationGranted = granted
        // The permission is only half of it: nothing else in the app acquires a fix,
        // and without a stored position the server has no radius to filter sensors or
        // events by. Started on the process scope because leaving onboarding must not
        // cancel the upload.
        if (granted) {
            val network = QuakeNetwork.from(context)
            network.applicationScope.launch {
                network.userLocationRepository.sync(force = true)
            }
        }
    }

    // Re-check requirements that are resolved outside the app (Settings screens).
    val settingsLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.StartActivityForResult()
    ) {
        batteryUnrestricted = isIgnoringBatteryOptimizations(context)
        fullscreenGranted = context.canUseFullScreenIntentCompat()
    }

    val requestNotification: () -> Unit = {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            notificationLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
        } else {
            notificationGranted = true
        }
    }

    val requestLocation: () -> Unit = {
        locationLauncher.launch(
            arrayOf(
                Manifest.permission.ACCESS_FINE_LOCATION,
                Manifest.permission.ACCESS_COARSE_LOCATION
            )
        )
    }

    val requestBattery: () -> Unit = {
        openBatteryOptimizationSettings(context, settingsLauncher::launch)
    }

    // Full-screen intent has no runtime dialog (unlike POST_NOTIFICATIONS):
    // the only path is the app's notification settings screen, like the
    // battery exemption above — hence the settings launcher for the return.
    val requestFullscreen: () -> Unit = {
        openFullscreenSettings(context, settingsLauncher::launch)
    }

    // Local to the screen: the modal owns its own playback, so there is nothing
    // about it for onboarding state to hold. Hosted here rather than inside
    // TestAlertControls so it survives the page recomposing under a swipe.
    var showTestAlertSound by remember { mutableStateOf(false) }
    if (showTestAlertSound) {
        TestAlertSoundDialog(onDismissRequest = { showTestAlertSound = false }, lang = lang)
    }

    val fireTestAlert: () -> Unit = {
        coroutineScope.launch {
            when (AlertRaiser.runLocalTest(context)) {
                AlertRaiser.LocalTestOutcome.RAN -> Unit
                AlertRaiser.LocalTestOutcome.SWITCH_OFF -> {
                    Toast.makeText(context, strings.enableAlertsFirst, Toast.LENGTH_SHORT).show()
                }
                AlertRaiser.LocalTestOutcome.NO_PERMISSION -> {
                    Toast.makeText(context, strings.toastEnableFirst, Toast.LENGTH_SHORT).show()
                    requestNotification()
                }
            }
        }
    }

    Box(
        modifier = modifier
            .fillMaxSize()
            .background(OnboardingBackgroundBrush)
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .systemBarsPadding()
                .padding(vertical = 20.dp)
        ) {
            // The pager itself spans full width (no horizontal inset) so that
            // pageSpacing reads as a clean gap between pages while swiping.
            // Per-page horizontal padding lives inside OnboardingPageItem so the
            // resting content stays aligned with the bottom controls below.
            // Hoisted outside the pager lambda so the identical page inset isn't
            // re-allocated for every page composition.
            val pageContentModifier = Modifier.padding(horizontal = ScreenHorizontalPadding)
            HorizontalPager(
                state = pagerState,
                pageSpacing = 32.dp,
                modifier = Modifier
                    .fillMaxWidth()
                    .weight(1f)
            ) { pageIndex ->
                OnboardingPageItem(
                    page = pages[pageIndex],
                    notificationGranted = notificationGranted,
                    locationGranted = locationGranted,
                    batteryUnrestricted = batteryUnrestricted,
                    fullscreenGranted = fullscreenGranted,
                    onRequestNotification = requestNotification,
                    onRequestLocation = requestLocation,
                    onRequestBattery = requestBattery,
                    onRequestFullscreen = requestFullscreen,
                    onTestAlert = fireTestAlert,
                    onTestAlertSound = { showTestAlertSound = true },
                    strings = strings,
                    modifier = pageContentModifier
                )
            }

            Spacer(modifier = Modifier.height(20.dp))

            // Bottom controls share the same horizontal padding as the page
            // content so everything lines up when a page rests in place.
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = ScreenHorizontalPadding)
            ) {
                QuakePageIndicator(
                    pageCount = pages.size,
                    currentPage = pagerState.currentPage,
                    modifier = Modifier.align(Alignment.CenterHorizontally)
                )

                Spacer(modifier = Modifier.height(20.dp))

                // Bottom actions: single CTA on the first page, Back/Next otherwise.
                if (pagerState.currentPage == 0) {
                    QuakePrimaryButton(
                        text = pages[0].actionText ?: "Start",
                        onClick = {
                            if (pages.size > 1) {
                                coroutineScope.launch { pagerState.animateScrollToPage(1) }
                            } else {
                                onFinish()
                            }
                        },
                        modifier = Modifier.fillMaxWidth()
                    )
                } else {
                    val isLast = pagerState.currentPage == pages.lastIndex
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(20.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        QuakeSecondaryButton(
                            text = strings.back,
                            onClick = {
                                coroutineScope.launch {
                                    pagerState.animateScrollToPage(pagerState.currentPage - 1)
                                }
                            },
                            modifier = Modifier.weight(1f)
                        )
                        QuakePrimaryButton(
                            text = if (isLast) strings.getStarted else strings.next,
                            onClick = {
                                if (isLast) {
                                    onFinish()
                                } else {
                                    coroutineScope.launch {
                                        pagerState.animateScrollToPage(pagerState.currentPage + 1)
                                    }
                                }
                            },
                            modifier = Modifier.weight(1f)
                        )
                    }
                }
            }
        }
    }
}

/**
 * Renders a single page: centered illustration, title, description and any
 * page-specific interactive control selected by [OnboardingPage.kind].
 */
@Composable
fun OnboardingPageItem(
    page: OnboardingPage,
    notificationGranted: Boolean,
    locationGranted: Boolean,
    batteryUnrestricted: Boolean,
    fullscreenGranted: Boolean = true,
    onRequestNotification: () -> Unit,
    onRequestLocation: () -> Unit,
    onRequestBattery: () -> Unit,
    onRequestFullscreen: () -> Unit = {},
    onTestAlert: () -> Unit,
    onTestAlertSound: () -> Unit,
    modifier: Modifier = Modifier,
    strings: OnboardingStrings = onboardingStrings(DisplayLanguage.EN)
) {
    Column(
        modifier = modifier.fillMaxSize(),
        verticalArrangement = Arrangement.SpaceBetween,
        horizontalAlignment = Alignment.Start
    ) {
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .weight(1f),
            contentAlignment = Alignment.Center
        ) {
            // Illustrations are full-colour vector art; render them untinted
            // (Image, not Icon) so every page shows its original artwork
            // consistently instead of a single flat tint.
            Image(
                painter = painterResource(id = page.iconRes),
                contentDescription = page.title,
                modifier = Modifier.size(150.dp)
            )
        }

        Column(
            modifier = Modifier.fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(20.dp)
        ) {
            Text(
                text = page.title,
                color = TextPrimary,
                fontFamily = NunitoFontFamily,
                fontWeight = FontWeight.ExtraBold,
                fontSize = if (page.largeTitle) 32.sp else 24.sp,
                lineHeight = if (page.largeTitle) 36.sp else 26.sp,
                textAlign = TextAlign.Start,
                modifier = Modifier.fillMaxWidth()
            )

            when (page.kind) {
                OnboardingPageKind.READY -> ReadyText(
                    strings = strings,
                    modifier = Modifier.fillMaxWidth()
                )
                else -> Text(
                    text = page.description,
                    color = TextSecondary,
                    fontFamily = NunitoFontFamily,
                    fontWeight = FontWeight.Normal,
                    fontSize = 14.sp,
                    lineHeight = 24.sp,
                    textAlign = TextAlign.Start,
                    modifier = Modifier.fillMaxWidth()
                )
            }

            when (page.kind) {
                OnboardingPageKind.NOTIFICATION_PERMISSION -> PermissionCard(
                    title = page.cardTitle,
                    isGranted = notificationGranted,
                    grantedLabel = page.grantedLabel,
                    tapToAllowLabel = strings.tapToAllow,
                    onClick = onRequestNotification
                )

                OnboardingPageKind.BATTERY_OPTIMIZATION -> PermissionCard(
                    title = page.cardTitle,
                    isGranted = batteryUnrestricted,
                    grantedLabel = page.grantedLabel,
                    tapToAllowLabel = strings.tapToAllow,
                    onClick = onRequestBattery
                )

                OnboardingPageKind.FULLSCREEN_PERMISSION -> PermissionCard(
                    title = page.cardTitle,
                    isGranted = fullscreenGranted,
                    grantedLabel = page.grantedLabel,
                    tapToAllowLabel = strings.tapToAllow,
                    onClick = onRequestFullscreen
                )

                OnboardingPageKind.LOCATION_PERMISSION -> PermissionCard(
                    title = page.cardTitle,
                    isGranted = locationGranted,
                    grantedLabel = page.grantedLabel,
                    tapToAllowLabel = strings.tapToAllow,
                    onClick = onRequestLocation
                )

                OnboardingPageKind.TEST_ALERT -> TestAlertControls(
                    strings = strings,
                    onTestAlert = onTestAlert,
                    onTestAlertSound = onTestAlertSound
                )

                else -> Unit
            }
        }
    }
}

/**
 * Opens the "Manage full screen intents" system page, where the
 * full-screen-intent toggle lives — there is no runtime dialog for this
 * permission. Falls back to the app's notification settings on a device
 * without the dedicated page (pre-API 34).
 */
private fun openFullscreenSettings(
    context: Context,
    launch: (Intent) -> Unit
) {
    // Resolved synchronously (not via the launcher) because only the
    // fallback needs launching: the dedicated page, when present, is opened
    // directly like every other system screen from this flow.
    if (!context.openFullscreenIntentSettings()) {
        launch(Intent(Settings.ACTION_SETTINGS))
    }
}

/**
 * Opens the per-app battery-optimization exemption dialog, falling back to the
 * general battery-optimization settings list if the direct request is
 * unavailable on the device.
 */
private fun openBatteryOptimizationSettings(
    context: Context,
    launch: (Intent) -> Unit
) {
    val direct = Intent(
        Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS,
        Uri.parse("package:${context.packageName}")
    )
    if (direct.resolveActivity(context.packageManager) != null) {
        launch(direct)
    } else {
        launch(Intent(Settings.ACTION_IGNORE_BATTERY_OPTIMIZATION_SETTINGS))
    }
}

@Composable
private fun rememberOnboardingPages(strings: OnboardingStrings, lang: DisplayLanguage): List<OnboardingPage> {
    val id = lang == DisplayLanguage.ID
    return listOf(
        OnboardingPage(
            iconRes = R.drawable.ic_puzzle_piece,
            title = if (id) "Selamat datang di Aplikasi QuakeAlert." else "Welcome to QuakeAlert App.",
            description = if (id) {
                "QuakeAlert adalah sistem peringatan dini gempa bumi berbasis " +
                    "komunitas (platform). Tetap aman dengan peringatan EWS real time " +
                    "cerdas yang dapat diberitahukan lewat aplikasi ini!"
            } else {
                "QuakeAlert is community based earthquake early warning " +
                    "system (platform). Keep safe with intelligent real time EWS " +
                    "alert that can be notified through this app!"
            },
            actionText = strings.start,
            largeTitle = true
        ),
        OnboardingPage(
            iconRes = R.drawable.ic_sensor_chip,
            title = if (id) {
                "Berdasarkan Sensor murah yang dapat dipasang di seluruh dunia."
            } else {
                "Based on low cost Sensors that can be placed all over the world."
            },
            description = if (id) {
                "Ini adalah sistem peringatan dini yang didukung komunitas. Anda " +
                    "dapat memasang sensor murah Anda sendiri di rumah. Hanya butuh " +
                    "jaringan WiFi yang stabil dan Anda siap! Baca penafian dan " +
                    "panduan di sini pada halaman GitHub kami."
            } else {
                "This is a community supported early warning system. You can " +
                    "place your own low cost sensors on your home. Just need a stable WiFi " +
                    "network and you\u2019re good to go! Read disclaimer and guides here on " +
                    "our GitHub pages."
            }
        ),
        OnboardingPage(
            iconRes = R.drawable.ic_notification_permission,
            title = if (id) "Mohon izinkan izin notifikasi." else "Please allow notification permission.",
            description = if (id) {
                "Untuk menerima peringatan gempa bumi, Aplikasi QuakeAlert membutuhkan " +
                    "izin untuk mengirimi Anda notifikasi."
            } else {
                "To receive earthquake alerts, QuakeAlert App needs " +
                    "permission to send you notifications."
            },
            kind = OnboardingPageKind.NOTIFICATION_PERMISSION,
            cardTitle = if (id) "Izinkan Notifikasi" else "Allow Notification",
            grantedLabel = strings.allowed
        ),
        OnboardingPage(
            iconRes = R.drawable.ic_fullscreen_permission,
            title = if (id) "Mohon izinkan peringatan layar penuh." else "Please allow full-screen alerts.",
            description = if (id) {
                "Agar peringatan gempa dapat membangunkan ponsel ini saat terkunci."
            } else {
                "So earthquake warnings can wake this phone when it is locked."
            },
            kind = OnboardingPageKind.FULLSCREEN_PERMISSION,
            cardTitle = if (id) "Izinkan Layar Penuh" else "Allow Full-Screen Alerts",
            grantedLabel = strings.allowed
        ),
        OnboardingPage(
            iconRes = R.drawable.ic_battery_optimization,
            title = if (id) "Mohon atur pengaturan optimasi baterai." else "Please set battery optimization settings.",
            description = if (id) {
                "Agar peringatan tidak pernah tertunda, Aplikasi QuakeAlert perlu " +
                    "berjalan tanpa pembatasan baterai."
            } else {
                "To ensure alerts are never delayed, QuakeAlert App needs " +
                    "to run witout battery restrictions."
            },
            kind = OnboardingPageKind.BATTERY_OPTIMIZATION,
            cardTitle = if (id) "Matikan Pembatasan" else "Disable Restrictions",
            grantedLabel = strings.disabled
        ),
        OnboardingPage(
            iconRes = R.drawable.ic_location_permission,
            title = if (id) "Mohon izinkan akses lokasi presisi." else "Please allow precise location access.",
            description = if (id) {
                "Untuk menghitung lokasi Anda dari pusat gempa bumi dan " +
                    "memberikan peringatan gempa yang relevan dan akurat."
            } else {
                "To calculate your location from the earthquake center and " +
                    "give relevant and accurate earthquake alerts."
            },
            kind = OnboardingPageKind.LOCATION_PERMISSION,
            cardTitle = if (id) "Izinkan Akses Lokasi Presisi" else "Allow Precise Location Access",
            grantedLabel = strings.allowed
        ),
        OnboardingPage(
            iconRes = R.drawable.ic_alert_test,
            title = if (id) "Uji peringatan." else "Test alert.",
            description = if (id) {
                "Kirim notifikasi uji untuk memastikan layanan notifikasi bekerja."
            } else {
                "Send a test notification to make sure the notification service is working."
            },
            kind = OnboardingPageKind.TEST_ALERT
        ),
        OnboardingPage(
            iconRes = R.drawable.ic_ready_smiley,
            title = if (id) "Anda siap." else "You\u2019re ready.",
            description = "",
            kind = OnboardingPageKind.READY
        )
    )
}


@Preview(showBackground = true, widthDp = 402, heightDp = 874)
@Composable
private fun OnboardingScreenPreview() {
    QuakeAlertTheme {
        OnboardingScreen()
    }
}
