---
name: "db-adm-tui"
description: "Nightshift — a quiet, keyboard-first database instrument for the terminal. One dark theme, semantic color only, no decoration."
status: final
updated: 2026-11-18
colors:
  # Nightshift. Values are fixed for v1. The role names are the contract: a future
  # per-profile theme swaps these values and nothing else (see EXPERIENCE.md → Future/Open Items).
  background-base: '#04060A'
  surface: '#0A0F15'
  surface-raised: '#121A23'
  border-default: '#26333F'
  border-focus: '#3EE0D0'
  border-modal: '#F5B85C'
  text-primary: '#F0F6FA'
  text-secondary: '#94A6B8'
  accent-primary: '#3EE0D0'
  accent-secondary: '#B49BFF'
  success: '#58E6A0'
  warning: '#F5C65F'
  error: '#FF6B7D'
  info: '#6DB5FF'
  match-highlight: '#FFD166'
  selection: '#0F3A44'
  readonly-mode: '#7CC4FF'
  readwrite-mode: '#FF9D4D'
typography:
  # Terminal-native. The tool ships no font and must not attempt to set one.
  body:
    fontFamily: 'Terminal monospace (inherited; the tool sets no font)'
    fontSize: '1 cell'
    fontWeight: 'normal'
    lineHeight: '1 line'
    note: 'Default text. Hierarchy comes from color, weight, position, and borders — not size.'
  strong:
    fontFamily: 'Terminal monospace (inherited; the tool sets no font)'
    fontSize: '1 cell'
    fontWeight: 'bold'
    lineHeight: '1 line'
    note: 'Object names, modal titles, the active tab, and the current value in a detail row.'
  label:
    fontFamily: 'Terminal monospace (inherited; the tool sets no font)'
    fontSize: '1 cell'
    fontWeight: 'bold'
    lineHeight: '1 line'
    note: 'Short uppercase section labels: pane titles, tab labels, column headers, section headers in privileges. Twelve characters or fewer.'
  code:
    fontFamily: 'Terminal monospace (inherited; the tool sets no font)'
    fontSize: '1 cell'
    fontWeight: 'normal'
    lineHeight: '1 line'
    note: 'Query text, data cells, identifiers, and engine messages. Same face as body; the role marks unmodified, copy-exact text.'
rounded:
  none: 0
  DEFAULT: 0
spacing:
  '0': '0 cells'
  '1': '1 cell'
  '2': '2 cells'
  '3': '3 cells'
  '4': '4 cells'
  pane-gap: '1 cell'
  pane-padding: '1 cell'
  row-height: '1 line'
  modal-margin-min: '2 cells'
  modal-margin-max: '3 cells'
  viewport-min-width: '80 cells'
  viewport-min-height: '24 lines'
