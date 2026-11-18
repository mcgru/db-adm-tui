---
title: 'Common tree and Security'
type: 'feature'
created: '2026-11-18'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
baseline_commit: 'c3810dabbfdbc5b26465af69833a0176a94197f0'
context:
  - "{project-root}/_bmad-output/planning-artifacts/architecture/architecture-db-adm-tui-2026-11-18/ARCHITECTURE-SPINE.md"
  - "{project-root}/_bmad-output/planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/DESIGN.md"
  - "{project-root}/_bmad-output/planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/EXPERIENCE.md"
  - "{project-root}/_bmad-output/specs/spec-db-adm-tui/SPEC.md"
  - "{project-root}/_bmad-output/implementation-artifacts/epic-1-context.md"
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Story 2 connects, but the workspace still shows the story-1 preview: no real databases, schemas, tables, collections, users, roles, or access markers. CAP-2 is unfulfilled.

**Approach:** Replace the preview tree with real lazy catalog navigation for PostgreSQL, MySQL/MariaDB, and MongoDB, including a Security node with users and roles and native `ro`/`rw`/`—` access markers. Add context tabs with basic object details and explicit placeholders for statistics, data, and privileges, which land in later stories.

**Decision:** The visible tree starts at databases plus Security; the connection is the top status line. PostgreSQL lists objects only for the database of its one live connection, so expanding a different database reconnects the session to it (same profile, source re-read, old connection closed first); a failed switch returns to the picker with a row error. MySQL/MariaDB and MongoDB see all databases on one connection. `rw` means any native write privilege, `ro` means read without write, `—` means no read; an unknown check omits the marker.

## Boundaries & Constraints

**Always:** Keep one serialized engine operation per session; lazy-load children on expand; attach load errors to the parent; preserve selection across focus and tab changes; run metadata with a 30s timeout; use native engine terms for users, roles, and scope; quote every catalog identifier; sanitize all names before rendering; render 80x24 with no color-only state; keep the tree read-only.

**Never:** No object statistics, indexes, sampled structures, exact count, data rows, query execution, filtering, grants/memberships, GRANT/REVOKE, sorting, editing, export, or write mode in this story. No fake catalog data in the real workspace. No second concurrent connection.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| HAPPY_TREE | Connected PostgreSQL/MySQL/MongoDB profile | Tree shows databases plus Security; expanding loads real children lazily; access markers appear on tables/collections/views | Parent stays expanded with an inline error on failure |
| LAZY_LOAD | Expand a database, schema, or Security group | Only that node's children load; already-loaded siblings are reused; selection stays on the expanded node | Timeout or permission error stays attached to the parent; Enter retries |
| SECURITY | Expand Security, then users or roles | Native principals load lazily; PostgreSQL roles show attributes, MySQL shows `user@host`, MongoDB shows database-scoped names | `not permitted` under the group instead of an empty success |
| ACCESS | Table, collection, or view row | `rw` for any write privilege, `ro` for read without write, `—` for no read, no marker when unknown; sequences carry none | A failed check omits the marker, never invents `—` |
| PG_SWITCH | Expand a PostgreSQL database other than the current one | Session reconnects to that database; top status updates; only one connection is live | Failed switch returns to the picker with the profile row error |
| PLACEHOLDERS | Select a table, collection, view, sequence, or Security node | Context tabs appear; Details shows basic identity plus `statistics are not available in this story`; Data and Privileges are explicit placeholders | N/A |
| EMPTY | Database or Security group has no visible children | Explicit empty row under the expanded node | N/A |
| TOO_SMALL | Terminal below 80x24 | Story 1 too-small line; no tree renders | No panic; recovery on resize |

</frozen-after-approval>

## Code Map

