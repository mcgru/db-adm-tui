---
title: 'Data browsing and paging'
type: 'feature'
created: '2026-11-19'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
baseline_commit: '74883d50e9740d06e8a6a2bc10d73f40f975ca6e'
context:
  - "{project-root}/_bmad-output/planning-artifacts/architecture/architecture-db-adm-tui-2026-11-18/ARCHITECTURE-SPINE.md"
  - "{project-root}/_bmad-output/planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/DESIGN.md"
  - "{project-root}/_bmad-output/planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/EXPERIENCE.md"
  - "{project-root}/_bmad-output/specs/spec-db-adm-tui/SPEC.md"
  - "{project-root}/_bmad-output/implementation-artifacts/epic-1-context.md"
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Story 3 navigates real catalogs, but a table or collection still has no data view. The browsing half of CAP-4 is missing: no initial page, paging, bounded full fetch, grid navigation, or load status.

**Approach:** Add a cross-engine `DataReader` and a unified `Grid`, then a virtualized Data tab with row/column cursors, 100-row paging, a 10,000-row/64 MB offset window, explicit end-of-data gestures, and bounded full fetch. Table/collection selection returns to the Data tab. Search, server-side regexp, and case mode are deferred to the next story.

**Decision:** The Data tab opens for every table/collection. `m` appends the next 100 rows; after `Ctrl+A` the batch size becomes 10,000. The loaded window holds at most 10,000 rows or 64 MB; when it exceeds the cap the oldest rows drop and `windowStart` advances. Scrolling above the window start re-fetches the previous batch. `Catalog.OrderKey` supplies the stable order (Mongo `_id`); no order key warns that rows may repeat or skip. The first Mongo page establishes `_id` first and first-seen field order; later pages extend the union.

## Boundaries & Constraints

**Always:** Read-only fetches; one serialized engine operation per session; quote identifiers; pass values as parameters; cap at 100 rows per normal page and 10,000 rows/64 MB per loaded window; keep the previous view on error or cancellation; sanitize all database text; render null as `null` and empty string as empty; preserve mode/case indicators when counts truncate; use an offset window with stable order.

**Never:** No sorting controls, editing, export, write statements, query bar execution, `/` buffer search, `n`/`N` match navigation, `Ctrl+F` server regexp, or case-mode work in this story. No unbounded fetch, no second connection, no hidden per-keystroke database call. `/` and `Ctrl+F` render explicit "search lands in the next story" placeholders.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| INITIAL_PAGE | Table/collection selected | Data loads up to 100 rows; header + row gutter + grid; status shows loaded count and estimate or `total unavailable` | Empty object shows explicit empty state; failure keeps previous view with inline error |
| NEXT_PAGE | `m` in Data tab | Next 100 rows append; loaded count updates; at end `m` is a no-op with `end of data` | Failed fetch keeps rows and shows error |
| END_GESTURE | Cursor on last row, two `PgDown` or three `Down` | First press shows `press PgDown again` / `press Down twice more`; second/third fetches next page; counter resets on any other key/focus change | Disabled while a modal is open |
| WINDOW_CAP | `Ctrl+A` / repeated `m` beyond 10,000 rows or 64 MB | Window holds at most 10,000 rows/64 MB; oldest rows drop; status shows window range and more available; scrolling above window start re-fetches previous batch | Failed batch keeps the window and shows error |
| GRID_NAV | Data tab focused | Up/Down row cursor, Left/Right column cursor, Home/End first/last column, PgUp/PgDn page; horizontal scroll keeps cursor visible | No sort/edit affordances |
| EMPTY | No rows | `0 rows` / explicit empty state; grid headers remain if columns are known | N/A |
| FETCH_CANCEL | `Esc` during a fetch | Previous rows remain; status `cancelled`; no partial page | N/A |
| VIEW_SEQUENCE | View/sequence selected | Data tab absent; Details remains the v1 placeholder | N/A |
| TOO_SMALL | Terminal below 80x24 | Story 1 too-small line; no grid | No panic; recovery on resize |

</frozen-after-approval>

## Code Map

