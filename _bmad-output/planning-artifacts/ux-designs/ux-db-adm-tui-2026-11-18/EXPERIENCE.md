---
title: "Experience: db-adm-tui"
name: "db-adm-tui"
status: final
updated: 2026-11-18
sources:
  - _bmad-output/planning-artifacts/prds/prd-db-adm-tui-2026-11-18/prd.md
  - _bmad-output/planning-artifacts/briefs/brief-db-adm-tui-2026-11-18/brief.md
  - _bmad-output/planning-artifacts/briefs/brief-db-adm-tui-2026-11-18/addendum.md
  - _bmad-output/planning-artifacts/ux-designs/ux-db-adm-tui-2026-11-18/.memlog.md
---

# db-adm-tui — Experience Spine

> Paired with `DESIGN.md` (Nightshift visual system). This spine is the behavioral contract: navigation, keys, states, copy, and flows. Visual specs live in `DESIGN.md.Components`; token references here resolve there by name. Key-screen references: `mockups/key-picker.html`, `mockups/key-workspace.html`, `mockups/key-overlays.html`. Where this pair differs from `.working/color-themes-1.html` or those mocks, the spines win.

## Foundation

db-adm-tui is a single-binary terminal application for PostgreSQL, MongoDB, and MySQL/MariaDB, run on the database host inside a closed environment. The experience is one window, one keyboard, and one live connection. It is not a GUI, not an IDE, and not a dashboard: it answers small, urgent questions about what exists, what the user can do, where a row is, and how large a table is. The PRD holds the scope; this spine holds the movement.

The surface is fixed: a top status line, a tree on the left, a context-sensitive main pane on the right, a full-width query bar, and a bottom status line. Every surface is reachable by keyboard, and every state is legible at 80×24 with no color. There is exactly one theme, Nightshift; themes per profile are a later addition, not a v1 setting. v1 holds one live connection at a time; there is no side-by-side connection tree. `Ctrl+D` disconnects to the picker.

Session state: read-only mode starts on every connection and is never persisted; the global search case mode starts case-sensitive and lasts for the session; tree width and query bar height persist per profile; query history persists per profile under the history path (FR-24, FR-42).

**Security invariants.** Passwords never appear in process arguments, logs, history files, error output, or crash text. The tool only reads source files; it never writes them. History is JSONL + TXT under the history path, created `0600`, redacted (`***` for password literals), rotated at 10 MB × 5. There is no telemetry, update check, or network call beyond the selected database endpoints. `~/.pgpass` import follows psql semantics: `0600` or ignored, `\` escaping, `*` wildcards, first match wins, `PGPASSFILE`/`passfile` override. MongoDB has no session-level read-only; read-only there is the tool's statement guard only.

## Information Architecture

| Surface | Reached from | Purpose |
|---|---|---|
| Profile picker | Launch; connection loss | Saved profiles, `n` new connection, `i` import, `?` help, `q` quit |
| Workspace | A profile connects | Tree, context tabs, query bar, both status lines |
| Details tab | Tab 1–3 in the main pane | Estimated/exact count, size, indexes, structure |
| Data tab | Tab 1–3 in the main pane | Read-only rows or documents for a table or collection |
| Privileges tab | Tab 1–3 in the main pane | Native grants and memberships for the selected object or role |
| Results tab | After any statement execution | Query rows and query errors in one place |
| New connection modal | `n` at the picker | One no-password URI and an optional label |
| Import modal | `i` at the picker | Discovered `~/.pgpass` lines and `~/.mongo.*` files |
| Confirm modal | Read-write switch; save-as-profile | Blocking confirmation with consequence and keys |
| Help overlay | `?`; `Ctrl+F1` | Keymap, or context help for the focused element |
| Edit modal | `Ctrl+X Ctrl+E` | One logical statement with completion and context help |

Key-screen references: picker → `mockups/key-picker.html`; workspace → `mockups/key-workspace.html`; confirm modal and Results error → `mockups/key-overlays.html`.

The workspace has three focus zones in a fixed order: tree → main pane → query bar → tree. `Tab` advances, `Shift+Tab` reverses, and focus never disappears. `Tab` is never completion. The main pane's tab strip belongs to the main zone; `1`/`2`/`3` select a tab, `[` and `]` move between tabs. One modal at a time; the completion popup is a popup, not a modal, and closes before any other overlay opens.

**Tree.** A connection root expands into databases. PostgreSQL adds schemas between database and objects; MySQL/MariaDB lists tables and views directly under the database; MongoDB lists collections and views. A dedicated `Security` node sits at the connection root and holds `users` and `roles`. Children load lazily on expand and failures stay attached to the parent (FR-1, FR-2, FR-3). Tables, collections, and views carry an access marker in `{colors.text-secondary}` for the connected user: `ro`, `rw`, or `—`, never dimmed; sequences carry no marker. The tree is read-only in v1.

**Context tabs.** Tabs are a function of the selection, not a global mode:

| Selection | Tabs (key order) | Opens on |
|---|---|---|
| Connection | 1 Details · 2 Privileges | Details |
| Database or schema | 1 Details · 2 Privileges | Details |
| Table or collection | 1 Data · 2 Details · 3 Privileges | Data |
| View or sequence | 1 Details (placeholder) | Details |
| Security user or role | 1 Details · 2 Privileges | Details |
| Query result | 1 Results | Results |

Selecting a different object recomputes the tabs and opens the default tab. `Connection → Privileges` shows the connected user's cluster/server-level grants (FR-37); a table or collection's `Privileges` shows the connected user's access to that object; a security user or role's `Privileges` shows that principal's own grants. A tab that does not exist for the current context is a no-op with a status-line note, never a disabled tab. `[` and `]` cycle and wrap. View and sequence Details is a placeholder (`detail view not available in v1`), never count, size, indexes, or structure. [ASSUMPTION: tab selection resets to the context default on a new tree selection, while `Esc` from Results restores the tab that was active before execution.]

## Voice and Tone

English only in v1. Short labels and abbreviations are a width decision, not a style: long words break 80 columns. The voice is friendly and calm — the tone of a competent colleague who is not impressed with themselves. No jokes, no exclamation marks, no emoji, no encouragement, no apology loops.

Writing rules:

- **Error = what happened + what to do.** Name the profile, the source file, or the object; never the secret. `Could not connect to prod-ro: no matching line in ~/.pgpass. Press Enter to retry, or i to import.`
- **Empty = what is empty + what to do.** `No profiles yet. Press n to create one, or i to import from ~/.pgpass, ~/.mypass, or ~/.mongo.*.`
- **Confirmation = consequence + keys.** `Enable read-write mode? Writes will run on prod-ro. Enter enable · Esc cancel.`
- **Status = state word + count.** `loaded 100 of ~12,480`, `end of data`, `exact`, `estimated`, `unavailable`.
- Modes are `read-only` and `read-write` in sentences, `r/o` and `r/w` in the badge. `~` prefixes an estimate; `exact` follows an exact count.
- Sentence case for sentences; uppercase only for short labels, tabs, and column headers.
- No "successfully", no "please", no "oops". A completed action reports the result, not the effort.

| Do | Don't |
|---|---|
| `No rows. public.audit_log is empty.` | `Nothing to show.` |
| `No matches for "ali". Esc clears the filter.` | `0 results found.` |
| `end of data` | `You've reached the end!` |
| `Detail view not available in v1.` | `Coming soon…` |
| `Not permitted. This account cannot read role memberships.` | `Access denied.` |
| `Counting rows in public.orders. Esc cancels.` | `Please wait…` |
| `Exact count: 48,209,117 rows (exact).` | `Done!` |
| `Blocked: write statement. r/o is on. Ctrl+X Ctrl+W to switch.` | `Permission denied!` |
| `Connection lost. Back to profiles.` | `Fatal error.` |
| `This account has elevated rights. Read-only mode is a product guard, not a security boundary.` | `WARNING!!!` |

