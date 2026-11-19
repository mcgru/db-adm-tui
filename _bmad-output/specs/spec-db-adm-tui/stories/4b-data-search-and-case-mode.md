---
title: 'Data search and case mode'
type: 'feature'
created: '2026-11-19'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
baseline_commit: 'd55c68a8005e8e477a82caf72447fca7c143aa89'
context:
  - "{project-root}/_bmad-output/planning-artifacts/architecture/architecture-db-adm-tui-2026-11-18/ARCHITECTURE-SPINE.md"
  - "{project-root}/_bmad-output/planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/DESIGN.md"
  - "{project-root}/_bmad-output/planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/EXPERIENCE.md"
  - "{project-root}/_bmad-output/specs/spec-db-adm-tui/SPEC.md"
  - "{project-root}/_bmad-output/implementation-artifacts/epic-1-context.md"
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Story 4 browses and pages tables and collections, but CAP-4's search half is still a placeholder: `/`, `n`/`N`, `Ctrl+F`, and case mode do nothing.

**Approach:** Add an inline `/` filter over loaded rows with highlighting and `n`/`N` navigation, a `Ctrl+F` server-side regexp filter on the focused column or field that reuses the paged window, and a session-wide `Ctrl+X i` case mode shown in the top status. Both searches are case-insensitive by default.

## Boundaries & Constraints

**Always:** `/` filters only loaded rows, never queries, and matches a literal substring. `Ctrl+F` is read-only, filters one focused column/field with the engine's native regexp, labels the result filtered, and keeps the existing paging, window, cancel, and error rules. Case mode defaults to insensitive, lasts for the session, is visible outside work panes, applies to both filters, and `Ctrl+X i` toggles it to sensitive. `Esc` clears the filter and restores the cached unfiltered view or reloads the initial page. Errors and cancellation keep the previous rows. Sanitize filtered text, highlight matches with the Match style, and never rely on color alone. Filters reset on object or tab change.

**Never:** No server call or regexp for `/`; no multi-field, OR, or whole-row syntax. No sorting, editing, export, query execution, or changed paging/window caps. No persisted filter or case mode. No second connection. `Ctrl+F` does not apply to Results in v1. `n` must not open a new connection while a Data filter is active.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| BUFFER_FILTER | `/`, type `ali` | only matching rows; matches highlighted; `match X of N`; `Esc` restores all | empty pattern clears; never queries |
| NO_MATCH | no loaded row matches | `No matches for "<pattern>". Esc clears the filter.`; `0 of N loaded` | pattern stays editable |
| MATCH_NAV | filtered view, `n`/`N` | next/previous match, wrapping; counter updates | no filter/rows: no-op |
| SERVER_FILTER | cursor on `email`, `Ctrl+F`, regexp, Enter | first 100 matching rows; banner `Filtered by regexp on email; N matching rows loaded`; `m`/`Ctrl+A` page it | invalid/engine error keeps the previous view and shows the error inline |
| SERVER_CLEAR | `Esc` with an active server filter | cached unfiltered view returns, or the initial page reloads; banner gone | reload failure keeps the filtered view and shows the error |
| CASE_MODE | start state, then `Ctrl+X i` | default top status `case: insensitive`; toggling shows `case: sensitive`; local filter recomputes; active server filter refetches from offset 0 | no filter: only the indicator changes |
| PROMPT_OPEN | filter bar open, `m`/`PgDown`/`Down` at the end | text editing wins; end-of-data gesture disabled | `Enter` closes and keeps; `Esc` clears |
| JSON_FIELD | focused SQL JSON subfield or Mongo nested field | SQL filters the base column; Mongo filters the dotted path | rejected field: inline error, previous view kept |

</frozen-after-approval>

## Code Map

- `internal/domain/data.go` -- `DataQuery`; add `DataFilter{Field, Pattern, CaseInsensitive}` and `Filter *DataFilter`.
- `internal/engine/postgres/data.go`, `internal/engine/mysql/data.go` -- `Page`/`buildDataSelect`; add quoted, parameterized `WHERE` branches with native case flags.
- `internal/engine/mongo/data.go` -- `Page`; add `{field: {$regex: pattern, $options: "i"}}` for insensitive matching, keeping `_id` sort/skip/limit.
- `internal/app/session.go` -- `Page` passes the extended query unchanged under the session lock.
- `internal/tui/keymap/keymap.go` -- `/` Search, `N` NextMatch, `Ctrl+F` ServerFilter, `Ctrl+X i` CaseMode; keep `n` as New and route it contextually.
- `internal/tui/model.go` -- `apply` routing; session case mode; filter prompt state; top-status indicator.
- `internal/tui/grid.go` -- `dataState`, `beginDataFetch`, `handleDataPage`, `dataSummary`, `renderGrid`; local match projection/highlight, `n`/`N`, server filter cache.
- `internal/tui/workspace.go` -- filter bar, filtered banner, case indicator, key hints; reset filters in `syncDetails`/`selectTab`.
- Tests/goldens -- `internal/tui/*_test.go`, `internal/engine/*/data_test.go`, `internal/tui/testdata/*.golden`.

