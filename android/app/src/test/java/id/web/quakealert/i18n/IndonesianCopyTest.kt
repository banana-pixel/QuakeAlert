package id.web.quakealert.i18n

import id.web.quakealert.data.UnitSystem
import id.web.quakealert.data.network.ApiException
import id.web.quakealert.data.network.ServerHealth
import id.web.quakealert.data.network.mapper.QuakeFormat
import id.web.quakealert.domain.ChatChannel
import id.web.quakealert.domain.ChatChannelKind
import id.web.quakealert.domain.DisplayLanguage
import id.web.quakealert.domain.EmergencyContacts
import id.web.quakealert.domain.EventState
import id.web.quakealert.domain.ProtectionStatus
import id.web.quakealert.domain.standDownCopyFor
import id.web.quakealert.domain.unconfirmedActivityLabel
import id.web.quakealert.ui.addsensor.AddSensorWizardStep
import id.web.quakealert.ui.addsensor.DetailsError
import id.web.quakealert.ui.addsensor.LinkError
import id.web.quakealert.ui.addsensor.WizardFailure
import id.web.quakealert.ui.addsensor.failureCopy
import id.web.quakealert.ui.addsensor.headline
import id.web.quakealert.ui.addsensor.helperText
import id.web.quakealert.ui.addsensor.message
import id.web.quakealert.ui.chat.chatStrings
import id.web.quakealert.ui.chat.toChannelInfo
import id.web.quakealert.ui.common.QuakeIntensity
import id.web.quakealert.ui.common.QuakeStationStatus
import id.web.quakealert.ui.common.QuakeTimeWindow
import id.web.quakealert.ui.common.errorCopy
import id.web.quakealert.ui.common.filterStrings
import id.web.quakealert.ui.common.label
import id.web.quakealert.ui.common.stateStrings
import id.web.quakealert.ui.history.MmiSeverity
import id.web.quakealert.ui.history.distanceLabel
import id.web.quakealert.ui.history.historyStrings
import id.web.quakealert.ui.history.label
import id.web.quakealert.ui.onboarding.onboardingStrings
import id.web.quakealert.ui.sensors.sensorStrings
import id.web.quakealert.ui.settings.settingsStrings
import id.web.quakealert.ui.updates.updatesStrings
import id.web.quakealert.ui.warning.ActivityAvailability
import id.web.quakealert.ui.warning.RecentSeismicActivity
import id.web.quakealert.ui.warning.activeQuakeTips
import id.web.quakealert.ui.warning.emergencyStrings
import id.web.quakealert.ui.warning.noActiveQuakeTips
import id.web.quakealert.ui.warning.suggestedActions
import id.web.quakealert.ui.warning.warningStrings
import id.web.quakealert.ui.history.QuakeHistoryItem
import java.io.IOException
import java.time.Instant
import java.time.ZoneId
import java.util.Locale
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Test

/**
 * Indonesian copy: glossary pins + em-dash guard.
 *
 * Every user-visible Indonesian string produced by a unit-reachable function is
 * asserted here for its key term (per the BMKG/BPBD/Android-system glossary) and
 * for the absence of U+2014. Notifier-private and component-private literals are
 * covered by the drill screenshot instead — they need Android to render.
 */
class IndonesianCopyTest {

    private val EM_DASH = "—"
    private val ID = DisplayLanguage.ID
    private val EN = DisplayLanguage.EN
    private val ID_LOCALE = Locale("in")

    private fun assertNoEmDash(where: String, text: String) {
        assertFalse("$where memuat em dash: $text", text.contains(EM_DASH))
    }

    @Test
    fun `stand-down keeps its safety distinction in Indonesian`() {
        val cancelled = standDownCopyFor(EventState.CANCELLED, ID)
        assertEquals("Laporan Ditarik", cancelled.title)
        val resolved = standDownCopyFor(EventState.RESOLVED, ID)
        assertEquals("Aman", resolved.title)
        val unknown = standDownCopyFor(null, ID)
        assertEquals("Aman", unknown.title)
        listOf(cancelled, resolved, unknown).forEach {
            assertNoEmDash("standDown", it.title)
            assertNoEmDash("standDown", it.detail)
        }
    }