components:
  top-status-line:
    background: '{colors.surface}'
    foreground: '{colors.text-primary}'
    border: '{colors.border-default}'
    mode-readonly: '{colors.readonly-mode}'
    mode-readwrite: '{colors.readwrite-mode}'
    elevated-warning: '{colors.warning}'
    height: '1 line'
  status-line:
    background: '{colors.surface}'
    foreground: '{colors.text-secondary}'
    key-hint: '{colors.accent-primary}'
    message-info: '{colors.info}'
    message-success: '{colors.success}'
    message-warning: '{colors.warning}'
    message-error: '{colors.error}'
    height: '1 line'
  tree:
    background: '{colors.surface}'
    foreground: '{colors.text-secondary}'
    title: '{colors.text-secondary}'
    border-idle: '{colors.border-default}'
    border-focus: '{colors.border-focus}'
    node-expanded: '{colors.accent-primary}'
    row-selected-background: '{colors.selection}'
    row-selected-foreground: '{colors.text-primary}'
    access-marker: '{colors.text-secondary}'
    security-node: '{colors.info}'
    error: '{colors.error}'
  object-tabs:
    background: '{colors.surface}'
    active: '{colors.accent-primary}'
    inactive: '{colors.text-secondary}'
    separator: '{colors.border-default}'
    height: '1 line'
  details-pane:
    background: '{colors.surface}'
    foreground: '{colors.text-primary}'
    label: '{colors.text-secondary}'
    value: '{colors.text-primary}'
    unavailable: '{colors.text-secondary}'
    exact: '{colors.success}'
    running: '{colors.info}'
    error: '{colors.error}'
    border-idle: '{colors.border-default}'
    border-focus: '{colors.border-focus}'
  privileges-pane:
    background: '{colors.surface}'
    foreground: '{colors.text-primary}'
    section-label: '{colors.text-secondary}'
    native-term: '{colors.accent-secondary}'
    direct: '{colors.text-primary}'
    via-role: '{colors.text-secondary}'
    not-permitted: '{colors.warning}'
    no-privilege: '{colors.text-secondary}'
    error: '{colors.error}'
    border-idle: '{colors.border-default}'
    border-focus: '{colors.border-focus}'
  data-grid:
    background: '{colors.surface}'
    header-background: '{colors.surface-raised}'
    header-foreground: '{colors.text-secondary}'
    cell: '{colors.text-primary}'
    null-value: '{colors.text-secondary}'
    row-selected-background: '{colors.selection}'
    row-selected-foreground: '{colors.text-primary}'
    match-background: '{colors.match-highlight}'
    match-foreground: '{colors.background-base}'
    border: '{colors.border-default}'
  filter-bar:
    background: '{colors.surface}'
    foreground: '{colors.text-primary}'
    prompt: '{colors.accent-primary}'
    hint: '{colors.text-secondary}'
    no-match: '{colors.error}'
    height: '1 line'
  query-bar:
    background: '{colors.surface}'
    foreground: '{colors.text-primary}'
    prompt-readonly: '{colors.readonly-mode}'
    prompt-readwrite: '{colors.readwrite-mode}'
    keyword: '{colors.accent-primary}'
    object-name: '{colors.accent-secondary}'
    string: '{colors.accent-secondary}'
    cursor: '{colors.accent-primary}'
    hint: '{colors.text-secondary}'
    disabled: '{colors.text-secondary}'
    error: '{colors.error}'
    border-idle: '{colors.border-default}'
    border-focus: '{colors.border-focus}'
  completion-popup:
    background: '{colors.surface-raised}'
    foreground: '{colors.text-primary}'
    selected-background: '{colors.selection}'
    selected-foreground: '{colors.text-primary}'
    kind-label: '{colors.text-secondary}'
    border: '{colors.border-default}'
  results-view:
    background: '{colors.surface}'
    foreground: '{colors.text-primary}'
    banner: '{colors.info}'
    running: '{colors.info}'
    error: '{colors.error}'
    error-statement: '{colors.text-primary}'
    empty: '{colors.text-secondary}'
    border-idle: '{colors.border-default}'
    border-focus: '{colors.border-focus}'
  profile-picker:
    background: '{colors.background-base}'
    row: '{colors.text-primary}'
    label: '{colors.text-primary}'
    meta: '{colors.text-secondary}'
    engine: '{colors.accent-secondary}'
    selected-background: '{colors.selection}'
    selected-foreground: '{colors.text-primary}'
    error: '{colors.error}'
    empty: '{colors.text-secondary}'
    border-idle: '{colors.border-default}'
    border-focus: '{colors.border-focus}'
  new-connection-modal:
    background: '{colors.surface-raised}'
    foreground: '{colors.text-primary}'
    label: '{colors.text-secondary}'
    input: '{colors.text-primary}'
    uri: '{colors.text-primary}'
    hint: '{colors.text-secondary}'
    error: '{colors.error}'
    border: '{colors.border-modal}'
  import-modal:
    background: '{colors.surface-raised}'
    foreground: '{colors.text-primary}'
    row: '{colors.text-primary}'
    meta: '{colors.text-secondary}'
    selected-background: '{colors.selection}'
    selected-foreground: '{colors.text-primary}'
    malformed-warning: '{colors.warning}'
    empty: '{colors.text-secondary}'
    border: '{colors.border-modal}'
  confirm-modal:
    background: '{colors.surface-raised}'
    foreground: '{colors.text-primary}'
    title-warning: '{colors.warning}'
    title-readwrite: '{colors.readwrite-mode}'
    body: '{colors.text-secondary}'
    emphasis: '{colors.text-primary}'
    primary-action: '{colors.readwrite-mode}'
    cancel: '{colors.text-secondary}'
    border: '{colors.border-modal}'
  help-overlay:
    background: '{colors.surface-raised}'
    foreground: '{colors.text-primary}'
    heading: '{colors.accent-primary}'
    key: '{colors.accent-primary}'
    body: '{colors.text-primary}'
    muted: '{colors.text-secondary}'
    border: '{colors.border-modal}'
  edit-modal:
    background: '{colors.surface-raised}'
    foreground: '{colors.text-primary}'
    statement: '{colors.text-primary}'
    keyword: '{colors.accent-primary}'
    object-name: '{colors.accent-secondary}'
    cursor: '{colors.accent-primary}'
    hint: '{colors.text-secondary}'
    error: '{colors.error}'
    border: '{colors.border-modal}'
