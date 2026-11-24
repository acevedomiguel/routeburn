# RouteBurn UI redesign — btop as the reference

Status: implemented. Reference screenshot supplied by the user 2026-11-21
(`btop` running on macOS).

## What makes btop look good

1. **Bordered panels, not floating cards.** Every section is a box, and the section
   title is embedded *in the top border* (`╭─┤ cpu ├────────────`). The frame does the
   grouping work, so no card needs a background colour or a shadow.
2. **Dot leaders.** `Label ······ value` — the leader is what makes a terminal table
   look deliberate. It also solves the current ragged two-column card problem for free.
3. **Right-aligned values in a column.** Numbers line up digit-for-digit, so magnitudes
   are comparable at a glance.
4. **Inline meters.** `▇▇▇▇▇▇░░░░░` sits on the same row as the number it describes,
   with the percentage at the row end. No separate bar block.
5. **Bars in the header and footer.** The top line is a status strip (title left, clock
   middle, vitals right); the bottom line is a key strip with arrow glyphs and a counter
   on the far right. Both are full-width and flush to the edge.
6. **Colour as signal only.** Cyan = structure/titles, green = healthy, amber = warning,
   white = values. Nothing is coloured decoratively.

## Mapped onto RouteBurn

```
╭─ Credits ──────────────────────────── Updated 23:42 ─╮ ╭─ Daily spend ─────────────────────────────────────────╮
│ Remaining ··································· $42.10 │ │  DATE                                 SPEND   TOKENS  │
│ Runway ···························· 102 days ~Mar 03 │ │  Nov 21       ▇▇▇▇▇▇░░░░░░░░░░░░░░    $0.33     231K  │
│ Daily burn ······························· $0.41/day │ │  Nov 20       ▇▇▇▇▇▇▇▇░░░░░░░░░░░░    $0.50     350K  │
╰──────────────────────────────────────────────────────╯ │  Nov 19       ▇▇▇▇░░░░░░░░░░░░░░░░    $0.25     175K  │
                                                         │  Nov 18       ▇▇▇░░░░░░░░░░░░░░░░░    $0.15     105K  │
╭─ Last 30 days ───────────────────────────────────────╮ │  Nov 17       ▇▇▇▇▇░░░░░░░░░░░░░░░    $0.30     210K  │
│ Spend ······································· $11.29 │ │  Nov 16       ▇▇▇▇▇▇▇▇░░░░░░░░░░░░    $0.45     315K  │
│ Requests ····································· 1,240 │ │  Nov 15       ▇▇▇▇▇▇▇▇▇▇░░░░░░░░░░    $0.60     420K  │
│ Tokens ········································ 8.4M │ │  Nov 14       ▇▇▇░░░░░░░░░░░░░░░░░    $0.20     140K  │
│ Cache hit ···································· 18.4% │ │  Nov 13       ▇▇▇▇▇▇░░░░░░░░░░░░░░    $0.35     245K  │
╰──────────────────────────────────────────────────────╯ │  Nov 12       ▇▇▇▇▇▇▇░░░░░░░░░░░░░    $0.40     280K  │
                                                         │  Nov 11       ▇▇▇▇▇▇▇▇▇░░░░░░░░░░░    $0.55     385K  │
╭─ Tokens ─────────────────────────────────────────────╮ │  Nov 10       ▇▇▇▇░░░░░░░░░░░░░░░░    $0.24     168K  │
│ Input ·············· 7.1M  Output ············· 1.3M │ │  Nov 09       ▇▇▇░░░░░░░░░░░░░░░░░    $0.18     126K  │
│ Cached ············· 1.5M  Reasoning ·········· 480K │ │  Nov 08       ░░░░░░░░░░░░░░░░░░░░    $0.00        0  │
╰──────────────────────────────────────────────────────╯ │  Nov 07       ▇▇▇▇▇░░░░░░░░░░░░░░░    $0.31     217K  │
                                                         │  Nov 06       ▇▇▇▇▇▇▇▇▇▇▇▇▇░░░░░░░    $0.77     539K  │
╭─ Cost by model ──────────────────────────────────────╮ │  Nov 05       ▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇    $1.20     840K  │
│ claude-sonnet-4 ··············· $7.21 ▇▇▇▇▇▇░░░░ 58% │ │  Nov 04       ▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇░░░░░    $0.90     630K  │
│ gpt-5 ························· $3.82 ▇▇▇░░░░░░░ 31% │ │  Nov 03       ▇▇▇▇▇░░░░░░░░░░░░░░░    $0.27     189K  │
│ llama-4-70b ··················· $1.31 ▇░░░░░░░░░ 11% │ │  Nov 02       ▇▇▇▇▇░░░░░░░░░░░░░░░    $0.31     217K  │
╰──────────────────────────────────────────────────────╯ │  Nov 01       ▇▇▇▇▇▇▇░░░░░░░░░░░░░    $0.42     294K  │
                                                         │  Oct 31       ▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇░░░░░    $0.88     616K  │
                                                         │  Oct 30       ▇▇▇▇▇▇▇▇▇▇░░░░░░░░░░    $0.61     427K  │
                                                         │  Oct 29       ░░░░░░░░░░░░░░░░░░░░    $0.00        0  │
                                                         │  Oct 28       ▇░░░░░░░░░░░░░░░░░░░    $0.05      35K  │
                                                         │  Oct 27       ▇▇▇░░░░░░░░░░░░░░░░░    $0.20     140K  │
                                                         │  Oct 26       ▇▇▇▇▇▇▇░░░░░░░░░░░░░    $0.44     308K  │
                                                         │  Oct 25       ▇▇▇▇▇░░░░░░░░░░░░░░░    $0.31     217K  │
                                                         │  Oct 24       ▇▇░░░░░░░░░░░░░░░░░░    $0.12      84K  │
                                                         │  Oct 23       ░░░░░░░░░░░░░░░░░░░░    $0.00        0  │
                                                         │                                                       │
                                                         │                                                       │
                                                         │ ───────────────────────────────────────────────────── │
                                                         │  30 days                             $11.29     7.9M  │
                                                         ╰───────────────────────────────────────────────────────╯
 [r] refresh  [↑↓] scroll  [tab] table  [?] help  [q] quit                                                          
```

