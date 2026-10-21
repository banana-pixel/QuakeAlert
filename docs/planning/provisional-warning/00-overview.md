# 00 — Provisional-warning decision package (DRAFT — waiting on owner)

- Purpose: decision-PREPARATION package for an E2 PRELIM provisional trusted-local warning, per `docs/planning/admin-node-threshold/12-prelim-early-warning-research.md` conclusion #2 and its own governance list (#6). NOTHING here modifies production; no D-number is minted; every amendment is a draft the owner may adopt, renumber, or reject.
- Status: DRAFT for owner review. Evidence status is INSUFFICIENT EVIDENCE today (file 02 §2); the package exists so that when the owed measurements land, the decision step itself takes minutes instead of weeks.
- Files:
  | File | Content | Posture |
  | --- | --- | --- |
  | `01-decision-draft-e2.md` | The decision text E2 would ask the owner to adopt (veto relaxation, reason vocab, withdrawal rule, validity policy, bar policy) | DRAFT amendments, no D-numbers |
  | `02-measurements-acceptance.md` | The 8 missing measurements mapped to what now ships (server tool, device stamps, protocols doc 17) + adapted acceptance criteria + current verdict | Status ledger, blocking list |
  | `03-contract-addendum-draft.md` | Additive wirecontract delta (`provisional` marker FCM + WS), old-build semantics proof | DRAFT, inert until server adopt |
  | `04-algo-ver-ruling-draft.md` | V3/V6/ordering-anchoring ruling with recommendation + emission-row taxonomy question | DRAFT |
  | `05-sequencing-rollback.md` | Adoption order, activation record repair linkage, rollback story, guard interactions (S1/S6/D-003/D-037/D-040) | DRAFT |
  | `06-shadow-log-runbook.md` | The S1–S3 shadow evaluator (LOG-ONLY, owner-approved, implemented this session): records, retention, export | IMPLEMENTED (log-only) |
- Relationship to existing authority: exactly-once/no-bump sits on D-003/D-037; the veto, reason vocab and withdrawal semantics amend D-036; stand-down asymmetry travels with the D-040 deployment; validity policy consumes D-018. Where this package and `docs/DECISIONS.md` disagree BEFORE adoption, `docs/DECISIONS.md` wins (PROJECT_RULES §5).
- Relationship to Problem 1 (floor value): strictly decoupled, as doc 12 concluded — a provisional bar change and the floor value are separate edits with separate blast radii; compounding them in one decision is the highest-risk combination on record.