---

## Brand & Style

db-adm-tui is a quiet instrument, not a dashboard. It lives on a database host at the end of an SSH session, next to `htop` and `tail`, and it should feel like it belongs there: near-black, unhurried, legible at a glance, and visibly under the operator's control. Nightshift is the register — sharp and clinical, with near-black depth and one ice-teal signal for the next action. There is no brand mascot, no empty-state illustration, no decorative color. The interface is made of text, borders, and a small number of honest signals.

The posture is *operator's tool*: dense enough to respect the user's time, sparse enough that nothing competes with the data. The tree is the map. The main pane is the work. The two status lines are the instrument panel — the top line says what you are connected to and whether writes are allowed; the bottom line says what just happened and which keys apply. Everything else stays out of the way.

Visual references: `mockups/key-picker.html`, `mockups/key-workspace.html`, and `mockups/key-overlays.html` (Nightshift key-screen studies, 80×24); the color study is `.working/color-themes-1.html`. These are illustration, not layout contracts; where they differ from these spines, the spines win. The color study's clock, viewport counters (`2 db`, `3 cols`), and sample data are illustrative only.

This is a TUI. It never sets a font, never draws an image, never animates, and never assumes a terminal wider than 80 columns. Its visual identity is carried by color roles, one border weight, alignment, and short labels.

## Colors

Eighteen semantic roles, one dark theme. Every value is taken from the Nightshift study. Roles are stable; the values are what a future per-profile theme may replace. Color is used to separate surfaces, mark focus, and name a state — never to decorate, and never as the only carrier of meaning.

**Surfaces.** `{colors.background-base}` is the canvas: the profile picker, the gaps between panes, and the empty field behind a modal. `{colors.surface}` is every working pane and both status lines. `{colors.surface-raised}` is one step above the work: modal bodies, data-grid headers, and the completion popup. The steps are deliberately few; depth in a terminal comes from borders, not from a fourth shade.

**Borders.** `{colors.border-default}` is structure at 1.5:1 against `{colors.surface}` — visible as a frame, never asked to carry information by itself. `{colors.border-focus}` is the focus signal and the accent of last resort; it appears on the one pane that owns the keyboard and on the active selection marker. `{colors.border-modal}` is amber and reserved for modals: the only surface that interrupts the work.

**Text.** `{colors.text-primary}` is data, values, and object names. `{colors.text-secondary}` is labels, metadata, hints, and the tree's resting state. There is no third neutral text color; "dim" is the terminal's dim attribute on `{colors.text-secondary}`, with the plain color as fallback.

**Accents.** `{colors.accent-primary}` (ice teal) means *the next action*: the focused border, the active tab, the query prompt, the selected tree node's chevron, the key hints in the status line. `{colors.accent-secondary}` (violet) marks engine-native names — schemas in a path, object identifiers in a statement, native privilege terms. Two accents, two jobs, no exceptions.

**Status.** `{colors.success}`, `{colors.warning}`, `{colors.error}`, and `{colors.info}` are always paired with a word: `exact`, `warning`, `error`, `running`. They are never applied to a whole row, never used as fills, and never the sole difference between two states.

**View state.** `{colors.match-highlight}` is a fill behind matched text inside the data grid, with `{colors.background-base}` as the foreground. `{colors.selection}` is the selected-row fill; the selected row's text becomes `{colors.text-primary}`. `{colors.readonly-mode}` is the read-only mode indicator; `{colors.readwrite-mode}` is the read-write indicator and the confirmation action. They are different hues on purpose: blue is the calm default, orange says the guard is off.

