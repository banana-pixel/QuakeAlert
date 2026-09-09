package id.web.quakealert.ui.common

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import id.web.quakealert.R
import id.web.quakealert.data.network.ServerHealth
import id.web.quakealert.domain.DisplayLanguage
import id.web.quakealert.ui.theme.CardBorder
import id.web.quakealert.ui.theme.ConnectingBadgeFill
import id.web.quakealert.ui.theme.Dimens
import id.web.quakealert.ui.theme.FilterInactiveFill
import id.web.quakealert.ui.theme.HealthyBadgeFill
import id.web.quakealert.ui.theme.NunitoFontFamily
import id.web.quakealert.ui.theme.OfflineBadgeFill
import id.web.quakealert.ui.theme.PillFill
import id.web.quakealert.ui.theme.TextPrimary

/**
 * Shared screen header used by all five main tabs (Figma nodes 1:705 / 1:1082): a
 * large title on the left and the server-status badge on the right. Extracted to
 * [ui.common] so every screen shares a single source of truth for the layout and
 * token wiring instead of duplicating it.
 *
 * The badge is driven by the global [ServerHealth] verdict rather than by anything
 * the calling screen loaded. That is the point: it reports whether the *backend* can
 * do its job, so an empty station roll or an all-offline node fleet cannot make one
 * tab claim the network is down while another says it is up. Per-screen health
 * (station chips, active-sensor counts) stays inside the screens that own it.
 *
 * It is always present, in one of four states, rather than shown only while healthy.
 * A badge that vanishes when the link drops says nothing at all — and "nothing" is
 * indistinguishable from a healthy app with a quiet network, which is the one reading
 * this app must never allow. On Warning the badge is joined by
 * [id.web.quakealert.ui.warning.WarningOfflineNotice], which spells out what a dropped
 * link means for the alerts themselves.
 *
 * @param title the screen title (e.g. "History", "Sensors").
 * @param health the global verdict from [id.web.quakealert.data.network.ServerHealthMonitor];
 *   the label is rendered verbatim, so the UI never sees the internal word for degradation —
 *   users get "Limited", not jargon.
 * @param onUpdatesClicked opens the Updates overlay (Figma node 158-1645 places the
 *   notification-text glyph beside the server-status pill). Null — the default — renders no
 *   control, so a caller that cannot host the overlay simply leaves it out.
 */
@Composable
fun QuakeAppBar(
    title: String,
    health: ServerHealth,
    modifier: Modifier = Modifier,
    onUpdatesClicked: (() -> Unit)? = null,
    lang: DisplayLanguage = DisplayLanguage.EN
) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .statusBarsPadding(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = title,
            color = TextPrimary,
            fontFamily = NunitoFontFamily,
            fontWeight = FontWeight.ExtraBold,
            fontSize = 24.sp,
            lineHeight = 26.sp
        )

        Row(verticalAlignment = Alignment.CenterVertically) {
            if (onUpdatesClicked != null) {
                UpdatesIconButton(
                    onClick = onUpdatesClicked,
                    contentDescription = if (lang == DisplayLanguage.ID) {
                        "Buka pembaruan"
                    } else {
                        "Open Updates"
                    }
                )
                // Same gap as the All/Near pill pair, so the header's right
                // cluster reads at the same rhythm as the filter row below it.
                Spacer(Modifier.width(Dimens.FilterRowGap))
            }
            ServerHealthBadge(health = health, lang = lang)
        }
    }
}

/**
 * The Updates entry point in the app bar (Figma node 158-1645): the filter
 * sheet button's chrome (dark fill, 1dp stroke, radius, 20dp glyph) with the
 * status badge's box model, so it renders at the badge's height and the pair
 * reads as one proportional cluster. Hug size with no touch-target padding,
 * exactly like its filter twin (and unlike a Material `IconButton`, whose
 * 48dp minimum is what used to stretch this row a head taller than the
 * design).
 */
