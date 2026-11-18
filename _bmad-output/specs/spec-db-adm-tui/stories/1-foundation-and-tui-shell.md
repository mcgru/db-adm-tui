---
title: 'Foundation and TUI shell'
type: 'feature'
created: '2026-11-18'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
baseline_commit: '3958fa7a16c96d5f912bb281607c70543f13323d'
context:
  - "{project-root}/_bmad-output/planning-artifacts/architecture/architecture-db-adm-tui-2026-11-18/ARCHITECTURE-SPINE.md"
  - "{project-root}/_bmad-output/planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/DESIGN.md"
  - "{project-root}/_bmad-output/planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/EXPERIENCE.md"
  - "{project-root}/_bmad-output/specs/spec-db-adm-tui/SPEC.md"
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** The repository has planning, UX, and architecture artifacts but no Go code. There is no runnable shell proving the chosen stack, the 80x24 layout, the Nightshift theme, or the keyboard model before database adapters arrive.

**Approach:** Create the Go module and a Bubble Tea v2 application shell with a fake profile picker and a fake workspace. Render Nightshift tokens and the 80x24 layout from `DESIGN.md`, implement one central Latin/Cyrillic keymap and the terminal-too-small state, and lock the shell with golden tests. No database drivers, real profiles, imports, or query execution in this story.

## Boundaries & Constraints

**Always:** Go module `github.com/mcgru/db-adm-tui`; `charm.land/bubbletea/v2` v2.1.2 and `charm.land/lipgloss/v2` v2.1.1; `CGO_ENABLED=0`; 80x24 minimum; Nightshift tokens from `DESIGN.md`; one keymap for all letter shortcuts and chord finals; fake data only; `tea.View.AltScreen` enabled.

**Never:** No database drivers, `~/.pgpass`/`~/.mongo.*` parsing, real profiles or connections, query execution, network calls, help corpus, second theme, or literal hex outside `internal/tui/theme`.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| HAPPY_PATH | 80x24 terminal, app starts | Picker shows fake profiles and footer; `Enter` opens workspace with top status, tree, tabs, fake grid, query bar, bottom status | N/A |
| SMALL_TERMINAL | 72x20 or height 23 | One line: `Terminal too small: need 80×24. Current WxH.` No workspace | No panic; resizing back to 80x24 restores the shell |
| CYRILLIC_KEY | Key text `т`, `ш`, `й`, `ь` | Same actions as `n`, `i`, `q`, `m` | N/A |
| CHORD_ALIAS | `Ctrl+X` then `ы`, `ц`, `у`, or `ш` | Resolves exact-count, read/write, edit, or case actions; shell reports `not available in this story` | Unknown final key reports `cancelled` |

</frozen-after-approval>

## Code Map

- `go.mod`, `go.sum` — module and pinned dependencies.
- `cmd/db-adm-tui/main.go` — program entry and exit handling.
- `internal/tui/theme/nightshift.go` — Nightshift colors and Lip Gloss styles; the only hex source.
- `internal/tui/layout/layout.go` — 80x24 regions, fixed tree width/query height, too-small check.
- `internal/tui/keymap/keymap.go` — actions, Latin/Cyrillic aliases, `Ctrl+X` chord resolver.
- `internal/tui/model.go` — root model, screen/focus state, `Update`, `View`, alternate screen and background.
- `internal/tui/picker.go` — fake profile picker and footer.
- `internal/tui/workspace.go` — fake status lines, tree, tabs, grid, query bar.
- `internal/tui/render.go` — ANSI-safe padding, truncation, sanitizing, single-line boxes.
- `internal/tui/testdata/*.golden` — 80x24 picker, workspace, focus, wide, and too-small views.
- `Makefile` — `build`, `test`, `lint`.
- `README.md` — build, run, and test commands.

## Tasks & Acceptance

