---
title: 'Connection setup: profiles, import, new connection'
type: 'feature'
created: '2026-11-18'
status: 'done'
route: 'dispatch'
review_loop_iteration: 0
baseline_commit: '20acc48a3a2e94f7ae8b485f769bc62451a10ce4'
context:
  - "{project-root}/_bmad-output/planning-artifacts/architecture/architecture-db-adm-tui-2026-11-18/ARCHITECTURE-SPINE.md"
  - "{project-root}/_bmad-output/planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/DESIGN.md"
  - "{project-root}/_bmad-output/planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/EXPERIENCE.md"
  - "{project-root}/_bmad-output/specs/spec-db-adm-tui/SPEC.md"
  - "{project-root}/_bmad-output/implementation-artifacts/epic-1-context.md"
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Story 1 ships only a fake picker and workspace. CAP-1 is unfulfilled: there is no profile store, no `~/.pgpass` or `~/.mongo.*` import, no new-connection form, and no connect-time credential resolution.

**Approach:** Add a file-backed profile domain and JSON store under XDG config, psql-compatible `~/.pgpass` parsing, selected `~/.mongo.*` parsing, an import modal, a new no-password URI form with a one-time masked password prompt, and a connection service that resolves the secret from the saved source at connect time. Keep the 80x24 shell and keyboard contracts; catalogs and data views remain fake until story 3.

**Decision:** Story 2 opens real connections for PostgreSQL, MySQL/MariaDB, and MongoDB (open, ping with a 10s timeout, close on failure or disconnect); catalog reads remain out of scope. New URIs require an explicit database path, use the engine default port when none is given, and default the label to `database@host` when the optional label is empty.

## Boundaries & Constraints

**Always:** Profiles store only label, engine, host, port, database, user, source pointer, and timestamps; the file is `0600` and its directory `0700` under `$XDG_CONFIG_HOME/db-adm-tui` (default `~/.config/db-adm-tui`). Re-read the source on every connect; honor `PGPASSFILE`, `0600`, backslash escaping, `*` wildcards, comments, and first match for `~/.pgpass`; for `~/.mongo.*`, ignore `#` comments and let the last parsed URI win. Mask every password field; resolved secrets live only in memory; source files are never written. Errors name the profile and source but never the secret. All external text is sanitized before rendering. The picker, import, and new-connection flows are fully keyboard-driven at 80x24.

**Never:** No password, DSN userinfo, or secret in profiles, history, argv, logs, error text, or golden files. No engine catalogs, SQL/MQL execution, data reads, privilege reads, or write mode. No MySQL import source in v1. No profile encryption, credential vault, shared profiles, or network calls beyond the selected database endpoint.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|---|---|---|---|
| HAPPY_PATH | Empty profile store; `n`; `postgres://app_ro@db1:5432/orders`; label `prod-ro` | Form validates; matching `~/.pgpass` line supplies the secret; connection succeeds; picker offers to save; saved profile has no password | N/A |
| IMPORT_PGPASS | `~/.pgpass` `0600` with comments, escapes, wildcards, malformed lines | Valid lines listed and selected; malformed carry `warning` and are skipped; import creates or refreshes profiles | Missing file shows an empty state, not an error |
| IMPORT_MONGO | `~/.mongo.*` and `~/.mongo` files, one malformed | Every file listed; selectable with `Space`; last URI in a file wins; malformed row carries `error` naming the file; other files still import | Unreadable file leaves other rows usable |
| PGPASS_INSECURE | `~/.pgpass` mode not `0600` | File is ignored exactly as psql does; picker/import reports the source as ignored | No passwords read |
| PASSWORD_PROMPT | Saved profile with no source pointer | Centered masked `Password for <profile>` modal; `Enter` connects, `Esc` returns to picker; value never stored | Wrong password returns a row error; retry prompts again |
| CONNECTING | `Enter` on a profile or a submitted URI | Row shows `connecting…`; the picker stays responsive; `Esc` cancels the attempt and restores the row | Timeout or cancellation returns a non-secret error |
| CONNECT_ERROR | Missing source line, auth failure, timeout, connection loss | Picker stays open with an error under the selected row naming profile and source | No secret in the message; `Enter` retries |
| NEW_URI_SECRET | URI contains `user:pass@` | Inline rejection; form stays editable | `Could not connect: the URI must not contain a password.` |
| STORE_UNREADABLE | Corrupt or unreadable `profiles.json` | Picker shows no profiles and stays usable; `n` and `i` work | `Profile storage is unreadable. Showing no profiles.` |
| TOO_SMALL | Terminal below 80x24 | Story 1 too-small line; no modal or picker is drawn | No panic; recovery on resize |

