package id.web.quakealert.domain

import java.util.Locale

/**
 * Language a user-facing string is rendered in.
 *
 * Lives in domain (not ui.settings) so every copy function — including the
 * pure ones asserted without Android ([AlertLifecycleCopy], [ProtectionStatus])
 * — can take it as a parameter without importing UI state. The settings
 * [id.web.quakealert.ui.settings.AppLanguage] maps 1:1 onto this; the two enums
 * stay separate because one is a persisted preference and the other is a
 * render parameter, and conflating them would make "which language is stored"
 * and "which language is shown" the same question.
 */
enum class DisplayLanguage(val tag: String) {
    EN("en"),
    ID("id");

    companion object {
        /** The entry for [tag], or null when nothing was stored or chosen. */
        fun fromTagOrNull(tag: String?): DisplayLanguage? =
            entries.firstOrNull { it.tag.equals(tag, ignoreCase = true) }
    }

    /** The [Locale] its copy branch formats dates and relative times with. */
    fun locale(): Locale = when (this) {
        EN -> Locale.US
        ID -> Locale("in")
    }
}

/**
 * Resolves which language to show: the stored override wins, otherwise the
 * system locale decides (Indonesian devices read Indonesian), otherwise
 * English. The last arm is a fallback, never a preference — every string has
 * an English branch, so an unrecognised locale degrades to complete English
 * rather than a mix.
 */
fun resolveDisplayLanguage(overrideTag: String?, system: Locale = Locale.getDefault()): DisplayLanguage =
    DisplayLanguage.fromTagOrNull(overrideTag)
        ?: if (system.language == DisplayLanguage.ID.tag) DisplayLanguage.ID else DisplayLanguage.EN