@Composable
private fun UpdatesIconButton(
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    contentDescription: String = "Open Updates"
) {
    val shape = RoundedCornerShape(Dimens.RadiusSmall)
    Box(
        modifier = modifier
            .clip(shape)
            .background(FilterInactiveFill, shape)
            .border(Dimens.BorderThin, CardBorder, shape)
            .clickable(role = Role.Button, onClick = onClick)
            // Same box model as the status badge beside it (5dp horizontal,
            // 4dp vertical around a ~20dp-tall content row), so the two render
            // at the same height instead of merely similar ones.
            .padding(
                horizontal = Dimens.BadgePaddingHorizontal,
                vertical = Dimens.BadgePaddingVertical
            ),
        contentAlignment = Alignment.Center
    ) {
        Icon(
            painter = painterResource(id = R.drawable.ic_updates_notification),
            contentDescription = contentDescription,
            tint = TextPrimary,
            modifier = Modifier.size(Dimens.FilterTriggerGlyphSize)
        )
    }
}

/**
 * Server-status pill (Figma node 1:708 is the healthy variant). The other three
 * variants reuse its geometry unchanged and differ only in fill, glyph and word, so
 * the badge stays one recognisable object across all four.
 *
 * The severity ramp reads grey → green → amber → red: Checking has no verdict yet,
 * Healthy needs no explanation, Limited says "something behind the server is not
 * right" without borrowing Offline's alarm (the globe keeps standing for a place
 * that answers; the triangle is reserved for the one state where alerts cannot
 * arrive).
 */
@Composable
private fun ServerHealthBadge(
    health: ServerHealth,
    modifier: Modifier = Modifier,
    lang: DisplayLanguage = DisplayLanguage.EN
) {
    val (fill, glyph) = when (health) {
        // Neutral grey, not amber: no verdict yet must not read as a caution.
        ServerHealth.CHECKING -> Pair(PillFill, R.drawable.ic_globe)
        ServerHealth.HEALTHY -> Pair(HealthyBadgeFill, R.drawable.ic_globe)
        ServerHealth.LIMITED -> Pair(ConnectingBadgeFill, R.drawable.ic_globe)
        ServerHealth.OFFLINE -> Pair(OfflineBadgeFill, R.drawable.ic_alert_triangle)
    }
    val label = health.label(lang)
    val statusDescription = if (lang == DisplayLanguage.ID) {
        "Status server: $label"
    } else {
        "Server status: $label"
    }

    Row(
        modifier = modifier
            .semantics { contentDescription = statusDescription }
            .background(fill, RoundedCornerShape(Dimens.RadiusSmall))
            .border(Dimens.BorderThin, CardBorder, RoundedCornerShape(Dimens.RadiusSmall))
            .padding(
                horizontal = Dimens.BadgePaddingHorizontal,
                vertical = Dimens.BadgePaddingVertical
            ),
        horizontalArrangement = Arrangement.spacedBy(Dimens.BadgeIconGap),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Icon(
            painter = painterResource(id = glyph),
            // The row's semantics already announce the whole statement once; a
            // contentDescription here would make TalkBack read the glyph first.
            contentDescription = null,
            tint = TextPrimary,
            modifier = Modifier.size(16.dp)
        )
        Text(
            text = label,
            color = TextPrimary,
            fontFamily = NunitoFontFamily,
            fontWeight = FontWeight.Bold,
            fontSize = 15.sp
        )
    }
}

/**
 * Badge word in [lang]. Pure so the Indonesian copy stays pinned by unit test;
 * "Offline" is deliberately the same in both — it is the technical term users say.
 */
internal fun ServerHealth.label(lang: DisplayLanguage): String {
    if (lang == DisplayLanguage.ID) {
        return when (this) {
            ServerHealth.CHECKING -> "Memeriksa…"
            ServerHealth.HEALTHY -> "Normal"
            ServerHealth.LIMITED -> "Terbatas"
            ServerHealth.OFFLINE -> "Offline"
        }
    }
    return when (this) {
        ServerHealth.CHECKING -> "Checking…"
        ServerHealth.HEALTHY -> "Healthy"
        ServerHealth.LIMITED -> "Limited"
        ServerHealth.OFFLINE -> "Offline"
    }
}