Load-bearing combinations, measured:

| Pair | Ratio | Where |
|---|---|---|
| `{colors.text-primary}` on `{colors.background-base}` | 18.6:1 | Base text, picker |
| `{colors.text-secondary}` on `{colors.surface}` | 7.7:1 | Metadata, hints, labels |
| `{colors.text-primary}` on `{colors.selection}` | 11.3:1 | Selected row |
| `{colors.background-base}` on `{colors.match-highlight}` | 14.1:1 | Filter match |
| `{colors.readonly-mode}` on `{colors.surface}` | 10.3:1 | r/o indicator |
| `{colors.readwrite-mode}` on `{colors.surface}` | 9.3:1 | r/w indicator, confirm action |
| `{colors.error}` on `{colors.surface}` | 7.0:1 | Errors, blocked statements |
| `{colors.error}` on `{colors.surface-raised}` | 6.4:1 | Short error lines inside modals |
| `{colors.warning}` on `{colors.surface}` | 12.0:1 | Elevated-rights, malformed lines |
| `{colors.border-focus}` on `{colors.background-base}` | 12.3:1 | Focus border |

`{colors.border-default}` at 1.5:1 is structural only. Do not put text in it, and do not rely on it to tell two states apart.

## Typography

The terminal owns the font. db-adm-tui inherits the user's configured monospace face and never ships, selects, or styles a family. Type roles are cell styles, not sizes: `{typography.body}`, `{typography.strong}`, `{typography.label}`, and `{typography.code}` all occupy one cell per character and one line per row.

Hierarchy is built from weight, color, position, and borders. There are no font sizes to escalate with and no italics to rely on. `{typography.strong}` is bold and used for object names, modal titles, the active tab, and the current value in a detail row. `{typography.label}` is short uppercase text for pane titles, tab labels, column headers, and privilege section headers — twelve characters or fewer, never a sentence. `{typography.code}` is the same face as body but marks text that must be read exactly: query text, data cells, identifiers, and engine messages.

Line length is bounded by the layout, not by a measure: prose fields (help, errors, confirmations) wrap at the pane's inner width, capped at 72 cells, and are written to fit 48 characters without truncation. Long identifiers truncate with `…` at the end, never in the middle, except in a query bar where the cursor position wins and the line scrolls horizontally.

No letter-spacing token exists because cells cannot be tracked. No line-height token beyond `{spacing.row-height}` exists because a row is a line.

## Layout & Spacing

The workspace is a four-band vertical stack that fills the terminal:

```
┌ top status line ─────────────────────────────────────────── 1 line ─┐
│                                                                     │
│  tree pane                    │  main pane                          │  body
│  (default 28 cells)           │  (tab strip + active tab)           │  (fills)
│                               │                                     │
├ query bar ──────────────────────────────────────────────── 1–3 lines ┤
└ status line ─────────────────────────────────────────────── 1 line ─┘
```

At the 80×24 floor: one line for the top status line, twenty-one for the body, one for the query bar, one for the bottom status line. The body splits into a 28-cell tree pane, a one-cell gap, and a 51-cell main pane. The main pane spends its first line on the tab strip; the active tab gets the rest. A filter bar, when present, takes one line at the top of the data grid.

The tree width is adjustable with `Ctrl+Left` / `Ctrl+Right` and persists per profile. The query bar height is adjustable with `Ctrl+Up` / `Ctrl+Down` and persists per profile. [ASSUMPTION: the default tree width is 28 cells at 80 columns and stays fixed in cells as the terminal grows; the clamp runs from 16 cells to half the terminal width. The query bar is 1 line by default and clamps at 1–3 lines; extra height reveals additional wrapped segments of the same logical statement and never accepts a second statement. The resize step is 2 cells horizontally and 1 line vertically.]

Panes are separated by `{spacing.pane-gap}` and content is inset by `{spacing.pane-padding}`. Rows are exactly `{spacing.row-height}`; there is no leading inside a row and no vertical padding inside lists. Borders are single-line and occupy cells like any other glyph.