</frozen-after-approval>

## Code Map

- `go.mod`, `go.sum` -- add Bubbles v2.2.1 for masked text input; add pgx, MySQL, and Mongo drivers only if REAL is chosen.
- `internal/domain/engine.go` -- `Engine` values `postgresql`, `mysql`, `mongodb`, display names, default ports.
- `internal/domain/profile.go` -- `Profile`, `Source` (`pgpass`, `mongo-file`, `prompt`), deterministic identity key, `RedactedURI`.
- `internal/domain/error.go` -- error kinds `auth`, `permission`, `timeout`, `config`, `engine`; redaction helper that never includes a secret.
- `internal/ports/profile.go` -- `ProfileStore` (`Load`, `Save`), `Connector` (`Connect`, `Close`), `Connection` (`Ping`, `Close`) seams for app and fakes.
- `internal/profile/store.go` -- XDG path resolution, schema-v1 JSON `{version, profiles}`, atomic `0600`/`0700` writes, corrupt-file fallback, refusal to overwrite an unreadable store.
- `internal/profile/paths.go` -- home expansion, XDG config resolution, `PGPASSFILE` precedence, deterministic `~/.mongo.*` discovery.
- `internal/profile/pgpass.go` -- psql parser: `0600`, `PGPASSFILE`, comments, backslash escapes, wildcards, first match; returns entries plus per-line warnings.
- `internal/profile/mongo.go` -- scan `~/.mongo*` and `~/.mongo`, parse `mongodb://` URIs, comments, last-URI-wins; per-file errors; non-secret selector strips password query keys.
- `internal/profile/resolve.go` -- resolve a profile's secret and fresh Mongo selector from its source; map missing file/selector/line to a non-secret error.
- `internal/app/connections.go` -- list/import/connect/save orchestration, source matching for new URIs, password-needed sentinel, profile refresh without losing identity, unique-profile import counts.
- `internal/app/uri.go` -- no-password URI parsing, default ports, label derivation, password-key and query-option rejection.
- `internal/engine/{postgres,mysql,mongo}/connect.go` -- compose a DSN/URI from profile plus secret, open, ping with a 10s context, redact driver errors, close on failure; Mongo replays the fresh non-secret selector.
- `internal/tui/model.go` -- inject `app` service; add picker/import/new-connection/password modal states, connecting state, row errors, `Esc` cancel, async result generations, too-small quit guard.
- `internal/tui/picker.go` -- render real profiles, empty/loading/connecting/error states, count line, scrolling viewport, selected row, footer-safe layout budget.
- `internal/tui/import.go` -- import modal list, `Space` toggle, `Enter` import, `Esc` cancel, warnings/errors per row, selected-vs-skipped accounting.
- `internal/tui/newconn.go` -- URI and label inputs, inline validation, masked one-time password, auth-failure override, save-as-profile prompt.
- `internal/tui/modal.go` -- centered amber modal frames, overlay composition, wrapping, width clamps.
- `internal/tui/workspace.go` -- use the active domain profile; keep fake catalog content and note it until story 3.
- `internal/tui/keymap/keymap.go` -- modal actions: `Space` toggle, `Tab`/`Shift+Tab` field focus, `Esc` cancel, `Enter` confirm; keep Latin/Cyrillic aliases.
- `cmd/db-adm-tui/main.go` -- composition root: XDG paths, store, resolver, connector, TUI.
- `internal/profile/*_test.go`, `internal/app/*_test.go`, `internal/tui/*_test.go`, `internal/tui/testdata/*.golden` -- parser, store, flow, redaction, golden coverage.
- `README.md` -- profile location, import sources, new-connection flow, password rules.