## Component Patterns

Behavioral rules. Visual anatomy lives in `DESIGN.md.Components`.

| Component | Use | Behavioral rules |
|---|---|---|
| top-status-line | Every workspace surface | Shows profile label, engine, current database, mode badge, connected user, and a `privileged` tag when elevated rights are detected (FR-30). Mode is always present and changes the moment `Ctrl+X Ctrl+W` completes. Never shows a password or a full URI. On narrow terminals the profile label truncates first. |
| status-line | Global | One message at a time, plus context key hints on the right. Carries transient confirmations (`saved`), errors that have no pane home, and the data-view summary (FR-17): `loaded 100 of ~12,480`, the loaded window range, filter state, and `case: sensitive`. Errors persist until the next action; transient info clears on the next key or after five seconds. [ASSUMPTION: five-second transient lifetime.] Never used for blocking confirmation. |
| tree | Workspace left | Lazy expand with `Right` or `Enter`; collapse with `Left`. `Enter` or `Space` on a table or collection opens its default tab. Selection survives focus changes and tab switches. Failed child load leaves the parent expanded with an inline error. Permission failures show `not permitted` under the node. No action modifies data, grants, or roles. |
| object-tabs | Main pane top | Recomputed from the selection (see Information Architecture). `1`/`2`/`3` select by position, `[`/`]` cycle. Selecting a missing tab is a no-op with `tab not available here`. The active tab keeps its position; labels never reorder within a context. |
| details-pane | Details tab | Shows estimated count with source, size, indexes, and structure (FR-5–FR-8). `Ctrl+X s` runs the whole-object exact count, not the filtered view (FR-9). While counting, elapsed time is visible and `Esc` cancels; on cancel or failure the estimate stays and the error is inline. `unavailable` is shown, never `0`. MongoDB structure is labeled `sampled from N documents` and shows per-field counts and percentages. For views and sequences, this tab is a placeholder only (`detail view not available in v1`). Scrolls with `Up`/`Down`/`PgUp`/`PgDn`; a position indicator appears when content exceeds the pane. |
| privileges-pane | Privileges tab | Native engine terms, verbatim; no cross-engine vocabulary (FR-37–FR-39). Connected-user view and selected-user/role view share the anatomy. `direct` and `via role` are separate; `INHERIT FALSE` is marked `requires SET ROLE`. MySQL/MariaDB separates granted and active roles; PostgreSQL shows role attributes and memberships; MongoDB shows inherited roles and inherited privileges. Where effective computation is unavailable, the view says `effective unavailable`. `not permitted` (cannot inspect) and `no privilege` (inspected, absent) are different states. A failed section does not hide the others. No `GRANT`, `REVOKE`, or membership change is reachable. Scrolls with `Up`/`Down`/`PgUp`/`PgDn`; a position indicator appears when content exceeds the pane. |
| data-grid | Data tab; Results rows | First page is up to 100 rows or documents (FR-10). `m` appends the next page; at the end `m` is a no-op with `end of data`. At the last loaded row, two `PgDown` presses or three `Down` presses fetch the next page; the first press says `press PgDown again` or `press Down twice more` (FR-12). `Ctrl+A` loads up to 10,000 rows per request and slides the window thereafter (FR-13). `/` filters loaded rows only; `n`/`N` walk matches; `Ctrl+F` runs the server-side regexp on the focused cell (FR-14–FR-16). No sorting, no editing, no export. Null renders as `null`; empty string renders empty. MQL results use the union of top-level fields, `_id` first, then first-seen order; nested objects and arrays render as compact JSON (FR-25). |
| filter-bar | Data and Results | Opens on `/`; filters incrementally as the pattern is typed; count and `match 3 of 17` update live. Matching follows the global case mode. `Esc` clears and restores all loaded rows. No match turns the count into `No matches for "<pattern>". Esc clears the filter.` while the pattern stays editable. Never queries the server. Disabled while a modal is open. |
| query-bar | Workspace bottom | Bound to the current database and, for MongoDB, the current collection (FR-19). One statement per execution; multi-statement input and arbitrary JavaScript are rejected. MQL accepts `find`, `aggregate`, `count`, and `distinct` only. `Enter` executes when the completion popup is closed; `Ctrl+X Ctrl+X` executes from any zone. `Up`/`Down` browse history when focused and no popup is open (FR-24). Extra height wraps the same logical statement; it never accepts a second statement and never becomes an IDE. Disabled with a hint when no database or collection is selected. Keeps the statement after an error. `Ctrl+X Ctrl+E` moves it to the edit modal. |
| completion-popup | Query bar and edit modal | `Ctrl+Space` opens on the word at the cursor; `Up`/`Down` move, `PgUp`/`PgDn` scroll the list, `Enter` accepts, `Esc` closes, `Ctrl+Space` reopens (FR-21). Suggestions are contextual, never validation. `Tab` still moves focus. The count line shows `1–8 of N`. The status line says `no suggestions` when empty. |
| results-view | Main pane, after execution | One tab holds both rows and query errors. On success it shows rows with elapsed time and loaded count; on error it shows the engine message, the statement preserved verbatim, and `Esc returns to <object>.` Paging, `/`, and `n`/`N` work as in the data grid; `Ctrl+F` does not apply to arbitrary results in v1 (FR-25). `Esc` leaves Results and reselects the object with its previous tab. Running shows elapsed time and `Esc cancels`. |
| profile-picker | Startup; after disconnect | Rows show label, engine, host, user, source file, and last used; never a password (FR-31, FR-36). `Enter` connects, `n` opens new connection, `i` opens import, `?` opens the keymap, `q` quits. Connection failure keeps the picker and attaches the error to the row; the row stays selected for retry. The source file is re-read on every connect; a changed selector fails with the file named (FR-34). The list scrolls with `Up`/`Down`/`PgUp`/`PgDn` inside the viewport above the footer, with a count line. Empty list points to `n` and `i`. While connecting, the row shows `connecting…` and the picker stays responsive to `Esc`. |
| new-connection-modal | `n` at picker | One no-password URI line plus an optional label (FR-46). Accepts `postgres://`, `mysql://`, and `mongodb://` forms. Resolves the password from the matching source file; if no match, prompts once inside the modal and never stores it. Invalid URI or failed connection keeps the form with an inline error. On success, offers `Save <label> as a profile? Enter save · Esc skip`. Reject `user:pass@` in the URI; the password is resolved from a source file or prompted once. A saved profile without a source pointer opens a centered `Password for <profile>` modal at connect. |
| import-modal | `i` at picker | Lists every discovered `~/.pgpass` line and every `~/.mongo.*` file (plus `~/.mongo`), with engine, selector, and user; passwords are never shown (FR-32, FR-33). All valid `~/.pgpass` lines are selected by default; `Space` toggles, `Enter` imports the selection, `Esc` cancels. A `~/.pgpass` file with permissions other than `0600` is ignored; `PGPASSFILE` is honored. Malformed `~/.pgpass` lines are skipped with a `warning` while valid lines import. A malformed `~/.mongo.*` file is not imported; the row error names the file and asks the user to fix it. A missing file is an empty state pointing to `n`, not an error dialog. The list scrolls with `Up`/`Down`/`PgUp`/`PgDn` inside a viewport capped at terminal height minus margins, with a count line. Re-import refreshes non-secret fields and preserves profile identity. |
| confirm-modal | Read-write switch; save-as-profile | Blocking and one at a time. Body states the consequence on the named profile; footer states both keys. `Enter` confirms, `Esc` cancels; with no pointer, the focused action is the only default. Switching back to read-only never confirms. No nested confirmation, no "are you sure" beyond the stated consequence. |
| help-overlay | `?`; `Ctrl+F1` | `?` opens the keymap; `Ctrl+F1` opens help for the focused element or the word under the cursor, anywhere in the interface (FR-22). Offline content shipped with the binary. `Esc` closes. Unknown word or element: `no help available`. Scrolls with `Up`/`Down` when taller than the terminal. |
| edit-modal | `Ctrl+X Ctrl+E` | Opens the current statement with the cursor at the first editable position (FR-23). One logical statement; horizontal scroll, no multi-line IDE. `Enter` executes and closes; `Esc` cancels and leaves the query bar unchanged. Completion and `Ctrl+F1` work inside. On execution the query bar shows the executed statement. |

