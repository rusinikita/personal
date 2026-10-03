# Web UI Design System - Complete Specification

## Overview

Shared visual language and reusable Go `html/template` building blocks for the *new* `/web/*` dashboard pages (Money, Progress browse view, Workouts, Achievements) plus `/money/import`. Provides one page shell (header/nav/footer), a data table component, summary/stat tiles, a line chart, a bar chart, a combo chart, a dual-axis line chart, a month-grid calendar, a drill-down/detail layout, and an achievement-progress tile grid, all themed consistently (light + dark) from a single place.

Built on **Pico CSS** (classless CSS framework, loaded via CDN) for base typography/layout/forms, plus a small hand-written CSS layer on top for the pieces Pico doesn't cover (nav active state, stat tile emphasis, chart container). Charts are rendered with **Chart.js** (also CDN), fed by server-rendered JSON data — no hand-rolled SVG/canvas math to maintain.

This is a **presentation-only, infrastructure layer**: it owns no database tables and no MCP tools. It exposes exactly one HTTP route of its own — a **demo/style-guide page** (`GET /web/design-system`) showing every component with sample data, so the design language can be reviewed and iterated on before any of the real dashboards (Money, Progress browse, Workouts, Achievements) exist. All other pages are built by other actions' HTTP handlers (`action/money`, `action/progress`, `action/workout`, `action/achievements`), which import this package and call it to render their pages.

**Explicitly out of scope / untouched:** `action/progress/dashboard_web.go` (`/web/progress`) is a separate, purpose-built black-and-white fixed-viewport page for screenshot/e-ink display. It does not use this design system and must not be modified as part of this work.

**Replaces:** the hand-rolled inline `<style>` blocks in `action/money/import_web.go` (`importFormHTML`) — that page's form is rewired to use the shared shell, but its form logic/handler behavior is unchanged.

**Local preview:** `make preview-webui` (`cmd/webui-preview`) serves every `/web/*` route against a reusable Postgres testcontainer — restored on each start from a cached snapshot of migrations + fixture data — using the real `db.NewRepository`, so pages can be eyeballed (and forms submitted) without the production database or Telegram credentials. See "Local Preview" below.

## Architecture Diagrams

### Entity Relation Diagram

Not applicable — this is a presentation layer with no database tables of its own. It renders data owned by the Money, Progress, and Workout subdomains.

### C4 Context Diagram

```mermaid
graph TB
    Browser[Human user / browser]

    subgraph "CDN (static assets, no server dependency)"
        Pico[Pico CSS]
        ChartJS[Chart.js]
    end

    subgraph "action/webui (design system)"
        Demo[GET /web/design-system<br/>demo/style-guide handler — OWN route]
        Shell[Page shell template<br/>header/nav/footer]
        Table[Table component]
        Stat[Stat tile component]
        Line[Line chart component<br/>JSON data + Chart.js init script]
        Bar[Bar chart component<br/>JSON data + Chart.js init script]
        Combo[Combo chart component<br/>bar+line mixed dataset, Chart.js init script]
        Dual[Dual-axis line chart component<br/>two series, left + right Y axis, Chart.js init script]
        Detail[Drill-down/detail layout]
        Tile[Achievement tile grid component<br/>Pico native progress bar, no Chart.js]
    end

    subgraph "Consuming HTTP handlers"
        MoneyH[action/money handlers]
        ProgressH[action/progress handlers<br/>NEW browse view only]
        WorkoutH[action/workout handlers]
        ImportH[action/money import_web.go]
        AchievementsH[action/achievements handlers<br/>+ BuildAchievementTiles helper, called by<br/>MoneyH/ProgressH/WorkoutH for their own embedded tiles]
    end

    Browser -->|GET /web/design-system| Demo
    Browser -->|GET /web/money, /web/workouts, /web/progress/browse, /money/import| MoneyH
    Browser --> ProgressH
    Browser --> WorkoutH
    Browser --> ImportH
    Browser -->|GET /web/achievements| AchievementsH
    Browser -->|loads via link/script tag in shell head| Pico
    Browser --> ChartJS

    Demo -->|renders fixture data through every component| Shell
    MoneyH -->|builds Table/StatTile/BarChart/DetailView data,<br/>calls webui.Render*| Shell
    ProgressH -->|builds LineChartData| Shell
    WorkoutH -->|builds DualAxisChartData| Shell
    ImportH --> Shell
    AchievementsH -->|builds AchievementTilesData, calls webui.RenderAchievementTiles| Shell
    MoneyH -.->|"achievements.BuildAchievementTiles types=money_saving/money_spend"| AchievementsH
    ProgressH -.->|"achievements.BuildAchievementTiles types=activity_*"| AchievementsH
    WorkoutH -.->|"achievements.BuildAchievementTiles types=exercise_*"| AchievementsH

    Shell --> Table
    Shell --> Stat
    Shell --> Line
    Shell --> Bar
    Shell --> Combo
    Shell --> Dual
    Shell --> Detail
    Shell --> Tile

    style Browser fill:#e1f5ff
    style Shell fill:#ffe1e1
    style Demo fill:#fff3cd
```