- `internal/domain/grid.go` -- `Grid`, `Column`, `Row`, `Cell`, `CellKind`, null/empty distinction, compact JSON, approximate bytes.
- `internal/domain/data.go` -- `DataQuery` (`Ref`, `Offset`, `Limit`), `DataPage` (`Grid`, `Offset`, `HasMore`, `EstimatedTotal`, `EstimateSource`, `OrderKey`, `OrderKeyKnown`).
- `internal/ports/data.go` -- `DataReader.Page(ctx, DataQuery)`; add it to `SessionBundle`.
- `internal/engine/postgres/data.go` -- `SELECT` with schema-qualified quoted identifiers, `OrderKey`, estimate from `pg_class.reltuples`.
- `internal/engine/mysql/data.go` -- `SELECT` with backtick identifiers, primary-key `OrderKey`, estimate from `information_schema.TABLES`.
- `internal/engine/mongo/data.go` -- `find` with `_id` sort/skip/limit, ordered `bson.D` fields, union columns, `estimatedDocumentCount`.
- `internal/engine/{postgres,mysql,mongo}/connect.go` -- return a bundle with `DataReader`.
- `internal/app/session.go` -- serialize `Page`, switch PostgreSQL database first, expose data operations without ports.
- `internal/tui/grid.go` -- grid viewport, row/column cursors, horizontal scroll, null/empty/JSON rendering, row-number gutter.
- `internal/tui/model.go`, `workspace.go`, `details.go` -- Data tab default, fetch commands/generations/cancel, paging gestures, window slide/re-fetch, status, placeholders.
- `internal/tui/keymap/keymap.go` -- wire `More`, `FetchAll`, `Home`/`End`, `PgUp`/`PgDn`, arrows; `/`/`Ctrl+F` show placeholders.
- `README.md` -- document Data tab, paging, window, and read-only limits.
- Tests/goldens -- grid mapping, paging/window, Mongo union, status, empty/error/cancel, 80x24 goldens.

## Tasks & Acceptance

**Execution:**
- [x] `internal/domain/grid.go`, `internal/domain/data.go` -- define grid/data types and byte estimation -- one cross-engine shape.
- [x] `internal/ports/data.go`, `internal/ports/catalog.go` -- add `DataReader` to the bundle -- keep TUI and drivers apart.
- [x] `internal/engine/postgres/data.go`, `mysql/data.go`, `mongo/data.go` -- initial pages, offset paging, estimates, ordering, and Mongo field union -- CAP-4 engine core.
- [x] `internal/app/session.go` -- serialize and expose `Page`, including PostgreSQL database switching -- one live connection.
- [x] `internal/tui/grid.go`, `model.go`, `workspace.go`, `details.go` -- Data tab, grid navigation, paging gestures, window slide/re-fetch, status, placeholders -- CAP-4 surface.
- [x] `internal/tui/keymap/keymap.go`, `internal/tui/model.go` -- wire existing paging/navigation actions -- keyboard contract.
- [x] Tests and goldens -- query builders, grid mapping, window/fetch, Mongo union, status, empty/error/cancel -- lock behavior.
- [x] `README.md` -- document data browsing, paging, window, and limits.

**Acceptance Criteria:**
- Given a table/collection, when selected, then Data opens with up to 100 rows, a row-number gutter, column headers, null/empty distinction, and status with loaded count and estimate or `total unavailable`.
- Given the Data tab, when `m`, the end gesture, or `Ctrl+A` runs, then pages append and the loaded window never exceeds 10,000 rows/64 MB.
- Given a loaded window, when it slides or the user scrolls above its start, then evicted rows are re-fetched using `Catalog.OrderKey` and stable order.
- Given a grid, when arrows/Home/End/PgUp/PgDn are pressed, then the row/column cursor moves and the viewport scrolls without sorting or editing.
- Given an empty object, fetch error, cancellation, view/sequence, or below-floor terminal, then the explicit empty/error/placeholder/too-small state remains.
- Given `/` or `Ctrl+F`, then the Data tab shows `search lands in the next story` without querying.
- Given `make test`, `make lint`, and `make build`, then all pass and the binary remains static.

## Implementation Notes