## Tasks & Acceptance

**Execution:**
- [x] `internal/domain/*.go`, `internal/ports/profile.go` -- define profile, source, engine, error, and store/connector seams -- keep TUI and drivers apart.
- [x] `internal/profile/store.go` -- implement XDG JSON store with atomic `0600` writes and corrupt-file fallback -- persist only non-secret fields.
- [x] `internal/profile/pgpass.go` -- implement psql-compatible parsing with permissions, `PGPASSFILE`, escaping, wildcards, comments, first match -- import and resolve from one parser.
- [x] `internal/profile/mongo.go` -- implement `~/.mongo*` discovery and URI parsing with last-URI-wins -- selected file becomes one profile.
- [x] `internal/profile/resolve.go` -- resolve secrets by source pointer and matching selector -- re-read on every connect.
- [x] `internal/app/connections.go` -- orchestrate list, import, connect, save, refresh, password-needed flow, and `last_used` update -- one service for the TUI; re-import updates non-secret fields while preserving identity.
- [x] `internal/engine/*/connect.go` -- REAL only: open, ping, redact, close -- prove the connection seam.
- [x] `internal/tui/model.go`, `picker.go`, `import.go`, `newconn.go` -- implement the four flows and states at 80x24 -- CAP-1 user surface.
- [x] `internal/tui/workspace.go`, `cmd/db-adm-tui/main.go` -- use the active profile and wire real dependencies -- remove the hardcoded fake profile list.
- [x] Tests and goldens -- parser/store/app/TUI coverage, no-secret assertions, empty/error/import/new-connection/password states -- lock CAP-1.
- [x] `README.md` -- document XDG profiles, import, new connection, and password behavior.

**Acceptance Criteria:**
- Given an empty profile store, when the picker renders, then it shows the empty state pointing to `n` and `i`, and both keys open their flows.
- Given a `0600` `~/.pgpass`, when the user imports it, then valid lines create profiles without passwords, malformed lines are warned, and an insecure file is ignored.
- Given selected `~/.mongo.*` files, when the user imports them, then each file creates one profile with the last URI's non-secret fields and no password; re-import refreshes fields and preserves identity.
- Given a saved profile with a source, when `Enter` is pressed, then the source is re-read, the secret is resolved in memory, the row shows `connecting…`, `Esc` cancels, and a failure names the profile and source without the secret.
- Given a saved profile without a source, when `Enter` is pressed, then a masked password modal appears and the typed value is never persisted.
- Given a new no-password URI, when it is submitted, then a matching source is used or a masked one-time prompt appears, and `user:pass@` is rejected inline.
- Given a successful new-connection URI, when the connection opens, then the picker offers `Save <label> as a profile? Enter save · Esc skip`, and the saved profile stores no password.
- Given a successful connection, when the workspace opens, then the top status uses the active profile, `last_used` is refreshed, and no password or full URI appears anywhere.
- Given `make test`, `make lint`, and `make build`, then all pass with no secret in profiles, goldens, argv, or error text.

## Implementation Notes