- `internal/domain/catalog.go` -- `ObjectRef`, `Node`, `ObjectKind`, `Access`, `Principal`, tree identity keys, marker words.
- `internal/ports/catalog.go` -- `Catalog` (`Databases`, `Children`, `Principals`, `OrderKey`), `SessionBundle` (`Session`, `Catalog`).
- `internal/ports/profile.go` -- change `Connector.Connect` to return `SessionBundle`; keep `Session` (`Ping`, `Close`).
- `internal/engine/postgres/catalog.go` -- databases, schemas, tables/views/sequences, roles/users, access checks, `OrderKey` from the primary key.
- `internal/engine/mysql/catalog.go` -- databases, tables/views, MariaDB sequences, users/roles, access checks, `OrderKey` from the primary key.
- `internal/engine/mongo/catalog.go` -- databases, collections/views, scoped users/roles, access from `connectionStatus`, `_id` order key.
- `internal/engine/{postgres,mysql,mongo}/connect.go` -- return a bundle whose catalog shares the live connection.
- `internal/app/session.go` -- `app.Session` wraps the bundle, serializes one operation, tracks the current database, reconnects PostgreSQL on switch, retains only in-session credentials.
- `internal/app/connections.go` -- return the bundle on connect; expose catalog operations to the TUI without ports.
- `internal/tui/tree.go` -- flattened visible rows, lazy expansion, selection, collapse, errors, loading state, markers.
- `internal/tui/details.go` -- context tab sets, basic identity rows, placeholders, scroll offset.
- `internal/tui/model.go` -- async tree load commands and generations; connect opens the real tree; tab switching by available tabs.
- `internal/tui/workspace.go` -- top status current database, real tree count/title, main pane tabs/details, status line.
- `internal/tui/keymap/keymap.go` -- existing Left/Right/Space/Enter/PgUp/PgDn/Home/End actions wired to tree behavior; no new letters.
- `cmd/db-adm-tui/main.go` -- unchanged wiring; the connector returns bundles.
- `internal/**/*_test.go`, `internal/tui/testdata/*.golden` -- fake catalog, query-builder tests, tree/Security/access/PG-switch states, 80x24 goldens.

## Tasks & Acceptance

**Execution:**
- [x] `internal/domain/catalog.go` -- define catalog, access, principal, and node types with stable identity -- one cross-engine vocabulary.
- [x] `internal/ports/catalog.go`, `internal/ports/profile.go` -- define and wire the catalog port and `SessionBundle` -- keep TUI and drivers apart.
- [x] `internal/engine/postgres/catalog.go` -- implement lazy catalog, access markers, Security, and `OrderKey` -- cover schemas, views, sequences, and roles.
- [x] `internal/engine/mysql/catalog.go` -- implement databases, tables/views, MariaDB sequences, users/roles, access, and `OrderKey` -- native terms.
- [x] `internal/engine/mongo/catalog.go` -- implement databases, collections/views, scoped users/roles, access, and `_id` order key -- native terms.
- [x] `internal/app/session.go` -- serialize operations, track current database, reconnect PostgreSQL on switch, retain only in-session credentials -- one live connection.
- [x] `internal/tui/tree.go`, `details.go`, `model.go`, `workspace.go` -- real lazy tree, Security groups, context tabs, basic details, placeholders -- CAP-2 surface.
- [x] `internal/tui/keymap/keymap.go`, `internal/tui/model.go` -- wire Left/Right/Space/Enter and scrolling keys -- keyboard contract.
- [x] Tests and goldens -- catalog mapping, access markers, tree expansion/error, PG switch, Security, placeholders -- lock behavior.
- [x] `README.md` -- document real tree, Security, access markers, placeholders, and PostgreSQL database switching.

**Acceptance Criteria:**
- Given a connected profile, when the workspace opens, then real databases and Security appear; expanding a database loads only that node's children and keeps selection.
- Given a table/collection/view node, then its access marker matches the native read/write check; sequences carry none; a failed check omits the marker.
- Given a PostgreSQL database other than the current one, when expanded, then the single session switches to it, the top status updates, and a failed switch returns to the picker with the row error.
- Given Security, when users or roles are expanded, then native principals load lazily with scope/attributes and permission failures render `not permitted`.
- Given a table, collection, or Security node, when selected, then context tabs and basic details render; view and sequence selection shows the v1 placeholder; Data, statistics, and Privileges are explicit placeholders.
- Given a catalog failure, then the parent shows an inline error and the rest of the workspace remains usable.
- Given `make test`, `make lint`, and `make build`, then all pass and the binary remains static.