## Concrete changes

| Area | Now | After |
| --- | --- | --- |
| Framing | bare text, no boxes | `panel(title, body)` helper: rounded border, title inlaid in the top border, cyan |
| Card rows | `Label` on one line, `Value` on the next | `leader(label, value)` helper: dots between, value right-aligned, both same row |
| Column grid | two columns of different height, ragged | equal-height columns; panels sized to content, grid fills the frame |
| Empty space | ~10 filler rows above the footer | panels stretch to fill the frame; slack absorbed inside the last panel, not left blank |
| Sparkline | blocks only | blocks + right-aligned `peak $2.30 · Nov 21` caption on the same row |
| Model table | full-slug names, spend right-aligned far off | short display name (strip `vendor/` and trailing date), dot leader, inline proportional meter, percentage at row end |
| Model names | `deepseek/deepseek-v4.1-flash-20260910` | `deepseek-v4.1-flash` — `internal/domain` already has the vendor/date; add `ShortModelName` |
| Header | title left, `Updated 15:42` right, void between | status strip: `╭─┤ routeburn ├──` left, spinner/updated right, framed by the outer box |
| Footer | bubbles help bar | keep it (it is already good), switch key glyphs to `↑↓`, add right-aligned page counter |
| Palette | cyan accent + green bar | unchanged, but borders get cyan and need a dimmer variant for the outer frame |

## Constraints that must not break

`internal/ui/dashboard_test.go` is the contract:

- exactly `Height` lines for every width 60..140 × height 16..60;
- footer on the last rendered row;
- no line exceeds the terminal width;
- `--no-color` output contains zero escape sequences;
- the daily panel is a bordered table, never below six rows (two borders, a
  column header, one day, a separator, and the pinned total row); on wide
  terminals it fills a full-height right column beside the sidebar.

`internal/ui/columns_test.go` requires model spend values (`$7.21`, `$3.82`, `$1.31`)
to stay visible at 100/110/120/140 columns.

## Shipped

1. `panel()` and `leader()` helpers in `internal/ui/panel.go`, with tests.
2. Outer frame plus header strip rebuilt; the bubbles help bar kept as the footer.
3. Credits and last-30 as two equal-height leader panels, side by side at every
   usable width (stacking costs six rows and loses the model panel at 60x16).
4. Sparkline caption on the same row as the blocks; the chart grows with the
   terminal instead of staying at three rows.
5. `ShortModelName` in `internal/domain` plus a model panel with inline meters.
6. Full suite green; renders captured at 60x20, 72x24, 80x30, 100x40 and 140x50.
7. The model panel and the token split share a row, model on the left and the
   split on the right (issue #12). On a 26-row body or taller the two panels are
   padded to the same height so their borders end together; below that the split
   is dropped and the model panel keeps the full width.

Note on the footer: the body is fitted to the frame at every supported size, so
pagination dots are unreachable in practice. The dots code stays as a safety net
for a terminal shorter than the 16-row minimum.