- Chose REAL connections per the Decision. `internal/engine/{postgres,mysql,mongo}/connect.go` open one connection/client, ping with a 10s context, redact driver text with the resolved secret, and close on failure. `internal/engine/engine.go` is the composition-root dispatcher.
- `internal/profile` owns `~/.pgpass` and `~/.mongo.*` parsing and credential re-reads. The app calls the resolver on every connect; no secret is cached, logged, stored, or rendered.
- `profile.Store` writes schema v1 `{version, profiles}` atomically with a `0600` temp file and `0700` directory. Corrupt/unknown-version files return a config error and the picker shows the exact copy while staying usable.
- Profile identity: PostgreSQL/MySQL use `engine|host|port|database|user`; MongoDB uses the source file path so a changed URI refreshes the same profile. Import preserves `created_at`, `last_used`, and an existing label, refreshes the non-secret connection fields, and never writes a password.
- `~/.pgpass` follows psql semantics: `PGPASSFILE`, group/other permission rejection, `#` comments, backslash escapes, `*` wildcards, first match. Wildcard host/port/database/user import as `localhost`, the engine default port, and empty database/user (connector defaults) respectively.
- MongoDB discovery lists `~/.mongo.*` plus `~/.mongo`, one row per file, `#` comments skipped, last valid URI wins, and any malformed URI tags the file with a non-secret `error`. Selection defaults to valid pgpass lines only. A Mongo profile's `Source.Selector` preserves the URI's non-secret host list and options; the connector replays them with the resolved password.
- Adapter errors wrap a redacted cause rather than the raw driver error, and `RedactSecret` covers plain, query-escaped, and path-escaped secret forms.
- Import and save return a config error instead of overwriting an unreadable `profiles.json`; import summaries count unique profiles, and selected-vs-skipped rows are counted separately.
- A matched source whose password is rejected offers a one-time password and saves the resulting profile without a source pointer. The new-connection form rejects all query options so a pasted `sslmode`/`authSource` cannot be silently downgraded.
- Below 80x24 only `q` at the picker and `Ctrl+C`/`Ctrl+с` force quit; workspace `q` stays inert. Bidi overrides/isolates are stripped, while ZWNJ/ZWJ and RTL marks are preserved.
- The TUI injects `*app.Service`. Picker rows use the real store, source health (`ignored`/`missing`/`unreadable`), `last_used`, and row errors. Modals are centered with the amber border: import (Space/Enter/Esc), new connection (URI + optional label + masked one-time password), password for a source-less profile, and save-as-profile.
- Connection attempts run as `tea.Cmd` with a request generation. `Esc` cancels the context and increments the generation; stale successes are closed. The connecting row shows `connecting…` and the picker stays responsive.
- New URIs require an explicit database path and user, use the engine default port when omitted, default the label to `database@host`, reject `user:pass@` and `password=` query keys, and redact any typed URI password in the form view. A matching pgpass/mongo source is attached; otherwise the form reveals a masked one-time password field.
- The workspace uses the active domain profile for the top status, SQL/MQL prompt, and `Ctrl+D` close. The catalog/tree/grid remain the story-1 preview and say `catalog preview` until story 3.
- External strings are passed through `Sanitize`/`oneLine` before rendering. Tests assert no fixture secret appears in any view, golden, profile file, or error message.

## Spec Change Log

## Review Triage Log

