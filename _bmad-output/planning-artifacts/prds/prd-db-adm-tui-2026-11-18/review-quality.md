# PRD Quality Review — db-adm-tui

- **PRD:** `_bmad-output/planning-artifacts/prds/prd-db-adm-tui-2026-11-18/prd.md`
- **Rubric:** `.agents/skills/bmad-prd/assets/prd-validation-checklist.md`
- **Context read:** brief and addendum in `_bmad-output/planning-artifacts/briefs/brief-db-adm-tui-2026-11-18/`
- **Reviewed:** 2026-11-18
- **Verdict:** CONCERNS — 1 critical, 6 high, 11 medium, 1 low

## Overall verdict

This is a well-crafted PRD: the FR consequences are unusually concrete, the Non-Goals do real work, and the read-only guard is described with more honesty than most safety features get. What's at risk is coverage and self-consistency at the top of the document — the product's headline differentiator (privilege inspection) has no functional requirements at all, the Vision says v1 blocks writes in the query bar while FR-30 ships a read-write switch, and several test targets (the "reference host", the filtered-row estimate in FR-16) are undefined or unsatisfiable. On the synthesis ladder this lands at Poor because of the critical finding, but the defects are bounded: three added FR groups and a handful of reconciliations, not a rewrite.

## Decision-readiness — thin

The document states decisions rather than considerations: read-only default, restricted MQL, 100-row pages, the 10,000 cap, blocklist-before-engine. Trade-offs are named where it matters most — estimates over exact counts "because exact counts are expensive", a sliding window over unbounded memory, and the repeated insistence that the guard is "not a security boundary". A pushback about safety or scope would find its objection acknowledged.

The problem is the first paragraph of the Vision. It says: "For v1, read-only is a product promise enforced by the tool's own input filter: write statements are blocked in the query bar and the UI offers no data editing or privilege changes." That is categorical, and §2.6 plus FR-30 contradict it: every connection starts read-only, but `Ctrl+X Ctrl+W` switches to read-write "after confirmation", after which the query bar will run `DELETE`. The brief is explicit that the switch is in v1 ("writes require an explicit, confirmed switch to read-write mode"), so the Vision sentence is the outlier — but a reader skimming §1 will come away believing v1 cannot write at all. Because this is the product's headline safety promise, it must be one sentence, consistently, in both places.

Two smaller decision gaps: sorting is cut with no rationale, and the Open Questions list is mostly architecture in disguise. Q3 (blocklist coverage) and Q4 (profile storage) are explicitly "owned by architecture"; Q1 is a decision deferred with a stated trigger. Only Q2 is a live product question, and it has no owner or date — which matters because FR-22 ships the help corpus in the binary.

### Findings
- **[high] Vision contradicts the v1 read-write switch** (§1 vs §2.6 / FR-30) — "write statements are blocked in the query bar" (v1, unconditional) versus "`Ctrl+X Ctrl+W` switches between read-only and read-write" with writes then permitted. *Fix:* state the v1 posture once — read-only default, query-bar writes only after a confirmed switch, no editing UI — and make §1 match §7.1's wording.
- **[medium] Sorting is cut without a rationale and referenced as if it exists** (§6, §7.2 vs FR-18) — "No column sorting or data export in v1; both are deferred to v3", yet FR-18 says the case mode "does not change SQL or MQL execution, command completion, or sorting". For a tool whose job is browsing data, deleting sort is a real decision. *Fix:* decide whether sorting ships in v1; if not, drop the FR-18 reference and record why sorting can wait.
- **[medium] Open Questions are three-quarters architecture-owned** (§9) — Q3 and Q4 are architecture decisions by the PRD's own admission; Q1 is deferred by decision. Q2 ("Who curates the offline SQL/MQL help set, and at what cadence") is the only live product question and has no owner or timing. *Fix:* keep Q2 with an owner, move Q3/Q4 into an architecture-inputs note.

## Substance over theater — strong

Nothing here is furniture. The personas are replaced by JTBD plus a Non-Users list that does work; the three UJs are short, named, and each carries an edge case that changes a requirement (the `INHERIT FALSE` nuance in UJ-1, the invalid regexp in UJ-2, the cancelled exact count in UJ-3). The UJ density note — "this is an internal tool with a single operator role" — is the right calibration for the shape.

The NFRs are product-specific rather than boilerplate: 300 ms to the profile picker, 50 ms keypress feedback, 80x24, UTF-8 without color, version floors per engine, `0600` history files, ANSI-injection sanitization, no listening ports. NFR-10 ("no network calls except to the database endpoints the user selected") is the kind of constraint that only comes from knowing the deployment context.

