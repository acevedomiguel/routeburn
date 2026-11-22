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
╭─┤ routeburn ├──────────────────────────────────────────── 15:42 ─╮
│                                                                  │
│ ╭─┤ Credits ├─────────────────╮ ╭─┤ Last 30 days ├────────────╮  │
│ │ Remaining ········· $18.43  │ │ Spend ············· $5.13  │  │
│ │ Runway ············ 195 days│ │ Tokens ··········· 195.9M  │  │
│ │ Daily burn ········ $0.09   │ │ Models ·············· 2    │  │
│ ╰─────────────────────────────╯ ╰────────────────────────────╯  │
│ ╭─┤ Daily credit spend ├────────────────────────────────────╮   │
│ │ ▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▃▂▅█   peak $2.30 · Nov 21      │   │
│ ╰──────────────────────────────────────────────────────────╯   │
│ ╭─┤ Cost by model ├─────────────────────────────────────────╮   │
│ │ deepseek-v4.1-flash ·········· $2.61 ▇▇▇▇▇▇▇▇▇▇  52%     │   │
│ │ kimi-k3.1-pro ··············· $2.52 ▇▇▇▇▇▇▇▇▇  50%        │   │
│ ╰──────────────────────────────────────────────────────────╯   │
╰──────────────────────────────────────────────────────────────────╯
 ↑↓ scroll  r refresh  ? help  q quit                       0/1
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
- the sparkline is a bordered panel and grows with the terminal, never below
  three rows (top border, one row of blocks, bottom border).

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

Note on the footer: the body is fitted to the frame at every supported size, so
pagination dots are unreachable in practice. The dots code stays as a safety net
for a terminal shorter than the 16-row minimum.