Modals are centered, never full-screen, and keep `{spacing.modal-margin-min}` to `{spacing.modal-margin-max}` cells of clear space from every terminal edge. A modal is at most 76 cells wide; confirmation and connection modals stay within 60. Long error text wraps inside it rather than widening it. One modal at a time. [ASSUMPTION: the modal is horizontally centered with equal margins, and its height is content + two border lines. A background-dim treatment is optional and terminal-dependent.]

The 80×24 arithmetic is a hard constraint, not a guideline. Any new label, column, or badge must either fit in the cells named here or replace something that does. The `…` truncation ladder is defined in EXPERIENCE.md → Responsive & Platform.

## Elevation & Depth

There are three tonal levels and no shadows. `{colors.background-base}` sits behind everything; `{colors.surface}` is the working layer; `{colors.surface-raised}` is the interrupting layer. A modal is legible because it steps up one tone and takes the amber `{colors.border-modal}` frame — not because of a shadow, a glow, or transparency.

Focus is the only other depth cue. The pane that owns the keyboard draws its frame in `{colors.border-focus}` and sets its title in bold `{colors.accent-primary}`; every other pane uses `{colors.border-default}` and a plain `{colors.text-secondary}` title, dim where available. When a modal is open, the workspace behind it is inactive: its borders return to `{colors.border-default}`, dim where supported, and its titles drop to plain secondary.

Stacking is flat and total: exactly one modal may be open. No popover over a modal, no nested confirmation, no help on top of a confirmation. The completion popup is the one exception — it is a popup, not a modal, and it belongs to the focused input.

## Shapes

Zero radius. Terminal cells are square, and the interface accepts that: `{rounded.none}` on every pane, modal, button, badge, and popup. There are no rounded corners, no pills, no circles, and no filled panels.

Frames use single-line box drawing: `┌ ┐ └ ┘ ─ │ ├ ┤`. Corners are square, joins are single, and no double-line or heavy-border style is used anywhere. Tabs are labels, not boxes: the active tab is bold, `{colors.accent-primary}`, and underlined with the terminal's underline attribute on the same row — never a second glyph row. The selected row is a one-line fill of `{colors.selection}` spanning the pane's inner width. A cursor is a filled cell, not a bar glyph.

The working glyph set is small and fixed: `▸` collapsed, `▾` expanded, `·` leaf, `—` no access, `›` query prompt, `…` truncation. The access marker `—` is an em dash and reads as a marker, not a minus sign. [ASSUMPTION: this glyph set and single-line border style were not decided in discovery; UTF-8 is required, and no ASCII fallback mode is in scope for v1.]

## Components

Seventeen components carry the whole product. Behavioral rules live in EXPERIENCE.md → Component Patterns; this section fixes their visual anatomy.