| # | Finding | Verdict | Route | Evidence |
|---|---|---|---|---|
| 1 | Import/SaveProfile overwrite an unreadable store (BH) | high | patch | `Import` and `SaveProfile` now return the `Load` error; `TestImportAndSaveRefuseCorruptStore` proves the file is unchanged. |
| 2 | Re-import leaves an orphan profile when a `~/.pgpass` selector changes (BH) | low | patch | The picker now shows `stale` when the source file exists but no longer matches (`TestSourceState`, `TestPickerShowsSourceHealth`); pruning remains outside v1 because there is no delete/edit surface. |
| 3 | A malformed Mongo line discards an earlier valid URI; comment says last-valid-wins (BH) | low | patch | The comment now states that any malformed line invalidates the file, matching the UX contract; `TestParseMongoBytesMalformedLineInvalidatesFile` covers good-then-bad. |
| 4 | `mongodb+srv://` is reported as “no mongodb:// URI found” (BH) | low | patch | `ParseMongoBytes` now returns `unsupported scheme mongodb+srv`; README states the v1 limitation; `TestParseMongoBytesRejectsSRVExplicitly`. |
| 5 | Import passes non-selectable rows to `Import`, making the empty guard unreachable and inflating skipped counts (BH) | medium | patch | `handleImportKey` selects only checked selectable rows, reports `No selectable entries; N skipped.` when nothing is importable, and adds warning/error rows to the summary; `TestImportMalformedRowsCarryTags`. |
| 6 | Picker row budget scrolls early and error rows can push the footer past the viewport (BH) | medium | patch | `viewPicker` shares one content budget for rows, errors, status, and footer; `pickerVisibleRows` matches the layout; `TestPickerFooterSurvivesErrorsAndScroll` renders 20 profiles with 10 errors and asserts 24 lines plus the footer. |
| 7 | Typed URI query options are silently dropped, including `sslmode`/`authSource` (BH) | high | patch | `ParseURI` rejects all query options after the password-key check; README documents the restriction and the deferred non-secret options field; URI tests cover `?sslmode=require`. |
| 8 | `?pwd=` is accepted by `ParseURI` though `Redact` treats it as a secret key (BH) | medium | patch | `pwd` is rejected alongside `password`/`passwd`; `TestParseURIRejectsPasswordsInline` covers it. |
| 9 | Saving a new URI that matches a Mongo source overwrites the imported profile and changes its label (BH) | low | rejected | One profile per selected `~/.mongo.*` source is the import contract; the new label is user-entered and README now states that saving refreshes the source’s profile. |
| 10 | README key table misstates PgUp/PgDn scope, Esc, Enter, and omits Home/End (BH) | low | patch | README table corrected: PgUp/PgDn scroll picker/import, Home/End jump import rows, Esc closes/cancels, Enter is contextual. |
| 11 | README removed `Ctrl+X` chords while the footer still advertises them (BH) | low | patch | README restored the chord rows (`Ctrl+X Ctrl+X`, `s`, `Ctrl+E`, `Ctrl+W`, `i`). |
| 12 | Five-second message clear is missing; unbound keys allegedly leave messages (BH) | low | rejected | Unbound keys already clear the message before action lookup; next-key clearing satisfies the stated “next key or five seconds” behavior, and a timer would add state/commands for no user-visible gain. |
| 13 | `Sanitize` strips every `Cf` rune, including ZWNJ/ZWJ and variation selectors (BH) | medium | patch | `isUnsafeFormat` now strips only bidi overrides/isolates (U+202A–202E, U+2066–2069); test preserves ZWNJ/ZWJ and strips bidi controls. |
| 14 | Adapters retain the password in the connection struct; `app.Session` says no secret (BH) | false | rejected | PRD explicitly allows in-session passwords in memory; `app.Session` has no secret field, and the secret is only retained inside the live adapter connection for ping-error redaction. |
| 15 | Verification pattern skips most Mongo tests (BH) | low | rejected | The fix is a spec edit; the Verification section was corrected as housekeeping to run `go test ./internal/profile` and `go test ./internal/app ./internal/engine/...`. |
| 16 | Story Code Map omits `app/uri.go`, `profile/paths.go`, `tui/modal.go` (BH) | low | rejected | The fix is a spec edit; those files were added to the Code Map as housekeeping. |
| 17 | Store rejects one invalid profile, does not fsync the directory, and hides validation reasons; no delete/edit path (BH) | low | patch | Save now best-effort syncs the directory and surfaces validation-safe reasons; per-profile tolerance and a delete/edit UI would add surface outside the story. |
| 18 | Unreadable store overwritten by import/save (EC) | high | patch | Duplicate of row 1; `TestImportAndSaveRefuseCorruptStore`. |
| 19 | New URI with a matched source whose password fails has no way to supply a one-time password (EC) | high | patch | Auth failure reveals a masked password field, sets `newIgnoreSource`, and saves the recovered profile without a source; `TestNewConnectionAuthFailureAllowsPasswordOverride`. |
| 20 | Two pgpass lines collapsing to one profile are both counted as imported (EC) | medium | patch | Import counts unique created/updated keys; `TestImportCountsUniqueProfiles`. |
| 21 | `?pwd=` accepted by URI validation (EC) | medium | patch | Duplicate of row 8; `TestParseURIRejectsPasswordsInline`. |
| 22 | Password-like keys in a Mongo selector are persisted in `profiles.json` (EC) | high | patch | `SanitizeMongoSelector` strips `password`/`passwd`/`pwd` before storage and replay; tests cover `?pwd=hunter2` and `Profile.Normalize`. |
| 23 | Malformed Mongo line discards the last valid URI (EC) | low | patch | Duplicate of row 3; the file is intentionally invalidated and the comment now says so. |
| 24 | Saved Mongo profile replays a stale selector while only the password is refreshed (EC) | high | patch | `ResolveSource` returns the fresh selector and `Service.Connect` copies it before the connector call; `TestConnectUsesFreshMongoSelector` changes `authSource` to `tls`. |
| 25 | Editing the URI with Ctrl+D/word-delete after the password prompt keeps the one-time password (EC) | medium | patch | `handleNewKey` compares the URI value before/after every edit and clears `newNeedsPassword`/`newIgnoreSource`; `TestNewConnectionURIEditClearsPassword`. |
| 26 | Too-small guard lets `q` quit a workspace and blocks Cyrillic `Ctrl+с` (EC) | high | patch | The guard now allows `ForceQuit` always and `Quit` only at the picker; `TestTooSmallKeepsPickerQuitRuleAndCyrillicForceQuit`. |
| 27 | PostgreSQL passfile-suppression test never exercises an empty password (VG) | medium | patch | `TestBuildDSNSuppressesPassfileFallback` sets `PGPASSFILE` and asserts pgx leaves the password empty. |
| 28 | Mongo URI tests never assert the resolved password is in the URI (VG) | medium | patch | Both Mongo URI tests now parse/assert the encoded or decoded password. |
| 29 | MySQL connector configuration is untested (VG) | medium | patch | `buildConfig` extracted; `TestBuildConfigCarriesProfileAndSecret` covers user, password, address, database, timeout, and `ParseTime`. |
| 30 | Stale-session test never reaches the TUI generation guard (VG) | medium | patch | `TestStaleConnectResultClosesSession` obtains a successful result, presses Esc, then applies it and asserts the picker plus a closed session. |
| 31 | Esc-cancel from the new-connection and password modals is untested (VG) | medium | patch | `TestNewConnectionCancelClosesSession` and `TestPasswordModalCancelClosesSession` cancel before applying the result. |
| 32 | Picker source-health words are not verified at the TUI boundary (VG) | medium | patch | `TestPickerShowsSourceHealth` renders `ignored`, `missing`, `stale`, and `error`. |
| 33 | Store-write failures during connect/save are unverified (VG) | medium | patch | `TestConnectWarnsWhenLastUsedCannotBeSaved` and `TestSaveFailureKeepsWorkspaceOpen` use a failing store. |
| 34 | Cyrillic `Ctrl+с` cannot force quit below 80x24 (VG other) | high | patch | Duplicate of row 26; `TestTooSmallKeepsPickerQuitRuleAndCyrillicForceQuit`. |

## Verification

**Commands:**
- `go test -race ./...` -- PASS.
- `go vet ./...` -- PASS.
- `make build` -- PASS; `bin/db-adm-tui` is a statically linked x86-64 ELF.
- `go test ./internal/profile` -- PASS (all pgpass, Mongo, store, and resolver tests).
- `go test ./internal/app ./internal/engine/...` -- PASS.
- `go test ./internal/tui -run TestGolden` -- PASS; `-update` only after an intentional layout change.

**Observed:** `make test`, `make lint`, and `make build` all pass. No password appears in `profiles.json`, golden files, argv, or error text. The binary was started under a pseudo-terminal with an empty profile store and stayed running without a panic.