    @Test
    fun `unconfirmed label has no plural in Indonesian`() {
        assertEquals(
            "2 stasiun melaporkan guncangan - belum dikonfirmasi oleh stasiun yang terpisah",
            unconfirmedActivityLabel(2, ID)
        )
        assertEquals(
            "1 stasiun melaporkan guncangan - belum dikonfirmasi oleh stasiun yang terpisah",
            unconfirmedActivityLabel(1, ID)
        )
    }

    @Test
    fun `error copy speaks Indonesian`() {
        val offline = errorCopy(IOException("x"), lang = ID)
        assertEquals("Anda sedang offline", offline.title)
        val unauth = errorCopy(ApiException(401, "UNAUTHENTICATED", "x"), lang = ID)
        assertEquals("Sesi login kedaluwarsa", unauth.title)
        val throttled = errorCopy(ApiException(429, "RATE_LIMITED", "x"), lang = ID)
        assertEquals("Terlalu banyak permintaan", throttled.title)
        listOf(offline, unauth, throttled).forEach {
            assertNoEmDash("errorCopy", it.title)
            assertNoEmDash("errorCopy", it.message)
        }
    }

    @Test
    fun `wizard copy is Indonesian`() {
        assertEquals("Di mana sensor ini dipasang?", AddSensorWizardStep.LOCATION.headline(ID))
        assertEquals("Tidak bisa menambah sensor", AddSensorWizardStep.RATE_LIMIT.headline(ID))
        assertEquals(
            "Pilih jaringan Wi-Fi untuk sensor ini.",
            LinkError.SSID_REQUIRED.message(ID)
        )
        assertEquals(
            "Masukkan nama tempat untuk sensor ini.",
            DetailsError.NAME_REQUIRED.message(ID)
        )
        val failure = failureCopy(WizardFailure.OFFLINE, ID)
        assertEquals("Anda sedang offline", failure.title)
    }

    @Test
    fun `protection status is Indonesian`() {
        val status = ProtectionStatus(
            alertsEnabled = true,
            notificationsPermitted = false,
            autoSyncEnabled = true,
            batteryUnrestricted = true,
            lastSyncLabel = null
        )
        assertEquals("Peringatan diblokir oleh pengaturan sistem", status.headline(ID))
        status.lines(ID).forEach { assertNoEmDash("protection", it) }
    }

    @Test
    fun `relative time and counts are Indonesian`() {
        val now = Instant.parse("2026-09-07T12:00:00Z")
        assertEquals("baru saja", QuakeFormat.relativeTime(now, now, ID_LOCALE))
        assertEquals(
            "20 menit yang lalu",
            QuakeFormat.relativeTime(now.minusSeconds(20 * 60), now, ID_LOCALE)
        )
        assertEquals("3 stasiun", QuakeFormat.reportingNodes(3, ID_LOCALE))
        assertEquals("1 stasiun", QuakeFormat.reportingNodes(1, ID_LOCALE))
        assertEquals(
            "20 Jun 2026",
            QuakeFormat.date(Instant.parse("2026-06-20T00:00:00Z"), ZoneId.of("UTC"), ID_LOCALE)
        )
        assertEquals(
            "Intensitas : IV (sedang)",
            QuakeFormat.intensityBanner("IV", "moderate", "MODERATE", ID_LOCALE)
        )
    }

    @Test
    fun `severity words are Indonesian`() {
        assertEquals("Sedang", MmiSeverity.MODERATE.label(ID))
        assertEquals("Kuat", MmiSeverity.SEVERE.label(ID))
    }

    @Test
    fun `filter option labels are Indonesian`() {
        assertEquals("Semua", filterStrings(ID).all)
        assertEquals("Semua Intensitas", QuakeIntensity.ALL.label(ID))
        assertEquals("Kapan saja", QuakeTimeWindow.ALL.label(ID))
        assertEquals("Semua stasiun", QuakeStationStatus.ALL.label(ID))
        assertEquals(
            "Semua stasiun di area ini sedang offline.",
            QuakeStationStatus.ONLINE.emptyRollSubtitle(ID)
        )
    }

