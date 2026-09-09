package id.web.quakealert.ui.settings

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import id.web.quakealert.R
import id.web.quakealert.domain.DisplayLanguage
import id.web.quakealert.ui.common.QuakeModalHeader
import id.web.quakealert.ui.theme.AboutActionDonateFill
import id.web.quakealert.ui.theme.AboutActionEmailFill
import id.web.quakealert.ui.theme.AboutActionGithubFill
import id.web.quakealert.ui.theme.AboutModalGradient
import id.web.quakealert.ui.theme.BorderLight
import id.web.quakealert.ui.theme.CardBorder
import id.web.quakealert.ui.theme.ChipLabel
import id.web.quakealert.ui.theme.Dimens
import id.web.quakealert.ui.theme.ModalBodyText
import id.web.quakealert.ui.theme.QuakeAlertTheme

/**
 * About-specific external destinations (contact, support). Project URLs
 * (repository, profile, published site) live in [ProjectLinks] instead, so the
 * two pages that link at GitHub can never disagree again.
 */
object AboutLinks {

    /** Author contact — "Email" action. Pre-fills a subject line. */
    const val EMAIL = "mailto:wiratara006@gmail.com?subject=QuakeAlert%20Feedback"

    /** Support the project via Saweria — "Donate" action. */
    const val DONATE = "https://saweria.co/bananapixel"
}

/** Mission statement (Figma node 4:672, first paragraph). */
private fun aboutMission(lang: DisplayLanguage): String =
    if (lang == DisplayLanguage.ID) {
        "QuakeAlert dibangun untuk menyediakan sistem peringatan yang dapat " +
            "diakses semua orang, terutama untuk negara atau tempat yang belum " +
            "memiliki sistem peringatan dini gempa bumi. Saya harap aplikasi " +
            "ini dapat menyelamatkan nyawa."
    } else {
        "QuakeAlert is built to provide a warning system that can be accessed for " +
            "everyone, especially for the countries or places that don’t have early " +
            "warning system for earthquake. I hope this app can save lives."
    }

/** Feedback invitation (Figma node 4:672, second paragraph). */
private fun aboutFeedback(lang: DisplayLanguage): String =
    if (lang == DisplayLanguage.ID) {
        "Jika Anda punya saran atau menemukan bug, jangan ragu menghubungi saya."
    } else {
        "If you have some suggestion or found any bugs, feel free to contact me."
    }

/** Author attribution (Figma node 4:672, closing line). Not translated: a name. */
private const val ABOUT_ATTRIBUTION = "by @banana-pixel (Vito Wiratara)"

/**
 * The About overlay hosted in its own [Dialog] window (Figma node 4:654). Sits on
 * top of the Settings screen with the platform scrim behind it; back press and
 * taps outside the card both route to [onDismiss], so navigation is never
 * trapped.
 *
 * [DialogProperties.usePlatformDefaultWidth] is disabled so the card can span the
 * full content width the design calls for, inset by the shared
 * [Dimens.ScreenHorizontalPadding] rather than Material's narrower dialog width.
 *
 * @param onDismiss invoked by the close button, back press or an outside tap.
 * @param onGithubClick invoked by the "GitHub" action.
 * @param onEmailClick invoked by the "Email" action.
 * @param onDonateClick invoked by the "Donate" action.
 */
@Composable
fun AboutModalDialog(
    onDismiss: () -> Unit,
    onGithubClick: () -> Unit,
    onEmailClick: () -> Unit,
    onDonateClick: () -> Unit,
    lang: DisplayLanguage = DisplayLanguage.EN
) {
    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false)
    ) {
        AboutModal(
            onDismiss = onDismiss,
            onGithubClick = onGithubClick,
            onEmailClick = onEmailClick,
            onDonateClick = onDonateClick,
            lang = lang,
            modifier = Modifier.padding(Dimens.ScreenHorizontalPadding)
        )
    }
}

/**
 * Stateless About modal card (Figma node 4:668): a dark rounded surface filled
 * with the teal → near-black vertical gradient and a subtle white-10% stroke,
 * stacking four sections 26dp apart:
 *  1. Header — the shared [QuakeModalHeader]: centered "About" title with a
 *     circular close (X) button trailing.
 *  2. Logo badge — concentric glowing discs around the seismograph glyph.
 *  3. Body copy — mission, feedback note and the author attribution.
 *  4. Actions — "GitHub" + "Email" on one row, full-width "Donate" below.
 *
 * The card scrolls internally so the copy stays reachable on short viewports
 * (landscape, large font scales) instead of being clipped by the dialog window.
 *
 * Exposed separately from [AboutModalDialog] so it can be previewed and tested
 * without a dialog window.
 */
