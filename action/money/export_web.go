// Spending export: GET /web/money/export (filter + editable preview) and
// GET /web/money/export/download (the CSV itself) — a dedicated screen for
// pulling a category-level spending breakdown out of the system, separate
// from the read-only /web/money overview. See money-spec.md's "Spending
// Export" section.
package money

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

// exportPreviewFormSrc renders the date filter fields, one row per category
// (checkbox + real total + count + editable amount), a running total, and
// two submit buttons targeting the preview and download routes via
// formaction — one <form method="GET"> so incl[]/amt[] state always lives in
// the URL, never server-side session state (money-spec.md's "Export state
// lives entirely in the URL").
const exportPreviewFormSrc = `<form method="GET">
<fieldset role="group">
<input type="date" name="from" value="{{.From}}">
<input type="date" name="to" value="{{.To}}">
<button type="submit" formaction="/web/money/export">Update</button>
<a href="/web/money/export" role="button" class="outline">Clear</a>
</fieldset>
<article>
<table>
<thead><tr><th style="vertical-align:middle"></th><th style="vertical-align:middle">Category</th><th style="text-align:right;vertical-align:middle">Total</th><th style="text-align:right;vertical-align:middle">Count</th><th style="text-align:right;vertical-align:middle">Export amount</th></tr></thead>
<tbody>
{{range .Rows}}<tr>
<td style="vertical-align:middle"><input type="checkbox" name="incl[{{.Category}}]" {{if .Included}}checked{{end}} style="margin:0"></td>
<td style="vertical-align:middle">{{.Category}}</td>
<td style="text-align:right;vertical-align:middle">{{.RealTotal}}</td>
<td style="text-align:right;vertical-align:middle">{{.Count}}</td>
<td style="text-align:right;vertical-align:middle"><input type="text" name="amt[{{.Category}}]" value="{{.Amount}}" style="margin:0;width:8rem"></td>
</tr>
{{end}}
</tbody>
<tfoot><tr><td></td><td>Total</td><td></td><td></td><td style="text-align:right">{{.RunningTotal}}</td></tr></tfoot>
</table>
</article>
<button type="submit" formaction="/web/money/export/download">Export CSV</button>
</form>`

var exportPreviewFormTemplate = template.Must(template.New("exportPreviewForm").Parse(exportPreviewFormSrc))

type exportPreviewRowData struct {
	Category  string
	RealTotal string
	Count     int
	Amount    string
	Included  bool
}

type exportPreviewFormData struct {
	From         string
	To           string
	Rows         []exportPreviewRowData
	RunningTotal string
}

func renderExportPreviewForm(data exportPreviewFormData) (template.HTML, error) {
	var b strings.Builder
	if err := exportPreviewFormTemplate.Execute(&b, data); err != nil {
		return "", err
	}
	return template.HTML(b.String()), nil
}

// exportDateRange resolves ?from/?to to display-timezone day bounds,
// defaulting to [start of current month, today] when both are absent
// (money-spec.md's "Export default range is the current calendar month").
// The returned from/to strings are the resolved "2006-01-02" values, used to
// pre-fill the filter form and re-submitted on every subsequent request.
func exportDateRange(c *gin.Context, loc *time.Location) (from, to time.Time, fromStr, toStr string, err error) {
	now := time.Now().In(loc)
	fromStr = strings.TrimSpace(c.Query("from"))
	toStr = strings.TrimSpace(c.Query("to"))

	if fromStr == "" && toStr == "" {
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		to = now
		fromStr = from.Format("2006-01-02")
		toStr = to.Format("2006-01-02")
		return from, to, fromStr, toStr, nil
	}

	if fromStr != "" {
		from, err = parseDayBound(loc, fromStr, false)
		if err != nil {
			return time.Time{}, time.Time{}, "", "", fmt.Errorf("invalid from date")
		}
	}
	if toStr != "" {
		to, err = parseDayBound(loc, toStr, true)
		if err != nil {
			return time.Time{}, time.Time{}, "", "", fmt.Errorf("invalid to date")
		}
	} else {
		to = now
	}
	if fromStr == "" {
		from = time.Time{}
	}
	return from, to, fromStr, toStr, nil
}

// exportCategories loads the top-level (depth=1) spending breakdown for
// [from, to] — money-spec.md's "Spending export reuses GetSpendingByCategory,
// no new repository method".
func exportCategories(ctx context.Context, db gateways.DB, userID int64, from, to time.Time) ([]domain.SpendingByCategory, error) {
	return db.GetSpendingByCategory(ctx, userID, from, to, exportCategoryDepth)
}