- Added `domain.Grid`/`Cell`/`Row`/`Column` plus `DataQuery`/`DataPage`; null and empty are distinct, JSON compacts, and grids expose an approximate byte count used by the TUI window cap.
- Added `ports.DataReader` to `SessionBundle`; all three connectors return it. `app.Session.Page` serializes the live connection, checks the capability, and switches the PostgreSQL database before querying.
- PostgreSQL and MySQL build quoted, parameterized `SELECT *` windows with primary-key ordering and catalog/`information_schema` estimates. MongoDB uses an `_id`-sorted find with skip/limit, ordered `bson.D` decoding, `_id`-first first-seen column union, and `estimatedDocumentCount`.
- The Data tab is the table/collection default. It keeps `windowStart`, rows, `hasMore`, byte/row estimates, batch size, absolute cursor, and viewport offsets; paging, end gestures, `Ctrl+A`, eviction, previous-batch refetch, cancellation, and generation drops live in `internal/tui/grid.go`. Forward appends evict the oldest rows at the front; a previous-page fetch evicts the far end so the rows just fetched survive and the cursor can move above the old window start.
- Search keys render `search lands in the next story`; no filter, regexp, sort, edit, export, or case-mode work is present.
- Tests cover domain cell mapping, SQL query builders and type mapping, Mongo union/columns, app serialization and PostgreSQL switching, TUI paging/window/cancel/error/empty states, search placeholders, absolute gutters, and 80×24 goldens.
- Review hardening: data loads on Enter/Data focus rather than every tree-cursor move; `DataPage` carries the offset and actual order key; SQL pages close the row set before estimating; `Ctrl+A` fills the existing window before sliding; previous-page fetches honor the returned offset; scroll-back restores `hasMore`; cancel drops late success; oversized rows have a distinct state; the 64 MB/row cap and SQL `Page` arguments are tested; case-mode text was removed until the search story.
- Live PostgreSQL, MySQL/MariaDB, and MongoDB data reads were not run against real servers; only fakes, sqlmock, and pure helpers were exercised. Mongo `Page` and the integration suite remain opt-in.

## Spec Change Log

- 2026-11-19: Split the original CAP-4 story into story 4 (browsing/paging) and story 4b (search/case mode). Trigger: the combined spec was roughly twice the token target and the search surfaces depend on a stable loaded grid. Amended: story title, frozen scope, Code Map, tasks, acceptance, Design Notes, and `stories.yaml`; search is recorded in `deferred-work.md`. Known-bad state avoided: a single oversized implementation that mixed paging, windowing, filters, server regexp, and case mode. KEEP: the DataReader/Grid seam, 10,000-row/64 MB window, absolute row-number gutter, Mongo union columns, and Data-tab default.

## Review Triage Log