## State Patterns

| State | Surface | Treatment |
|---|---|---|
| Cold launch | Profile picker | Profiles render from local storage. No spinner; if storage is slow, the empty frame with `loading profiles…` in the status line. |
| No profiles | Profile picker | `No profiles yet. Press n to create one, or i to import from ~/.pgpass, ~/.mypass, or ~/.mongo.*.` |
| Connecting | Profile picker | Row shows `connecting…`; status line shows the target; `Esc` cancels and restores the row. |
| Connection error | Profile picker | Error attached under the row, naming profile and source file: `Could not connect to prod-ro: no matching line in ~/.pgpass. Press Enter to retry, or i to import.` |
| Profile storage unreadable | Profile picker | `Profile storage is unreadable. Showing no profiles.` The picker stays usable; `n` and `i` still work. |
| Profile save failure | Profile picker / bottom status | `Could not save <profile>. The connection is still open.` The profile stays unsaved; the session continues. |
| Password required | New connection modal / centered password modal | New connection: a masked field inside the modal. A saved profile without a source pointer: a centered `Password for <profile>` modal with `Enter connect · Esc cancel`. Masked input; never echoed, never stored. |
| Read-only default | Top status line | `r/o` in `{colors.readonly-mode}`. Every connection starts here and every reconnect resets to it. On MongoDB, a one-time status line adds `Read-only is enforced by the tool; MongoDB has no session-level read-only.` The badge stays `r/o`. |
| Read-write | Top status line | `r/w` in `{colors.readwrite-mode}`; status `read-write mode on. Read-only guard is off.` Switching back is immediate and needs no confirmation. |
| No import sources | Import modal | `No ~/.pgpass, ~/.mypass, or ~/.mongo.* files found. Press n to create a connection.` |
| Malformed import line | Import modal | The row carries a `warning` tag; valid lines still import; the summary says `1 line skipped.` |
| Import file unreadable | Import modal | The row carries an `error` tag and the file name; other rows remain selectable. This is not the same as a malformed line. |
| Tree cold load | Tree | Root shows `loading databases…`; the rest of the workspace shows `select a database` until the first expand. |
| Child load | Tree | Node stays expanded with a one-line `loading…` child; failure replaces it with an inline error and keeps the node expanded. |
| Permission denied | Tree / details / privileges | Inline `not permitted` row in the failing section; the rest of the pane remains. Never a crash or an empty pane pretending success. |
| Empty database | Tree | `no objects visible to app_ro` under the expanded database. |
| No selection | Main pane | `Select a table or collection to browse.` before any object is opened; `Nothing selected.` when the tree cursor sits on a non-object node. |
| Data cold load | Data tab | Previous rows stay visible; status line shows `loading rows…` with elapsed on slow fetches. No blank flash. |
| Empty object | Data tab | `No rows. public.audit_log is empty.` |
| No filter match | Data grid | `No matches for "<pattern>". Esc clears the filter.` Pattern stays editable; count is `0 of N loaded`. |
| Filtered view | Data grid | Banner: `Filtered by regexp on <field>; N matching rows loaded`, then `of ~M total rows (estimated)` or `total unavailable`. `Esc` clears and restores the cached unfiltered view or reloads the first page. |
| End of data | Data grid | `end of data` in the status line; `m` is a no-op. The end-of-data gesture hint appears once at the last row. |
| Window cap | Data grid | Status line shows the loaded window (`rows 1–10,000`) and `more available`; the window holds at most 10,000 rows or 64 MB, and evicted rows re-fetch on scroll. A missing stable order adds `rows may repeat or be skipped`. |
| Fetch error | Data grid | Previous view stays; error inline in the pane; status line points to retry. |
| Cancelled fetch | Data grid | Previous view stays; `cancelled` in the status line. No partial page is appended. |
| Invalid regexp | Data tab | `Invalid regexp: missing closing ]. The previous view is unchanged.` Prompt stays editable. |
| Query running | Results | `running 00:00:03 · Esc cancels`; elapsed updates once per second. The query bar keeps the statement and is not editable during execution. |
| Results cursor open | Global | While a query-result cursor is open, other engine-backed actions show `results open — Esc to release` and wait; UI navigation still works. |
| Query cancelled | Results | Previous view stays; `cancelled · 00:00:02` in the status line. If there was no previous result, the object's tab returns. |
| Query error | Results | Error block with engine message and preserved statement; `Esc returns to <object>.` The query bar keeps the original statement. |
| Empty result | Results | `0 rows. Elapsed 00:00:00.041.` The tab still exists; `Esc` returns. |
| Details unavailable | Details | `unavailable` per row, with the reason in the status line on focus. Other rows remain. |
| No indexes | Details | `no indexes.` under the `indexes` header; the rest of the pane stays. |
| Exact count running | Details | Row shows `counting… 00:00:12` and `Esc cancels`; estimate remains visible next to it. |
| Count cancelled | Details | Estimate stays; `count cancelled. estimate kept.` No error styling. |
| Exact count not permitted | Details | `error: exact count not permitted.` The estimate stays; the rest of Details is untouched. |
| Privileges loading | Privileges | `loading privileges…` for the first load; local and instant, no spinner. |
| Privilege section failure | Privileges | `not permitted. This account cannot read <section>.` in that section only. |
| Query bar disabled | Query bar | `Select a database to run a query.` or `Select a collection to run a query.` Whole bar dim; focus still enters for help. |
| Regexp outside data | Data tab | `Move focus to the data view? Enter continue · Esc cancel.` On confirm the prompt names the current field (FR-16). |
| Blocked statement | Query bar / status line | Nothing is sent. `Blocked: write statement. r/o is on. Ctrl+X Ctrl+W to switch.` The statement stays editable; the full reason replaces the hint line when the pane has room. Input rejection (multi-statement, arbitrary JavaScript, MQL outside the four commands) gets the same inline treatment with the reason, without the mode-switch hint. |
| Input rejected | Query bar / status line | `Rejected: <reason>.` The statement stays editable; no mode-switch hint. |
| Connection lost | Profile picker | The workspace closes and the picker returns with the row error `Connection lost. Press Enter to reconnect.` Other profiles stay usable (NFR-6). |
| Disconnected | Workspace → picker | `Ctrl+D` returns to the picker; if a query or count is running, a confirm modal appears first. The workspace closes; the profile row is selected for reconnect. |
| Elevated rights | Top status line | Persistent `privileged` tag in `{colors.warning}` for an elevated account or a root launch; one status-line warning at connect and at the read-write switch. |
| History failure | Status line | `History is not writable. Queries still run.` One warning per session; never blocks a query (FR-45). |
| No color | Global | Every state keeps its word: `r/o`, `r/w`, `ro`, `rw`, `—`, `error`, `warning`, `exact`, `estimated`. Selection is visible through the reverse/dim attributes as well as `{colors.selection}`. |
| Focus | Global | The focused pane draws `{colors.border-focus}` and sets its title bold and accent; key hints switch to that zone. A monochrome terminal still sees the bold title and the hints, never color alone. |
| Unknown help | Help overlay | `no help available.` `Esc` closes. |
| Terminal too small | Global | `Terminal too small: need 80×24. Current 72×20.` No workspace is drawn; the message stays until the terminal grows. A running query or count continues; `Esc` cancels. An open modal is dismissed without applying. [ASSUMPTION: the exact string and the decision to block rather than squeeze.] |
| Modal open | Global | One modal, blocking; workspace borders go inactive; `Esc` closes unless a confirmation is mid-flight. Completion popup closes first. |