## Database Schema

### SQL DDL

None — no new tables. This layer only renders data already owned by other subdomains.

## Go Code Structure

### Domain Models

```go
// action/webui/types.go
package webui

// NavItem is one link in the shared page nav.
type NavItem struct {
    Label  string
    URL    string
    Active bool // current page, for highlighting
}

// PageData is the top-level data passed to the shell template.
type PageData struct {
    Title    string
    Nav      []NavItem
    UserName string        // logged-in user's name, shown in the header dropdown; empty hides it
    Content  template.HTML // pre-rendered content template output
}

// StatTileData is one summary number (e.g. "Current balance: €4,231").
type StatTileData struct {
    Label     string
    Value     string
    SubLabel  string // optional, e.g. "vs last month: +€120"
    Emphasis  bool   // visually highlighted tile (e.g. the primary metric)
}

// TableColumn describes one column header.
type TableColumn struct {
    Label string
    Align string // "left" | "right", default left
}

// TableRow is one row's cells plus an optional drill-down link.
type TableRow struct {
    Cells   []string
    LinkURL string        // empty = not clickable
    Tags    []TableRowTag // this row's tags, rendered in the table's tags column right after the first Cells column (see TableData.TagsColumnLabel); nil = none, only meaningful when TagsColumnLabel is set
    Heading string        // when non-empty, row renders as a full-width section-heading row (Cells/LinkURL/Tags ignored) instead of normal cells; empty = normal row, same "empty = off" convention as the rest of the component
}

// TableData is a full table component.
type TableData struct {
    Columns         []TableColumn
    Rows            []TableRow
    Pagination      *PaginationData // nil = no pagination controls rendered
    TagsColumnLabel string          // header for the tags column, inserted right after the first Cells column; empty = no tags column at all
}

// PaginationData drives the prev/next + "page X of Y" controls rendered
// below a table. Page is 1-indexed. PrevURL/NextURL are pre-built by the
// caller (so the component stays agnostic of each page's own query params)
// and are empty on the first/last page respectively, which hides that link.
type PaginationData struct {
    Page       int
    TotalPages int
    PrevURL    string
    NextURL    string
}

// DetailViewData is a drill-down page: a header plus a table of related records.
type DetailViewData struct {
    Title       string
    Description template.HTML // optional subtitle under Title, e.g. an activity's free-text description; empty renders nothing
    BackURL     string
    BackText    string
    Stats       []StatTileData // optional summary tiles at top
    Table       TableData
}

// LineChartPoint is one (x, y) sample in a trend line chart.
type LineChartPoint struct {
    Label string  // x-axis label, e.g. "2026-08-01"
    Value float64 // y-axis value, e.g. weight in kg or rep count
}

// LineChartData is a single-series line chart (e.g. a Progress activity's
// value-over-time drill-down).
type LineChartData struct {
    ID         string // unique DOM id for this chart's <canvas>, caller-supplied
    Title      string
    SeriesName string // legend label, e.g. "Weight (kg)"
    Points     []LineChartPoint
}

// BarChartBar is one labeled bar in a bar chart.
type BarChartBar struct {
    Label string  // e.g. category name "groceries"
    Value float64 // e.g. average monthly spend
}

// BarChartData is a single-series categorical bar chart (e.g. Money spend-by-category).
type BarChartData struct {
    ID         string // unique DOM id for this chart's <canvas>, caller-supplied
    Title      string
    SeriesName string // legend label, e.g. "Avg monthly spend (EUR)"
    Bars       []BarChartBar
}

// ComboChartPoint is one (x, y) sample plotted as both a bar and a line —
// the same value drawn two ways, not two different series.
type ComboChartPoint struct {
    Label string  // x-axis label, e.g. "2026-08"
    Value float64 // e.g. balance at this point in the trend
}

// ComboChartData is a single-series chart rendered as both a bar and an
// overlaid line for the same values (e.g. Money's balance trend) — for
// when a plain LineChartData or BarChartData reads less clearly alone than
// the two drawing styles combined on one series.
type ComboChartData struct {
    ID         string // unique DOM id for this chart's <canvas>, caller-supplied
    Title      string
    SeriesName string // e.g. "Balance (EUR)" — shown in the tooltip, no legend
    Points     []ComboChartPoint
}

// DualAxisChartPoint is one x position with up to two values, one per Y axis.
// nil = no value for that series at this point (gap in that line only).
type DualAxisChartPoint struct {
    Label string   // x-axis label, e.g. "2026-08-01"
    Left  *float64 // left Y axis value, e.g. weight in kg
    Right *float64 // right Y axis value, e.g. rep count
}

// DualAxisChartData is a two-series line chart, each series on its own Y
// axis (e.g. Workouts exercise drill-down: weight left, reps right).
type DualAxisChartData struct {
    ID              string // unique DOM id for this chart's <canvas>, caller-supplied
    Title           string
    LeftSeriesName  string // legend + left axis label, e.g. "Weight (kg)"
    RightSeriesName string // legend + right axis label, e.g. "Reps"
    Points          []DualAxisChartPoint
}

// TableRowTag is one small tag shown next to a table row's first cell (e.g.
// a life_part tag on the Progress browse table), styled as a Pico CSS
// contrast button (role="button" class="contrast") — no custom color CSS of
// ours. Tooltip renders via Pico's own data-tooltip attribute (pure CSS, no
// JS) instead of the native HTML title attribute — Chrome's native title
// tooltip proved unreliable in practice.
type TableRowTag struct {
    Label   string
    Tooltip string // shown on hover via Pico's data-tooltip attribute; empty = no tooltip
}

// CalendarDay is one cell in a month-grid calendar.
type CalendarDay struct {
    Day      int    // day-of-month number shown in the cell, e.g. 5
    InMonth  bool   // false for the leading/trailing days of adjacent months
    Count    string // e.g. "3 transactions" — empty hides the count line
    Total    string // e.g. "€42.10" — empty hides the total line
    LinkURL  string // empty = not clickable, same convention as TableRow.LinkURL
}

// CalendarData is a month-grid calendar (e.g. Money's transaction calendar).
type CalendarData struct {
    Title    string        // e.g. "August 2026"
    PrevURL  string        // previous month link; empty hides it (and the whole nav row when NextURL is also empty)
    NextURL  string        // next month link; empty hides it (and the whole nav row when PrevURL is also empty)
    Weekdays []string      // 7 column headers, e.g. ["Mon", ..., "Sun"]
    Weeks    [][]CalendarDay // each inner slice has exactly 7 CalendarDay entries
}

// AchievementTileData is one achievement rendered as a card with a progress bar (see achievements-spec.md).
type AchievementTileData struct {
    Name            string
    ProgressLabel   string  // e.g. "€420 / €1,000 (42%)", "82kg / 100kg", "14 / 30 day streak"
    PercentComplete float64 // 0-100, clamped for the bar width (ProgressLabel still shows the real, unclamped numbers)
    Deadline        string  // formatted, e.g. "by Dec 31, 2026" — empty hides the deadline line (no deadline set)
    OverTarget      bool    // current exceeds target — tile renders with a warning tint. Only meaningful for achievement
                             // types where exceeding is bad (e.g. money_spend over budget); left false otherwise
    LinkURL         string  // drill-down link to the underlying category/exercise/activity page, empty = not clickable
                             // (same "empty = not clickable" convention as TableRow.LinkURL/CalendarDay.LinkURL);
                             // derived per achievement_type by action/achievements' toAchievementTileData, see achievements-spec.md
}

// AchievementTilesData is a grid of achievement tiles — either every active achievement (the
// dedicated /web/achievements page) or a subset pre-filtered to one domain's achievement
// types (embedded in the Money/Progress-browse/Workouts pages).
type AchievementTilesData struct {
    Tiles        []AchievementTileData
    EmptyMessage string // shown instead of the grid when Tiles is empty; empty string means render nothing (see achievements-spec.md's embedded-vs-dedicated empty-state convention)
}
```