    @Test
    fun `screen holders are Indonesian and dash-free`() {
        val holders: List<Pair<String, List<String>>> = listOf(
            "settings" to settingsStrings(ID).let {
                listOf(
                    it.appBar, it.syncNow, it.alerts, it.testNotification,
                    it.testNotificationDetail, it.units, it.permFullscreen,
                    it.language, it.disableTitle, it.disableBody, it.turnOff, it.cancel,
                    it.resetTitle, it.resetBody, it.reset, it.tapToAllow, it.moreAboutUs
                )
            },
            "chat" to chatStrings(ID).let {
                listOf(
                    it.appBar, it.loading, it.emptyTitle, it.emptySubtitle,
                    it.inputPlaceholder, it.sending, it.notSentRetry
                )
            },
            "history" to historyStrings(ID).let { listOf(it.appBar, it.shareChooser, it.loading) },
            "sensors" to sensorStrings(ID).let { listOf(it.appBar, it.loading) },
            "updates" to updatesStrings(ID).let {
                listOf(it.title, it.loading, it.emptyTitle, it.emptySubtitle)
            },
            "onboarding" to onboardingStrings(ID).let {
                listOf(
                    it.back, it.next, it.getStarted, it.testNotification,
                    it.testNotificationDetail, it.enableAlertsFirst, it.readyPara1
                )
            },
            "warning" to warningStrings(ID).let {
                listOf(
                    it.appBar, it.seeDetails, it.alertTitle, it.estimatedIntensity,
                    it.suggestedActions, it.emergencyCta, it.offlineMessage, it.cardTitle,
                    it.statusTitle, it.detailTitle, it.checkingNetwork, it.drillBadge,
                    it.endTest
                )
            },
            "filter" to filterStrings(ID).let {
                listOf(it.title, it.intensityGroup, it.reset, it.apply, it.all, it.near)
            },
            "states" to stateStrings(ID).let {
                listOf(it.retry, it.resetFilters, it.noHistory, it.noCoverage, it.syncLocation)
            }
        )
        holders.forEach { (name, texts) ->
            texts.forEach { assertNoEmDash(name, it) }
        }
        // Glossary pins.
        assertEquals("Riwayat", historyStrings(ID).appBar)
        assertEquals("Peringatan", warningStrings(ID).appBar)
        assertEquals("Chat", chatStrings(ID).appBar)
        assertEquals("Pengaturan", settingsStrings(ID).appBar)
        // Drill-test entry points name the earthquake alert, not a generic ping.
        assertEquals("Uji Peringatan Gempa", settingsStrings(ID).testNotification)
        assertEquals("Test Earthquake Alert", settingsStrings(EN).testNotification)
        assertEquals("Uji Peringatan Gempa", onboardingStrings(ID).testNotification)
        assertEquals("AKHIRI TES", warningStrings(ID).endTest)
        assertEquals("END TEST", warningStrings(EN).endTest)
        assertEquals("Peringatan Layar Penuh", settingsStrings(ID).permFullscreen)
        assertEquals("Full-Screen Alerts", settingsStrings(EN).permFullscreen)
    }

    @Test
    fun `warning activity copy is Indonesian`() {
        val quiet = RecentSeismicActivity(
            availability = ActivityAvailability.MEASURED,
            eventCount = 0
        )
        assertEquals("Tidak Ada Gempa Terkini", quiet.bannerTitle(ID))
        val unmeasured = RecentSeismicActivity()
        assertEquals("Tidak Ada Gempa Aktif", unmeasured.bannerTitle(ID))
        assertEquals("Perlu lokasi Anda", unmeasured.countValue(ID))
        assertEquals(
            "Tidak ada kejadian",
            unmeasured.copy(availability = ActivityAvailability.MEASURED).countValue(ID)
        )
        val item = QuakeHistoryItem(
            id = "e1",
            intensity = "V",
            severity = MmiSeverity.MODERATE,
            location = "Bandung",
            date = "20 Jun 2026",
            time = "07:19:18",
            distanceKm = null,
            relativeTime = "baru saja",
            pgaLabel = "61.5 gal",
            reportingNodesLabel = "3 stasiun",
            coordinates = "-6.9, 107.6",
            latitude = -6.9,
            longitude = 107.6
        )
        assertEquals("Jarak tidak diketahui", item.distanceLabel(UnitSystem.METRIC, ID))
        val global = ChatChannel(id = "global", kind = ChatChannelKind.GLOBAL, displayName = "Global")
        assertEquals("Semua pengguna QuakeAlert", global.toChannelInfo(false, ID).subtitle)
        val actions = suggestedActions(ID).map { it.label }
        assertEquals(listOf("Menunduk!", "Lindungi Kepala!", "Berpegangan!"), actions)
        val tips = (activeQuakeTips(ID) + noActiveQuakeTips(ID)).flatMap { listOf(it.title, it.description) }
        tips.forEach { assertNoEmDash("tips", it) }
    }