**Execution:**
- [x] `go.mod` -- initialize `github.com/mcgru/db-adm-tui`, add `charm.land/bubbletea/v2` v2.1.2 and `charm.land/lipgloss/v2` v2.1.1; `go 1.26.0` minimum -- Bubble Tea v2 requires 1.26+.
- [x] `internal/tui/theme/nightshift.go` -- map `DESIGN.md` colors and the styles used by top status, tree, tabs, grid, query bar, bottom status -- one token source.
- [x] `internal/tui/layout/layout.go` -- compute top/body/query/bottom, tree 28 cells, 1-cell gap, main remainder, too-small below 80x24 -- enforce the floor.
- [x] `internal/tui/keymap/keymap.go` -- define actions and a single alias table for Latin/Cyrillic physical positions; resolve `Ctrl+X` chords -- prevent layout drift.
- [x] `internal/tui/render.go` -- pad/truncate to width, draw single-line frames, strip ANSI for tests -- deterministic views.
- [x] `internal/tui/picker.go` -- render the fake profiles `prod-ro`, `analytics`, `billing` and footer; `Enter` selects, `n`/`i` show stubs, `q` quits -- prove the entry surface.
- [x] `internal/tui/workspace.go` -- render top `prod-ro · PostgreSQL · orders · r/o`, fake tree with `ro`/`rw`/`—`, tabs `1 Data · 2 Details · 3 Privileges`, fake grid, query bar, bottom status -- prove the 80x24 layout.
- [x] `internal/tui/model.go` -- root model; WindowSizeMsg; focus tree→main→query bar; tabs via `1/2/3` and `[`/`]`; `Ctrl+D` returns to picker; status messages -- wire the shell.
- [x] `cmd/db-adm-tui/main.go` -- run `tea.NewProgram`, set `AltScreen`, handle error without a stack trace -- runnable binary.
- [x] `internal/tui/layout/layout_test.go`, `internal/tui/keymap/keymap_test.go`, `internal/tui/model_test.go`, `internal/tui/render_test.go`, `internal/tui/theme/nightshift_test.go` -- table tests for regions and aliases, golden views with ANSI stripped, ANSI-safe primitives, raw focus/selection styles, pinned palette -- lock behavior.
- [x] `Makefile`, `README.md` -- build/test/lint and run instructions -- developer entry.
- [x] `internal/tui/testdata/*.golden` -- generate from the approved views -- stable references.

**Acceptance Criteria:**
- Given an 80x24 terminal, when the app starts, then the picker shows `db-adm-tui`, three fake profiles, and the footer; pressing `Enter` opens the workspace.
- Given a 72x20 terminal or height 23, when the view renders, then only the too-small line appears; resizing to 80x24 restores the shell without a panic.
- Given `n`, `i`, `q`, or `m` in either Latin or Cyrillic, when pressed, then each pair resolves to the same action; `Ctrl+X` followed by `ы`/`ц`/`у`/`ш` resolves the corresponding chord action.
- Given the workspace, when `1`/`2`/`3` or `[`/`]` are pressed, then the active tab changes; `Tab` cycles focus; `Ctrl+D` returns to the picker; `q` quits only at the picker.
- Given `make build`, then a `CGO_ENABLED=0` binary is produced in `bin/`; `make test` and `go vet ./...` pass; golden views match after ANSI stripping.

## Implementation Notes

- Implemented directly because subagent balance was unavailable; the spec was the sole source.
- Toolchain resolved to Go 1.26.0+ (`go 1.26.0` module minimum, no `toolchain` pin); Bubble Tea v2 requires 1.26+.
- Corrected module paths to `charm.land/bubbletea/v2` and `charm.land/lipgloss/v2`; the architecture spine stack was updated.
- Fake picker and workspace only; no database drivers, profiles, imports, or query execution.
- `Pad` originally truncated strings at exact width and split ANSI sequences; fixed to preserve exact-width styled lines. Golden tests caught it.
- Review patches: `Ctrl+X Ctrl+X` now executes; chords expire after two seconds; truncation is ANSI-safe; picker rows move and select profiles; reconnect resets tab/tree/focus; too-small output is one padded line; keymap covers the v1 table plus Cyrillic `Ctrl+с`; route tests and raw-style tests were added.
- Verification: `make test` (race), `make lint` (`go vet`), `make build` all pass; binary is a static ELF x86-64.

