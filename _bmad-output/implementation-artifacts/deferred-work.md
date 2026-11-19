- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/2-connection-setup-profiles-import-new-connection.md`
  summary: Real PostgreSQL/MySQL/MongoDB connectors are compiled and unit-tested for DSN construction, timeouts, and redaction, but no live database was available, so open/ping/close was not exercised against a real endpoint.
  evidence: The story's verification contract has no integration environment; `internal/engine/*/connect_test.go` covers URI/DSN construction and secret redaction, while the app/TUI tests use a fake connector. Story 3+ should run the opt-in DB_ADM_TEST_* integration suite.

- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/2-connection-setup-profiles-import-new-connection.md`
  summary: The new-connection form rejects all query options, so a typed URI cannot set PostgreSQL/MySQL TLS or MongoDB `authSource`/`tls`; imported MongoDB files do replay hosts and options via the source selector.
  evidence: `ParseURI` stores only host, port, database, and user, and rejecting options prevents a silent `sslmode` downgrade. Servers that require non-default TLS/auth options for a typed URI need a follow-up non-secret connect-options field.

- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/2-connection-setup-profiles-import-new-connection.md`
  summary: A wildcard `~/.pgpass` line imports as `localhost` with the engine default port and empty database/user; it cannot preserve the wildcard itself because a profile stores a concrete endpoint.
  evidence: `internal/app/connections.go` maps `*` host/port/database/user to concrete values before creating a profile. The source line still matches at resolve time through the stored wildcard file, but the label and picker row show the concrete fallback.

- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/2-connection-setup-profiles-import-new-connection.md`
  summary: Connection loss while idle in the workspace is not detected; the story only opens and pings at connect time.
  evidence: `app.Service.Connect` pings with a 10s context, but no background ping or cursor error path exists until the catalog/data stories wire the session. The picker row error and `Ctrl+D` paths are implemented.

- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/2-connection-setup-profiles-import-new-connection.md`
  summary: The workspace still shows the story-1 preview tree, grid, details, and privileges; only the top status, database, user, and SQL/MQL prompt now come from the active profile.
  evidence: The story explicitly defers catalogs and data views to story 3; the tree title and status note `catalog preview`.

- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/3-common-tree-security-object-details.md`
  summary: Object statistics, indexes, sampled MongoDB structure, and exact count are deferred to a follow-up CAP-3 story after the common tree and Security ship.
  evidence: The original story combined CAP-2 and CAP-3, making a spec far above the 900–1600 token target. Tree navigation and Security are independently useful and must land first; the follow-up story will add StatsReader, engine statistics queries, exact count/cancel, and the full Details tab.

- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/3-common-tree-security-object-details.md`
  summary: Live PostgreSQL, MySQL/MariaDB, and MongoDB catalog queries were not run against real engines; only query builders, mapping helpers, fakes, and goldens were exercised.
  evidence: No database was available in the environment. `make test` covers the TUI tree, app session switching, catalog interfaces, and pure MySQL/Mongo helpers; a follow-up integration run should exercise real databases, including MySQL role-derived markers and MongoDB custom-role privileges.

- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/4-data-browsing-and-paging.md`
  summary: Buffer filtering with `/`, `n`/`N` match navigation, server-side `Ctrl+F` regexp, the focus prompt, filter banner, and session-wide case mode are deferred to a follow-up search story.
  evidence: The original story combined browsing/paging and search, far above the 900–1600 token target. The search surfaces layer on the loaded grid and are independently reviewable once paging, windowing, and the Data tab are stable.

- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/4-data-browsing-and-paging.md`
  summary: Live data pages were not run against real PostgreSQL, MySQL/MariaDB, or MongoDB servers; query construction, type mapping, Mongo union, session switching, fakes, and 80x24 goldens were exercised instead.
  evidence: No database was available in the environment. The first opt-in `DB_ADM_TEST_*` integration run should verify real estimates, quoted identifiers, binary/JSON values, stable offset paging, and MongoDB documents whose fields appear only on later pages.

- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/4-data-browsing-and-paging.md`
  summary: A primary-key column name containing a comma cannot be represented in `Catalog.OrderKey`'s comma-joined string, so its stable ordering falls back to an invalid column list.
  evidence: `Catalog.OrderKey` returns a single string and the data adapters split it on commas. Normal identifiers, including quoted mixed-case names, are covered; a structured order-key list would be needed before this edge can be handled safely.

- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/4b-data-search-and-case-mode.md`
  summary: MongoDB `connection.Page` does not exercise the built regexp filter in a unit test because the concrete driver client has no find seam; only the filter builder and the TUI fake are covered.
  evidence: `internal/engine/mongo/data.go` calls `Collection.Find` directly and `internal/engine/mongo` has no driver mock. PostgreSQL and MySQL have sqlmock Page tests. Close this with the live MongoDB integration suite or a small find seam.

- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories.yaml` (story 5 follow-up)
  summary: Story 5b — Bracket-pair highlighting in the statement editor. Deferred for later; it is a presentation feature, not a query or guard dependency.
  evidence: Matching `()[]{}` around the cursor needs a string/comment-aware scanner plus a custom single-line renderer for the query bar and edit modal, because `bubbles/textinput` exposes neither per-rune styling nor its horizontal scroll offset. Planned scope includes a theme token for the matched pair, an unmatched-bracket state, tests for nesting/quoting/comments, and keeping the existing editing, history, completion popup, and Esc+Tab behavior intact.
