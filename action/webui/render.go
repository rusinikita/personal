package webui

import (
	"encoding/json"
	"html/template"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
)

const shellTemplateSrc = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}}</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css">
<script src="https://cdn.jsdelivr.net/npm/chart.js@4"></script>
<style>
:root {
  --webui-accent: #0d6efd;
  --webui-stat-bg: #f4f4f4;
  --webui-stat-emphasis-bg: #e7f1ff;
  --webui-chart-grid: #dddddd;
  --webui-chart-line: #0d6efd;
}
@media (prefers-color-scheme: dark) {
  :root {
    --webui-accent: #4dabf7;
    --webui-stat-bg: #22262b;
    --webui-stat-emphasis-bg: #14315e;
    --webui-chart-grid: #444444;
    --webui-chart-line: #4dabf7;
  }
}
nav a[aria-current="page"] { color: var(--webui-accent); font-weight: 700; }
.webui-stats { display: flex; flex-wrap: wrap; gap: 1rem; margin-bottom: 1.5rem; }
.webui-stat-tile { flex: 1 1 10rem; padding: 1rem; border-radius: var(--pico-border-radius, 0.25rem); background: var(--webui-stat-bg); }
.webui-stat-tile.emphasis { background: var(--webui-stat-emphasis-bg); border: 1px solid var(--webui-accent); }
.webui-stat-label { font-size: 0.8rem; opacity: 0.7; text-transform: uppercase; letter-spacing: 0.03em; }
.webui-stat-value { font-size: 1.6rem; font-weight: 600; }
.webui-stat-sub { font-size: 0.8rem; opacity: 0.7; }
.webui-chart-container { max-width: 100%; margin-bottom: 1.5rem; }
.webui-detail-back { display: inline-block; margin-bottom: 1rem; }
</style>
</head>
<body>
<header class="container">
<nav>
<ul><li><strong>{{.Title}}</strong></li></ul>
<ul>
{{range .Nav}}<li><a href="{{.URL}}"{{if .Active}} aria-current="page"{{end}}>{{.Label}}</a></li>
{{end}}
</ul>
</nav>
</header>
<main class="container">
{{.Content}}
</main>
</body>
</html>`

var shellTemplate = template.Must(template.New("shell").Parse(shellTemplateSrc))

// RenderPage executes the shared page shell (head/nav/footer + design
// tokens, light/dark via prefers-color-scheme) with data.Content dropped
// into the content slot, and writes the resulting HTML to w.
func RenderPage(w io.Writer, data PageData) error {
	return shellTemplate.Execute(w, data)
}

const tableTemplateSrc = `<table>
<thead><tr>
{{range .Columns}}<th{{if eq .Align "right"}} style="text-align:right"{{end}}>{{.Label}}</th>
{{end}}</tr></thead>
<tbody>
{{range .Rows}}<tr>
{{$link := .LinkURL}}
{{range $i, $cell := .Cells}}<td>{{if and (eq $i 0) $link}}<a href="{{$link}}">{{$cell}}</a>{{else}}{{$cell}}{{end}}</td>
{{end}}</tr>
{{end}}</tbody>
</table>`

var tableTemplate = template.Must(template.New("table").Parse(tableTemplateSrc))

// RenderTable renders a TableData into the shared table component markup.
func RenderTable(data TableData) template.HTML {
	return execToHTML(tableTemplate, data)
}

const statTilesTemplateSrc = `<div class="webui-stats">
{{range .}}<div class="webui-stat-tile{{if .Emphasis}} emphasis{{end}}">
<div class="webui-stat-label">{{.Label}}</div>
<div class="webui-stat-value">{{.Value}}</div>
{{if .SubLabel}}<div class="webui-stat-sub">{{.SubLabel}}</div>{{end}}
</div>
{{end}}</div>`

var statTilesTemplate = template.Must(template.New("statTiles").Parse(statTilesTemplateSrc))

// RenderStatTiles renders a row of summary/stat tiles.
func RenderStatTiles(data []StatTileData) template.HTML {
	return execToHTML(statTilesTemplate, data)
}

const chartTemplateSrc = `<div class="webui-chart-container">
<canvas id="{{.ID}}"></canvas>
<script>
(function() {
  var ctx = document.getElementById({{.IDJSON}});
  var style = getComputedStyle(document.documentElement);
  new Chart(ctx, {
    type: {{.TypeJSON}},
    data: {
      labels: {{.LabelsJSON}},
      datasets: [{
        label: {{.SeriesNameJSON}},
        data: {{.ValuesJSON}},
        borderColor: style.getPropertyValue('--webui-chart-line').trim(),
        backgroundColor: style.getPropertyValue('--webui-chart-line').trim(),
        tension: 0.25
      }]
    },
    options: {
      responsive: true,
      plugins: { title: { display: true, text: {{.TitleJSON}} } }
    }
  });
})();
</script>
</div>`

var chartTemplate = template.Must(template.New("chart").Parse(chartTemplateSrc))

type chartTemplateData struct {
	ID             string
	IDJSON         template.JS
	TypeJSON       template.JS
	TitleJSON      template.JS
	SeriesNameJSON template.JS
	LabelsJSON     template.JS
	ValuesJSON     template.JS
}

var chartIDCounter int64

func nextChartID() string {
	n := atomic.AddInt64(&chartIDCounter, 1)
	return "webui-chart-" + strconv.FormatInt(n, 10)
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
	return renderChart("line", data.Title, data.SeriesName, labels, values)
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
	return renderChart("bar", data.Title, data.SeriesName, labels, values)
}

func renderChart(chartType, title, seriesName string, labels []string, values []float64) template.HTML {
	id := nextChartID()
	return execToHTML(chartTemplate, chartTemplateData{
		ID:             id,
		IDJSON:         toJS(id),
		TypeJSON:       toJS(chartType),
		TitleJSON:      toJS(title),
		SeriesNameJSON: toJS(seriesName),
		LabelsJSON:     toJS(labels),
		ValuesJSON:     toJS(values),
	})
}

const detailViewTemplateSrc = `<a class="webui-detail-back" href="{{.BackURL}}">← {{.BackText}}</a>
<h2>{{.Title}}</h2>
{{.StatsHTML}}
{{.TableHTML}}`

var detailViewTemplate = template.Must(template.New("detailView").Parse(detailViewTemplateSrc))

type detailViewRenderData struct {
	Title     string
	BackURL   string
	BackText  string
	StatsHTML template.HTML
	TableHTML template.HTML
}

// RenderDetailView renders the drill-down/detail layout: back link, optional
// stat tiles, then a table.
func RenderDetailView(data DetailViewData) template.HTML {
	var statsHTML template.HTML
	if len(data.Stats) > 0 {
		statsHTML = RenderStatTiles(data.Stats)
	}
	return execToHTML(detailViewTemplate, detailViewRenderData{
		Title:     data.Title,
		BackURL:   data.BackURL,
		BackText:  data.BackText,
		StatsHTML: statsHTML,
		TableHTML: RenderTable(data.Table),
	})
}

// execToHTML executes tmpl against data and returns the result as
// template.HTML, so it can be embedded into a parent template without
// double-escaping. tmpl itself still HTML-escapes every field it prints,
// so untrusted strings passed in via data are safe.
func execToHTML(tmpl *template.Template, data any) template.HTML {
	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		// Component templates are fixed at compile time and only ever fail
		// on a programmer error (bad field access), which template.Must
		// already catches for parse errors — an exec error here means the
		// data shape stopped matching the template, so fail loudly.
		panic(err)
	}
	return template.HTML(b.String())
}

// toJS marshals v to JSON and returns it as template.JS so it can be
// embedded verbatim into a <script> block. "</" is escaped to "<\/" so a
// string value can never prematurely close the surrounding <script> tag.
func toJS(v any) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return template.JS(strings.ReplaceAll(string(b), "</", `<\/`))
}
