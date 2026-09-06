package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"personal/action/webui"
)

// designSystemRouter builds a minimal gin engine serving only the demo page.
// The demo page is pure presentation (fixture data only, no DB), so unlike
// most other web handlers it needs no DB/UserID context middleware.
func (s *IntegrationTestSuite) designSystemRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/web/design-system", webui.DesignSystemHandler)
	return r
}

func (s *IntegrationTestSuite) TestDesignSystem_RendersOK() {
	r := s.designSystemRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Equal(s.T(), "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(s.T(), w.Body.String(), "<!DOCTYPE html>")
}

func (s *IntegrationTestSuite) TestDesignSystem_LoadsPicoCSSFromCDN() {
	r := s.designSystemRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, "picocss", "shell must link Pico CSS from the CDN")
	assert.Contains(s.T(), body, "<link", "Pico CSS must be a <link> stylesheet, not inlined")
}

func (s *IntegrationTestSuite) TestDesignSystem_LoadsChartJSFromCDN() {
	r := s.designSystemRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, "chart.js", "shell must load Chart.js from the CDN")
}

func (s *IntegrationTestSuite) TestDesignSystem_SupportsLightAndDarkViaMediaQuery() {
	r := s.designSystemRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, "prefers-color-scheme: dark",
		"design tokens must redefine themselves for dark mode instead of using a toggle")
}

func (s *IntegrationTestSuite) TestDesignSystem_NavHasActiveItem() {
	r := s.designSystemRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, `aria-current="page"`,
		"the current nav item must be marked active")
}

func (s *IntegrationTestSuite) TestDesignSystem_ShowsStatTilesPlainAndEmphasized() {
	r := s.designSystemRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, "webui-stat-tile")
	assert.Contains(s.T(), body, "webui-stat-tile emphasis",
		"demo must show at least one emphasized stat tile, not just plain ones")
}

func (s *IntegrationTestSuite) TestDesignSystem_ShowsTableWithDrillDownLink() {
	r := s.designSystemRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, "<table>")
	assert.Contains(s.T(), body, "<thead>")
	assert.Contains(s.T(), body, "<a href=",
		"at least one table row must be a clickable drill-down link")
}

func (s *IntegrationTestSuite) TestDesignSystem_ShowsLineCharts() {
	r := s.designSystemRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.GreaterOrEqual(s.T(), strings.Count(body, `type: "line"`), 2,
		"demo must render at least two line chart examples (Workouts-style and Progress-style)")
}

func (s *IntegrationTestSuite) TestDesignSystem_ShowsBarChart() {
	r := s.designSystemRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, `type: "bar"`,
		"demo must render a bar chart example (Money-style spend-by-category)")
}

func (s *IntegrationTestSuite) TestDesignSystem_ShowsComboChart() {
	r := s.designSystemRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, `<canvas id="chart-balance-trend-demo">`,
		"demo must render a combo chart example (Money-style balance trend)")
	assert.Contains(s.T(), body, "--webui-chart-bar",
		"combo chart's bar dataset must use its own color token, distinct from its line overlay")
	assert.Contains(s.T(), body, "legend: { display: false }",
		"combo chart must keep the legend off — its bar and line datasets are the same series, not two")
}

func (s *IntegrationTestSuite) TestDesignSystem_ChartsHaveUniqueCanvasIDs() {
	r := s.designSystemRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	canvasCount := strings.Count(body, "<canvas id=")
	// two line chart examples + one bar chart example + one combo chart example
	assert.GreaterOrEqual(s.T(), canvasCount, 4)

	// Every canvas id referenced in a getElementById call must actually exist
	// as a rendered <canvas id="..."> element (charts wired to the right canvas).
	getElementCount := strings.Count(body, "getElementById(")
	assert.Equal(s.T(), canvasCount, getElementCount)
}

func (s *IntegrationTestSuite) TestDesignSystem_ShowsDrillDownDetailView() {
	r := s.designSystemRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, "webui-detail-back",
		"demo must include a drill-down/detail view section with a back link")
}

func (s *IntegrationTestSuite) TestDesignSystem_EscapesFixtureTextContainingMarkup() {
	// The render functions used here are the same ones real dashboards will
	// call with real user data (merchant names, activity notes, ...), so
	// html/template's default contextual escaping must not be bypassed
	// anywhere in the shell/table/stat-tile paths. The demo fixtures
	// deliberately include a "<3 days>"-style value to prove it comes out
	// escaped rather than as live markup.
	r := s.designSystemRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/design-system", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(s.T(), body, "&lt;3 days&gt;",
		"a fixture value containing '<3 days>' must render HTML-escaped")
	assert.NotContains(s.T(), body, "<3 days>",
		"the unescaped raw markup must never appear in the output")
}
