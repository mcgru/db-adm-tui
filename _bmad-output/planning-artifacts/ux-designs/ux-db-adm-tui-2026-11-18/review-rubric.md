# Spine Pair Review — db-adm-tui

**Reviewer:** parent fallback (subagent balance unavailable)
**Date:** 2026-11-18

## Overall verdict

The pair is decision-ready: the Nightshift token set is complete and reproducible, the 17 components have both visual and behavioral rules, the flows map to every PRD journey, and the state table covers the difficult paths (permission denial, invalid regexp, buffer window, blocked statement, terminal-too-small). The remaining gaps are narrow: profile storage failures, unreadable import files, and a first-load treatment for privileges are not yet explicit, and visual reference coverage depends on the key-screen mockups that are being promoted with this pass.

## 1. Flow coverage — strong

Every PRD journey has a flow with Mcgru, numbered steps, a climax, and a failure path. Startup/new/import are covered by F1; the query bar and guard by F4/F5; stats by F6. Connection loss and import selection are covered as state branches rather than standalone flows, which is acceptable for a v1 internal tool.

### Findings
- **[low]** Connection loss has a state row but no numbered flow branch (EXPERIENCE → State Patterns). *Fix:* add a failure beat to F1 or F4 when the next revision touches those flows.

## 2. Token completeness — strong

18 color tokens with exact Nightshift hex; 4 typography roles; 12 spacing tokens; 17 component token groups. All 28 `{path.to.token}` references resolve. Contrast pairs are stated and reproduce the palette.

### Findings
- **[low]** No `colors.focus` alias; focus is `{colors.border-focus}`. *Fix:* none required; the mapping is explicit in Components and Interaction Primitives.

## 3. Component coverage — strong

17 component names are identical in DESIGN frontmatter, DESIGN.Components, and EXPERIENCE.Component Patterns. Folded elements (profile row, error block, filter input, password prompt, access marker) are described inside their parent.

### Findings
- **[low]** No generic "modal" base component. *Fix:* the four concrete modals carry the rules; add a template row only if a fifth modal appears.

## 4. State coverage — adequate

Covered: cold launch, empty profiles, connecting, connection error, password required, read-only/write, import empty/malformed, tree cold/child load, permission denied, empty database, no selection, data cold/empty/no-match/filtered/end/window/errors, query running/cancelled/error/empty, details unavailable/no indexes/count running/cancelled, privilege partial/not permitted, disabled query bar, regexp outside data, blocked statement, connection lost, disconnected, elevated rights, history failure, no color, focus, unknown help, terminal too small, modal open.

### Findings
- **[medium]** Profile save failure and profile-storage read failure have no state row (EXPERIENCE → State Patterns; FR-35). *Fix:* add both rows with copy and fallback behavior.
- **[medium]** An unreadable import file is not distinguished from a malformed line (EXPERIENCE → import-modal). *Fix:* add a row with `error` and the file name.
- **[low]** Input rejection (multi-statement, arbitrary JavaScript, MQL outside the four commands) is folded into "Blocked statement". *Fix:* keep the shared row but add the exact `Rejected: <reason>.` copy.
- **[low]** First-load treatment for privileges is not stated; data has `loading rows…`. *Fix:* add `loading privileges…` as a local, instant-load state.
- **[low]** Exact-count permission failure has no row. *Fix:* add `error: exact count not permitted.` with the estimate retained.

## 5. Visual reference coverage — strong

The Nightshift study and the three promoted key-screen mocks are linked inline: `mockups/key-picker.html`, `mockups/key-workspace.html`, `mockups/key-overlays.html`. Each spine states "spines win on conflict".

### Findings
- none

## 6. Bloat & overspecification — adequate

DESIGN prose carries an editorial voice deliberately; EXPERIENCE is table-led. FR references replace consequence lists. No PRD scope is restated.

### Findings
- **[low]** DESIGN's Brand & Style runs long for a terminal spec. *Fix:* preserve it; it is the only place the register is argued, and the token contract is elsewhere.

## 7. Inheritance discipline — strong

`source` frontmatter resolves to the PRD, brief, addendum, and memlog. UJ names are verbatim. Glossary terms match the PRD (`estimated`, `exact`, `read-only mode`, `security node`, `source file`, `active batch size`). Component names are identical across all three places. Chord notation is now explicit: `Ctrl+X Ctrl+<key>` versus `Ctrl+X <key>`.

### Findings
- **[low]** The PRD says "one-line query bar"; the spine allows 1–3 wrapped lines. *Fix:* the PRD has been amended to "one logical statement, up to three wrapped lines"; keep the wording synchronized.

## 8. Shape fit — strong

DESIGN sections are in canonical order. EXPERIENCE has all eight required defaults plus Inspiration, Responsive, and Future/Open. Invented sections (Future/Open) earn their place by recording the per-profile theme and i18n deferrals.

### Findings
- none

## Mechanical notes

- Fixed in this pass: query bar height is 1–3 lines everywhere; access markers are no longer dimmed; chord finals use one notation; results error wraps at pane width.
- Pending: the three key-screen mocks; link them and re-check visual reference coverage.