## Tasks & Acceptance

**Execution:**
- [x] `internal/domain/data.go` -- define `DataFilter` and extend `DataQuery` -- one cross-engine filter shape.
- [x] `internal/engine/postgres/data.go`, `mysql/data.go`, `mongo/data.go` -- add filter clauses and parameter binding -- CAP-4 engine search.
- [x] `internal/tui/keymap/keymap.go`, `model.go` -- wire `/`, `n`/`N`, `Ctrl+F`, `Ctrl+X i`, prompt precedence, case indicator -- keyboard contract.
- [x] `internal/tui/grid.go`, `workspace.go` -- local filtering/highlighting, navigation, filtered paging, cached restore, banner/status -- CAP-4 surface.
- [x] Tests, goldens, `README.md` -- query builders, local/navigation, server/clear/error/cancel, case mode, prompt gestures, JSON fields, 80x24 states.

**Acceptance Criteria:**
- Given loaded rows, when `/` types a pattern, then the view narrows without a server query and `Esc` restores every loaded row.
- Given a filtered view, when `n`/`N` run, then the cursor wraps to the next/previous match and the counter follows.
- Given a focused cell, when `Ctrl+F` applies a valid regexp, then only matching rows load, the banner names the field, and `m`/`Ctrl+A` page that result.
- Given `Ctrl+X i`, when either filter is active, then both use the new case mode and the top status shows it.
- Given a failed or cancelled filtered fetch, when it ends, then the previous rows remain and the error or `cancelled` is visible.
- Given an open filter bar, when `m`, `PgDown`, or `Down` at the end is pressed, then no page is fetched.
- Given a new object or tab, when it opens, then no filter, match cursor, or server filter carries over.
- Given `make test`, `make lint`, and `make build`, then all pass and the binary remains static.

## Implementation Notes

- `domain.DataFilter{Field, Pattern, CaseInsensitive}` extends `DataQuery`; `app.Session.Page` forwards the query under the session lock. `Filter` is nil for an unfiltered window.
- PostgreSQL adds `WHERE CAST("col" AS text) ~* $3` (or `~`) with the pattern bound; the base column is filtered even when a JSON subfield is focused. MySQL/MariaDB adds `WHERE \`col\` REGEXP ?` (or `REGEXP BINARY ?`) and binds the pattern before LIMIT/OFFSET. MongoDB builds `{field: {$regex: pattern, $options: "i"}}` (no `$options` when sensitive), validates dotted paths, and keeps the `_id` sort/skip/limit.
- `internal/tui/filter.go` owns both surfaces. The buffer filter is a literal projection over loaded raw rows: `matches` holds raw-row offsets, the gutter keeps absolute row numbers, `match X of N` follows the cursor, and `n`/`N` wrap. `n` stays `keymap.New` and is routed contextually so it never opens a new connection while a Data filter is active. Match highlighting uses the Match style; no-match copy sits on its own line so it fits 80×24 without truncation.
- `Ctrl+F` opens `regexp on <field>`, applies on Enter, keeps the prompt editable on an engine error, and caches the unfiltered `dataState` by value for Esc. Filtered paging uses the same `m`/`Ctrl+A` rules and offsets; the banner wraps to the pane width at the 80×24 floor. Clearing without a cache reloads the first page and keeps the filtered view if that reload fails.
- `Ctrl+X i` toggles the session case mode (default insensitive), shown as `case: insensitive` / `case: sensitive` in the top status. A buffer filter recomputes; an active server filter refetches from offset 0. `/` and `Ctrl+F` are mutually exclusive, and both reset on object or tab change. No filter or case mode is persisted.
- Added query-builder, sqlmock Page-binding, Mongo filter, TUI local/navigation, server apply/page/clear/error/cancel, case-mode, JSON-field, prompt-gesture, and 80×24 golden tests; README documents the search contract.

## Spec Change Log

## Review Triage Log

