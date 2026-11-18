- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/1-foundation-and-tui-shell.md`
  summary: Below the 80x24 floor the picker is replaced by the too-small line, while EXPERIENCE.md allows the picker to keep rendering when it still fits.
  evidence: The story acceptance (SMALL_TERMINAL) requires only the too-small line, so the shell follows the story; the UX text and the story need a single ruling before a later story changes the behavior.

- source_spec: `_bmad-output/specs/spec-db-adm-tui/stories/1-foundation-and-tui-shell.md`
  summary: The tree width and query height are fixed in story 1; Ctrl+Left/Right/Up/Down are inert stubs and sizes do not persist per profile.
  evidence: The story fixes the 80x24 regions and has no persistence layer; EXPERIENCE.md -> Resize assigns the adjustable 16..half-width tree and 1..3-line query bar to the workspace polish story, so layout.Compute has no resize parameters or clamp branches yet.

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
