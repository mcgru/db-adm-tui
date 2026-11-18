---
title: "Pass 1 coverage — db-adm-tui UX spines"
date: 2026-11-18
scope: "DESIGN.md + EXPERIENCE.md mechanical coverage per bmad-ux/references/validate.md"
status: working
---

# Pass 1 Coverage — db-adm-tui

Checked files:

- `_bmad-output/planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/DESIGN.md`
- `_bmad-output/planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/EXPERIENCE.md`

Sources: final PRD, brief, addendum, `.memlog.md`, `.working/color-themes-1.html`.

| Category | Verdict |
|---|---|
| 1. Flow coverage | strong |
| 2. Token completeness | strong |
| 3. Component coverage | strong |
| 4. State coverage | adequate |
| 5. Visual reference coverage | strong |

## 1. Flow coverage — strong

Every PRD journey maps to a Key Flow with Mcgru as protagonist, numbered steps, a **Climax** beat, and a failure path:

| Source | Flow | Failure covered |
|---|---|---|
| UJ-1. Mcgru answers an access question on a PostgreSQL host. | F2 — Privilege question | A section is not permitted; the rest stays visible |
| UJ-2. Mcgru finds one customer's document in a closed MongoDB. | F3 — Find one row | Invalid regexp keeps the previous view |
| UJ-3. Mcgru checks a table before a migration. | F6 — Table statistics and exact count | Count cancelled or timed out; estimate stays |
| UJ-4. Mcgru runs a read-only check on MySQL. | F4 — Query and Results; F5 — Read-only guard and r/w switch | Query error; guard block; elevated-rights warning |
| Startup, new connection, import (FR-31–FR-36, FR-46) | F1 — First connect | Missing `~/.pgpass` line; invalid URI; malformed import line |

Gaps:

- **[medium]** Connection loss and reconnect have a state row but no flow. NFR-6 defines the behavior; a short failure branch in F1 or F4 would make it testable end to end. (EXPERIENCE.md → State Patterns, "Connection lost".)
- **[low]** Import is a paragraph in F1's failure path, not its own numbered sequence. FR-32/FR-33 are covered behaviorally in `import-modal`, but no flow walks the selection and warning path.
- **[low]** Profile save is one step of F1; save failure (unwritable config directory) has no flow and no state. See §4.
- **[low]** No flow covers `Ctrl+X Ctrl+E`, completion, or help; these are specified as component behavior only (FR-21–FR-23).

## 2. Token completeness — strong

- **Colors:** 18/18 roles defined with six-digit hex. Script comparison against `.t-night` in the theme study shows **0 mismatches**. Roles preserved exactly as decided: `background-base`, `surface`, `surface-raised`, `border-default`, `border-focus`, `border-modal`, `text-primary`, `text-secondary`, `accent-primary`, `accent-secondary`, `success`, `warning`, `error`, `info`, `match-highlight`, `selection`, `readonly-mode`, `readwrite-mode`.
- **Typography:** 4 roles (`body`, `strong`, `label`, `code`), each with family note, size in cells, weight, line height. Terminal-native; no font is shipped.
- **Rounded:** 2 tokens, both 0.
- **Spacing:** 12 tokens in cells/lines, including the 80×24 viewport floor.
- **Components:** 17 component token groups, all semantic references.
- **Cross-references:** 28 distinct `{path.to.token}` references across both files; **0 unresolved**.
- **Contrast:** 11 load-bearing pairs stated; lowest text pair is 6.4:1 (short modal errors); body/data pairs are 7:1 or better.

Gaps:

- **[low]** There is no token for "dim"; it is intentionally a terminal attribute on `text-secondary` (documented in DESIGN.md → Colors/Text and Accessibility).
- **[low]** There is no `colors.focus` alias; focus is `{colors.border-focus}`. Every reference resolves, but a consumer looking for the prompt's example name will need the mapping.
- **[info]** Single dark mode is intentional; no light-mode pairs exist, so the spec's light/dark-pair rule does not apply.

## 3. Component coverage — strong

17 component names are identical in DESIGN frontmatter, DESIGN.Components, and EXPERIENCE.Component Patterns:

`top-status-line`, `status-line`, `tree`, `object-tabs`, `details-pane`, `privileges-pane`, `data-grid`, `filter-bar`, `query-bar`, `completion-popup`, `results-view`, `profile-picker`, `new-connection-modal`, `import-modal`, `confirm-modal`, `help-overlay`, `edit-modal`.

Each has a visual row (anatomy, tokens, sizing, states) and a behavioral row (use, keys, transitions, failure). No one-word descriptions.

Folded elements, all described inside a parent rather than as separate components: profile row (inside `profile-picker`), query error block (inside `results-view`), filter input line (inside `filter-bar`), password prompt (inside `new-connection-modal`), access marker (inside `tree`). No orphan components.

Gaps: none critical. **[low]** "Modals" is not a base component; four concrete modals carry the rules, which is enough for build but means a new modal has no template row.

## 4. State coverage — adequate

Surfaces walked: profile picker; import; new connection; tree; main pane (no selection, Details, Data, Privileges, Results); query bar; overlay layer; global/terminal.

