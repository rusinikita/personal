package money

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

const defaultUserID int64 = 1

// BasicAuthMiddleware checks HTTP Basic Auth credentials from env.
// Expected env vars: IMPORT_USERNAME, IMPORT_PASSWORD.
func BasicAuthMiddleware(username, password string) gin.HandlerFunc {
	return gin.BasicAuth(gin.Accounts{username: password})
}

// importFormContentSrc is embedded into the shared webui.RenderPage shell —
// it owns no <html>/<head>/<style> of its own, just the form and result
// message. Layout/typography come from Pico CSS via the shell.
const importFormContentSrc = `<h1>💰 Bank CSV Import</h1>
<form method="POST" enctype="multipart/form-data">
    <label for="account">Account name</label>
    <select id="account" name="account" required>
        <option value="">— select account —</option>
        <option value="revolut">Revolut</option>
        <option value="bank of cyprus">Bank of Cyprus</option>
    </select>

    <label for="file">CSV file</label>
    <input type="file" id="file" name="file" accept=".csv" required>

    <button type="submit">Import</button>
</form>
{{if .Message}}
<pre{{if .IsError}} style="color: var(--pico-del-color)"{{end}}>{{.Message}}</pre>
{{end}}`

var importFormTemplate = template.Must(template.New("importForm").Parse(importFormContentSrc))

type importPageData struct {
	Message string
	IsError bool
}

// ImportGETHandler renders the CSV upload form.
func ImportGETHandler(c *gin.Context) {
	renderImportPage(c, importPageData{})
}

// ImportPOSTHandler processes the uploaded CSV file.
func ImportPOSTHandler(c *gin.Context) {
	db := gateways.DBFromContext(c.Request.Context())
	if db == nil {
		renderImportPage(c, importPageData{Message: "database not available", IsError: true})
		return
	}

	account := strings.TrimSpace(c.PostForm("account"))
	if account == "" {
		renderImportPage(c, importPageData{Message: "account name is required", IsError: true})
		return
	}

	parser := ParserFor(account)
	if parser == nil {
		renderImportPage(c, importPageData{
			Message: fmt.Sprintf("unknown account %q — supported: Revolut, Bank of Cyprus", account),
			IsError: true,
		})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		renderImportPage(c, importPageData{Message: "file is required", IsError: true})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		renderImportPage(c, importPageData{Message: "cannot open file: " + err.Error(), IsError: true})
		return
	}
	defer file.Close()

	// Stage 1 — parse CSV.
	rawTxs, err := parser.Parse(file)
	if err != nil {
		renderImportPage(c, importPageData{Message: "parse error: " + err.Error(), IsError: true})
		return
	}

	if len(rawTxs) == 0 {
		renderImportPage(c, importPageData{Message: "no transactions found in file", IsError: true})
		return
	}

	// Stages 2 & 3 — enrich and build domain transactions.
	domainTxs := make([]*domain.Transaction, 0, len(rawTxs))
	skipped := 0

	for _, raw := range rawTxs {
		if raw.Amount == 0 {
			skipped++
			continue
		}

		// Stage 2: merchant recognition.
		origDesc := raw.Description
		merchant := RecognizeMerchant(origDesc)

		// Stage 3: category inference.
		category := InferCategory(merchant, origDesc)
		if category == "" {
			category = "uncategorized"
		}

		// Determine type.
		txType := domain.TransactionTypeExpense
		amt := math.Abs(raw.Amount)
		if override := InferTypeOverride(raw.Description); override != "" {
			txType = domain.TransactionType(override)
		} else if raw.Amount > 0 {
			txType = domain.TransactionTypeIncome
		}

		// amount_eur = original if EUR, else 0 (manual correction later).
		amountEUR := amt
		if raw.Currency != "EUR" {
			amountEUR = 0
			skipped++ // non-EUR without conversion — skip for now
			continue
		}

		domainTxs = append(domainTxs, &domain.Transaction{
			UserID:              defaultUserID,
			Type:                txType,
			AmountOriginal:      amt,
			Currency:            raw.Currency,
			AmountEUR:           amountEUR,
			Account:             account,
			Category:            category,
			Merchant:            merchant,
			OriginalDescription: &origDesc,
			IdempotencyKey:      buildIdempotencyKey(parser, raw),
			TransactedAt:        raw.Date,
		})
	}

	if len(domainTxs) == 0 {
		renderImportPage(c, importPageData{
			Message: fmt.Sprintf("imported 0, skipped %d (no importable rows)", skipped),
			IsError: true,
		})
		return
	}

	ctx := c.Request.Context()
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		userID = defaultUserID
	}
	// Rewrite user_id on all prepared transactions.
	for _, tx := range domainTxs {
		tx.UserID = userID
	}

	saved, err := db.AddTransactions(ctx, domainTxs)
	if err != nil {
		renderImportPage(c, importPageData{Message: "database error: " + err.Error(), IsError: true})
		return
	}

	duplicates := len(domainTxs) - len(saved)

	if len(saved) == 0 {
		renderImportPage(c, importPageData{
			Message: fmt.Sprintf("✅ imported 0, skipped %d (invalid), %d duplicates", skipped, duplicates),
		})
		return
	}

	renderImportPage(c, importPageData{
		Message: fmt.Sprintf(
			"✅ imported %d, skipped %d (invalid), %d duplicates\nlast imported: %s — %s (%.2f %s)",
			len(saved), skipped, duplicates,
			saved[len(saved)-1].TransactedAt.Format(time.DateOnly),
			saved[len(saved)-1].Merchant,
			saved[len(saved)-1].AmountOriginal,
			saved[len(saved)-1].Currency,
		),
	})
}

// buildIdempotencyKey derives a stable dedup key for a raw transaction so
// re-importing the same CSV export skips rows already stored in the DB.
// The unique index is scoped to (user_id, account, idempotency_key), so the
// key itself only needs to be unique within one account's export.
// Returns nil when there isn't enough stable data to key on (caller then
// stores no key, and the row is never treated as a duplicate).
func buildIdempotencyKey(parser Parser, raw RawTransaction) *string {
	switch parser.(type) {
	case *RevolutParser:
		key := fmt.Sprintf("%s:%.2f", raw.Date.UTC().Format(time.RFC3339), raw.Amount)
		return &key
	case *BankOfCyprusParser:
		if raw.Reference == "" {
			return nil
		}
		key := fmt.Sprintf("ref:%s", raw.Reference)
		return &key
	default:
		return nil
	}
}

func renderImportPage(c *gin.Context, data importPageData) {
	var content strings.Builder
	if err := importFormTemplate.Execute(&content, data); err != nil {
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:   "Money Import",
		Content: template.HTML(content.String()),
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}
