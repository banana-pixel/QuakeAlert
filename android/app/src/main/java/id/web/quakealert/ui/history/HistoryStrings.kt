package id.web.quakealert.ui.history

import id.web.quakealert.domain.DisplayLanguage

/**
 * Every literal on the History screen, in one place.
 *
 * Built once per composition from the selected language. Pure and
 * unit-testable.
 */
data class HistoryStrings(
    val appBar: String,
    val shareChooser: String,
    val loading: String,
    val detailTitle: String
)

private fun historyStringsEn(): HistoryStrings = HistoryStrings(
    appBar = "History",
    shareChooser = "Share earthquake details",
    loading = "Loading earthquake history...",
    detailTitle = "Earthquake Details"
)

private fun historyStringsId(): HistoryStrings = HistoryStrings(
    appBar = "Riwayat",
    shareChooser = "Bagikan detail gempa bumi",
    loading = "Memuat riwayat gempa bumi...",
    detailTitle = "Detail Gempa Bumi"
)

internal fun historyStrings(lang: DisplayLanguage): HistoryStrings =
    if (lang == DisplayLanguage.ID) historyStringsId() else historyStringsEn()