### Repository Interface

None — this package performs no database access. Consuming handlers fetch their own data and pass it in as the structs above.

### Template File Layout

Every template is a real `.html` file, embedded at compile time and parsed once into a single named-template set:

```
action/webui/templates/
  layout.html                    {{define "layout"}}      — shell: head/nav/footer, design tokens
  components/
    table.html                   {{define "components/table"}}       — renders Pagination controls below the rows when set
    pagination.html               {{define "components/pagination"}} — prev/next + "page X of Y", included by table.html
    stat_tiles.html               {{define "components/stat_tiles"}}
    chart.html                    {{define "components/chart"}}       — shared by line + bar (differ by .Type)
    combo_chart.html               {{define "components/combo_chart"}} — one series, bar+line mixed-dataset overlay
    dual_axis_chart.html           {{define "components/dual_axis_chart"}} — two series, left + right Y axis, legend on
    calendar.html                  {{define "components/calendar"}}
    detail_view.html               {{define "components/detail_view"}}
    achievement_tiles.html                 {{define "components/achievement_tiles"}}  — one <article> card per AchievementTileData, wrapped in a responsive grid
  pages/
    design_system.html             {{define "pages/design_system"}}   — demo page's own composition
```

A page's `.html` file (e.g. `pages/design_system.html`) never calls a component template directly — the Go handler renders each component to a `template.HTML` fragment first (via `RenderTable`, `RenderStatTiles`, ...), then the page template just places those fragments into `<section>`s declaratively. This keeps every `.html` file focused on markup/layout only, and keeps the small data-shaping (e.g. splitting `LineChartData.Points` into parallel label/value slices) in Go where it belongs.