// exportCategoryDepth is the category grouping depth for the spending
// export, same as the dashboard's own category table.
const exportCategoryDepth = 1

// buildExportPreviewRows merges the real per-category totals with any
// submitted incl[]/amt[] state. incl/amt are nil on first visit (no
// selections made yet), in which case every category defaults to included
// with amount == its real total. Once present, a category absent from incl
// is excluded, and amt[category] (when present) is the amount that will be
// exported instead of the real total — money-spec.md's "There is no
// separate 'override' flag".
func buildExportPreviewRows(categories []domain.SpendingByCategory, incl, amt map[string]string, hasSelection bool) ([]exportPreviewRowData, float64) {
	rows := make([]exportPreviewRowData, 0, len(categories))
	var runningTotal float64
	for _, cat := range categories {
		included := true
		if hasSelection {
			_, included = incl[cat.Category]
		}
		amount := cat.TotalEUR
		amountStr := formatAmountInput(cat.TotalEUR)
		if v, ok := amt[cat.Category]; ok {
			amountStr = v
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				amount = parsed
			}
		}
		if included {
			runningTotal += amount
		}
		rows = append(rows, exportPreviewRowData{
			Category:  cat.Category,
			RealTotal: formatEUR(cat.TotalEUR),
			Count:     cat.Count,
			Amount:    amountStr,
			Included:  included,
		})
	}
	return rows, runningTotal
}

// formatAmountInput renders a EUR total for the editable amount input's
// default value — plain "120.50", not the "€"-prefixed formatEUR used for
// read-only display, so it round-trips through strconv.ParseFloat unchanged
// if the user submits it back without editing.
func formatAmountInput(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// ExportWebHandler renders GET /web/money/export: the date filter, an
// editable preview table (checkbox + real total + count + amount per
// category), and a running total computed from the submitted incl[]/amt[]
// state so an exclusion or override is reflected immediately.
func ExportWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	loc, err := time.LoadLocation(displayTimezone)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load timezone: %v", err)
		return
	}

	from, to, fromStr, toStr, err := exportDateRange(c, loc)
	if err != nil {
		c.String(http.StatusBadRequest, "%v", err)
		return
	}

	categories, err := exportCategories(ctx, db, userID, from, to)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load spending by category: %v", err)
		return
	}

	incl, hasIncl := c.GetQueryMap("incl")
	amt, hasAmt := c.GetQueryMap("amt")
	rows, runningTotal := buildExportPreviewRows(categories, incl, amt, hasIncl || hasAmt)

	formHTML, err := renderExportPreviewForm(exportPreviewFormData{
		From:         fromStr,
		To:           toStr,
		Rows:         rows,
		RunningTotal: formatEUR(runningTotal),
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:    "Money — Export",
		Nav:      moneyNav,
		UserName: c.GetString("user_name"),
		Content:  formHTML,
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// csvField quotes a CSV field per RFC 4180 whenever it contains a comma,
// quote, or newline — category names are user-entered free text (via
// edit_transactions) and can't be assumed comma-free.
func csvField(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// ExportDownloadWebHandler renders GET /web/money/export/download: re-runs
// the same category breakdown for from/to, filters to categories present in
// incl[], and streams a CSV — Category, Amount (EUR), Count — using
// amt[category] (falling back to the real total when absent) as the
// exported amount.
func ExportDownloadWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)

	loc, err := time.LoadLocation(displayTimezone)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load timezone: %v", err)
		return
	}

	from, to, fromStr, toStr, err := exportDateRange(c, loc)
	if err != nil {
		c.String(http.StatusBadRequest, "%v", err)
		return
	}

	categories, err := exportCategories(ctx, db, userID, from, to)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load spending by category: %v", err)
		return
	}

	incl, hasIncl := c.GetQueryMap("incl")
	amt, hasAmt := c.GetQueryMap("amt")
	hasSelection := hasIncl || hasAmt

	var b strings.Builder
	b.WriteString("Category,Amount (EUR),Count\n")
	for _, cat := range categories {
		if hasSelection {
			if _, included := incl[cat.Category]; !included {
				continue
			}
		}
		amount := cat.TotalEUR
		if v, ok := amt[cat.Category]; ok {
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				amount = parsed
			}
		}
		fmt.Fprintf(&b, "%s,%.2f,%d\n", csvField(cat.Category), amount, cat.Count)
	}

	filename := fmt.Sprintf("spending-export-%s-%s.csv", fromStr, toStr)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "text/csv", []byte(b.String()))
}