The one asserted-not-earned claim is the closing line of the Vision: "It wins by being small, fast, predictable, comfortable, and honest about what it does not do." The brief's addendum holds the evidence that would make this checkable (no surveyed TUI combines on-server operation, MongoDB and SQL in one static binary; none imports `~/.pgpass`), but the PRD does not borrow even one clause of it.

### Findings
- **[low] Differentiation is asserted, not grounded** (§1) — "It wins by being small, fast, predictable, comfortable, and honest" is unfalsifiable on its own. *Fix:* name the specific gap in one sentence (on-server + MongoDB + SQL in one static binary; native `~/.pgpass` import) so the claim can be checked.

## Strategic coherence — thin

The thesis is clear and it is a good one: the tool is already where the data is, so the winning move is to unify *movement* (one tree, one keyboard model, one set of views) while keeping each engine's native vocabulary. Features follow from it — static binary, no ports, layout-independent shortcuts, restricted MQL, conservative guard. The MVP is a problem-solving/experience scope, and the scope logic matches. The counter-metrics are genuine rather than decorative: SM-C2 ("Do not relax the read-only blocklist to make a query pass") and SM-C3 (bounded memory) both defend the thesis against its own convenience features.

The arc breaks at its strongest claim. The brief calls privileges "a first-class view" and "the moat"; SM-1 is the first success metric ("Privilege answer time"), and UJ-1 is entirely a privilege-inspection journey, down to `CONNECT`/`USAGE`/`SELECT` and the `SET ROLE` nuance. But §2.1 FR-3 only requires that the Security node *list* users and roles. §2.2 defines details for tables and collections. No FR anywhere says selecting a user or role displays its grants, scope, memberships, or effective privileges, and §7.1's in-scope list says only "a Security node with users and roles". The product's differentiator is therefore unfunded — and SM-1's "Validates FR-1, FR-2, FR-3, FR-4" is a trace to requirements that cannot produce the measurement.

A second, smaller gap: §2.5 (the query bar, eight FRs) has no user journey and no success metric, and MySQL/MariaDB — one of three supported engines — appears in no UJ and no SM. The MySQL-specific consequences (FR-3's "native scope labels", FR-18's regexp flags) are the only places the engine is specified.

### Findings
- **[critical] No functional requirements for privilege inspection** (§2.1 FR-3, §2.2, §7.1 vs UJ-1, SM-1) — FR-3 stops at listing users and roles; nothing requires the grants view that UJ-1 describes ("sees its native grants: `CONNECT`, `USAGE` on `public`, `SELECT` on three tables") or its `INHERIT FALSE`/`SET ROLE` behavior. SM-1 measures privilege answer time and claims validation by FR-1..FR-4, none of which display a grant. *Fix:* add FRs for user/role detail — native grants, scope, memberships, granted vs effective, `INHERIT FALSE` handling — with per-engine testable consequences; add them to §7.1 and re-map SM-1.
- **[medium] The query bar and MySQL/MariaDB have no journey and no success signal** (§2.5, §7.1) — eight FRs for the query bar, zero UJs; one third of the engine surface, zero UJs and zero SMs. *Fix:* add one journey that uses the query bar on MySQL/MariaDB, or state explicitly that the query bar is validated only by SM-2.

## Done-ness clarity — thin

At the FR level this is the best part of the document. Almost every consequence is a verifiable condition: "at most 100 rows or documents", "the row shows `unavailable`, not `0`", "exactly 10,000 are loaded and the status line says `10,000 loaded; more available`", "`Ctrl+X s`, not `Ctrl+S` (terminal flow control)", the literal banner in FR-16, the `0600` permissions in NFR-7. An engineer could build from these.

The failures are at the level of measurement and of one stated contract. "The reference host" appears in NFR-1 and SM-5 (and NFR-3 has no host qualifier at all) and is never defined anywhere in the PRD or the brief — so the 300 ms start, 50 ms keypress, and 1 s first-page targets cannot be verified by anyone but the author. FR-16 promises a banner reading "N matching rows loaded of ~M total rows (estimated)" for an arbitrary regexp on one column; none of the three engines can supply an estimated *matching* total for a regexp from catalogs, and the FR provides no `unavailable` fallback even though FR-5 establishes exactly that pattern. FR-13 and NFR-5 disagree about what "loaded" means: FR-13 says a second fetch means "all are loaded and the status line reflects the complete set", while NFR-5 caps each data view at 10,000 rows or 64 MB and evicts the oldest pages — on any object larger than the window, "the complete set" is unreachable, and FR-13's "the cap applies per request, not to the total that can eventually be loaded" describes a total the view is not allowed to hold. NFR-5's "re-fetched on scroll" also assumes a stable row order that no FR requires; on a live table, offset re-fetch can duplicate or skip rows.

