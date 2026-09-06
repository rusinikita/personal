package webui

import (
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"strings"
)

//go:embed templates/layout.html templates/components/*.html templates/pages/*.html
var templatesFS embed.FS

var funcMap = template.FuncMap{
	// toJS marshals v to JSON for embedding into a <script> block. html/template
	// treats a template.JS return value as already safe for JS context and
	// inserts it verbatim instead of re-escaping it.
	"toJS": toJS,
}

var tmpl = template.Must(template.New("webui").Funcs(funcMap).ParseFS(templatesFS,
	"templates/layout.html",
	"templates/components/*.html",
	"templates/pages/*.html",
))

// RenderPage executes the shared page shell ("layout" — head/nav/footer +
// design tokens, light/dark via prefers-color-scheme) with data.Content
// dropped into the content slot, and writes the resulting HTML to w.
func RenderPage(w io.Writer, data PageData) error {
	return tmpl.ExecuteTemplate(w, "layout", data)
}

// tableCellRenderData is one <td>'s text plus its resolved alignment — cells
// don't carry an Align field themselves (TableRow.Cells is just []string),
// so RenderTable resolves each cell's alignment from its column position
// before handing the template a shape it can render without doing that
// lookup itself.
type tableCellRenderData struct {
	Text  string
	Align string // "left" | "right", always set (never empty)
}

type tableRowRenderData struct {
	Cells   []tableCellRenderData
	LinkURL string
}

type tableRenderData struct {
	Columns    []TableColumn
	Rows       []tableRowRenderData
	Pagination *PaginationData
}

// RenderTable renders a TableData into the shared table component markup.
// Each column's Align applies to both its header and every cell in that
// column, so numeric ("right") columns line up between the header and the
// rows instead of the header alone being right-aligned.
func RenderTable(data TableData) template.HTML {
	rows := make([]tableRowRenderData, len(data.Rows))
	for i, row := range data.Rows {
		cells := make([]tableCellRenderData, len(row.Cells))
		for j, cell := range row.Cells {
			align := "left"
			if j < len(data.Columns) && data.Columns[j].Align == "right" {
				align = "right"
			}
			cells[j] = tableCellRenderData{Text: cell, Align: align}
		}
		rows[i] = tableRowRenderData{Cells: cells, LinkURL: row.LinkURL}
	}
	return execToHTML("components/table", tableRenderData{
		Columns:    data.Columns,
		Rows:       rows,
		Pagination: data.Pagination,
	})
}

// RenderStatTiles renders a row of summary/stat tiles.
func RenderStatTiles(data []StatTileData) template.HTML {
	return execToHTML("components/stat_tiles", data)
}

// chartTemplateData is the shape components/chart.html renders — both line
// and bar charts share one Chart.js init script, differing only by Type.
type chartTemplateData struct {
	ID         string
	Type       string
	Title      string
	SeriesName string
	Labels     []string
	Values     []float64
}

// RenderLineChart renders a <canvas> plus LineChartData's points as embedded
// JSON and a small inline script that initializes a Chart.js line chart
// against it, colored from the page's CSS variables (so it follows
// light/dark automatically).
func RenderLineChart(data LineChartData) template.HTML {
	labels := make([]string, len(data.Points))
	values := make([]float64, len(data.Points))
	for i, p := range data.Points {
		labels[i] = p.Label
		values[i] = p.Value
	}
	return execToHTML("components/chart", chartTemplateData{
		ID: data.ID, Type: "line", Title: data.Title, SeriesName: data.SeriesName,
		Labels: labels, Values: values,
	})
}

// RenderBarChart is the same as RenderLineChart, but initializes a Chart.js
// bar chart from BarChartData.
func RenderBarChart(data BarChartData) template.HTML {
	labels := make([]string, len(data.Bars))
	values := make([]float64, len(data.Bars))
	for i, b := range data.Bars {
		labels[i] = b.Label
		values[i] = b.Value
	}
	return execToHTML("components/chart", chartTemplateData{
		ID: data.ID, Type: "bar", Title: data.Title, SeriesName: data.SeriesName,
		Labels: labels, Values: values,
	})
}

