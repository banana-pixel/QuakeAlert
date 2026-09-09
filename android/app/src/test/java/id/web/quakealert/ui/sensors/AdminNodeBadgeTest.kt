package id.web.quakealert.ui.sensors

import id.web.quakealert.data.network.mapper.toDomain
import id.web.quakealert.data.network.mapper.toStationItem
import id.web.quakealert.data.network.model.SensorDto
import id.web.quakealert.domain.DisplayLanguage
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The Admin Node badge chain (D-036): `is_admin_node` on the wire → domain →
 * card item, defaulting to false at every hop so the badge is never granted
 * by absence — the same rule trust itself follows.
 */
class AdminNodeBadgeTest {

    private fun dto(admin: Boolean? = null): SensorDto {
        val base = SensorDto(
            stationId = "NODE-163A149F",
            sensorModel = "MPU 6050",
            locationName = "Cimahi, West Java, ID",
            latitude = -6.87,
            longitude = 107.54,
            status = "Online",
            verified = true
        )
        return if (admin == null) base else base.copy(isAdminNode = admin)
    }

    @Test
    fun `absent flag defaults to no badge at every hop`() {
        val item = dto().toDomain().toStationItem()

        assertFalse(item.isAdminNode)
    }

    @Test
    fun `an explicit true survives to the card item`() {
        val item = dto(admin = true).toDomain().toStationItem()

        assertTrue(item.isAdminNode)
        assertEquals(SensorStatus.ONLINE, item.status)
    }

    @Test
    fun `an explicit false stays off`() {
        val item = dto(admin = false).toDomain().toStationItem()

        assertFalse(item.isAdminNode)
    }

    @Test
    fun `the badge copy is one proper noun in both languages`() {
        assertEquals("Admin Node", sensorStrings(DisplayLanguage.EN).adminNodeBadge)
        assertEquals("Admin Node", sensorStrings(DisplayLanguage.ID).adminNodeBadge)
    }
}