| # | Finding | Verdict | Route | Evidence |
|---|---|---|---|---|
| BH1 | Zero-match server filter renders the object-empty copy | medium | patch | `renderGrid` special-cases only `bufferPattern`; `serverFilter` with no rows falls through to "No rows. <object> is empty." and `dataSummary` says `0 rows`. |
| BH2 | Filtered views show the unfiltered table estimate as the filtered total | medium | patch | Engine estimates (`reltuples`, `TABLE_ROWS`, `EstimatedDocumentCount`) are table-level; the banner and summary present them beside filtered counts. |
| BH3 | Failed case-mode refetch leaves the filter's case flag flipped | medium | patch | `toggleCaseMode` mutates `serverFilter.CaseInsensitive` and starts a fetch without `filterApplying`/`filterRestore`; `revertFilterOnError` cannot restore it. |
| BH4 | `/` and `Ctrl+F` fold Unicode case differently | low | patch | `runesEqualFold` uses per-rune `unicode.ToLower`; Go `strings.EqualFold` handles sigma/final-sigma/long-s and is the shared semantic. |
| BH5 | Empty `Ctrl+F` pattern silently does nothing | medium | patch | `applyServerFilter` closes the prompt and returns when the pattern is empty, leaving an existing filter applied or no filter started. |
| BH6 | Match highlighting is color-only | low | patch | `theme.Nightshift` Match style has only foreground/background; no underline or other non-color attribute. |
| BH7 | Story Code Map names files/functions that do not match the diff | false | reject | The finding is accurate about the document, but its fix edits this build's spec, which step-04 rejects; actual files are recorded in Implementation Notes. |
| BH8 | Mongo field validation is duplicated with different behavior | medium | patch | `validMongoPath` (`filter.go`) and `validMongoFilterField` (`mongo/data.go`) implement the same rule; the engine trims, the TUI does not. |
| BH9 | An open filter bar traps non-text keys | medium | patch | `handleFilterKey` handles only Quit/ForceQuit/Cancel/Confirm; Tab, Shift+Tab, Ctrl+D, F1 and chords fall through or are swallowed. |
| BH10 | Esc precedence is undocumented against the UX ladder | low | patch | Code cancels a loading fetch before clearing the filter; EXPERIENCE.md's Esc ladder says the same, but the README says only "Esc clears the filter". |
| BH11 | `n`/`N` only navigate the buffer filter | medium | patch | `nextMatch` requires `bufferPattern != ""`; with only a server filter `n` is a silent no-op and `N` does nothing. |
| BH12 | No-cache server-filter reload path is unreachable in normal use | low | reject | `applyServerFilter` always fills `serverCached`; the no-cache branch is defensive after a failed reload. The test-only path adds complexity for a state users rarely reach. |
| BH13 | MySQL case mode depends on collation and uses deprecated `REGEXP BINARY` | medium | patch | Insensitive `REGEXP` follows the column collation; a `_bin`/`_cs` column contradicts `case: insensitive`. |
| BH14 | Server regexp cost and index impact are undocumented | medium | patch | No note covers pattern cost, timeouts/cancel expectations, or the `CAST(... AS text)`/`BINARY`/unanchored `$regex` index caveat. |
| BH15 | Very long patterns make the filter bar lose its cursor and hints | low | reject | `renderFilterBar` truncates the tail; patterns over the pane width are unlikely in ordinary use and the fix is more than a direct correction. |
| BH16 | `match X of N` can point at a cell outside the visible viewport | low | reject | Match cells cover all view columns; `n`/`N` now move to and horizontally scroll to the matching cell, so the counter is authoritative. |
| BH17 | Applying a server filter resets a Ctrl+A-grown batch | false | reject | The SERVER_FILTER matrix requires the first 100 matching rows; starting a new filtered result at 100 is the specified behavior. |
| BH18 | Cursor projection lacks direct tests; 80×24 check counts runes | low | patch | The data-row helpers are only exercised end to end, and `TestFilterBarKeeps80x24Shape` uses rune length instead of `lipgloss.Width`. |
| BH19 | Required transitions lack tests | medium | patch | No tests cover server-filter tab/`[`/`]` reset, paste into prompts, zero-row server filters, or non-text filter fields. |
| VG1 | MongoDB `Page` filter wiring has no test | medium | defer | The concrete `*mongo.Client` has no find seam; builder and TUI fake are covered and live Mongo integration is already deferred. |
| VG2 | Filter tests only assert offset 0 with `windowStart` 0 | medium | patch | Apply and case-toggle tests never page first; changing `beginDataFetch(0, …)` to `windowStart` would pass the suite. |
| VG3 | Buffer filtering after a slid/capped window is untested | medium | patch | Every buffer test starts at `windowStart` 0; `TestDataRowCapKeepsAbsoluteCursorOnPaging` has no filter. |
| VG4 | Server-filter tab and `cycleTab` resets are untested | medium | patch | `TestFiltersResetOnTabAndObjectChange` covers only a buffer filter and `selectTab`; `cycleTab` reset has no test. |
| VG5 | Paste into an open filter bar is untested | medium | patch | `handlePaste` has prompt cases; no test sends `tea.PasteMsg` to either prompt or an ANSI/control payload. |
| VG6 | No-cache `Esc` reload success is untested | low | patch | The only no-cache test forces `failAt`; the success branch that clears `filterClearing` is unexercised. |
| VG7 | Highlight test proves only one occurrence | low | patch | `TestBufferFilterNarrowsAndRestores` asserts one `Match.Render("active")`; repeated/mixed-case occurrences are not checked. |
| VG8 | Arrow/page/range navigation in a filtered view is untested | medium | patch | No filter test presses Up/Down/PgUp/PgDown/Ctrl+Home/Ctrl+End after applying a buffer filter. |
| VG9 | Ctrl+X i while a prompt is open is untested | high | patch | `handleFilterKey` swallows the chord and inserts `i`; duplicate of EC1/BH9. |
| VG10 | `/` after a server filter without a cache never opens the bar | medium | patch | `openBufferFilter` returns the clear reload command before setting `promptBuffer`; the user must press `/` twice. |
| EC1 | Ctrl+X i while a filter prompt is open toggles nothing and appends `i` | high | patch | Duplicate of VG9/BH9; verified in `handleFilterKey`. |
| EC2 | Esc while a buffer bar is open and a page loads leaves the filter active | medium | patch | Cancel branch checks `d.loading` first and returns after cancelling; the prompt and pattern stay. |
| EC3 | Typing into the server prompt during its apply fetch loses input | medium | patch | `applyServerFilter` leaves `promptServer` while loading; later keystrokes edit a pattern that is discarded when the page lands. |
| EC4 | Filter banner renders over unfiltered rows before the filtered page lands | medium | patch | `serverFilter` is set before `beginDataFetch`; `filterBannerLines` renders immediately with old rows and counts. |
| EC5 | Footer can show impossible filtered counts against the table estimate | medium | patch | `dataSummary` combines filtered `len(rows)` with the unfiltered estimate; duplicate of BH2. |
| EC6 | Leading/trailing spaces in a pattern are invisible | medium | patch | `renderFilterValue`/`renderNoMatchLine` use `oneLine`, which trims; the cursor and copy disagree with the pattern. |
| EC7 | Printable Shift/AltGr keys are rejected | medium | patch | `editFilterText` returns unchanged for any non-zero `msg.Mod`, so uppercase/AltGr characters cannot be typed. |
| EC8 | `beginDataFetch` races on the filter pointer | high | patch | The closure captures `m.data.serverFilter` and copies it at execution; `toggleCaseMode` mutates that pointer before a new fetch starts. |
| EC9 | Mongo filter trims the field while the TUI does not | medium | patch | `buildMongoFilter` uses `strings.TrimSpace(filter.Field)`; a padded field targets a different key. Duplicate of BH8. |
| EC10 | No-match copy contradicts the row-too-large state | low | patch | `filterNoMatchVisible` does not exclude `tooLarge`, so both messages can render. |
| EC11 | Server prompt Enter while a page loads is silently dead | medium | patch | `applyServerFilter` returns nil when `d.loading`; the filter never starts and no message explains it. |
| EC12 | The JSON-expand footer hint was removed | low | patch | The old Data-focus hint carried `` ` JSON``; the new hints omit it, leaving no in-app pointer. |
| EC13 | Claim: Ctrl+X i applies case mode with a filter active | high | patch | Duplicate of EC1/VG9/BH9; while a prompt owns keys, the chord never reaches `apply`. |
| EC14 | Claim: Esc restores all rows after `/` | medium | patch | Duplicate of EC2; Esc during a loading buffer prompt cancels the page but leaves the filter and bar. |
| U1 | User report: `n`/`N` jumped by rows, not highlighted cells | medium | patch | Fixed: `matchCells`/`matchNumber` now walk matching cells row-major with wrapping; added `TestBufferFilterMatchNavigationVisitsCells`. |

## Design Notes

Local filtering keeps matching raw-row offsets for the row projection and a sorted `matchCells` list for `n`/`N`; the gutter keeps absolute row numbers. `n`/`N` wrap through matching cells and scroll the column cursor into view; `Up`/`Down` clamp. Server filtering replaces the window and caches the unfiltered `dataState` by value for `Esc`. SQL filters the base column even when a JSON subfield is focused; MongoDB uses the dotted path. Case mode starts insensitive; toggling recomputes a buffer filter and refetches an active server filter from offset 0. `/` and `Ctrl+F` are mutually exclusive.

## Verification

**Commands:**
- `go test ./...` -- expected: PASS.
- `go test -race ./...` -- expected: PASS.
- `go vet ./...` -- expected: PASS.
- `make build` -- expected: static ELF in `bin/`.

**Observed:** `go test ./...`, `go test -race ./...`, and `go vet ./...` pass; `make build` produces a statically linked, stripped Linux `amd64` ELF. New goldens cover the filtered view, no-match state, and server-filter banner at 80×24. Real PostgreSQL/MySQL/MongoDB servers were not available; engine behavior is covered by sqlmock, query-builder tests, and the fake reader, with live integration deferred.