## Interaction Primitives

**Focus.** Three zones, one owner: tree, main pane, query bar. `Tab` and `Shift+Tab` cycle. Focus is visible through the teal frame and a title marker, not color alone. The main pane keeps its own tab state and row selection while unfocused. Help and context help never steal focus permanently.

**Modal focus.** A modal traps `Tab`/`Shift+Tab`; focus cycles its fields top to bottom and never reaches the workspace behind it. The focused field carries a leading `>` and the modal's amber frame. `Enter` confirms, `Esc` cancels. The completion popup closes before any modal opens.

**Key chords.** `Ctrl+X` starts a chord; the second key follows within two seconds. The status line shows `Ctrl+X …` while the chord is pending. [ASSUMPTION: two-second timeout; not decided in discovery.] Notation: `Ctrl+X Ctrl+<key>` means the second key is held with Ctrl; `Ctrl+X <key>` means a plain second key. An unknown final key cancels with `cancelled` in the status line.

| Key | Scope | Action |
|---|---|---|
| `Tab` / `Shift+Tab` | Global | Move focus tree → main → query bar → tree |
| `Enter` | Contextual | Tree: open or expand. Picker: connect. Query bar: execute. Import: import selected. Modal: confirm or accept. |
| `Space` | Tree | Same as `Enter` on a table or collection |
| `Up` / `Down` | Tree, data, results, details, privileges, help, query bar | Move the cursor or scroll the pane; in the query bar, browse history |
| `Left` / `Right` | Tree: collapse / expand. Data / Results: move the cell cursor; at the edge, scroll columns |
| `Home` / `End` | Data / Results | First / last column |
| `1` `2` `3` | Main pane | Select tab by position |
| `[` / `]` | Main pane | Previous / next tab, wrapping |
| `m` | Data, Results | Fetch the next page |
| `PgUp` / `PgDown` | Data, Results, details, privileges, help, completion popup | Scroll one screen; at the last row, `PgDown` twice fetches the next page |
| `Down` ×3 at last row | Data, Results | Fetch the next page after the hint |
| `Ctrl+A` | Data, Results | Fetch all remaining, capped at 10,000 |
| `/` | Data, Results | Buffer filter over loaded rows |
| `n` / `N` | Filtered data, filtered results | Next / previous match, wrapping |
| `Ctrl+F` | Data | Server-side regexp on the focused field; offer to move focus first |
| `Ctrl+X s` | Global (object selected) | Exact count for the whole object |
| `Ctrl+X Ctrl+X` | Global | Execute the current statement |
| `Ctrl+X Ctrl+E` | Global | Edit the statement in the edit modal |
| `Ctrl+X Ctrl+W` | Global | Switch read-only / read-write (read-write confirms) |
| `Ctrl+X i` | Global | Toggle case-insensitive search; status shows `case: sensitive` / `case: insensitive` |
| `Ctrl+Space` | Query bar, edit modal | Completion popup |
| `Ctrl+F1` | Global | Context help |
| `F1` | Global | Keymap help; layout-independent |
| `?` | Global | Keymap help alias on Latin layouts |
| `Esc` | Global | Cancel / back, in the ladder below |
| `Ctrl+D` | Global | Disconnect to the picker; confirm first if a query or count is running |
| `Ctrl+Left` / `Ctrl+Right` | Global | Tree width −2 / +2 cells |
| `Ctrl+Up` / `Ctrl+Down` | Global | Query bar height −1 / +1 line; clamp 1–3 lines |
| `n` / `i` / `q` | Profile picker | New connection / import / quit |
| `Space` / `Enter` / `Esc` | Import modal | Select / import / cancel |

