package money_import

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"personal/domain"
	"personal/gateways"
)

const defaultUserID int64 = 1

// BasicAuthMiddleware checks HTTP Basic Auth credentials from env.
// Expected env vars: IMPORT_USERNAME, IMPORT_PASSWORD.
func BasicAuthMiddleware(username, password string) gin.HandlerFunc {
	return gin.BasicAuth(gin.Accounts{username: password})
}

const importFormHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Money Import</title>
<style>
body { font-family: monospace; max-width: 600px; margin: 40px auto; padding: 0 20px; }
h1 { font-size: 18px; margin-bottom: 24px; }
label { display: block; margin-bottom: 6px; font-size: 13px; font-weight: bold; }
input[type=text], input[type=file], select {
    display: block; width: 100%; padding: 8px; margin-bottom: 16px;
    border: 1px solid #ccc; font-family: monospace; font-size: 13px; box-sizing: border-box;
}
button { padding: 8px 20px; font-family: monospace; font-size: 13px; cursor: pointer; }
.result { margin-top: 24px; padding: 12px; border: 1px solid #000; font-size: 13px; white-space: pre-wrap; }
.error { border-color: red; color: red; }
</style>
</head>
<body>
<h1>💰 Bank CSV Import</h1>
<form method="POST" enctype="multipart/form-data">
    <label>Account name:</label>
    <select name="account" required>
        <option value="">— select account —</option>
        <option value="revolut">Revolut</option>
        <option value="bank of cyprus">Bank of Cyprus</option>
    </select>

    <label>CSV file:</label>
    <input type="file" name="file" accept=".csv" required>

    <button type="submit">Import</button>
</form>
{{if .Message}}
<div class="result{{if .IsError}} error{{end}}">{{.Message}}</div>
{{end}}
</body>
</html>`

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

	renderImportPage(c, importPageData{
		Message: fmt.Sprintf(
			"✅ imported %d transactions, skipped %d\nlast imported: %s — %s (%.2f %s)",
			len(saved), skipped,
			saved[len(saved)-1].TransactedAt.Format(time.DateOnly),
			saved[len(saved)-1].Merchant,
			saved[len(saved)-1].AmountOriginal,
			saved[len(saved)-1].Currency,
		),
	})
}

func renderImportPage(c *gin.Context, data importPageData) {
	tmpl, err := template.New("import").Parse(importFormHTML)
	if err != nil {
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, buf.String())
}
