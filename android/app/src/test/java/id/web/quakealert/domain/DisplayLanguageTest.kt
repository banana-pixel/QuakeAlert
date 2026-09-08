package id.web.quakealert.domain

import id.web.quakealert.data.network.mapper.QuakeFormat
import java.time.Instant
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
    fun `unknown tag defers to system, English only as last resort`() {
        assertEquals(DisplayLanguage.EN, resolveDisplayLanguage("xx", Locale.US))
        assertEquals(DisplayLanguage.ID, resolveDisplayLanguage("xx", Locale("in")))
        assertNull(DisplayLanguage.fromTagOrNull("xx"))
    }

    @Test
    fun `both Indonesian ISO codes are recognised`() {
        // Android reports `in` (legacy); the JDK normalizes Locale("in") to `id`.
        // Either spelling must resolve Indonesian — comparing one spelling only
        // silently breaks on the other platform.
        assertEquals(DisplayLanguage.ID, resolveDisplayLanguage(null, Locale("in")))
        assertEquals(DisplayLanguage.ID, resolveDisplayLanguage(null, Locale("id")))
        assertEquals("baru saja", QuakeFormat.relativeTime(
            Instant.EPOCH, Instant.EPOCH, Locale("id")
        ))
    }
}
