package id.web.quakealert.ui.updates

import id.web.quakealert.domain.DisplayLanguage

/**
 * Chrome copy around the operator announcements, in one place.
 *
 * Announcement titles/bodies themselves arrive from the server and are shown
 * verbatim, never translated.
 */
data class UpdatesStrings(
    val title: String,
    val loading: String,
    val emptyTitle: String,
    val emptySubtitle: String
)

private fun updatesStringsEn(): UpdatesStrings = UpdatesStrings(
    title = "Updates",
    loading = "Loading updates...",
    emptyTitle = "No Updates Yet",
    emptySubtitle = "Announcements from the QuakeAlert team will appear here. " +
        "Earthquake warnings are never sent this way."
)

private fun updatesStringsId(): UpdatesStrings = UpdatesStrings(
    title = "Pembaruan",
    loading = "Memuat pembaruan...",
    emptyTitle = "Belum Ada Pembaruan",
    emptySubtitle = "Pengumuman dari tim QuakeAlert akan muncul di sini. " +
        "Peringatan gempa bumi tidak pernah dikirim lewat sini."
)

internal fun updatesStrings(lang: DisplayLanguage): UpdatesStrings =
    if (lang == DisplayLanguage.ID) updatesStringsId() else updatesStringsEn()