Covered states: cold launch, empty profile list, connecting, connection error, password required, read-only, read-write, import empty, import malformed, tree cold load, child load, permission denied, empty database, no selection, data cold load, empty object, no filter match, filtered view, end of data, window cap, fetch error, cancelled fetch, invalid regexp, query running/cancelled/error/empty, details unavailable/no indexes/count running/count cancelled, privilege partial/not permitted, query bar disabled, blocked/rejected statement, regexp outside data, connection lost, elevated rights, history failure, no color, focus, unknown help, terminal too small, modal open.

Gaps:

- **[medium]** Profile save failure has no state or string. FR-35 stores profiles under `$XDG_CONFIG_HOME/db-adm-tui`; if that path is unwritable, the UI needs a status-line message that the connection still works. Likely: `Could not save prod-ro. The connection is still open.`
- **[medium]** Profile-storage read failure at startup has no state. The picker assumes local storage renders; a corrupted profile file should degrade to an empty list plus a warning, not a crash.
- **[low]** An unreadable import file (permissions, not malformed content) is not distinguished from a malformed line.
- **[low]** Input rejection (MQL outside the four commands, multi-statement SQL) is a sentence inside "Blocked statement", not its own state row.
- **[low]** Loading states for privileges and help assume local, instant data; no "loading…" treatment is stated.
- **[low]** Exact-count permission failure is covered by "on failure the estimate stays" in `details-pane`, but has no state-table row.

## 5. Visual reference coverage — adequate

- `.working/color-themes-1.html` is linked in DESIGN.md → Brand & Style, EXPERIENCE.md intro, and EXPERIENCE.md → Future/Open Items. It is named as a Nightshift 80×24 color study, not a layout contract; spines win on conflict.
- Three key-screen mocks are promoted to `mockups/`: `key-picker.html`, `key-workspace.html`, `key-overlays.html`. `wireframes/` and `imports/` are empty. There are no orphan files.
- Unillustrated surfaces: profile picker, new connection, import, edit modal, help, privileges. These are specified in prose and token anatomy only.
- The theme study contains chrome the spines do not adopt: clock `14:02`, `2 db`, `3 cols`, sample customer rows, status LEDs. DESIGN.md → Brand & Style flags these as illustrative so a builder does not copy them.
- Resolved: the picker/workspace/overlay mocks were rendered and linked; the profile-picker working scheme is now illustrated.

## Assumptions index

Tagged in the spines:

1. Tree default 28 cells, clamp 16 to half the terminal; query bar 1–3 lines; resize steps 2 cells / 1 line. (DESIGN.md → Layout & Spacing)
2. Modal centering, content + 2 border lines, optional background dim. (DESIGN.md → Layout & Spacing)
3. Glyph set and single-line borders; no ASCII fallback. (DESIGN.md → Shapes)
4. One live connection; no side-by-side tree; no connection-switch gesture. (EXPERIENCE.md → Foundation)
5. Tab selection resets to the context default; **Esc** from Results restores the previous tab. (EXPERIENCE.md → Information Architecture)
6. Transient status message lifetime of five seconds. (EXPERIENCE.md → Component Patterns)
7. Password-prompt placement: resolved — inside the new-connection modal; centered modal for a saved profile without a source. (EXPERIENCE.md → State Patterns)
8. Terminal-too-small behavior and string. (EXPERIENCE.md → State Patterns)
9. Chord timeout of two seconds. (EXPERIENCE.md → Interaction Primitives)
10. Resolved: `Ctrl+D` disconnects to the picker; `q` quits only at the picker. (EXPERIENCE.md → Interaction Primitives)
11. Resize clamp ranges; saved size is not overwritten on shrink. (EXPERIENCE.md → Interaction Primitives)
12. No spinner; static elapsed text. (EXPERIENCE.md → Accessibility Floor)
13. Write statements report affected rows in Results. (EXPERIENCE.md → Flow F5)

Implicit, worth confirming:

- Status-line truncation order (FR-17 gives the rule; the exact ladder is designed here).
- `q` quits only at the picker; the workspace has no quit binding.
- Profile rows never show a password, but the picker also never shows a redacted URI; only host/user/source.

## Pass 2 notes (brief)

- **Bloat:** DESIGN prose is editorial by design; EXPERIENCE uses tables and one source line per flow. No PRD section restatement. FR references replace consequence lists.
- **Inheritance:** frontmatter `sources` resolve to real relative paths. UJ names are verbatim in each flow's source line. Component names are identical in all three places. Glossary terms (`estimated`, `exact`, `read-only mode`, `Security node`, `source file`, `active batch size`) match the PRD.
- **Shape fit:** DESIGN sections are in canonical order. EXPERIENCE has all eight required defaults plus Inspiration, Responsive, Future/Open.
- **Consistency fixes made during Pass 1:** connection-loss copy now matches the Esc ladder (workspace returns to picker); tab underline is an attribute, not a second row; modal max width raised to 76 so `edit-modal` fits; contrast claim corrected to 4.5 minimum / 7 target; access markers inherit `text-primary` on a selected row.
- **Remaining risk:** the "one live connection / no disconnect gesture" assumption and the missing start-screen mockup are the two items most likely to change layout or keys after a build review.
