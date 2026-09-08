package id.web.quakealert.ui.chat

import id.web.quakealert.domain.DisplayLanguage

/**
 * Every literal on the Chat screen, in one place.
 *
 * Built once per composition from the selected language. Pure and
 * unit-testable. User messages themselves are never translated — only this
 * chrome is.
 */
data class ChatStrings(
    val appBar: String,
    val loading: String,
    val loadingOlder: String,
    val emptyTitle: String,
    val emptySubtitle: String,
    val inputPlaceholder: String,
    val sendDescription: String,
    val sending: String,
    val notSentRetry: String,
    val switchGlobal: String,
    val switchArea: String
)

private fun chatStringsEn(): ChatStrings = ChatStrings(
    appBar = "Chat",
    loading = "Loading messages...",
    loadingOlder = "Loading older messages...",
    emptyTitle = "No messages yet",
    emptySubtitle = "Be the first to say what it is like where you are.",
    inputPlaceholder = "Message the mesh...",
    sendDescription = "Send message",
    sending = "Sending...",
    notSentRetry = "Not sent. Tap to retry",
    switchGlobal = "Switch to the global channel",
    switchArea = "Switch to your area's channel"
)

private fun chatStringsId(): ChatStrings = ChatStrings(
    appBar = "Obrolan",
    loading = "Memuat pesan...",
    loadingOlder = "Memuat pesan lama...",
    emptyTitle = "Belum ada pesan",
    emptySubtitle = "Jadilah yang pertama menceritakan keadaan di tempat Anda.",
    inputPlaceholder = "Kirim pesan ke mesh...",
    sendDescription = "Kirim pesan",
    sending = "Mengirim...",
    notSentRetry = "Tidak terkirim. Ketuk untuk mencoba lagi",
    switchGlobal = "Beralih ke kanal global",
    switchArea = "Beralih ke kanal daerah Anda"
)

internal fun chatStrings(lang: DisplayLanguage): ChatStrings =
    if (lang == DisplayLanguage.ID) chatStringsId() else chatStringsEn()
