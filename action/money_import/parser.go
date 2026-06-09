package money_import

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// RawTransaction is the normalized output of any account-specific CSV parser.
type RawTransaction struct {
	Date        time.Time
	Description string
	Amount      float64 // positive = income, negative = expense
	Currency    string
}

// Parser parses a bank CSV export into raw transactions.
type Parser interface {
	Parse(r io.Reader) ([]RawTransaction, error)
}

// ParserFor returns the parser for the given account name.
// Returns nil if account is unknown.
func ParserFor(account string) Parser {
	switch strings.ToLower(strings.TrimSpace(account)) {
	case "revolut":
		return &RevolutParser{}
	case "bank of cyprus", "bankofcyprus", "boc":
		return &BankOfCyprusParser{}
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// Revolut parser
// ---------------------------------------------------------------------------

// RevolutParser parses Revolut CSV exports.
// Expected columns (v10+):
// Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
type RevolutParser struct{}

func (p *RevolutParser) Parse(r io.Reader) ([]RawTransaction, error) {
	records, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("revolut: csv read error: %w", err)
	}
	if len(records) < 2 {
		return nil, nil
	}

	header := records[0]
	idx := csvIndex(header)

	descCol := firstOf(idx, "Description")
	amtCol := firstOf(idx, "Amount")
	curCol := firstOf(idx, "Currency")
	dateCol := firstOf(idx, "Completed Date", "Started Date")
	stateCol := firstOf(idx, "State")

	var result []RawTransaction
	for _, row := range records[1:] {
		if len(row) == 0 {
			continue
		}
		// Skip pending/failed rows; empty state = no filter.
		if stateCol >= 0 && stateCol < len(row) {
			state := strings.ToLower(strings.TrimSpace(row[stateCol]))
			if state != "" && state != "completed" {
				continue
			}
		}

		date, err := parseDate(safeGet(row, dateCol))
		if err != nil {
			continue // skip unparseable rows
		}

		amtStr := strings.ReplaceAll(safeGet(row, amtCol), ",", "")
		amt, err := strconv.ParseFloat(strings.TrimSpace(amtStr), 64)
		if err != nil {
			continue
		}
		if amt == 0 {
			continue
		}

		result = append(result, RawTransaction{
			Date:        date,
			Description: strings.TrimSpace(safeGet(row, descCol)),
			Amount:      amt,
			Currency:    strings.TrimSpace(safeGet(row, curCol)),
		})
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Bank of Cyprus parser
// ---------------------------------------------------------------------------

// BankOfCyprusParser parses Bank of Cyprus CSV exports.
// Expected columns:
// Date,Description,Debit,Credit,Currency,Balance
type BankOfCyprusParser struct{}

func (p *BankOfCyprusParser) Parse(r io.Reader) ([]RawTransaction, error) {
	// Strip UTF-8 BOM present in BOC exports.
	br := bufio.NewReader(r)
	if bom, _ := br.Peek(3); len(bom) == 3 && bom[0] == 0xEF && bom[1] == 0xBB && bom[2] == 0xBF {
		_, _ = br.Discard(3)
	}

	records, err := csv.NewReader(br).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("boc: csv read error: %w", err)
	}

	// Real BOC exports have ~5 metadata rows before the "Date,..." header row.
	headerIdx := -1
	for i, row := range records {
		if len(row) > 0 && strings.TrimSpace(row[0]) == "Date" {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 || headerIdx+1 >= len(records) {
		return nil, nil
	}

	header := records[headerIdx]
	idx := csvIndex(header)

	descCol := firstOf(idx, "Description", "Narrative")
	debitCol := firstOf(idx, "Debit")
	creditCol := firstOf(idx, "Credit")
	curCol := firstOf(idx, "Currency")
	dateCol := firstOf(idx, "Date", "Value Date", "Transaction Date")

	var result []RawTransaction
	for _, row := range records[headerIdx+1:] {
		if len(row) == 0 {
			continue
		}

		date, err := parseDate(safeGet(row, dateCol))
		if err != nil {
			continue
		}

		var amount float64
		// BOC uses European decimal format: "1.234,56" means 1234.56.
		debitStr := parseEuropeanAmount(safeGet(row, debitCol))
		creditStr := parseEuropeanAmount(safeGet(row, creditCol))

		if debitStr != "" && debitStr != "0" && debitStr != "0.00" {
			v, err := strconv.ParseFloat(debitStr, 64)
			if err == nil {
				amount = -v // debit = expense
			}
		} else if creditStr != "" && creditStr != "0" && creditStr != "0.00" {
			v, err := strconv.ParseFloat(creditStr, 64)
			if err == nil {
				amount = v // credit = income
			}
		}

		if amount == 0 {
			continue
		}

		currency := strings.TrimSpace(safeGet(row, curCol))
		if currency == "" {
			currency = "EUR"
		}

		result = append(result, RawTransaction{
			Date:        date,
			Description: strings.TrimSpace(safeGet(row, descCol)),
			Amount:      amount,
			Currency:    currency,
		})
	}
	return result, nil
}

// parseEuropeanAmount normalizes BOC-style amounts: "1.234,56" → "1234.56".
// Period is thousands separator, comma is decimal separator.
func parseEuropeanAmount(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	return s
}

// ---------------------------------------------------------------------------
// Merchant recognition
// ---------------------------------------------------------------------------

// knownMerchants maps lowercased substrings to clean merchant names.
var knownMerchants = []struct {
	keyword  string
	merchant string
}{
	{"revpoints spare change", "Revolut Rounding"},
	{"top-up by", "Bank of Cyprus"},
	{"yandex cafe", "Yandex Cafe"},
	{"yandex.taxi", "Yandex Taxi"},
	{"yango", "Yandex Taxi"},
	{"yandex", "Yandex"},
	{"mms", "MMS"},
	{"the melting pot", "The Melting Pot"},
	{"buffalo wings", "Buffalo Wings"},
	{"starbucks", "Starbucks"},
	{"costa coffee", "Costa Coffee"},
	{"costa", "Costa Coffee"},
	{"lidl", "Lidl"},
	{"carrefour", "Carrefour"},
	{"bolt", "Bolt"},
	{"uber eats", "Uber Eats"},
	{"uber", "Uber"},
	{"wolt", "Wolt"},
	{"amazon", "Amazon"},
	{"apple", "Apple"},
	{"netflix", "Netflix"},
	{"spotify", "Spotify"},
	{"zuma", "Zuma"},
	{"ikea", "IKEA"},
	{"h&m", "H&M"},
	{"zara", "Zara"},
	{"mcdonald", "McDonald's"},
	{"kfc", "KFC"},
	{"burger king", "Burger King"},
	{"subway", "Subway"},
	{"papa john", "Papa John's"},
	{"domino", "Domino's"},
	{"circle k", "Circle K"},
	{"bp ", "BP"},
	{"shell", "Shell"},
	{"total ", "TotalEnergies"},
}

// RecognizeMerchant extracts a clean merchant name from a raw description.
func RecognizeMerchant(description string) string {
	lower := strings.ToLower(description)
	for _, km := range knownMerchants {
		if strings.Contains(lower, km.keyword) {
			return km.merchant
		}
	}
	// Fallback: take first meaningful word(s)
	parts := strings.Fields(description)
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts[:min(2, len(parts))], " ")
}

// ---------------------------------------------------------------------------
// Category inference
// ---------------------------------------------------------------------------

// categoryRules maps lowercased merchant/description keywords to categories.
var categoryRules = []struct {
	keyword  string
	category string
}{
	{"revpoints spare", "finance/rev_rounding"},
	{"top-up by", "transfer/topup"},
	{"maria ochirova", "housing/rent"},
	{"mariia kruglova", "transfer/to_masha"},
	{"maria sofokleous", "education/driving"},
	{"phivos charalambous", "education/driving"},
	{"christakis christoforou", "education/driving"},
	{"starbucks", "food/cafe"},
	{"costa", "food/cafe"},
	{"lidl", "groceries"},
	{"carrefour", "groceries"},
	{"supermarket", "groceries"},
	{"grocery", "groceries"},
	{"bolt", "transport/taxi"},
	{"electra", "transport/scootersharing"},
	{"ridenow", "transport/carsharing"},
	{"uber eats", "food/delivery"},
	{"uber", "transport/taxi"},
	{"wolt", "food/delivery"},
	{"amazon", "shopping/online"},
	{"netflix", "entertainment/streaming"},
	{"spotify", "entertainment/streaming"},
	{"youtube", "services/youtube"},
	{"google", "services/subscriptions"},
	{"claude", "services/ai"},
	{"anthropic", "services/ai"},
	{"linode", "services/hosting"},
	{"akamai", "services/hosting"},
	{"neon.tech", "services/hosting"},
	{"apple", "shopping/digital"},
	{"zuma", "food/restaurant"},
	{"mcdonald", "food/fast_food"},
	{"mc donald", "food/fast_food"},
	{"kfc", "food/fast_food"},
	{"burger king", "food/fast_food"},
	{"subway", "food/fast_food"},
	{"domino", "food/delivery"},
	{"papa john", "food/delivery"},
	{"ikea", "shopping/home"},
	{"h&m", "shopping/clothes"},
	{"zara", "shopping/clothes"},
	{"bfj", "shopping/clothes"},
	{"ecco", "shopping/clothes"},
	{"circle k", "transport/fuel"},
	{"bp ", "transport/fuel"},
	{"shell", "transport/fuel"},
	{"total ", "transport/fuel"},
	{"pharmacy", "health"},
	{"doctor", "health"},
	{"hospital", "health"},
	{"gym", "health/sport"},
	{"fitness", "health/sport"},
	{"salary", "salary"},
	{"payroll", "salary"},
	{"rent", "housing/rent"},
	{"electric", "housing/utilities"},
	{"water bill", "housing/utilities"},
	{"internet", "housing/internet"},
	// Cafes & coffee
	{"yandex cafe", "food/cafe"},
	{"uluwatu", "food/cafe"},
	{"tamper", "food/cafe"},
	{"paradosiaki", "food/cafe"},
	{"cafe toucan", "food/cafe"},
	{"wagmi", "food/cafe"},
	{"nomad bread", "food/cafe"},
	{"deja brew", "food/cafe"},
	{"javion", "food/cafe"},
	{"kiku", "food/cafe"},
	{"bean bar", "food/cafe"},
	{"blend coffee", "food/cafe"},
	{"lula coffee", "food/cafe"},
	{"nutry", "food/cafe"},
	{"t lounge", "food/cafe"},
	{"java lounge", "food/cafe"},
	{"intercaff", "food/cafe"},
	{"aroma", "food/cafe"},
	{"artist specialty", "food/cafe"},
	{"outpost lanka", "food/cafe"},
	{"cafe kumbuk", "food/cafe"},
	{"lolami", "food/cafe"},
	{"tziamouda", "food/cafe"},
	{"the melting", "food/cafe"},
	{"franz by", "food/cafe"},
	{"iyers", "food/cafe"},
	{"evgeniou grains", "food/cafe"},
	{"lucky's", "food/cafe"},
	{"food for", "food/cafe"},
	{"buffalo wings", "food/cafe"},
	{"nuovo caf", "food/cafe"},
	// Restaurants
	{"thymari", "food/restaurant"},
	{"ocean basket", "food/restaurant"},
	{"tasters", "food/restaurant"},
	{"malindi", "food/restaurant"},
	{"elefante", "food/restaurant"},
	{"crispy duck", "food/restaurant"},
	{"wagamama", "food/restaurant"},
	{"smash burger", "food/restaurant"},
	{"pan orient", "food/restaurant"},
	{"libabon", "food/restaurant"},
	{"manoushe", "food/restaurant"},
	{"street dogs", "food/restaurant"},
	{"submarines by", "food/restaurant"},
	{"pokeloha", "food/restaurant"},
	{"potato king", "food/restaurant"},
	{"restoran brankovina", "food/restaurant"},
	{"ristorante bella", "food/restaurant"},
	{"tt bistro", "food/restaurant"},
	{"the dutchman", "food/restaurant"},
	{"kafana", "food/restaurant"},
	{"kai beach", "food/restaurant"},
	{"berezka", "food/restaurant"},
	{"factory kitchen", "food/restaurant"},
	{"barel", "food/restaurant"},
	{"indian street", "food/restaurant"},
	{"just beer", "food/bar"},
	// Bakery
	{"koulouromag", "food/bakery"},
	{"koulourades", "food/bakery"},
	// Ice cream
	{"oeskimo", "food/icecream"},
	// Food delivery
	{"glovo", "food/delivery"},
	// Groceries
	{"sklavenitis", "groceries"},
	{"papanicolaou", "groceries"},
	{"alphamega", "groceries"},
	{"cargills", "groceries"},
	{"freshmart", "groceries"},
	{"global foodcity", "groceries"},
	{"limassol agora", "groceries"},
	{"nour daily", "groceries"},
	{"nour fresh", "groceries"},
	{"tharanga", "groceries"},
	{"urban fresh", "groceries"},
	{"mms", "groceries"},
	// Transport
	{"yandex.taxi", "transport/taxi"},
	{"yango", "transport/taxi"},
	{"yandex", "transport/taxi"},
	{"eko", "transport/fuel"},
	{"omv", "transport/fuel"},
	// Shopping
	{"sports direct", "shopping/sport"},
	{"fat burner", "shopping/sport"},
	{"superhome", "shopping/home"},
	{"lilly drog", "shopping/beauty"},
	{"cyprus duty", "shopping"},
	{"kelly's", "shopping"},
	{"olympus plaza", "shopping"},
	// Personal care
	{"oldboy", "personal_care"},
	// Travel
	{"premier inn", "travel/hotel"},
	{"soul temple", "travel/hotel"},
	{"weligama", "travel/hotel"},
	{"lm botanique", "travel/hotel"},
	{"astry", "travel/hotel"},
	{"airbnb", "travel/hotel"},
	{"bandaranaike", "travel/airport"},
	{"k-eta", "travel/visa"},
	{"papadopoulos dimitrios", "housing/rent"},
	{"papandopolous", "housing/rent"},
	{"qatar", "travel/flight"},
	{"kiwi.com", "travel/flight"},
	{"wizz", "travel/flight"},
	{"airarabia", "travel/flight"},
	{"air arabia", "travel/flight"},
	// Utilities & finance
	{"primetel", "housing/utilities"},
	{"waterboard", "housing/utilities"},
	{"ibu-maintenance", "housing/utilities"},
	{"revolut", "transfer/topup"},
	{"atm cash", "transfer/cash"},
	// Cafes
	{"stories", "food/cafe"},
	{"a2mnomad", "food/cafe"},
	{"a2m nomad", "food/cafe"},
	// Country catch-alls (must be last — overridden by more specific rules above)
	{"ge ", "travel/georgia"},
	{"lk ", "travel/srilanka"},
	{"ae ", "travel/uae"},
	{"tips out transfer fees", "finance/fees"},
	{"transfer commission", "finance/bank-fees"},
	{"processing fees", "finance/bank-fees"},
	{"opo trnsfr", "finance/bank-fees"},
	{"o.p.o", "finance/bank-fees"},
}

// typeOverrides maps lowercased description keywords to a forced transaction type.
var typeOverrides = []struct {
	keyword string
	txType  string
}{
	{"top-up by", "transfer"},
	{"revolut", "transfer"},
	{"atm cash", "transfer"},
}

// InferTypeOverride returns a forced transaction type for known description patterns.
// Returns empty string if no override applies — caller keeps the default type.
func InferTypeOverride(description string) string {
	lower := strings.ToLower(description)
	for _, rule := range typeOverrides {
		if strings.Contains(lower, rule.keyword) {
			return rule.txType
		}
	}
	return ""
}

// InferCategory infers a category from merchant name and raw description.
func InferCategory(merchant, description string) string {
	combined := strings.ToLower(merchant + " " + description)
	for _, rule := range categoryRules {
		if strings.Contains(combined, rule.keyword) {
			return rule.category
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func csvIndex(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, h := range header {
		m[strings.TrimSpace(h)] = i
	}
	return m
}

func firstOf(idx map[string]int, cols ...string) int {
	for _, col := range cols {
		if i, ok := idx[col]; ok {
			return i
		}
	}
	return -1
}

func safeGet(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}

var dateFormats = []string{
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05",
	"2006-01-02",
	"02/01/2006",
	"01/02/2006",
	"02-01-2006",
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	for _, layout := range dateFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse date: %q", s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type Order struct {
	StartTS int64
	EndTS   int64
}

type Aggregation struct {
	StartTS             int64
	EndTS               int64
	ParallelOrdersCount int
}

func MakeTimeline(orders []Order) (timeline []Aggregation) {
	return timeline
}