## Implementation Notes

- Implemented the real catalog tree with lazy children for PostgreSQL, MySQL/MariaDB, and MongoDB; Security expands to users/roles groups and native principals; access markers are computed from native checks.
- PostgreSQL database expansion closes the current bundle, re-resolves the source, and connects the same profile to the requested database; a failed switch marks the session dead and returns the picker row error.
- The TUI tree caches loaded children, retries failed parents on Enter, preserves selection across focus/tab changes, and shows context tabs with explicit Data/statistics/Privileges placeholders.
- Details, indexes, sampled structures, and exact count are deferred (see `deferred-work.md`); story 4 will restore the Data default.
- Live database integration was not available. Catalog query construction and mapping are unit-tested through pure helpers/fakes; real SQL/Mongo catalog queries remain unexercised against live engines.
- MySQL role-derived access markers and MongoDB custom-role privileges are best-effort; unknown checks omit the marker. MariaDB sequence visibility depends on the engine's `information_schema.TABLES` type.
- Review hardening: switch resolution uses the target database, the active picker profile is no longer overwritten, render paths use atomic display snapshots, stale tree generations are dropped, Home/End and paged movement skip info rows correctly, tree/detail names are sanitized, MySQL role discovery fails closed, Mongo auth caching retries, and foreign tables are listed.
- Verification: `make test` (race), `make lint`, and `make build` pass; the binary is a static ELF.

## Spec Change Log

## Review Triage Log