**Esc ladder.** In order: cancel a running query or count; close the completion popup; clear an active filter; close the topmost modal or help; leave Results for the object; otherwise do nothing. `Esc` never quits and never disconnects; `Ctrl+D` is the disconnect gesture. `q` quits only at the picker.

**Resize.** `Ctrl+Left`/`Ctrl+Right` change the tree width by 2 cells; `Ctrl+Up`/`Ctrl+Down` change the query bar height by 1 line, clamped at 1–3 lines. Both persist per profile in state. If the terminal shrinks below the saved size, the display clamps but the saved value is not overwritten. [ASSUMPTION: tree clamp 16 cells to half the terminal width; the no-overwrite rule.]

**Banned.** `Ctrl+S` and `Ctrl+Q` are never bound (terminal flow control). `Tab` never completes. `Ctrl+X Ctrl+R` is reserved for the v2 privilege matrix and is not bound in v1. No mouse-only action; mouse input is ignored in v1. No nested modals, no modal over a confirmation, no toast that covers the query bar.

## Accessibility Floor

- **Keyboard only.** Every action, including resize, import selection, and modal dismissal, has a key path (NFR-12). No drag, hover, or pointer requirement.
- **80×24 minimum.** The layout is laid out for 80 columns and 24 lines; below that, a single `Terminal too small` line replaces the workspace rather than clipping content. UTF-8 is required, but no glyph is the sole carrier of meaning.
- **Color is never alone.** The mode badge says `r/o` or `r/w`; access markers say `ro`, `rw`, or `—`; statuses carry words; focus carries a frame and a bold title; matches carry a highlight and a count. A monochrome terminal loses emphasis, not information.
- **Contrast.** Load-bearing pairs are listed in `DESIGN.md.Colors`. All text meets at least 4.5:1 (WCAG AA); body text, data, labels, and status words meet 7:1 or better on `{colors.surface}`. The one sub-7 pair is short error text in modals at 6.4:1 on `{colors.surface-raised}`. `{colors.border-default}` is structural at 1.5:1 and never carries text.
- **No motion.** No spinners, blink, or marquee. Running states use static text and a once-per-second elapsed counter. [ASSUMPTION: no spinner, because animation is unreliable and distracting on a shared terminal.]
- **No screen reader requirement in v1.** This is an explicit product decision, not an oversight; there is no ARIA or live-region analogue to satisfy. The text-first layout still leaves every value readable by a screen-scraping tool.
- **Layout-independent letters.** Every letter shortcut, including chord finals (`Ctrl+X Ctrl+W`, `Ctrl+X Ctrl+E`, `Ctrl+X s`, `Ctrl+X i`), matches the physical key position in Latin and Cyrillic layouts. `F1` is the layout-independent keymap help; `?` is an alias on Latin layouts.
- **Terminal safety.** All external strings — database content, pasted query text, source-derived values, and filenames — are rendered as plain cells; ANSI escape sequences, control characters, and title/clipboard sequences are stripped (NFR-8). No external string can change the terminal state.
- **Truncation preserves meaning.** Mode, case, and key hints survive longest; loaded counts and totals truncate first (FR-17). No state is represented only by a truncated string.

