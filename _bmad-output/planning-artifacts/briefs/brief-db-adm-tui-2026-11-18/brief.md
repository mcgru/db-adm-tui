---
title: "Product Brief: db-adm-tui"
status: ready
created: 2026-11-18
updated: 2026-11-18
---

# Product Brief: db-adm-tui

## Executive Summary

db-adm-tui is a keyboard-first terminal application for administering databases from the server that hosts them. An engineer SSHes into a closed environment, runs one binary, and works with PostgreSQL, MongoDB, and MySQL/MariaDB without tunnels, proxies, or a GUI. [ASSUMPTION: shipped as a single static binary with no runtime dependencies]

It is the tool for the moment someone needs a quick answer: see which databases exist, browse tables and collections, understand what the connected user is allowed to do, and find the rows that answer a question. Queries are typed as one logical statement with autocomplete and inline help rather than in a full SQL editor; the bar may wrap to three lines. Every action has a keyboard path.

v1 is read-only by default: browse, filter, and inspect privileges; writes require an explicit, confirmed switch to read-write mode.

## The Problem

Access to production and closed-environment databases is constrained by network architecture, not by missing features. GUI clients cannot reach the database directly; teams set up SSH tunnels, jump hosts, and proxies, or forbid direct access entirely. The path that always works is to SSH onto the database host and use what is installed there: psql, mongosh, mysql.

That path is powerful and unpleasant. Each engine has its own shell and vocabulary; there is no shared view of databases, tables, and collections; privileges are inspected by recalling engine-specific catalog queries; searching data means writing ad-hoc SQL or MongoDB queries by hand. Existing terminal clients cover SQL engines or MongoDB, rarely both; none combine on-server operation with privilege visibility, and privilege management is absent across the category.

The cost is time and errors: every "quick look" becomes a context switch, and every privilege question becomes a query written from memory. [ASSUMPTION: this is the core frustration; severity to be validated with the first external tester]

## The Solution

db-adm-tui opens as a profile picker and connects using credentials already on the machine: a PostgreSQL host, port, and database name or prefix with the password resolved through `~/.pgpass` (psql semantics: 0600 permissions, escaping, wildcards), or a MongoDB URI read from a `~/.mongo.*` file (or `~/.mongo`). It also accepts a new no-password connection URI; the password is resolved from the same source files or prompted once. Saved profiles keep only the non-secret connection details and an optional pointer to the source file; the password is re-read at connect time and never stored. The tool then presents one navigable tree of databases, schemas, tables, collections, users, and roles — the same tree across engines.

Three views carry the work:

- **Browse and filter data**, with less-style search on `/`.
- **Inspect effective privileges** in each engine's native terms.
- **Query** on a single line, with `Ctrl+Space` word completion and `Ctrl+F1` context help.

A read-only indicator is always visible. Every connection starts read-only; switching to read-write requires an explicit, confirmed action. In read-only mode the tool blocks write statements in its own input; this is a product guard, not a security boundary against a privileged user. Connections with elevated rights and root launches warn about the added risk. Command history is written under `~/.local/state/db-adm-tui/history` with credentials and password literals redacted, and terminal output is sanitized against ANSI escape injection.

## What Makes This Different

- **On-server first.** Competitors assume a workstation with a GUI or a Python runtime; this one is designed for the host, where the data is.
- **One tree across engines.** The same navigation, object views, and role views for PostgreSQL, MongoDB, and MySQL/MariaDB.
- **Privileges as a first-class view.** The category treats them as a query, not a screen.
- **Deliberately small.** No multi-line SQL IDE, no data-grid editing, no telemetry, no dashboard.

The moat is not technology. It is the combination, and the discipline to stay small.

## Who This Serves

**Personal-first:** v1 succeeds if the author stops reaching for psql and mongosh on the host.

**Primary:** the engineer who is already on the host — an SRE, backend developer, or DB generalist wearing a DBA hat for the moment. They know SQL and MongoDB well enough to ask questions but do not want to recall catalog queries or switch shells. Success is a question answered in seconds without leaving the terminal.

**Secondary:** a DBA or teammate who needs the same view without learning a new tool.

## Success Criteria

- A privilege question is answered in under 30 seconds, down from minutes of catalog queries.
- A week of real on-host work completes without the tool being abandoned out of frustration.
- The first external tester completes two real tasks unaided: find a record, and answer what a given user can do on a given database.
- A credential or password literal never appears in history files, logs, process arguments, or error output.
- The binary runs on a stock hardened host with no runtime installed. [ASSUMPTION: static binary]
- While in read-only mode, the tool blocks write statements in its input; this is a product guard, not a defense against a privileged user.

## Scope

**In v1**

- PostgreSQL 14+, MySQL 8.0+, MariaDB 10.4+, and MongoDB 6.0+ drivers, shipped as a Linux `x86_64`/`arm64` static binary; other Unix-like hosts are best-effort.
- Profiles imported from `~/.pgpass` and `~/.mongo.*` (or `~/.mongo`), or created from a new no-password connection string; passwords resolved at connect time or prompted once, never stored.
- Browse databases, schemas, tables, collections, views, and sequences where present, plus users and roles; data browsing and detailed statistics apply to tables and collections in v1.
- Paged data browsing with a bounded full-fetch mode.
- Object details: estimated count, size, indexes, and structure, with an exact count on demand.
- Read-only privilege inspection in each engine's native terms.
- One logical statement per execution in a query bar that may wrap to three lines, with `Ctrl+Space` completion and `Ctrl+F1` context help; SQL plus a restricted MQL command set (`find`, `aggregate`, `count`, `distinct`); `Ctrl+X Ctrl+E` opens an edit modal; statements run read-only unless read-write mode is enabled.
- Read-only by default for every connection, with an explicit, confirmed transition; elevated-rights and root warnings; ANSI sanitization; redacted, rotating JSONL and TXT history.

**Out of v1**

- Interactive data-grid editing, privilege assignment, a full multi-line SQL IDE, arbitrary JavaScript execution, a GUI, additional engines, telemetry, saved passwords.

**Planned v2**

- Privilege assignment matrix with native command preview, edit, and a separate audit trail; optional data edits.

**Later (v3)**

- Column sorting and data export.

## Vision

If it succeeds, db-adm-tui becomes the default way to answer a database question inside a closed perimeter: installed on the host like htop, trusted because it is small, quiet, and predictable — not a DBeaver replacement, but the terminal-first tool that is already there when a GUI cannot be.

## Open Questions

- **Public release posture:** built for personal use first, kept ready for public release; decide after v1 proves itself in daily use.
- **Blocklist parser coverage:** the PRD defines the blocked statement classes; exact parser coverage is validated in architecture.