// RenderComboChart renders a <canvas> plus ComboChartData's points as
// embedded JSON and a small inline script that initializes a single
// Chart.js chart plotting the same values as both a bar dataset (colored
// from --webui-chart-bar) and an overlaid line dataset (colored from
// --webui-chart-line, same as RenderLineChart) — one series, two drawing
// styles, not two different series, so the legend stays off same as
// RenderLineChart/RenderBarChart.
func RenderComboChart(data ComboChartData) template.HTML {
	labels := make([]string, len(data.Points))
	values := make([]float64, len(data.Points))
	for i, p := range data.Points {
		labels[i] = p.Label
		values[i] = p.Value
	}
	return execToHTML("components/combo_chart", chartTemplateData{
		ID: data.ID, Title: data.Title, SeriesName: data.SeriesName,
		Labels: labels, Values: values,
	})
}

// RenderCalendar renders a CalendarData into the shared month-grid calendar
// component markup.
func RenderCalendar(data CalendarData) template.HTML {
	return execToHTML("components/calendar", data)
}

// detailViewRenderData is the shape components/detail_view.html renders —
// StatsHTML/TableHTML are already-rendered fragments so the component
// template doesn't need to know how to build a table or stat tile itself.
type detailViewRenderData struct {
	Title       string
	Description template.HTML
	BackURL     string
	BackText    string
	StatsHTML   template.HTML
	TableHTML   template.HTML
}

// RenderDetailView renders the drill-down/detail layout: back link, optional
// stat tiles, then an optional table.
func RenderDetailView(data DetailViewData) template.HTML {
	var statsHTML template.HTML
	if len(data.Stats) > 0 {
		statsHTML = RenderStatTiles(data.Stats)
	}
	var tableHTML template.HTML
	if len(data.Table.Columns) > 0 {
		tableHTML = RenderTable(data.Table)
	}
	return execToHTML("components/detail_view", detailViewRenderData{
		Title:       data.Title,
		Description: data.Description,
		BackURL:     data.BackURL,
		BackText:    data.BackText,
		StatsHTML:   statsHTML,
		TableHTML:   tableHTML,
	})
}

// RenderGoalTiles renders data.Tiles as a responsive card grid, each card
// showing the goal name, a Pico native progress bar, the ProgressLabel text,
// and the Deadline line when set. When Tiles is empty, renders
// EmptyMessage if set, or nothing at all if it's also empty — the dedicated
// /web/goals page always sets a message, while Money/Progress-browse/
// Workouts leave it unset so an embedded section with no goals of that
// domain's types simply doesn't appear (see goals-spec.md).
func RenderGoalTiles(data GoalTilesData) template.HTML {
	return execToHTML("components/goal_tiles", data)
}

// execToHTML executes the named template (defined in one of the embedded
// templates/ files) against data and returns the result as template.HTML,
// so it can be embedded into a parent template without double-escaping.
// The template itself still HTML-escapes every field it prints, so
// untrusted strings passed in via data are safe.
func execToHTML(name string, data any) template.HTML {
	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, name, data); err != nil {
		// Component templates are embedded at compile time and only ever
		// fail on a programmer error (bad field access), which
		// template.Must already catches for parse errors — an exec error
		// here means the data shape stopped matching the template, so fail
		// loudly.
		panic(err)
	}
	return template.HTML(b.String())
}

// toJS marshals v to JSON and returns it as template.JS so it can be
// embedded verbatim into a <script> block. "</" is escaped to "<\/" so a
// string value can never prematurely close the surrounding <script> tag.
func toJS(v any) (template.JS, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return template.JS(strings.ReplaceAll(string(b), "</", `<\/`)), nil
}