    @Test
    fun `server badge speaks Indonesian`() {
        assertEquals("Normal", ServerHealth.HEALTHY.label(ID))
        assertEquals("Terbatas", ServerHealth.LIMITED.label(ID))
        assertEquals("Memeriksa…", ServerHealth.CHECKING.label(ID))
        assertEquals("Offline", ServerHealth.OFFLINE.label(ID))
        assertEquals("Healthy", ServerHealth.HEALTHY.label(EN))
    }

    @Test
    fun `emergency overlay follows the in-app language`() {
        val id = emergencyStrings(ID)
        assertEquals("Langkah Darurat & Kontak", id.title)
        assertEquals("1. Menunduk", id.dropTitle)
        assertEquals("Hubungi 112", id.dialAction("112"))
        listOf(
            id.title, id.duringTitle, id.dropTitle, id.dropDetail,
            id.coverTitle, id.coverDetail, id.holdTitle, id.holdDetail,
            id.afterTitle, id.aftershocks, id.gas, id.exit, id.injuries,
            id.contactsTitle, id.contactsNote, id.dialAction("112"),
            id.positionTitle, id.positionNote, id.positionUnknown, id.offlineNote
        ).forEach { assertNoEmDash("emergency", it) }
        val numbers = EmergencyContacts.forCountry("ID", ID)
        assertEquals(
            listOf("112", "110", "113", "118", "115"),
            numbers.map { it.number }
        )
        assertEquals("Darurat (semua jaringan)", numbers.first().label)
        assertEquals(
            listOf("Darurat (semua jaringan)", "Polisi", "Pemadam kebakaran", "Ambulans", "SAR (Basarnas)"),
            numbers.map { it.label }
        )
    }

    @Test
    fun `english branches are byte-identical to the pre-i18n copy`() {
        // Spot checks that the B1 refactor changed no English text.
        assertEquals("All Clear", standDownCopyFor(EventState.RESOLVED, EN).title)
        assertEquals("Di mana sensor ini dipasang?", AddSensorWizardStep.LOCATION.headline(ID))
        assertEquals("Where would you like to provision this sensor?", AddSensorWizardStep.LOCATION.headline(EN))
        assertEquals(
            "2 stations are reporting shaking - not yet confirmed by separated stations",
            unconfirmedActivityLabel(2, EN)
        )
        assertEquals(
            "20 minutes ago",
            QuakeFormat.relativeTime(
                Instant.parse("2026-09-07T11:40:00Z"),
                Instant.parse("2026-09-07T12:00:00Z"),
                Locale.US
            )
        )
    }

    @Test
    fun `local warning copy names itself and its source in both languages`() {
        val en = warningStrings(EN)
        assertEquals("Local Warning", en.localWarningTitle)
        assertEquals("LOCAL WARNING - SINGLE STATION REPORT", en.localWarningBadge)
        assertEquals("Admin Node", en.localWarningSource)

        val id = warningStrings(ID)
        assertEquals("Peringatan Lokal", id.localWarningTitle)
        assertEquals("PERINGATAN LOKAL - LAPORAN SATU STASIUN", id.localWarningBadge)
        // A proper noun, identical in both languages — like "GitHub" in the
        // About links — so the source label cannot drift between them.
        assertEquals("Admin Node", id.localWarningSource)

        listOf(en, id).forEach {
            assertNoEmDash("localWarning", it.localWarningTitle)
            assertNoEmDash("localWarning", it.localWarningBadge)
            assertNoEmDash("localWarning", it.localWarningSource)
        }
    }
}

// Placeholder assertion helpers live beside the copy they pin; the HistoryStrings
// data class itself carries no language logic to test.