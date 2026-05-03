# Money Import — Dry Run

Tool for testing CSV parsing and merchant categorization without writing to the database.

## Run

```sh
go run tests/dry_run/dryrun.go <account> <path/to/file.csv>
```

Supported accounts: `revolut`, `bank of cyprus` (aliases: `bankofcyprus`, `boc`).

Example:
```sh
go run tests/dry_run/dryrun.go "bank of cyprus" tests/dry_run/tmp/statement.csv
```

Put CSV files in `tests/dry_run/tmp/` — that folder is gitignored.

## Output

The script prints three sections:

1. **Transaction list** — one row per transaction: date, type, currency, amount, recognized merchant, inferred category
2. **Category totals** — sum of EUR amounts per category
3. **Uncategorized breakdown** — merchants that got no category, ranked by total amount

## What to debug for Bank of Cyprus

BOC exports have quirks handled in `action/money_import/parser.go` → `BankOfCyprusParser`:

- **UTF-8 BOM** at file start — stripped automatically
- **Metadata rows** before the actual header — parser scans for the row starting with `"Date"`
- **European decimal format** — `"1.234,56"` → `1234.56` (period = thousands, comma = decimal)
- **Debit/Credit columns** instead of a single Amount — debit becomes negative (expense), credit becomes positive (income)
- **Currency column** — may be empty, defaults to `"EUR"`

If the import produces 0 transactions, check:
1. Does the file have a BOM? (`xxd file.csv | head -1` — look for `ef bb bf`)
2. Does any row start exactly with `"Date"` in the first column?
3. Are Debit/Credit column names spelled exactly as expected (check `firstOf(idx, "Debit")` call)?

## Adding merchant rules

All rules are in `action/money_import/parser.go`:

- **`knownMerchants`** — maps description substring → clean merchant name. Add here when the fallback (first 2 words of description) gives a messy result.
- **`categoryRules`** — maps description/merchant substring → category path. Order matters: first match wins. Put specific rules (person names) before generic ones.
- **`typeOverrides`** — forces transaction type (`expense`/`income`/`transfer`) based on description substring. Used for top-ups and similar patterns where sign alone isn't enough.

After editing rules, re-run the dry run to see the effect — no rebuild needed.