## HTTP Handlers

### GET /web/design-system
The one real route this package owns. Renders a single page built entirely from fixture data, exercising every component in this design system so the whole visual language can be reviewed in a browser before any real dashboard is wired up:
- Page shell with populated nav (multiple items, one marked active)
- A stat tile row (mix of plain and emphasized tiles)
- A table (with at least one clickable/drill-down row)
- A line chart with sample time-series data (styled to preview both a Workouts-style "weight over time" and a Progress-style "value over time" use)
- A bar chart with sample categorical data (previewing a Money-style "spend by category" use)
- A combo chart with sample data (previewing Money's balance trend: one series drawn as both a bar and a line)
- A dual-axis line chart with sample data (previewing Workouts' weight + reps per set, including one point with no weight → gap in the weight line)
- A month-grid calendar with real, correctly-computed leading/trailing days and a few sample linked/unlinked days (previewing the Money transaction calendar use)
- A drill-down/detail view section (stat tiles + table, as a linked sub-page)
- An achievement tile grid with a few sample tiles (mixed progress percentages, one with a deadline, one `OverTarget`) previewing both the dedicated Achievements page and an embedded subset use

Behind `WebMiddleware` (see `auth-spec.md`'s "Authorization for web dashboards" flow) like every other `/web/*` dashboard — it renders through the same shell, and the shell now shows the logged-in username, so it needs a real session to demo that correctly.

The handler itself only builds fixture data and calls the render functions below; the actual page layout lives in `templates/pages/design_system.html` (see Template File Layout). Future dashboard handlers (Money, Progress, Workouts) follow the same shape: build data, call `RenderTable`/`RenderStatTiles`/etc. for each fragment, put the fragments into their own `PageData`/content template — never assembling HTML by hand in Go.

### Render functions
Everything else is a Go function, not a route — called from other subdomains' handlers to build their pages:

### `webui.RenderPage(w io.Writer, data PageData) error`
Executes the shared shell template (head/nav/footer + design tokens, light/dark via `prefers-color-scheme`) with `data.Content` dropped into the content slot. When `data.UserName` is set, the header also renders it as a `<details class="dropdown">` menu (Pico CSS's built-in disclosure pattern, no custom JS) containing one item: a "Logout" link to `GET /web/logout` (see `auth-spec.md`). Used by every `/web/*` handler, including the demo page above.

### `webui.RenderTable(data TableData) template.HTML`
Renders a `TableData` as a Pico CSS card (`<article>` — no extra class needed, Pico styles a bare `<article>` as a card) containing the `<table>`, to be embedded as `PageData.Content` (directly, or composed inside a page's own content template). When `data.Pagination` is non-nil, also renders the pagination controls (prev/next links, "page X of Y") inside a card `<footer>` below the rows — callers needing a paginated list (e.g. Progress browse/finished/future lists, or a drill-down's history table) just set `TableData.Pagination` instead of calling a separate render function. When `data.TagsColumnLabel` is non-empty, the table also gets a column with that header text inserted right after the first `Columns` entry (e.g. Name), and each row's `Tags` render inside its own cell in that column as small `<span role="button" class="outline contrast webui-tag">` elements — Pico's own `.contrast` button styling (dark background/border, no custom color CSS of ours), sized down to fit via `.webui-tag` (padding/font-size only, no color) — with `data-tooltip="{{.Tooltip}}"` set when `Tooltip` is non-empty for Pico's own CSS-only tooltip on hover, no extra script. `TagsColumnLabel` empty (every caller but Progress's activity lists) omits the column entirely. A row with `Heading` set renders as `<tr><td colspan="N">{{.Heading}}</td></tr>` (`N` = total column count, tags column included when present) instead of its (ignored) `Cells`/`Tags` — used by Progress's active-list grouping (see `progress-spec.md`).

### `webui.RenderStatTiles(data []StatTileData) template.HTML`
Renders a row of summary/stat tiles, each its own Pico card (`<article class="webui-stat-tile">`).

### `webui.RenderLineChart(data LineChartData) template.HTML`
Renders a Pico card (`<article class="webui-chart-container">`) with a `<header>` holding the chart title and a `<canvas>` below it, fed the `LineChartData` points as embedded JSON via a small inline script that initializes a Chart.js line chart, colored from the page's CSS variables (so it follows light/dark automatically). The card is capped at a fixed max-width (`--webui-chart-container` CSS) plus a fixed Chart.js `aspectRatio`, so charts render compact rather than stretching to the full page width.

### `webui.RenderBarChart(data BarChartData) template.HTML`
Same as `RenderLineChart`, but initializes a Chart.js bar chart from `BarChartData`.

### `webui.RenderComboChart(data ComboChartData) template.HTML`
Renders a Pico card (`<article class="webui-chart-container">`) with a `<header>` holding the chart title and a `<canvas>` below it, initializing a single Chart.js chart (`type: "bar"`) built from `ComboChartData.Points` with two datasets both plotting the *same* `Points[].Value` against the shared `Points[].Label` x-axis: a bar dataset colored from `--webui-chart-bar`, and a second dataset overridden to `type: "line"` colored from the existing `--webui-chart-line` (same color `RenderLineChart` uses). Legend stays off, same as `RenderLineChart`/`RenderBarChart` — both datasets are the one series, so a legend would just repeat `SeriesName` twice.

### `webui.RenderDualAxisChart(data DualAxisChartData) template.HTML`
Renders a Pico card (`<article class="webui-chart-container">`) with a `<header>` title and a `<canvas>`, initializing one Chart.js `type: "line"` chart with two datasets: `Points[].Left` on the left `y` axis (`--webui-chart-line`), `Points[].Right` on the right `y1` axis (`yAxisID: "y1"`, `--webui-chart-bar`). nil values render as JSON `null` → gap in that line. Legend on. Used by the Workouts exercise drill-down (`workout-spec.md`).

### `webui.RenderCalendar(data CalendarData) template.HTML`
Renders a Pico card (`<article>`) with a `<header>` holding the month title, and a 7-column grid below (`<table>`, one `<tr>` per `Weeks` entry) — each cell shows the day number plus, when set, `Count` and `Total`. When `PrevURL`/`NextURL` are set, the header also shows a prev/next nav — but a caller stacking several months on one page (e.g. Money's calendar) typically leaves both empty per grid and renders a single page-level Prev/Next control of its own instead, so the header falls back to just the title. A day with `InMonth == false` renders muted/de-emphasized; a day with `LinkURL` set is a clickable link, matching `RenderTable`'s row-link convention. Used by the Money transaction calendar (`money-spec.md`).

### `webui.RenderDetailView(data DetailViewData) template.HTML`
Renders the drill-down/detail layout: back link, title, optional description paragraph, optional stat tiles, then an optional table. `Description` renders (as pre-escaped `template.HTML`, so callers can pass through markdown-rendered content) only when non-empty — used by the Progress activity drill-down (`progress-spec.md`) to show `Activity.Description`. When `data.Table.Columns` is empty, the table section is skipped entirely (mirrors the existing "skip stat tiles when `Stats` is empty" behavior) — used by the Workouts exercise drill-down (`workout-spec.md`), which has stat tiles and a chart but no table.

### `webui.RenderAchievementTiles(data AchievementTilesData) template.HTML`
Renders `data.Tiles` as a responsive card grid (`<article class="webui-achievement-tile">` per tile, CSS grid wrapper — same "no fixed viewport sizing" responsiveness as every other component), each card showing the achievement name, a Pico native `<progress value="{{.PercentComplete}}" max="100">` bar, the `ProgressLabel` text below it, and the `Deadline` line when set. When `LinkURL` is set, the achievement name is rendered as `<a href="{{.LinkURL}}">{{.Name}}</a>` instead of plain text — the same "link the primary identifier, not the whole card" convention `RenderTable` uses for a row's first cell — otherwise it's plain text (some achievement types have no drill-down target, see `achievements-spec.md`). A tile with `OverTarget` true gets a warning-tint class (`webui-achievement-tile--over`), styled from the same `:root` design tokens as the rest of the custom CSS layer. When `data.Tiles` is empty, renders `data.EmptyMessage` if set, or nothing at all if it's also empty — the dedicated `/web/achievements` page always sets a message ("No active achievements yet"), while Money/Progress-browse/Workouts leave it unset so an embedded section with no achievements of that domain's types simply doesn't appear (see `achievements-spec.md`).

## Local Preview

### `make preview-webui` (`cmd/webui-preview`)
Dev-only command, not deployed. Reuses (or creates) the `personal-webui-preview` Postgres testcontainer, restores the cached migrations + fixtures snapshot (or builds it on first run / after a migration or `cmd/webui-preview/fixtures.sql` change), and serves every `/web/*` route (via `transport/web.Register`) on the real repository at `http://localhost:$PORT` (default 8082). Auth behaves as before: real `WebMiddleware` with `USERS`/`JWT_SECRET` from `.env.local`, or none when `AUTH_DISABLED` is set. Needs a running Docker (colima). The container stays up after exit; `docker rm -f personal-webui-preview` drops it. `gateways/db/mock.go` is removed.

No E2E test: it's a developer tool, not an HTTP handler; the pages it serves are already covered by the existing `tests/*_web_test.go` suites on the same repository.

## E2E Tests

In `tests/webui_design_system_test.go`:

- `TestDesignSystem_*`: renders OK; loads pico CSS from CDN; loads chart JS from CDN; supports light and dark via media query; nav has active item; shows stat tiles plain and emphasized; shows table with drill down link; shows line charts; shows bar chart; shows combo chart; shows dual axis chart; charts have unique canvas ids; shows drill down detail view; escapes fixture text containing markup

In `tests/webui_user_menu_test.go`:

- `TestRenderPage_*`: user name set, shows dropdown with logout link; user name empty, hides dropdown; user name containing markup, is escaped
- `TestDesignSystem_*`: shows logged in user dropdown

## Changelog

- **03-10-26** — migrated to the new spec template: removed Best Practices Applied and the sequence diagrams together with every reference to them; E2E Tests rewritten as a list of the current tests; added Changelog (Architecture Diagrams, E2E Tests, Changelog)
- **29-09-26** — local preview runs on a Postgres testcontainer restored from a cached fixture snapshot (Overview, Local Preview)
- **26-09-26** — goal tiles renamed to achievement tiles (Overview, Architecture Diagrams, Go Code Structure, HTTP Handlers)
- **26-09-26** — added the dual-axis chart component (Overview, Architecture Diagrams, Go Code Structure, HTTP Handlers, E2E Tests)
- **22-09-26** — table rows can be section headings (`TableRow.Heading`) (Go Code Structure, HTTP Handlers)
- **22-09-26** — row tags moved into their own table column (`TableData.TagsColumnLabel`) (Go Code Structure, HTTP Handlers)
- **21-09-26** — added table row tags (`TableRow.Tags`) (Go Code Structure, HTTP Handlers)
- **06-09-26** — added the combo chart component (Architecture Diagrams, Go Code Structure, HTTP Handlers)
- **06-09-26** — goal tiles get a drill-down `LinkURL` (Go Code Structure, HTTP Handlers)
- **06-09-26** — detail view gets an optional `Description` subtitle (Go Code Structure, HTTP Handlers)
- **02-09-26** — added the goal tile grid component (Overview, Architecture Diagrams, Go Code Structure, HTTP Handlers)
- **22-08-26** — Overview no longer lists the navigation home page (Overview)
- **22-08-26** — added the calendar component (Overview, Go Code Structure, HTTP Handlers)
- **22-08-26** — detail view's table became optional (HTTP Handlers)
- **22-08-26** — components render as Pico cards (`<article>`) (HTTP Handlers)
- **22-08-26** — added table pagination (`TableData.Pagination`) (Go Code Structure, HTTP Handlers)
- **22-08-26** — added the user menu with a logout link to the shell (Go Code Structure, HTTP Handlers)
- **22-08-26** — templates moved to real `.html` files; added Template File Layout (Go Code Structure, HTTP Handlers)
- **22-08-26** — initial version
