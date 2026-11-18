# Reconciliation: brief + addendum vs UX spines

Compares the 2026-11-18 brief and addendum with `DESIGN.md` and `EXPERIENCE.md`. Wording, palette, and purely visual choices are ignored. One blocker, six should-fix, five notes.

| Gap | Brief/addendum location | Spine location | Impact |
|---|---|---|---|
| Credential safety ends at on-screen display. No rule against passwords in process arguments, logs, or error output; no history redaction/rotation spec (`***`, JSONL+TXT, 10 MB × 5). | brief §Solution l.36, §Success Criteria l.60; addendum l.66–67 | EXPERIENCE l.90–91, l.101, l.149 (history only as a failure state) | blocker |
| `~/.pgpass` is reduced to "discovered lines" and "no matching line". 0600 enforcement, backslash escaping, wildcards, first-match-wins, and `PGPASSFILE`/`passfile` override are absent. | brief l.28; addendum l.27 | EXPERIENCE l.36, l.101–103, l.115 | should-fix |
| Query bar becomes 1–8 lines, persisted per profile; the brief promises a one-line bar and rejects IDE-style editing. Wrapped segments soften, but contradict, that intent. | brief l.14, l.74, l.79 | DESIGN l.301, l.307, l.343; EXPERIENCE l.98, l.187–188 | should-fix |
| Resizable tree (`Ctrl+Left/Right`) and query-bar height (`Ctrl+Up/Down`), persisted per profile, are new keys and state absent from the working keymap. | not in brief; not in addendum l.31–54 | DESIGN l.307; EXPERIENCE l.187–188, l.194, l.224 | should-fix |
| Layout-independent letters are narrowed to `n`/`i`/`q`/`m`. Chord finals `s`, `w`, `e`, `i`, and `?` are unguarded, so Cyrillic layouts can break exact count and the r/w switch. | addendum l.54 | EXPERIENCE l.206 | should-fix |
| A uniform `r/o` badge hides that session-level read-only exists for PostgreSQL/MySQL only; MongoDB has no second layer. No UX state marks a weaker guard. | addendum l.61; brief l.36, l.62 | EXPERIENCE l.90, l.117–118, l.148 | should-fix |
| Views and sequences get a Details tab (count/size/indexes), though the brief limits detailed statistics to tables and collections; Future/Open Items also calls these views deferred. | brief l.70 | EXPERIENCE l.51 vs l.321 | should-fix |
| `Ctrl+X Ctrl+R` "show the native command" is dropped without resolution; the addendum both authorizes it and reserves it for the v2 matrix. | addendum l.38 vs l.54 | EXPERIENCE l.321 (v2 only) | note |
| A `?` keymap overlay is added; the brief authorizes only `Ctrl+F1` context help. | brief l.34; addendum keymap l.31–54 | EXPERIENCE l.38, l.105, l.185 | note |
| Re-import merge semantics ("refreshes non-secret fields, preserves identity") and the save-as-profile confirmation are new behavior not described in the brief. | brief l.69 | EXPERIENCE l.102–103 | note |
| A malformed `~/.mongo.*` is skipped line-by-line by the spine; the addendum expects the user to fix the file. | addendum l.29 | EXPERIENCE l.103, l.120 | note |
| The no-telemetry / no-network trust claim is never restated; the anti-pattern list implies "not a dashboard" but does not bind network behavior. | brief l.43, l.79 | EXPERIENCE l.226–235 | note |