- **top-status-line** — One line, `{colors.surface}` fill, `{colors.border-default}` bottom rule. Left: profile label in `{typography.strong}`, then `·` separators and engine, database, and the mode badge (`r/o` in `{colors.readonly-mode}` or `r/w` in `{colors.readwrite-mode}`). Right: connected user, then a `privileged` tag in `{colors.warning}` when elevated rights are detected. Never shows a password or a full URI.
- **status-line** — One line, `{colors.surface}` fill, `{colors.border-default}` top rule. Left: the current message, colored by kind (`{colors.info}`, `{colors.success}`, `{colors.warning}`, `{colors.error}`) and always containing the state word. Right: context key hints in `{colors.text-secondary}` with the key itself in `{colors.accent-primary}`. Counts and mode/case indicators live here when the data view is active (FR-17).
- **tree** — Framed pane on `{colors.surface}`. Title `objects` in `{typography.label}` / `{colors.text-secondary}`; count on the right, same style. Rows are one line, indented two cells per depth. Expanded node chevrons and the active path use `{colors.accent-primary}`; names use `{colors.text-secondary}` at rest. The selected row is `{colors.selection}` fill with `{colors.text-primary}` text. The `Security` node and its children use `{colors.info}`. Access markers `ro` / `rw` / `—` sit after table, collection, and view names in `{colors.text-secondary}`; they are never dimmed, so they keep the 7.7:1 pair. Sequences carry none. On a selected row every character, including the access marker, switches to `{colors.text-primary}`. A node's inline error starts with the word `error` in `{colors.error}`, indented under the node.
- **object-tabs** — One line at the top of the main pane, `{colors.surface}` fill. Labels are `{typography.label}` with a leading number (`1 Details`). Active label is `{colors.accent-primary}` bold with the terminal underline attribute; inactive labels are `{colors.text-secondary}`; separators are `{colors.border-default}`. Tab set changes with the selected object's type; no disabled or greyed tabs are drawn — absent tabs are absent.
- **details-pane** — `{colors.surface}` fill, `{colors.border-default}` or `{colors.border-focus}` frame. Two-column rows: label in `{colors.text-secondary}`, value in `{colors.text-primary}`. Numeric values are right-aligned in their column. `estimated` and `exact` are trailing words in `{colors.text-secondary}` and `{colors.success}`; `unavailable` is `{colors.text-secondary}`, never `0`. A running count shows elapsed text in `{colors.info}`. An inline error starts with the word `error`. Section headers (`structure`, `indexes`) are `{typography.label}`.
- **privileges-pane** — Same frame and row rhythm as details-pane. Sections (`grants`, `memberships`, `effective`) use `{colors.text-secondary}` headers. Native terms (`CONNECT`, `USAGE`, `SELECT`, `INHERIT FALSE`, `user@host`) are `{colors.accent-secondary}`; `direct` grants use `{colors.text-primary}`, `via role` grants use `{colors.text-secondary}`, `not permitted` uses `{colors.warning}` with the word present, and `no privilege` is plain `{colors.text-secondary}`. A failed section starts with the word `error`.
- **data-grid** — Header row on `{colors.surface-raised}` with `{typography.label}` / `{colors.text-secondary}` column names. Cells are `{typography.code}` / `{colors.text-primary}`; null renders as `null` in `{colors.text-secondary}`, an empty string renders as nothing. The selected row is `{colors.selection}` fill. Matches are `{colors.match-highlight}` fill with `{colors.background-base}` text. Column separators are not drawn; a `{colors.border-default}` rule closes the header. Wide tables scroll horizontally; the row-number gutter stays pinned.
- **filter-bar** — One line between the tab strip and the header when a buffer filter is active. `/` prompt in `{colors.accent-primary}`, pattern in `{colors.text-primary}`, count and `Esc clear` hint in `{colors.text-secondary}`. A no-match state turns the count `{colors.error}` but leaves the pattern editable.
- **query-bar** — Full-width band on `{colors.surface}` with a `{colors.border-default}` top rule; gains `{colors.border-focus}` when focused. Prompt is `SQL` or `MQL` in `{colors.readonly-mode}` or `{colors.readwrite-mode}`, followed by `›`. Statement text is `{typography.code}`: keywords `{colors.accent-primary}`, identifiers `{colors.accent-secondary}`, strings `{colors.accent-secondary}`, literals `{colors.text-primary}`. The cursor is a filled cell in `{colors.accent-primary}`. Right-aligned hint `Ctrl+X Ctrl+X run` in `{colors.text-secondary}`. Disabled state uses `{colors.text-secondary}` for the whole bar and shows the reason.
- **completion-popup** — `{colors.surface-raised}` panel with `{colors.border-default}` frame, anchored above or below the cursor line, never wider than 40 cells. Suggestions are one line each; the highlighted one uses `{colors.selection}` fill. A right-aligned kind label (`keyword`, `table`, `column`, `field`) is `{colors.text-secondary}`. Maximum 8 visible suggestions with a count line; `PgUp`/`PgDn` scroll the list.
- **results-view** — Occupies the main pane like a tab. Result rows reuse data-grid anatomy exactly; the tab strip shows `1 Results` in `{colors.accent-primary}`. A filtered banner is one line in `{colors.info}`. A running query shows elapsed time in `{colors.info}` with `Esc cancels` in `{colors.text-secondary}`. An error block starts with an `error` header line in `{colors.error}`, the preserved statement in `{colors.text-primary}` on `{colors.surface-raised}`, and the engine message in `{colors.error}`, wrapped at the pane's inner width and capped at 72 cells.
- **profile-picker** — Full-screen on `{colors.background-base}`. Title `db-adm-tui` in `{typography.strong}`; subtitle `choose a connection` in `{colors.text-secondary}`. Rows show label, engine, host, user, source file, and last used; label `{colors.text-primary}`, engine `{colors.accent-secondary}`, the rest `{colors.text-secondary}`. Selected row is `{colors.selection}` fill. An inline error sits under its row in `{colors.error}`. Footer keys (`n new`, `i import`, `? help`, `q quit`) use the status-line hint style. The profile list scrolls inside the viewport above the footer, with a count line. The `border-focus` color is used only on the focused row's leading marker.
- **new-connection-modal** — Centered `{colors.surface-raised}` panel with `{colors.border-modal}` frame. Title `new connection` in `{typography.strong}`. One URI input line prefixed `uri` (`{colors.text-secondary}` label, `{colors.text-primary}` value) and one optional `label` input. Hint line explains password resolution in `{colors.text-secondary}`. Reject `user:pass@` in the URI; the password comes from a source file or a one-time prompt. Errors appear above the actions in `{colors.error}`. Actions `Enter connect` / `Esc cancel` in the confirm-modal action style.
- **import-modal** — Centered `{colors.surface-raised}` panel with `{colors.border-modal}` frame. Title `import` plus the number of discovered entries. Rows are one line: a `☐`/`☑` selection cell, source file and selector, engine, and user; passwords never appear. Selected rows use `{colors.selection}`. A malformed line carries a `warning` tag in `{colors.warning}`. The list scrolls inside a viewport capped at terminal height minus margins, with a count line. Footer: `Space select · Enter import · Esc cancel`.
- **confirm-modal** — Centered `{colors.surface-raised}` panel with `{colors.border-modal}` frame. Title in `{colors.warning}` or `{colors.readwrite-mode}` depending on consequence. Body in `{colors.text-secondary}` with the affected label in `{colors.text-primary}` bold. Actions right-aligned: `Esc cancel` in `{colors.text-secondary}`, primary action in `{colors.readwrite-mode}`. Width 40–60 cells.
- **help-overlay** — Centered `{colors.surface-raised}` panel with `{colors.border-modal}` frame. Section headings in `{colors.accent-primary}` and `{typography.label}`; keys in `{colors.accent-primary}`, descriptions in `{colors.text-primary}`. Scrolls vertically with `Up`/`Down` when taller than the terminal; `Esc closes` pinned at the bottom. Same anatomy serves the keymap (`?`) and context help (`Ctrl+F1`).
- **edit-modal** — Centered `{colors.surface-raised}` panel with `{colors.border-modal}` frame, 60–76 cells wide and never taller than the terminal minus twice `{spacing.modal-margin-min}`. Title `edit statement` in `{typography.strong}`. The statement is one logical line in `{typography.code}` with the same keyword, identifier, and string colors as the query bar; it scrolls horizontally rather than becoming a multi-line editor. Cursor is a filled cell in `{colors.accent-primary}`. Footer hints `Enter run · Esc cancel · Ctrl+Space complete` in `{colors.text-secondary}`. Errors replace the hint line in `{colors.error}` and keep the statement intact.

## Do's and Don'ts

| Do | Don't |
|---|---|
| Let one ice-teal signal mark the next action: focus border, active tab, prompt | Use `{colors.accent-primary}` for decoration, headings, or status text |
| Pair every color-coded state with a word (`r/o`, `error`, `exact`, `not permitted`) | Let color alone distinguish read-only from read-write, or success from error |
| Keep panes flat: `{colors.surface}` inside `{colors.background-base}` with a one-cell border | Add drop shadows, glows, gradients, or transparency |
| Use square corners and single-line box drawing | Round a corner, double a border, or fill a panel with an accent |
| Reserve amber `{colors.border-modal}` for modals and nothing else | Color a focused pane amber — focus is teal |
| Right-align numbers and keep one value per row | Add sparklines, bars, gauges, or badges to the details pane |
| Truncate long identifiers with `…` at the end | Wrap a tree row or a data cell onto a second line |
| Keep the 80×24 arithmetic intact before adding any label | Let a new badge push the status line into a second row |
| Use `{colors.text-secondary}` for metadata and access markers; reserve dim for non-essential chrome | Dim an access marker below the contrast floor, or introduce a third grey |
| Keep the tab strip to labels and numbers | Draw tab boxes, arrows, or scroll buttons |