## Spec Change Log

- 2026-11-18: Recorded the module-path correction to `charm.land/bubbletea/v2` v2.1.2 and `charm.land/lipgloss/v2` v2.1.1 and the toolchain claim as Go 1.26.0+. Trigger: the architecture spine still listed stale import paths and Go 1.26.11 while `go.mod` and the installed toolchain are 1.26.0. Amended: Implementation Notes, Code Map, Verification command, and the spine stack row. Known-bad state avoided: source importing a non-existent module path and an unreproducible toolchain pin. KEEP: the approved 80x24 layout, Nightshift tokens, one keymap, fake-only data, and `CGO_ENABLED=0` build.

## Review Triage Log

| # | Finding | Verdict | Route | Evidence / disposition |
|---|---|---|---|---|
| 1 | `Ctrl+X Ctrl+X` resolves to exact count, no execute action (BH) | medium | patch | Added `Execute`; chord finals `ctrl+x`/`ctrl+ч` resolve it; `TestChordExecute` verifies the message. |
| 2 | Chords never expire (BH) | medium | patch | `tea.Tick` arms a two-second `chordTimeoutMsg`; stale sequence numbers are ignored; `TestChordTimeout` covers expiry and staleness. |
| 3 | Truncate splits ANSI escapes and no production sanitize step (BH) | medium | patch | `Truncate` now uses `ansi.Truncate`; `Sanitize` strips escapes and controls; `TestTruncateIsANSISafe` and `TestStripAndSanitize` pin it. |
| 4 | Picker selection is inert (BH) | high | patch | `Up`/`Down` move `pickerSel`, `Enter` stores `activeProfile`; `TestPickerNavigationAndProfileSwitch` covers bounds and profile switch. |
| 5 | Workspace ignores the model's selection (BH) | low | rejected | Active profile now drives label, engine, user, and SQL/MQL prompt. Tree- and tab-driven panes require catalog state that story 1 intentionally does not have; changing the fixture is not a direct correction. |
| 6 | Fake data contradicts itself (BH) | medium | patch | Customers Details now names `customers_pkey` and the matching columns; billing no longer reuses `~/.pgpass`; tree count is `objects 3`; goldens updated. |
| 7 | Reconnect keeps stale tab and selection (BH) | medium | patch | Connect and disconnect reset `tab`, `treeSel`, and `focus`, and align `pickerSel` with `activeProfile`; `TestReconnectResetsWorkspaceState`. |
| 8 | Status messages never clear (BH) | medium | patch | `handleKey` clears transient messages before resolving the next key; `TestMessageClearsOnNextKey`; errors still survive as the current action's message. |
| 9 | Focus is not rendered per design (BH) | medium | patch | Focused box border and title use `Focused`; active tab is accent bold underlined; query prompt uses the focus style; tree movement is gated by focus; raw-style test. |
| 10 | Nightshift surface/rule/padding/label styles missing (BH) | low | patch | Box gained one-cell padding, `Header` is bold, and `Surface`/`Raised`/`BorderModal` styles exist. Full surface banding and status rules would change the fixed 80x24 arithmetic and belong to the visual pass; the direct style gaps are closed. |
| 11 | Too-small contract inconsistent and message can wrap (BH) | medium | patch | `viewTooSmall` truncates and pads to one line; story acceptance is followed; the EXPERIENCE.md conflict is recorded in `deferred-work.md`. |
| 12 | Keymap is a subset of the v1 table (BH) | medium | patch | Added `Space`, `Home`/`End`, `PgUp`/`PgDn`, `N`, `Ctrl+A`, `Ctrl+F`, `Ctrl+Space`, `Ctrl+F1`, `Ctrl+arrows`, `Ctrl+с`, and execute; stubs report `not available in this story`; picker footer now advertises `? help`. |
| 13 | Goldens and negative tests omit reachable states (BH) | medium | patch | Added Privileges, main-focus, query-focus, 100x30, and narrow too-small goldens; pending-chord and negative paths are asserted in model tests. |
| 14 | Render primitives have no unit tests (BH) | medium | patch | `render_test.go` covers truncate, pad, compose, fit, join, box dimensions, focus, clipping, and tiny degradation. |
| 15 | Story artifact names missing files/tests and has an empty change log (BH) | low | rejected | The Code Map, task paths, and Verification command were corrected as housekeeping and the change log records the module-path correction. Per the workflow rule, a finding whose fix is editing this build's spec is not a patch item. |
| 16 | Go toolchain claim is not reproducible (BH) | medium | patch | Spine and Implementation Notes now state Go 1.26.0+; `go.mod` has `go 1.26.0` with no toolchain pin, matching the installed 1.26.0. |
| 17 | README omits prerequisites and operational details (BH) | low | patch | README adds Go 1.26+, the 80x24 floor, the key table, the golden-update command, `-race`/cgo, and the fake-shell limitation. |
| 18 | Layout has dead clamps and no resize seam (BH) | low | patch | `Compute` is reduced to the fixed floor arithmetic; resizable tree/query sizes are recorded in `deferred-work.md`. |
| 19 | Picker Up/Down inert (EC) | medium | patch | Same correction as row 4; picker navigation is now bounded and tested. |
| 20 | Tree cursor moves while main or query is focused (EC) | medium | patch | `Up`/`Down` require `FocusTree` in the workspace; `TestTreeNavigationBounds` presses arrows from main and asserts no move. |
| 21 | Keys act invisibly while the terminal is too small (EC) | medium | patch | Non-quit keys are dropped below 80x24; `TestTooSmallKeyGuard`; `q` and `Ctrl+C` remain usable. |
| 22 | Plain `ц` is missing as a chord final (EC) | medium | patch | `"ц": ToggleMode` added; `TestChords`. |
| 23 | Plain `у` is missing as a chord final (EC) | medium | patch | `"у": Edit` added; `TestChords` and `TestChordExecute`. |
| 24 | `q` in the workspace never reaches the no-quit branch in tests (VG) | medium | patch | `TestWorkspaceQuitDoesNotExit` asserts no command, unchanged screen, and the exact message. |
| 25 | Focus cycling verifies only the first Tab hop (VG) | medium | patch | `TestFocusCycle` covers three Tab and three Shift+Tab transitions plus the query-focused bottom line. |
| 26 | Tab selection omits `1`, `3`, `[`, wrapping, and Privileges (VG) | medium | patch | `TestTabSelection` covers every transition; `workspace-privileges.golden` locks the pane. |
| 27 | Goldens strip selection/focus/active-tab styling (VG) | medium | patch | `TestRawSelectionAndFocusStyles` checks the raw `Select`, `Focused`, and active-tab sequences; `theme/nightshift_test.go` pins the palette. |
| 28 | Alias-pair tests assert only equality (VG) | low | patch | `TestLatinCyrillicAliases` pins the resolved action for every pair; `TestSingleActions` pins the full table; `TestMoreAndHelpMessages` covers `m`/`?`. |
| 29 | Tree Up/Down navigation is never exercised (VG) | medium | patch | `TestTreeNavigationBounds` asserts movement, both clamps, and the focus guard. |
| 30 | Focused title, active tab, query focus, selection reverse are below DESIGN.md (VG other) | medium | patch | Covered by row 9: focused title is bold accent, active tab is bold underlined, query prompt changes on focus, and `Select` adds bold+reverse for no-color terminals. |
| 31 | MySQL `billing` points at `~/.pgpass` (VG other) | low | patch | Source changed to `manual`; picker golden updated. |

## Design Notes

Fake data is illustrative and never persisted. The root `View` always sets `tea.View.AltScreen = true`. Golden comparisons strip ANSI so the tests assert layout, not terminal color support.

## Verification

**Commands:**
- `go test -race ./...` -- expected: PASS.
- `go test ./... -run 'TestGolden|TestLatinCyrillicAliases|TestSingleActions|TestChords|TestComputeAtFloor'` -- expected: PASS.
- `go vet ./...` -- expected: PASS.
- `make build` -- expected: `bin/db-adm-tui` exists; `file bin/db-adm-tui` reports an ELF executable.