## Responsive & Platform

Terminal size is the only breakpoint system. There is no mobile or web surface.

| Terminal | Behavior |
|---|---|
| 80×24 (floor) | Tree 28 cells + 1 gap; main 51. One-line query bar. All chrome present; status counts truncate first. |
| 100×30 | Tree keeps its saved width (default 28); main gains columns and rows. Tabs and details gain room, not new elements. |
| 120×40+ | Main pane gains data columns and visible detail rows; the tree does not grow on its own. Results show more columns before horizontal scroll. |
| Height < 24 | Workspace is replaced by `Terminal too small`; the picker still renders its rows and footer if it fits. |
| Width < 80 | Same `Terminal too small` message; no horizontal squeeze, no hidden panes. |

For messages, the explanatory clause truncates first and the action key hint is preserved. For the status line, truncation order: total/estimate counts → loaded-window range → filter pattern → connected user → profile label. Preserved to the end: mode badge, `privileged` tag, `case:` indicator, key hints. Tree rows truncate names at the pane's inner width with `…`; a selected row always keeps its access marker visible. Data cells truncate at their column width and are horizontally scrollable; the row gutter stays pinned.

Widths and heights are persisted per profile. Window resize does not reset a tab, a filter, a selection, or a scroll position. Platform: Linux `x86_64` and `arm64` static binary; other Unix-like hosts best-effort; the UI assumes a UTF-8 locale and a terminal that understands single-line box drawing (NFR-11, NFR-12).