Finally, the read-only guarantee is stated more absolutely than a keyword blocklist can deliver. NFR-9 says "no write statement reaches the engine"; FR-28 blocks keywords and nested CTE writes and blocks when "the parser is in doubt". Side-effecting reads pass a keyword filter untouched: `SELECT nextval(...)` / `setval(...)`, `pg_terminate_backend`, `SELECT ... FOR UPDATE`, `NOTIFY`. That is not a reason to abandon the guard — it is a reason to state the limit in the same breath as §2.6 does for privileged operators.

### Findings
- **[high] "Reference host" is undefined, so the performance NFRs are untestable** (§5 NFR-1, NFR-3, SM-5) — "The binary starts ... in under 300 ms on the reference host" with no host specification (CPU, RAM, disk, database locality, "normal load"). *Fix:* define the reference host numerically, or mark the numbers as provisional and hand the definition to architecture as an explicit input.
- **[high] FR-13 and NFR-5 contradict each other about the complete set** (§2.3 FR-13 vs §5 NFR-5) — "If 10,000 or fewer remain, all are loaded and the status line reflects the complete set" versus a 10,000-row / 64 MB window that evicts oldest pages; FR-13's "cap applies per request, not to the total that can eventually be loaded" describes a total the view cannot retain. *Fix:* state that fetch-all is windowed, define what the status line says after eviction, and require a stable order key or name the duplicate/skip risk.
- **[high] FR-16's filtered total cannot be produced** (§2.3 FR-16) — "`Filtered by regexp on <field>; N matching rows loaded of ~M total rows (estimated).`" requires a matching-count estimate for an arbitrary regexp, which PostgreSQL, MySQL and MongoDB do not expose. *Fix:* make the total optional and fall back to `total unavailable`, mirroring FR-5.
- **[high] NFR-9 overclaims what the blocklist guarantees** (§5 NFR-9, §2.6 FR-28) — "no write statement reaches the engine" versus a conservative keyword blocklist; side-effecting reads (`nextval`, `setval`, `pg_terminate_backend`, `SELECT ... FOR UPDATE`, `NOTIFY`) pass. *Fix:* restate as "no statement matching the blocklist reaches the engine; unclassifiable statements are blocked" and name the side-effect limitation alongside the "not a security boundary" caveat.
- **[medium] FR-18's regexp flags leave MariaDB unimplementable** (§2.4 FR-18) — the table names PostgreSQL `(?i)`, MySQL `REGEXP_LIKE(..., 'i')`, MongoDB `$options: 'i'`; MariaDB 10.4+ (supported by NFR-13) has no `REGEXP_LIKE` and needs `REGEXP`/`RLIKE` with PCRE inline flags. *Fix:* add the MariaDB row or defer the per-engine mapping to architecture.
- **[medium] Status-bar capacity is never budgeted** (§2.3 FR-17, §2.4 FR-18, §2.6 FR-27, §2.2 FR-16) — FR-17 alone requires loaded count, estimated total, filter state and case mode; FR-18 adds the case mode "outside the work areas", FR-27 the mode indicator, FR-16 a persistent banner — at 80x24 (NFR-12). *Fix:* define status-bar priority and truncation rules, or move some indicators into the pane chrome.
- **[medium] MQL result columns are unstable across pages** (§2.5 FR-25) — "columns are the union of top-level fields in the loaded documents, with `_id` first"; the union grows as pages load and no rule fixes column order or identity after a page arrives. *Fix:* state column ordering and whether existing columns move when a new field appears.
- **[medium] Load-bearing terms inside FRs are undefined** (§2.5 FR-21, §2.6 FR-30, §2.5 FR-24) — "recent identifiers" (source, window, persistence), "elevated rights" and "root launches" (which privilege per engine; how MongoDB's root role is detected), and "the history path". *Fix:* define each, or convert the detection rule into an architecture input.

## Scope honesty — adequate

The explicit Non-Goals section is the strongest instrument in the document: 14 bullets, each a real exclusion, and §7.2 maps deferrals to v2/v3 rather than hiding them. Six assumptions are tagged inline and all six appear in the index; the open-item count (4 OQs + 6 assumptions) is defensible for a personal-first v1.

What is not honest is the coverage gap between §7.1 and §2. The In Scope list opens with "Connection profiles imported from `~/.pgpass` and all `~/.mongo.*` files (including `~/.mongo`); passwords resolved at connect time and never stored" — the first thing every user does — and no FR covers it. The Glossary defines "Connection profile" and "Source file" precisely enough to imply the behavior, but there is no requirement for scanning the files, the `0600` rule, wildcards, `PGPASSFILE`, which line wins, multiple `~/.mongo.*` files, malformed files, or duplicate profiles. The same pattern applies to history: §7.1 requires "Command history in JSONL and TXT with redaction and rotation", while FR-24 covers only navigation; the rotation policy (10 MB × 5 per format) exists only in the brief addendum, and the dual format is never justified.

There are also zero `[NOTE FOR PM]` callouts. The document's real tensions — writes through the query bar in a "read-only" v1, the sorting cut, help-content ownership, the feasibility of FR-16's estimate — are all written as settled prose, which makes them harder to find than they should be.

### Findings
- **[high] Profile import has no requirements** (§7.1 vs §2, Glossary) — the v1 entry path is specified in the brief addendum and summarized in a scope bullet, but no FR defines import, precedence, failure handling, or the picker itself (ordering, creation, refresh after the files change). *Fix:* add a profile-import FR group with testable consequences per credential format, and reference it from §7.1.
- **[medium] History persistence, format and rotation have no FR** (§7.1, FR-24) — §7.1 requires JSONL + TXT, redaction and rotation; FR-24 requires only navigation, per-profile persistence and duplicate collapsing; rotation lives in the addendum. *Fix:* add an FR for history file format, rotation thresholds, redaction markers and failure behavior, or drop the capability from §7.1.
- **[medium] FR-22 hides an unbounded content deliverable** (§2.5 FR-22) — "Help covers SQL and MQL keywords, functions, operators, commands, and interface elements", "ships with the binary and is versioned with the product; releases update the help set" — three dialects of curated content, with ownership still open in §9 Q2 and no size bound. *Fix:* bound v1 coverage (e.g. N keywords/operators per dialect plus all UI elements) or move corpus growth to v1.x.
- **[medium] No `[NOTE FOR PM]` at any real tension** (§1, §6, §9) — read-write in v1, sorting to v3, help ownership, and the FR-16 estimate are all asserted rather than flagged. *Fix:* add 2–3 notes at those points so the next reader sees the seams.

## Downstream usability — adequate

This is close to clean. FR-1..FR-30, NFR-1..NFR-15, UJ-1..UJ-3 and SM-1..SM-7 (plus SM-C1..SM-C4) are contiguous and unique; cross-references to §2.2, §2.3, §2.4 and §2.6 all resolve; the Glossary is real vocabulary with 19 entries rather than a decoration; each UJ names Mcgru and carries its context inline; sections stand alone well enough for source extraction. For a chain-top PRD, that is the right level of traceability.

The gap is that the Glossary omits several nouns the requirements depend on — "reference host", "read-write mode" (only "Read-only mode" is defined), "elevated rights", "recent identifiers", "history path" — and that UJ-1 cannot be traced to an FR (the critical finding above), which is exactly the failure mode downstream story creation is least able to detect on its own.

### Findings
- **[medium] Glossary omits load-bearing requirement terms** (§4 vs §5, §2.5, §2.6) — "reference host", "read-write mode", "elevated rights", "recent identifiers" and "history path" are used as if defined. *Fix:* add them, or reword the requirements to reuse existing terms.

## Shape fit — strong

The shape matches the product. This is an internal, single-operator, terminal tool that feeds UX → architecture → stories, and the PRD handles that correctly: a capability-spec spine with globally numbered FRs, exactly three lightweight UJs with an explicit note that they are "a short named scene", operational NFRs, and a Non-Goals section doing more work than the personas would have. There is no UJ padding, no invented consumer narrative, no market-sizing theater. The one shape-adjacent risk is the unfunded privilege view (§2.1), which is a coverage problem rather than a form problem: the document's shape has room for it, and nothing else needs restructuring.

## Mechanical notes

- **Assumptions Index roundtrip is off by one section.** The index says "**§2.3 FR-18** — case-sensitive is the default search mode", but FR-18 lives under §2.4. The other five entries resolve.
- **FR-24's assumption tag is misplaced and duplicates a requirement.** "History is per profile, persisted under the history path" is stated as a consequence, then three lines later "[ASSUMPTION: history is per profile.]" is attached to "Consecutive duplicate statements are collapsed" — the actually unconfirmed claim is the collapsing, not per-profile history.
- **Glossary drift on the source-file set.** "Source file — `~/.pgpass` or a `~/.mongo.*` file" omits the plain `~/.mongo` that §7.1 explicitly includes.
- **Boilerplate attachment.** FR-1 carries "Letter shortcuts work in both Latin and Cyrillic layouts" although FR-1 defines no letter shortcut; the same sentence is repeated verbatim in FR-11, FR-12, FR-15 and FR-18. It belongs to NFR-12 once, with FRs referencing it.
- **Brief keymap residue.** The addendum keymap lists `Ctrl+X Ctrl+R` (show native command). The PRD correctly omits it — §7.2 defers native command preview to v2 — but never says so; UX sourcing the addendum may reintroduce it in v1.
- **"Deferred to v3" points at a version nobody defines.** §6 and §7.2 defer sorting and export to v3; the brief and PRD name only v1 and v2.
- **Case drift on the version label** — "V1" in FR-2 versus "v1" elsewhere; harmless but worth one pass.