| # | Finding | Verdict | Route | Evidence |
|---|---|---|---|---|
| B1 | Home/End land one row off and cannot reach the last row (BH) | medium | patch | `selectFirst`/`selectLast` scan in place; `TestTreeHomeEndAndPageMovement` checks first/last. |
| B2 | Paged tree movement skips rows when a page boundary lands on an info row (BH) | medium | patch | `treeState.move` clamps a target once, then walks one row at a time; the same test covers an info row. |
| B3 | PostgreSQL switch resolves the source against the old database (BH) | high | patch | `switchDatabaseLocked` builds `next` first and resolves `next`; `TestSessionSwitchesPostgresDatabase` uses distinct `orders`/`billing` passwords. |
| B4 | A failed switch after a successful switch loses the picker row error (BH) | high | patch | Removed the `m.active` overwrite on switch; `TestPostgresSwitchFailureReturnsToPicker` asserts the stored row key and rendered error. |
| B5 | Rendering locks the session operation mutex (BH) | medium | patch | `Session` now keeps an atomic display profile/alive flag; render paths no longer take the operation mutex. |
| B6 | Reconnect has no timeout and catalog work is uncancellable (BH) | medium | patch | Switch uses `ConnectTimeout`; the TUI owns a workspace context cancelled on disconnect/leave. |
| B7 | Stale catalog replies mutate the picker after leaving the workspace (BH) | high | patch | `treeGen` increments in disconnect/leave; `TestStaleTreeRootAfterDisconnect` delivers a late root and asserts no mutation. |
| B8 | Access markers can invent `—` for column grants and `rw` for DDL-only users (BH) | low | patch | MySQL write detection narrowed to INSERT/UPDATE/DELETE; column-level grants are documented as out of scope in README. |
| B9 | MySQL role-discovery failure silently lists roles as users (BH) | medium | patch | `mysqlPrincipals` returns the role error for both groups, so the group renders `not permitted`. |
| B10 | Connection Details surface is unreachable (BH) | low | rejected | The spec edit removes the unreachable Connection tab; the story's visible tree intentionally starts at databases plus Security. |
| B11 | View/sequence acceptance claims basic details while code shows the v1 placeholder (BH) | low | rejected | The AC was corrected to match the UX-mandated placeholder; no code change is needed. |
| B12 | Database/schema details omit child count and render an empty state row (BH) | low | patch | `detailLines` shows `state` only for the current database and includes loaded child counts. |
| B13 | README key table is stale for context tabs/tree keys (BH) | low | patch | README now describes positional tabs and tree/space/home/end/detail-scroll bindings. |
| B14 | Credential retention is undocumented (BH) | low | patch | README documents in-memory retention for the session and clearing on disconnect/exit. |
| B15 | Deferred work still lists details scrolling (BH) | low | patch | Removed from `deferred-work.md`; scrolling and the position footer are implemented and tested. |
| B16 | Dead `database` fields and stale `NewCatalog` comment (BH) | low | patch | Removed the unused fields and corrected the comment. |
| B17 | Duplicated/dead helpers (`details.visible`, `nodeLabel`, alias, offset) (BH) | low | patch | Removed unused helpers/alias; `HasAccessMarker` delegates to `AccessMarker`; tree offset is computed locally. |
| B18 | Server-provided names are rendered unsanitized (BH) | medium | patch | `treeRowText` and unlabeled detail values pass through `oneLine`; tree names and attributes are neutralized. |
| B19 | New UI/catalog states and adapter mappings are undertested (BH) | medium | patch | Added root-error/retry, parent-error, Security principal, unknown-marker, scroll, Home/End, page, stale-root, and PG/MySQL/Mongo mapping tests; live engine runs remain deferred. |
| B20 | Reconnect test no longer tests a reset (BH) | low | patch | `TestReconnectResetsTableTabAndSelection` switches to Privileges, moves the cursor, reconnects, and asserts defaults. |
| B21 | Object count/copy semantics drift from UX (BH) | low | rejected | The count is documented as loaded object rows; the empty row is explicit. A user-specific empty sentence is cosmetic and does not change the story outcome. |
| B22 | `Right` on a leaf silently moves focus to main (BH) | medium | patch | `expandSelected` no longer changes focus for leaves; `TestRightOnLeafKeepsTreeFocus` covers it. |
| E1 | Home/End one row off (EC) | medium | patch | Duplicate of B1; same test. |
| E2 | Disconnect while catalog query is in flight repopulates the tree (EC) | high | patch | Duplicate of B7; generation bump and stale-root test. |
| E3 | Switch connect can hang without a deadline (EC) | medium | patch | Duplicate of B6; switch wrapped in `ConnectTimeout`. |
| E4 | Switch source matches the old database (EC) | high | patch | Duplicate of B3; distinct-password app test. |
| E5 | MySQL role edge error misclassifies roles (EC) | medium | patch | Duplicate of B9; role error now fails both groups. |
| E6 | PostgreSQL `pg_` prefix match hides legitimate names (EC) | low | patch | Schema/role filters now escape `_` with `ESCAPE '\'`. |
| E7 | Cursor jumps when children arrive after navigation (EC) | medium | patch | `addChildren` saves and restores the selected key. |
| E8 | Newline/escape in names can break tree rendering (EC) | medium | patch | Duplicate of B18; sanitization at tree/detail boundaries. |
| E9 | Mongo auth cache is poisoned by a failed first call (EC) | medium | patch | `auth.loaded` is set only after a successful `connectionStatus` decode. |
| E10 | Foreign tables are omitted from PostgreSQL schemas (EC) | low | patch | `relkind 'f'` is listed and mapped to a table; `TestPostgresObjectKind`. |
| E11 | Render blocks on the session operation mutex (EC) | medium | patch | Duplicate of B5; atomic display fields. |
| E12 | Failed switch row error uses the switched database key (EC) | high | patch | Duplicate of B4; active profile remains the saved profile. |
| E13 | View/sequence details claim contradicts UX/code (EC) | low | rejected | Duplicate of B11; AC corrected, placeholder is intentional. |
| V1 | Root catalog failure has no executing test (VG) | medium | patch | `TestTreeRootErrorAndRetry` covers permission failure, rendered error, and successful retry. |
| V2 | Parent errors are asserted only via the status line (VG) | medium | patch | `TestTreeParentErrorIsAttached` checks `treeNode.Err` and the indented tree row. |
| V3 | Save/skip commands are discarded by tests (VG) | medium | patch | Save/skip tests now `settle` the returned command and assert loaded roots. |
| V4 | Users assertion is satisfied by the top status line (VG) | medium | patch | `TestSecurityLoadsPrincipals` finds the user in `m.tree.rows` and checks its details. |
| V5 | Prompt-password PG switch is untested (VG) | medium | patch | `TestSessionSwitchUsesRetainedPromptPassword` verifies the retained password and database switch. |
| V6 | Switch-failure test does not check the picker row error (VG) | medium | patch | The test asserts the stored profile key, `rowErrors`, and the rendered picker error. |
| V7 | Real engine catalog mapping/access markers never run in tests (VG) | medium | patch | Added PostgreSQL object-kind, MySQL privilege/mapping, and Mongo role/order-key tests; live SQL remains deferred. |
| V8 | Unknown native access is never rendered by a test (VG) | medium | patch | `TestUnknownAccessOmitsMarker` verifies no `ro`/`rw`/`—` for `AccessUnknown`. |
| V9 | Principal Details content is never asserted (VG) | medium | patch | Security test selects the user and checks `scope` and `LOGIN`. |
| O1 | Session/TUI active profile identity breaks switch row errors (VG other) | high | patch | Duplicate of B4; active profile is no longer overwritten by the switched profile. |
| O2 | Home/End never select first/last row (VG other) | medium | patch | Duplicate of B1. |
| O3 | Details scrolling has no test and deferred-work contradicts code (VG other) | medium | patch | `TestDetailsScrollAndPosition` exercises `scrollDetails` and the position footer; deferred entry corrected. |
| O4 | Connection details/AC/README/credential/dead-helper findings (VG other) | low | rejected | These are duplicates of B10-B18; the spec/docs were corrected or the findings were rejected with the same evidence. |