## Inspiration & Anti-patterns

- **Lifted from k9s:** the resource tree as the primary map, with a persistent top line naming the context and a bottom line for actions. One navigation model across object types.
- **Lifted from lazygit:** focus is a place, not a mode; the bottom line always shows the keys for the focused pane. Single-key actions where the context is unambiguous.
- **Lifted from yazi:** adjacent panes, keyboard movement, and a preview that does not require opening a separate window.
- **Lifted from btop:** semantic color used consistently and legibly on a dark terminal. What we take is the color discipline, not the density.
- **Rejected — noisy dashboards (btop-style graphs, gauges, sparklines):** this tool answers questions; it does not monitor. Numbers and indexes, not charts.
- **Rejected — modal overload:** modals block the work. Only the read-write switch, save-as-profile, new connection, import, help, and statement editing use one.
- **Rejected — color-only status:** every signal has a word or a shape; monochrome terminals must lose nothing.
- **Rejected — IDE-style editors, tab stacks, and docked panels:** one logical statement, one query bar, one edit modal.

## Key Flows

### Flow F1 — First connect (Mcgru, Tuesday afternoon, first run on a production host)

*Source: startup, new connection, and import (FR-31–FR-36, FR-46); not a PRD journey.*

1. Mcgru SSHes into `prod-db-01` and runs `db-adm-tui`. Cold start, no profiles.
2. The picker shows `No profiles yet. Press n to create one, or i to import from ~/.pgpass, ~/.mypass, or ~/.mongo.*.`
3. He presses `n`. The new connection modal opens with the `uri` line focused. He types `postgres://app_ro@localhost:5432/orders` and a label, `prod-ro`.
4. `Enter`. The tool matches the `~/.pgpass` line, reads the password at connect time, and opens the connection. The password never appears.
5. On success: `Save prod-ro as a profile? Enter save · Esc skip.` He presses `Enter`.
6. The tree loads `orders` and its schemas; the top line reads `prod-ro · PostgreSQL · orders · r/o`; the status line reads `connected · 2 databases`.
7. **Climax:** without one catalog query, he expands `orders → public` and sees `customers`, `invoices`, `audit_log`, each with a muted `ro` access marker. The map is already the answer.

Failure: no matching `~/.pgpass` line → the modal asks for the password once, connects, and stores nothing; a malformed URI keeps the form with `Could not connect: the URI is not valid. Press Enter to retry, or Esc to cancel.` Import edge: from the picker, `i` lists three `~/.pgpass` selectors and two `~/.mongo.*` files; `Space` selects, `Enter` imports, no password is shown; a malformed line imports nothing and carries a `warning` tag while valid lines still import.

### Flow F2 — Privilege question (Mcgru, UJ-1, a teammate waiting in chat)

*Source: UJ-1. Mcgru answers an access question on a PostgreSQL host.*

1. Mcgru launches, picks `prod-ro`, and expands `orders`.
2. He scrolls to `Security`, expands it, opens `roles`, and selects `reporting`. The main pane opens on Details.
3. `2` switches to Privileges. The pane lists native grants under `grants` and memberships under `memberships`.
4. He reads `CONNECT` on `orders`, `USAGE` on `public`, and `SELECT` on `customers`, `invoices`, and `audit_log`; each is marked `direct` or `via role`.
5. **Climax:** one membership is marked `INHERIT FALSE — requires SET ROLE`, so the view does not count it as effective. He pastes the answer into chat in under thirty seconds and exits without changing anything.

Failure: `role memberships` cannot be read → that section shows `not permitted. This account cannot read role memberships.` while the grants section stays intact.

### Flow F3 — Find one row (Mcgru, UJ-2, a MongoDB host)

