package id.web.quakealert.domain

/**
 * Bounded coexistence of concurrent alert events (D-020, U-011).
 *
 * The single process-wide instance lives on [QuakeNetwork.activeAlerts] next to
 * [AlertDedup]: every delivery surface (push service, foreground socket
 * ViewModel, background bridge, notifier, Activity observer) reads the same
 * board, so two transports can never disagree about which events are live.
 * Like dedup it is process-lifetime by design — after a process death the map
 * is empty and redelivered frames rebuild as news — which is the documented
 * limitation, not a defect discovered later.
 *
 * Policy, exactly as decided (no more, no less):
 * - at most [MAX_FULL] full entries (3), insertion-ordered;
 * - a new id beyond the cap evicts the oldest full entry into [collapsedIds],
 *   which keeps its identity (for stand-down matching and the count) while
 *   dropping its presentation;
 * - [selectedId] is the newest entry with `sounded=true`, else the newest entry;
 * - mute is per event id, shared by every surface;
 * - stand-down removes one id; if it was selected, the next-latest is promoted
 *   visually and MUST NOT sound (the caller enforces silence; the board only
 *   reports who was promoted);
 * - blank-id legacy stand-down acts only when exactly one event is live.
 *
 * Pure Kotlin, no Android imports: fully unit-testable on the JVM. All members
 * are synchronized; concurrent FCM/WS delivery is the normal case, not an edge.
 */
class ActiveAlertBoard(val maxFull: Int = MAX_FULL) {

    data class Slot(
        val eventId: String,
        val notificationId: Int,
        var sounded: Boolean,
        var muted: Boolean = false
    )

    data class UpsertResult(
        val notificationId: Int,
        /** Previously live full ids still present (they persist silently). */
        val supersededIds: List<String>,
        /** Older full entry evicted into the collapsed set, if any. */
        val collapsedId: String?,
        /** Ids whose notifications must be cancelled (the evicted slot). */
        val cancelNotificationIds: List<Int>,
        val selectedId: String
    )

    data class StandDownResult(
        val cancelledNotificationIds: List<Int>,
        /** New selected id after removal; the caller renders it WITHOUT sound. */
        val promotedId: String?,
        /** Whether anything was actually removed (vs unknown id). */
        val removedAny: Boolean
    )

    sealed interface BlankStandDown {
        data object NoneActive : BlankStandDown

        /**
         * The single live event was removed. [notificationId] is null when it
         * held no notification (collapsed identity): nothing to cancel.
         */
        data class ClearedSingle(val notificationId: Int?) : BlankStandDown
        data object IgnoredAmbiguous : BlankStandDown
    }

    private val entries = LinkedHashMap<String, Slot>()
    private val collapsedIds = LinkedHashSet<String>()
    private val freeIds = ArrayDeque((BASE_NOTIFICATION_ID until BASE_NOTIFICATION_ID + MAX_FULL).toList())

    /** Full entries in insertion order (oldest first). */
    @Synchronized
    fun fullEntries(): List<Slot> = entries.values.toList()

    /** Collapsed live ids (identity retained for stand-down matching + count). */
    @Synchronized
    fun collapsedIds(): Set<String> = collapsedIds.toSet()

    /** Every live id: full entries plus collapsed. */
    @Synchronized
    fun totalActive(): Int = entries.size + collapsedIds.size

    /** Newest sounded entry, else newest entry, else null. */
    @Synchronized
    fun selectedId(): String? =
        entries.values.lastOrNull { it.sounded }?.eventId
            ?: entries.keys.lastOrNull()

    /** Live events beyond the selected one (full others + collapsed). */
    @Synchronized
    fun extraActiveCount(): Int = (totalActive() - if (selectedId() == null) 0 else 1)
        .coerceAtLeast(0)

    @Synchronized
    fun notificationIdFor(eventId: String): Int? = entries[eventId]?.notificationId

    /**
     * Records a live event. Idempotent per id: redelivery returns the same
     * notification id and changes nothing except OR-ing `sounded`.
     */
    @Synchronized
    fun upsert(eventId: String, sounded: Boolean): UpsertResult {
        require(eventId.isNotBlank()) { "eventId must be non-blank" }
        entries[eventId]?.let { existing ->
            existing.sounded = existing.sounded || sounded
            return UpsertResult(
                notificationId = existing.notificationId,
                supersededIds = entries.keys.filter { it != eventId },
                collapsedId = null,
                cancelNotificationIds = emptyList(),
                selectedId = selectedId() ?: eventId
            )
        }
        // Return path for a previously collapsed id: it left only its identity
        // behind, so re-admitting it is a fresh insertion, not a resurrection.
        collapsedIds.remove(eventId)
        var collapsedId: String? = null
        val cancelIds = mutableListOf<Int>()
        if (entries.size >= maxFull) {
            val oldestKey = entries.keys.first()
            val oldest = entries.remove(oldestKey)!!
            freeIds.addLast(oldest.notificationId)
            cancelIds.add(oldest.notificationId)
            collapsedIds.add(oldestKey)
            collapsedId = oldestKey
        }
        val nid = freeIds.removeFirst()
        entries[eventId] = Slot(eventId = eventId, notificationId = nid, sounded = sounded)
        return UpsertResult(
            notificationId = nid,
            supersededIds = entries.keys.filter { it != eventId },
            collapsedId = collapsedId,
            cancelNotificationIds = cancelIds,
            selectedId = selectedId() ?: eventId
        )
    }

    /**
     * Removes one event by id, from full entries or the collapsed set.
     * Never sounds the promoted entry — the caller renders it silently.
     */
    @Synchronized
    fun standDown(eventId: String): StandDownResult {
        if (eventId.isBlank()) return StandDownResult(emptyList(), selectedId(), false)
        val removed = entries.remove(eventId)
        if (removed != null) {
            freeIds.addLast(removed.notificationId)
            return StandDownResult(
                cancelledNotificationIds = listOf(removed.notificationId),
                promotedId = selectedId(),
                removedAny = true
            )
        }
        if (collapsedIds.remove(eventId)) {
            return StandDownResult(emptyList(), selectedId(), removedAny = true)
        }
        return StandDownResult(emptyList(), selectedId(), removedAny = false)
    }

    /** Legacy blank-id stand-down: acts only when exactly one event is live. */
    @Synchronized
    fun standDownBlank(): BlankStandDown = when (totalActive()) {
        0 -> BlankStandDown.NoneActive
        1 -> {
            val single = entries.keys.firstOrNull() ?: collapsedIds.first()
            val slot = entries.remove(single)
            if (slot != null) {
                freeIds.addLast(slot.notificationId)
                BlankStandDown.ClearedSingle(slot.notificationId)
            } else {
                collapsedIds.remove(single)
                BlankStandDown.ClearedSingle(null)
            }
        }
        else -> BlankStandDown.IgnoredAmbiguous
    }

    @Synchronized
    fun setMuted(eventId: String, muted: Boolean) {
        entries[eventId]?.muted = muted
    }

    @Synchronized
    fun isMuted(eventId: String): Boolean = entries[eventId]?.muted ?: false

    companion object {
        /** First notification id; the pool is BASE..BASE+MAX_FULL-1. */
        const val BASE_NOTIFICATION_ID = 4301

        /** Maximum fully co-represented concurrent events (D-020). */
        const val MAX_FULL = 3
    }
}