## Design Notes

**Tab map.** Database/schema: Details · Privileges. Table/collection: Data · Details · Privileges, opening Details in story 3. View/sequence: Details placeholder. Security user/role: Details · Privileges. Missing tabs are absent, not disabled; `[`/`]` wrap. Data, statistics, and Privileges render explicit story placeholders.

**Tree shape.** Top level is databases plus Security. PostgreSQL: database → schema → tables/views/sequences. MySQL/MariaDB: database → tables/views/sequences. MongoDB: database → collections/views. Security → users/roles groups → principals. Chevrons `▸`/`▾`, leaves `·`; access markers only on table/collection/view rows. The tree title count is the number of loaded table, collection, view, sequence, user, and role rows.

**PostgreSQL switch.** The app closes the current bundle, re-resolves the source when applicable, opens the same profile against the requested database, and replaces the bundle. The tree cache stays; every later catalog call verifies the session database first.

**Simple details.** Database/schema shows name, kind, current-database marker, and loaded child count. Security principal shows name, kind, native scope, and attributes. These rows never query data. Connection identity is the top status line; there is no connection tree node in story 3, so the UX Connection tab is not present yet.

## Verification

**Commands:**
- `go test -race ./...` -- expected: PASS.
- `go vet ./...` -- expected: PASS.
- `make build` -- expected: static ELF in `bin/`.
- `go test ./internal/app ./internal/engine/...` -- expected: PASS.
- `go test ./internal/tui -run TestGolden` -- expected: PASS; `-update` only for intentional layout changes.

**Observed:** `make test` (race), `make lint`, and `make build` pass; the workspace golden shows the expanded PostgreSQL tree, access markers, Security, and Details placeholder. Review tests cover Home/End, paging, root/parent errors, stale generations, switch success/failure, prompt-password switching, unknown markers, principal details, and details scrolling.
