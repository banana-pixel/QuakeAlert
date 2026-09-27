# 00 — Overview: Admin Node threshold calibration (RESEARCH ONLY)

- Purpose: central entry point for the Admin Node trusted-local warning threshold research. Decides nothing; maps the question, the safety model, and the file set.
- Status: RESEARCH IN PROGRESS — planning package created 2026-09-11, no production change.
- Last updated: 2026-09-11 UTC.
- Authority note: this directory is a RESEARCH planning area. It does not override `/contracts`, `PROJECT_RULES.md`, `docs/DECISIONS.md`, or `docs/CURRENT_STATE.md` (PROJECT_RULES §5). Any conflict → higher document wins and this is the defect.
- Stop condition (binding): no production code, threshold, dispatch, contract, `algo_ver`, VPS, or ESP32-firmware change under this research. Allowed: docs, offline replay/benchmark scripts, test datasets, reports.

## Research question

Is 140 gal unnecessarily conservative, and could a lower threshold such as 100 gal provide earlier useful warnings while retaining acceptably low false-warning risk? No answer is assumed. Evidence decides; insufficient evidence → DO NOT CHANGE PRODUCTION.

## Temporary research value

`ADMIN_LOCAL_WARNING_PGA_GAL = 100 gal` is a CANDIDATE for benchmarking only. Production remains `AdminNodeMinPGAGal = 140.0` (`server/internal/event/admin_node.go:30`). 100 gal is NOT approved.

## Project goal (from task, consistent with PROJECT_RULES §1/S2/S4)

Detect potentially significant motion as quickly as reasonably possible; warn nearby (esp. sleeping) users quickly with loud/high-priority alarm; prioritize reliability, simplicity, safety; avoid excessive false alarms. NOT a perfect EEW system. Network density is never replaced by threshold tuning (S2).

## Admin Node safety model (current, FACT)

Designated Admin Node + verified + fresh heartbeat (≤5 min) + UNCONFIRMED event + admin contribution + FINAL phase + PGA ≥ 140 gal + token-targeted local dispatch (20 km, no GeoTopic fallback, no national broadcast). Normal CONFIRMED quorum (≥16.6 gal + ≥3 nodes + ≥2 cells/5 km) unchanged. Details: `01-current-state.md`.

## File map

| File | Contents |
| --- | --- |
| `00-overview.md` | this file |
| `01-current-state.md` | implementation as read from source (NOT summaries) |
| `02-research-questions.md` | questions + stop/allow list |
| `03-hypotheses.md` | falsifiable hypotheses, preserved when superseded |
| `04-data-plan.md` | what data, where, grouping by `algo_ver` |
| `05-experiment-plan.md` | benchmark matrix 60–160+ gal, metrics A–F |
| `06-replay-plan.md` | "if threshold had been X" method on existing rows |
| `07-threshold-comparison.md` | comparison table (filled as evidence arrives) |
| `08-safety-analysis.md` | per-candidate risk, governance impact |
| `09-findings.md` | evidence matrix + ledger pointer |
| `10-decision.md` | criteria (PROPOSED) + recommendation slot |
| `RESEARCH_LEDGER.md` | append-only ledger, every discovery |
| `CHANGELOG.md` | every planning-file change |

## Sources / evidence

- Source code as read 2026-09-11: `server/internal/event/admin_node.go`, `tracker.go`, `emit.go`, `event.go`, `classify.go`, `snapshot.go`, `store.go`; `server/internal/dispatch/event_frame.go`, `fcm.go`, `ws.go`, `severity.go`, `dispatcher.go`; `server/internal/consensus/engine.go`, `centroid.go`; `server/internal/store/admin_node.go`; `firmware/src/sensor.cpp`, `config.h`, `state.h`, `firmware.ino`, `mqtt.cpp`, `onset.cpp`; `android/.../domain/AlertGate.kt`, `SafetyPolicy.kt`.
- Governance: `PROJECT_RULES.md` §§1,2,5,6,8,10,11; `docs/DECISIONS.md` D-003/004/006/007/009/012/013/018/019/036; `docs/CURRENT_STATE.md`; `ROADMAP.md` Phase 4/F.
- Field evidence: `docs/evidence/live-eq-2026-09-12-001/AUDIT.md`; `docs/evidence/p4-m1`, `p4-m6`.
