# Backlog

List of ideas for future work. Each idea must be turned into a feature document (`docs/functions/{subdomain}-spec.md`) before implementation starts, per the AI-Driven Development Convention.

Each item is headed by the date it was added (DD-MM-YY), not a sequence number — that way removing a done item never forces renumbering the rest. When one item depends on another, reference it by date + title.

## 20-08-26 — Authorization for web dashboards

A shared, cookie-based login for human browser access to the *new* `/web/*` pages (Money, Progress, Workouts, Navigation home page below) plus `/money/import`, replacing the inconsistent per-page auth that exists today: `/money/import` uses HTTP Basic Auth with its own `IMPORT_USERNAME`/`IMPORT_PASSWORD` env creds. MCP/agent access keeps using the existing full OAuth 2.1 flow (`action/auth/oauth.go`) — that's built for machine clients exchanging a bearer token, not a human logging in once in a browser.

**Out of scope:** `action/progress/dashboard_web.go` (served at `/web/progress`) is explicitly excluded — it's purpose-built for black-and-white screenshot/e-ink display and stays exactly as it is today, unauthenticated included. Do not add login to it, do not route it through the new middleware.

**Why:** The Money, Progress, and Workouts dashboards below all need a logged-in user to scope their data and gate access. Building one shared login now avoids inventing a fourth one-off auth scheme, and lets `/money/import` drop its current inconsistent handling.

**Use cases:**
- User should be able to log in via a login page — reusing the existing username/password credential model from `action/auth`'s `USERS` env var and login page style — and receive a signed JWT stored as an httpOnly cookie, instead of doing the full OAuth authorization-code exchange MCP clients use
- User should stay logged in across all *new* web dashboard pages (Money, Progress, Workouts) via that single cookie, without logging in separately per page
- User should be able to log out, clearing the session cookie
- Unauthenticated visitors to any *new* `/web/*` page should be redirected to the login page — this excludes the existing `/web/progress` screenshot dashboard, which stays unauthenticated and untouched
- The login form (a state-changing POST) should be protected against CSRF, since cookie-based auth is vulnerable to it in a way the existing header-based bearer token isn't
- `/money/import` should be migrated from its own Basic Auth onto this shared cookie-based auth, and `IMPORT_USERNAME`/`IMPORT_PASSWORD` retired
- Dashboards should read the logged-in user's ID from the authenticated session (the same way MCP already does via `gateways.WithUserID`) instead of hardcoded defaults like money's `defaultUserID = 1`
- The existing MCP OAuth flow (`action/auth/oauth.go`) stays as-is for agent/API clients — the cookie-based flow is additive for human browser access, sharing the same user store and JWT secret

## 20-08-26 — Navigation home page for web dashboards

One entry-point page listing and linking to each web dashboard section.

**Why:** There's currently no single place to land — `/web/progress` and `/money/import` are pages you have to already know the URL for. Adding Money and Workouts dashboards without a shared entry point repeats that problem two more times.

**Use cases:**
- User should be able to open one home page listing/linking to each dashboard section (Money, Progress, Workouts)
- User should be able to get back to the home page from within any dashboard section, via the shared nav from the design system item
- New web routes should follow one consistent URL namespace, e.g. `/web/money`, `/web/workouts`, `/web/import` — TODO: confirm naming, since `/money/import` currently lives outside the `/web` prefix. Note `/web/progress` is already taken by the existing untouched screenshot dashboard, so the new Progress page needs a different path, e.g. `/web/progress/browse`
- Home page should rely on the shared login rather than being separately gated

**Depends on:** 20-08-26 Authorization for web dashboards.

## 19-08-26 — Web interface for Money

**Depends on:** 20-08-26 Authorization for web dashboards, 20-08-26 Navigation home page for web dashboards.

A **read-only** web interface on top of the existing money functionality for reviewing balance, income/spending trends, and category breakdowns. Adding/editing/deleting transactions stays out of scope — bulk import already exists at `/money/import` (`action/money/import_web.go`), and this UI should link out to it rather than duplicate it.

**Why:** The money system is currently MCP/bot-only, so there's no way to see the overall financial picture (net worth trend, category weight, sync freshness) at a glance without asking the agent. A read-only dashboard makes that a one-look check.

**Use cases:**
- User should be able to see the date of the last transaction sync
- User should be able to view a table of categories sorted by average monthly spend over all time, with "total" and "last month" columns
- User should be able to drill into a category from the table and see its transactions
- User should be able to see total income earned over all time
- User should be able to see the current balance
- User should be able to see net (income minus spend) for the last calendar month
- User should be able to see average monthly savings (net) over all time
- User should be able to see a projected balance 3 months, 6 months, and 1 year out, based on the average monthly savings rate
- User should be able to navigate from the money dashboard to the existing transaction import page (`/money/import`)

## 19-08-26 — Web interface for Progress

**Depends on:** 20-08-26 Authorization for web dashboards, 20-08-26 Navigation home page for web dashboards.

A **new, separate, read-only** web page for browsing progress data, built in the new design system. It sits alongside — not instead of — the existing `/web/progress` dashboard (`action/progress/dashboard_web.go`), which is purpose-built for black-and-white screenshot/e-ink display and stays completely untouched: same route, same fixed 100vw/100vh layout, same top-5-only, same code. Do not edit `dashboard_web.go` as part of this item.

**Why:** The existing dashboard is intentionally optimized for a screenshot (fixed viewport, B&W, top-5-only) and must keep working that way for its purpose — but that also means it can't show every active activity, has no drill-down into a single project's history, and has no way to browse finished or not-yet-started projects. A separate, free-scrolling, color, full-list page adds a real browsing surface without compromising the screenshot dashboard.

**Use cases:**
- User should be able to view all active projects and habits, not just a top-5 subset (no fixed viewport/size limit, no black-and-white restriction)
- User should be able to navigate from the main view to a list of finished projects
- User should be able to navigate from the main view to a list of future (not-yet-started) projects
- User should be able to drill into a specific project and see its history of progress points, each with its note

## 19-08-26 — Web interface for Workouts

**Depends on:** 20-08-26 Authorization for web dashboards, 20-08-26 Navigation home page for web dashboards.

A **read-only** web interface on top of the workout functionality for reviewing personal records and per-exercise trends. Logging/editing workouts stays in the Telegram bot — this is view-only.

**Why:** Personal records and progression trends are hard to review in a chat interface. A read-only web UI gives a scannable table of records and a drill-down view for tracking progression on a specific exercise.

**Use cases:**
- User should be able to view a table of personal records, sorted by how many times each exercise has been performed
- User should be able to drill into a specific exercise from the table
- User should be able to view a trend chart of weight over time for the selected exercise
- User should be able to view a trend chart of reps over time for the selected exercise
