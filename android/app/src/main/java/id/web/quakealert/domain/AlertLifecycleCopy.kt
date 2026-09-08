package id.web.quakealert.domain

/**
 * Copy for the two lifecycle facts the wire `type` cannot express: that an
 * unconfirmed tremor is unconfirmed, and that a stood-down alert was withdrawn
 * rather than ended.
 *
 * Lives here, and not in the ViewModel that renders it, for one reason: the wording
 * is the safety property (§13.3), so it has to be assertable without an Android
 * framework class around it.
 */

/**
 * What the user is told when an alert stands down.
 *
 * Two cases, and they are not the same claim (server §13.3):
 *
 *  - [EventState.RESOLVED] — the event ended: no station has reported new shaking
 *    for the server's resolution window. "All clear" is honest here.
 *  - [EventState.CANCELLED] — the *report* was withdrawn. Its evidence was
 *    invalidated (a node's verification was revoked) or an operator retracted it.
 *    Saying "the shaking has ended" would assert something the server never
 *    concluded, and in the invalidated case something it now believes never happened.
 *
 * Both arrive as `EVENT_RESOLVED` on the wire — the `type` enum is frozen so that an
 * un-updated install still clears its alarm (see [EventState]) — so this distinction
 * exists only in the copy, and only for a build that knows the state. A null or
 * unrecognised state therefore falls back to the all-clear wording: it is what every
 * pre-Phase-3 client already said, and an all-clear is never the unsafe direction.
 *
 * Neither wording may mention magnitude, epicentre or an arrival countdown, for the
 * same reason no other copy on this screen may.
 */
data class StandDownCopy(
    val title: String,
    val detail: String
)

/** All-clear copy for [state]; see [StandDownCopy] for why null is not a third case. */
fun standDownCopyFor(state: EventState?, lang: DisplayLanguage = DisplayLanguage.EN): StandDownCopy =
    when (lang) {
        DisplayLanguage.ID -> standDownCopyId(state)
        DisplayLanguage.EN -> standDownCopyEn(state)
    }

private fun standDownCopyEn(state: EventState?): StandDownCopy = when (state) {
    EventState.CANCELLED -> StandDownCopy(
        title = "Report Withdrawn",
        detail = "The earthquake report was withdrawn and is no longer active."
    )

    EventState.UNCONFIRMED,
    EventState.CONFIRMED,
    EventState.RESOLVED,
    null -> StandDownCopy(
        title = "All Clear",
        detail = "No further shaking reported nearby."
    )
}

// Indonesian branch (B2). Acuan: glosarium BMKG/BPBD — "Aman" untuk akhir
// kejadian, "Laporan Ditarik" untuk laporan yang dicabut; bukan literal.
private fun standDownCopyId(state: EventState?): StandDownCopy = when (state) {
    EventState.CANCELLED -> StandDownCopy(
        title = "Laporan Ditarik",
        detail = "Laporan gempa bumi ini ditarik dan sudah tidak aktif."
    )

    EventState.UNCONFIRMED,
    EventState.CONFIRMED,
    EventState.RESOLVED,
    null -> StandDownCopy(
        title = "Aman",
        detail = "Tidak ada guncangan susulan yang dilaporkan di dekat Anda."
    )
}

/**
 * Idle-banner read-out while an UNCONFIRMED tremor is being evaluated: "1 station is
 * reporting shaking - not yet confirmed by separated stations", and its plural.
 *
 * States what the network has felt and what it has not concluded, and nothing else.
 * No magnitude, no epicentre, no arrival countdown — none of which a surface MEMS
 * network estimates. [nodeCount] counts NODES, not observations: a node that sent
 * both a preliminary and a final reading counts once, which is the server's own
 * definition of the field. A zero or absent count drops the number rather than
 * printing "0 stations", which would read as no evidence at all.
 */
fun unconfirmedActivityLabel(nodeCount: Int, lang: DisplayLanguage = DisplayLanguage.EN): String {
    if (lang == DisplayLanguage.ID) return unconfirmedActivityLabelId(nodeCount)
    val subject = when {
        nodeCount <= 0 -> "A station is"
        nodeCount == 1 -> "1 station is"
        else -> "$nodeCount stations are"
    }
    return "$subject reporting shaking - not yet confirmed by separated stations"
}

// Indonesian branch (B2). Bahasa Indonesia tidak mengenal plural, sehingga
// tiga bentuk subjek Inggris menyatu menjadi satu.
private fun unconfirmedActivityLabelId(nodeCount: Int): String {
    val subject = when {
        nodeCount <= 0 -> "Satu stasiun"
        else -> "$nodeCount stasiun"
    }
    return "$subject melaporkan guncangan - belum dikonfirmasi oleh stasiun yang terpisah"
}