@Composable
fun AboutModal(
    onDismiss: () -> Unit,
    onGithubClick: () -> Unit,
    onEmailClick: () -> Unit,
    onDonateClick: () -> Unit,
    modifier: Modifier = Modifier,
    lang: DisplayLanguage = DisplayLanguage.EN
) {
    val shape = RoundedCornerShape(Dimens.RadiusCard)

    Column(
        modifier = modifier
            .fillMaxWidth()
            .clip(shape)
            .background(AboutModalGradient, shape)
            .border(Dimens.BorderThin, CardBorder, shape)
            .verticalScroll(rememberScrollState())
            .padding(Dimens.ModalPadding),
        verticalArrangement = Arrangement.spacedBy(Dimens.AboutModalSectionGap)
    ) {
        QuakeModalHeader(onDismiss = onDismiss, title = if (lang == DisplayLanguage.ID) "Tentang" else "About")
        AboutLogoBadge()
        AboutModalBody(lang)
        AboutModalActions(
            lang = lang,
            onGithubClick = onGithubClick,
            onEmailClick = onEmailClick,
            onDonateClick = onDonateClick
        )
    }
}

/**
 * Central logo badge (Figma node 4:670). Official QuakeAlert logo
 * (`R.drawable.ic_quake_logo`, transparent variant, D-023) drawn rounded via
 * `clip(RoundedCornerShape)`. Shape and motif verbatim from `docs/brand/`.
 */
@Composable
private fun AboutLogoBadge(modifier: Modifier = Modifier) {
    Box(
        modifier = modifier
            .fillMaxWidth()
            .height(Dimens.AboutModalLogoSize),
        contentAlignment = Alignment.Center
    ) {
        Image(
            painter = painterResource(id = R.drawable.ic_quake_logo),
            contentDescription = null,
            modifier = Modifier
                .size(Dimens.AboutModalLogoSize)
                .clip(RoundedCornerShape(Dimens.RadiusCard))
        )
    }
}

/**
 * Modal body copy (Figma node 4:672): a single centered text block whose three
 * paragraphs are separated by blank lines, so the gaps inherit the 22sp line
 * height rather than needing their own spacing token.
 *
 * The closing attribution is highlighted by stepping up to ExtraBold against the
 * surrounding Bold. A colour accent is deliberately avoided — the app reserves
 * tinted text for tappable links (`TextLink`), and this line is not one.
 */
@Composable
private fun AboutModalBody(lang: DisplayLanguage = DisplayLanguage.EN, modifier: Modifier = Modifier) {
    Text(
        text = buildAnnotatedString {
            append(aboutMission(lang))
            append("\n\n")
            append(aboutFeedback(lang))
            append("\n\n")
            withStyle(SpanStyle(fontWeight = FontWeight.ExtraBold)) {
                append(ABOUT_ATTRIBUTION)
            }
        },
        style = ModalBodyText,
        modifier = modifier.fillMaxWidth()
    )
}

/**
 * Bottom action block (Figma node 4:673): "GitHub" and "Email" share a row
 * as equal halves, with the full-width "Donate" button beneath. Rows and the
 * in-row gap both use the design's 20dp spacing.
 */
@Composable
private fun AboutModalActions(
    onGithubClick: () -> Unit,
    onEmailClick: () -> Unit,
    onDonateClick: () -> Unit,
    modifier: Modifier = Modifier,
    lang: DisplayLanguage = DisplayLanguage.EN
) {
    // "GitHub" is a proper noun: identical in both languages, so no new strings.
    val githubLabel = "GitHub"
    val emailLabel = "Email"
    val donateLabel = if (lang == DisplayLanguage.ID) "Donasi" else "Donate"
    Column(
        modifier = modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(Dimens.AboutModalActionGap)
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(Dimens.AboutModalActionGap),
            verticalAlignment = Alignment.CenterVertically
        ) {
            AboutActionButton(
                label = githubLabel,
                fill = AboutActionGithubFill,
                onClick = onGithubClick,
                modifier = Modifier.weight(1f)
            )
            AboutActionButton(
                label = emailLabel,
                fill = AboutActionEmailFill,
                onClick = onEmailClick,
                modifier = Modifier.weight(1f)
            )
        }

        AboutActionButton(
            label = donateLabel,
            fill = AboutActionDonateFill,
            onClick = onDonateClick,
            modifier = Modifier.fillMaxWidth()
        )
    }
}

/**
 * A single About action button (Figma nodes 4:677 / 4:681 / 4:686): a fixed 34dp
 * stadium-cornered box with a 31%-alpha [fill], a 2dp white-30% stroke and a
 * centered [ChipLabel]. Only the fill hue distinguishes the three actions, so
 * they share one implementation.
 */
@Composable
private fun AboutActionButton(
    label: String,
    fill: Color,
    onClick: () -> Unit,
    modifier: Modifier = Modifier
) {
    val shape = RoundedCornerShape(Dimens.RadiusSmall)

    Box(
        modifier = modifier
            .height(Dimens.ModalActionHeight)
            .clip(shape)
            .background(fill, shape)
            .border(Dimens.BorderMedium, BorderLight, shape)
            .clickable(onClick = onClick)
            .padding(horizontal = Dimens.ModalActionPaddingHorizontal),
        contentAlignment = Alignment.Center
    ) {
        Text(text = label, style = ChipLabel)
    }
}

@Preview(showBackground = true, backgroundColor = 0xFF000000)
@Composable
private fun AboutModalPreview() {
    QuakeAlertTheme {
        AboutModal(
            onDismiss = {},
            onGithubClick = {},
            onEmailClick = {},
            onDonateClick = {},
            modifier = Modifier.padding(Dimens.ScreenHorizontalPadding)
        )
    }
}