*Source: UJ-2. Mcgru finds one customer's document in a closed MongoDB.*

1. Mcgru connects to the MongoDB profile and selects `customers`. Data opens on the first 100 documents.
2. He presses `/` and types `ali`. The buffer has no match: `No matches for "ali". Esc clears the filter.`
3. `Esc`. He moves the cell cursor to `email` and presses `Ctrl+F`. The prompt names the field: `regexp on email`.
4. He types `^ali` and presses `Enter`. The banner reads `Filtered by regexp on email; 12 matching rows loaded of ~12,480 total rows (estimated)`.
5. He moves to the matching document and reads the value he needs.
6. **Climax:** he presses `Esc`, the filter clears, the cached full view returns, and the banner is gone. The answer came from one field-level filter, not a hand-written query.

Failure: the regexp is invalid → `Invalid regexp: missing closing ]. The previous view is unchanged.` The prompt stays open with the text intact.

### Flow F4 — Query and Results (Mcgru, UJ-4, a MySQL host)

*Source: UJ-4. Mcgru runs a read-only check on MySQL.*

1. Mcgru connects to `billing`, selects `invoices`, and tabs to the query bar. The prompt reads `SQL ›`.
2. He types `SELECT COUNT(*) FROM invoices WHERE status = 'open';` and presses `Enter`.
3. The status line shows `running 00:00:01 · Esc cancels`. The Results tab appears.
4. The result is one row; the status line shows `1 row · 00:00:00.084`.
5. **Climax:** he reads the count aloud, presses `Esc`, and lands back on `invoices`' Data tab with his selection intact.

Failure: he mistypes `invoies`. Results shows `ERROR: Table 'billing.invoies' doesn't exist` with the statement preserved below it and `Esc returns to invoices.` He presses `Esc`, corrects the statement in the query bar, and runs again. Nothing was written.

### Flow F5 — Read-only guard and the r/w switch (Mcgru, FR-27–FR-30)

*Source: UJ-4 edge case; FR-27–FR-30. The guard, the confirmation, and the mode indicator.*

1. In read-only mode on `prod-ro`, Mcgru types `UPDATE customers SET status = 'active' WHERE id = 1041;` and presses `Enter`.
2. Nothing is sent. The status line reads `Blocked: UPDATE is a write statement. Read-only mode is on. Press Ctrl+X Ctrl+W to switch.` The statement stays in the bar.
3. He decides the change is intended and presses `Ctrl+X Ctrl+W`. The confirm modal reads `Enable read-write mode? Writes will run on prod-ro. Enter enable · Esc cancel.`
4. `Enter`. The top badge flips to `r/w` in orange; the status line reads `read-write mode on. Read-only guard is off.`
5. He runs the statement; Results reports `Rows affected: 1 · 00:00:00.012`. [ASSUMPTION: a write statement reports affected rows in Results, consistent with FR-20's count.]
6. **Climax:** he presses `Ctrl+X Ctrl+W` again and the badge returns to blue `r/o` immediately — no confirmation to become safe. The guard's state was never a guess.

Failure: the account has elevated rights → the top line carries a `privileged` tag and the switch adds `This account has elevated rights. Read-only mode is a product guard, not a security boundary.` A read statement the parser misclassifies is blocked with the same reason and the same route to the switch; the user can edit instead of switching.

### Flow F6 — Table statistics and exact count (Mcgru, UJ-3, before a migration)

*Source: UJ-3. Mcgru checks a table before a migration.*

1. Mcgru selects `orders → public → events`, a 40 GB table. Data opens; `2` switches to Details.
2. Details shows `estimated 48,211,904 rows (catalog estimate)`, `size 40.2 GB`, the index list, and the column structure.
3. `Ctrl+X s`. The count row shows `counting… 00:00:12` and `Esc cancels`; the estimate stays visible.
4. The count runs long. He presses `Esc`. The status line reads `count cancelled. estimate kept.`
5. He runs it again before the maintenance window. It completes: `48,209,117 rows (exact)` in green, next to the unchanged estimate.
6. **Climax:** the exact number, the total size, and the missing index are on one screen. He writes the migration note from the terminal and never opens psql.

Failure: the count times out or is not permitted → an inline error under the count row, the estimate remains, and the rest of Details is untouched.

## Future/Open Items

- **Per-profile theme (not v1).** Nightshift ships. The five studies in `.working/color-themes-1.html` share the same role names; Graphite, Ember, Slate, and Signal are the alternates. A later theme picker must swap the 18 color values per profile and leave role names, contrast targets, and component anatomy unchanged.
- **i18n (not v1).** English only now. Future locales must keep labels at twelve cells or fewer and preserve the status-line truncation ladder.
- **Session edges.** One live connection at a time; `Ctrl+D` disconnects to the picker; password-prompt placement is decided (inside the new-connection modal, or a centered modal for a saved profile without a source). Remaining open: the exact confirm copy for disconnecting during a query.
- **Deferred surfaces.** Detail and data views for database views and sequences; the v2 privilege matrix and `Ctrl+X Ctrl+R` native command preview; v3 sorting and export.
