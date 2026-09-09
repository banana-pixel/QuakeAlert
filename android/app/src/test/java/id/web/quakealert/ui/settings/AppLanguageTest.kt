package id.web.quakealert.ui.settings

import id.web.quakealert.domain.DisplayLanguage
import org.junit.Assert.assertEquals
import org.junit.Test
import java.util.Locale

/**
 * Covers the D-022 language contract: null means System, an explicit choice
 * wins over the device locale, and unknown tags stay English.
 */
class AppLanguageTest {

    @Test
    fun `null means System`() {
        assertEquals(AppLanguage.SYSTEM, AppLanguage.fromTag(null))
    }

    @Test
    fun `explicit tags resolve to their entry`() {
        assertEquals(AppLanguage.EN, AppLanguage.fromTag("en"))
        assertEquals(AppLanguage.ID, AppLanguage.fromTag("id"))
    }

    @Test
    fun `unknown tags fall back to English`() {
        assertEquals(AppLanguage.EN, AppLanguage.fromTag("xx"))
    }

    @Test
    fun `System follows the device locale`() {
        assertEquals(DisplayLanguage.ID, AppLanguage.SYSTEM.toDisplay(Locale("in")))
        assertEquals(DisplayLanguage.ID, AppLanguage.SYSTEM.toDisplay(Locale("id")))
        assertEquals(DisplayLanguage.EN, AppLanguage.SYSTEM.toDisplay(Locale.US))
    }

    @Test
    fun `an explicit choice is never overridden by the device locale`() {
        assertEquals(DisplayLanguage.EN, AppLanguage.EN.toDisplay(Locale("in")))
        assertEquals(DisplayLanguage.ID, AppLanguage.ID.toDisplay(Locale.US))
    }

    @Test
    fun `fresh installs default to System`() {
        assertEquals(AppLanguage.SYSTEM, SettingsUiState().language)
    }
}