| # | Finding | Verdict | Route | Evidence |
|---|---|---|---|---|
| B1 | Tree navigation starts data fetches on cursor moves (BH) | medium | patch | `syncDetails` only resets state; `Enter`/Data focus calls `ensureData`; `TestDataTreeCursorDoesNotFetchUntilOpened`. |
| B2 | Initial-load errors lose status and retry affordance (BH) | medium | patch | Failed first page renders `press m to retry`; `TestDataRetryHintOnInitialError`. |
| B3 | Slow loads have no elapsed indication (BH) | low | rejected | Story 4a does not require elapsed text; the loading state is explicit and search/data polish can add a timer. |
| B4 | `Ctrl+A` immediately evicts the first page (BH) | medium | patch | It now fills the window to 10,000 (`limit = remaining`); `TestDataWindowCapSlidesAndRefetches`. |
| B5 | Previous-page fetches request the wrong span and cannot detect short pages (BH) | high | patch | `DataPage.Offset` is returned by engines; previous pages use it; `TestDataPreviousShortPageUsesOffset`. |
| B6 | README misstates `m`/`Ctrl+A` scope and batch (BH) | low | patch | README names Data focus, 100/10,000 batches, and the initial fill. |
| B7 | README key documentation regressed (BH) | low | patch | Home/End import, Enter Data/details, and `.` search alias are documented. |
| B8 | Stable-order errors are reported as no primary key; offset caveats undocumented (BH) | medium | patch | `DataPage.OrderKeyKnown` distinguishes failures; TUI warns when unknown/empty; README documents offset caveats and comma-PK deferral. |
| B9 | 64 MB cap is untested and `DataPage.Bytes` duplicates the TUI byte source (BH) | low | patch | Removed `DataPage.Bytes`; byte-cap tests use the TUI cap; realistic 64 MB data is deferred. |
| B10 | Goldens do not cover new states; `workspace-data.golden` duplicates `workspace.golden` (BH) | low | rejected | Duplicate removed; empty/error/loaded goldens cover the mandatory states, while window/hscroll behavior is covered by focused tests. |
| B11 | Value rendering is underdocumented (BH) | low | patch | README documents binary/date/JSON/truncation; engine cell tests cover mapping. |
| B12 | A row larger than the byte cap renders as an empty object (BH) | medium | patch | `tooLarge` state renders `Row exceeds the 64 MB window`; `TestDataOversizedWindowState`. |
| B13 | Planning artifacts still promise search in story 4 (BH) | low | patch | `stories.yaml` now has 4 browsing and 4b search; the spec and deferred work record the split. |
| B14 | Status advertises case mode that cannot be toggled (BH) | low | patch | Removed `case: sensitive` from the Data status until the search story. |
| B15 | SQL adapter logic is duplicated and validation diverges (BH) | low | rejected | PostgreSQL and MySQL differ in quoting, placeholders, and type mapping; a shared scanner would add indirection for little current benefit. |
| E1 | Completed fetches leak their cancel context (EC) | medium | patch | `handleDataPage` cancels and clears the stored cancel on completion. |
| E2 | Byte cap can evict every row and claim an empty object (EC) | high | patch | Duplicate of B12; oversized state and test. |
| E3 | Short previous pages corrupt offsets and gutters (EC) | high | patch | Duplicate of B5; offset-based previous-page test. |
| E4 | TUI warning OrderKey can disagree with the query's OrderKey (EC) | high | patch | Duplicate of B8; the page carries the key actually used and whether it was known. |
| E5 | Eviction direction contradicts the stated cap rule (EC) | low | patch | README documents forward drop-front and backward drop-tail behavior. |
| V1 | Real engine `Page` methods are never executed (VG) | medium | patch | sqlmock tests execute PostgreSQL/MySQL `Page`, bound `Limit+1`/offset, `HasMore`, and estimate; Mongo `Page` remains live-integration deferred. |
| V2 | Page requests after a window slide are never asserted (VG) | medium | patch | `TestDataWindowCapSlidesAndRefetches` asserts `Ctrl+A → m → up` offsets and batch. |
| V3 | Previous-page `hasMore` restoration is untested (VG) | medium | patch | The updated window test reaches `hasMore=false`, scrolls up, and asserts restored `hasMore`. |
| V4 | Esc can accept a late successful page (VG) | medium | patch | `TestDataCancelDropsLateSuccess` cancels through `cancelDataFetch` and applies a stale success. |
| V5 | Tree PgUp/PgDn do not verify data-fetch command propagation (VG) | low | rejected | The fix intentionally stops tree cursor movement from querying; Enter/Data focus is the load action, and `TestDataTreeCursorDoesNotFetchUntilOpened` covers it. |
| O1 | The estimate query runs before the SQL result set is closed (VG other) | high | patch | SQL `Page` closes rows before estimating; sqlmock Page tests assert the estimate and query order. |

## Design Notes

**Window model.** Keep `windowStart` (absolute first loaded row), `rows`, `nextOffset`, `hasMore`, `bytes`, and `batchSize` (100, or 10,000 after `Ctrl+A`). A forward fetch that exceeds 10,000 rows or 64 MB drops the oldest rows at the front and advances `windowStart`; a previous-page fetch prepends rows and drops the far end so the page just fetched survives. The cursor stays on the same absolute row where possible. Scrolling above `windowStart` fetches the previous batch at `windowStart - batchSize`.

**Mongo union.** Each document is decoded as ordered `bson.D`; `_id` is first, then fields in first-seen order. Nested documents/arrays render as compact JSON. New fields extend the union; older rows show empty cells. SQL columns keep driver order.

**Ordering.** `Catalog.OrderKey` is required for stable windows. An empty key warns `no stable order; rows may repeat or skip` and still allows browsing. The row-number gutter uses absolute offsets, not visible indices.

## Verification

**Commands:**
- `go test -race ./...` -- expected: PASS.
- `go vet ./...` -- expected: PASS.
- `make build` -- expected: static ELF in `bin/`.
- `go test ./internal/engine/... ./internal/app ./internal/tui` -- expected: PASS.
- `go test ./internal/tui -run TestGolden` -- expected: PASS; `-update` only for intentional layout changes.

**Observed:** `make test` (race), `make lint`, and `make build` pass; static ELF. Data goldens cover the loaded grid, empty object, and initial error state.
