package id.web.quakealert.domain

import java.util.Locale
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class DisplayLanguageTest {

    @Test
    fun `stored override wins over system`() {
        assertEquals(DisplayLanguage.ID, resolveDisplayLanguage("id", Locale.US))
        assertEquals(DisplayLanguage.EN, resolveDisplayLanguage("en", Locale("in")))
    }

    @Test
    fun `empty override follows system locale`() {
        assertEquals(DisplayLanguage.ID, resolveDisplayLanguage(null, Locale("in")))
        assertEquals(DisplayLanguage.ID, resolveDisplayLanguage("", Locale("in")))
        assertEquals(DisplayLanguage.EN, resolveDisplayLanguage(null, Locale.US))
    }

    @Test
    fun `unknown tag falls back to English, never mixed`() {
        assertEquals(DisplayLanguage.EN, resolveDisplayLanguage("xx", Locale("in")))
        assertNull(DisplayLanguage.fromTagOrNull("xx"))
    }
}
