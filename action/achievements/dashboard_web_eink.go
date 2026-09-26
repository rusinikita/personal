// Achievements' e-ink dashboard: GET /web/achievements/eink, a dedicated, unauthenticated,
// fixed-viewport black-and-white/monospace page for a physical e-ink
// display, mirroring action/progress/dashboard_web.go's screenshot
// dashboard. Shows only the active-achievements tile grid (the same data
// BuildAchievementTiles/webui.RenderAchievementTiles produce for GET /web/achievements) — no
// Refresh form, no past/completed table.
package achievements

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"personal/action/webui"
	"personal/gateways"
)

const einkHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Achievements</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, monospace;
            background: #fff;
        }

        .dashboard {
            width: 100vw;
            height: 100vh;
            background: #fff;
            border: 2px solid #000;
            padding: 16px;
            overflow: hidden;
        }

        .page-title {
            font-size: 11px;
            font-weight: 600;
            letter-spacing: 0.5px;
            text-transform: uppercase;
            color: #000;
            margin-bottom: 12px;
        }

        .webui-achievement-tiles {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
            gap: 12px;
            align-content: start;
        }

        .webui-achievement-tile {
            border: 1px solid #000;
            border-radius: 8px;
            padding: 10px 12px;
        }

        .webui-achievement-tile--over {
            border-width: 2px;
        }

        .webui-achievement-tile header {
            font-size: 18px;
            font-weight: 600;
            color: #000;
            margin-bottom: 6px;
        }

        .webui-achievement-tile header a {
            color: #000;
            text-decoration: none;
        }

        .webui-achievement-tile progress {
            width: 100%;
            height: 10px;
            accent-color: #000;
            color: #000;
            margin-bottom: 6px;
        }

        .webui-achievement-tile-label {
            font-size: 15px;
            color: #000;
        }

        .webui-achievement-tile-deadline {
            font-size: 12px;
            color: #808080;
            margin-top: 2px;
        }

        .dashboard > p {
            font-size: 12px;
            color: #808080;
            padding: 8px 0;
        }
    </style>
</head>
<body>
    <div class="dashboard">
        <div class="page-title">Achievements</div>
        {{.Tiles}}
    </div>
</body>
</html>`

var einkTemplate = template.Must(template.New("achievementsEink").Parse(einkHTMLTemplate))

// EinkDashboardWebHandler renders GET /web/achievements/eink: every active achievement as
// a tile (progress bar + label + deadline), restyled black-and-white for a
// glanceable e-ink view. No Refresh form, no past/completed table — those
// stay exclusive to GET /web/achievements.
func EinkDashboardWebHandler(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(http.StatusInternalServerError, "Database not available")
		return
	}
	userID := webui.CurrentUserID(c)
	now := time.Now().UTC()

	tiles, err := BuildAchievementTiles(ctx, db, userID, now, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load achievements: %v", err)
		return
	}

	tilesHTML := webui.RenderAchievementTiles(webui.AchievementTilesData{Tiles: tiles, EmptyMessage: "No active achievements"})

	var buf strings.Builder
	if err := einkTemplate.Execute(&buf, struct{ Tiles template.HTML }{Tiles: tilesHTML}); err != nil {
		c.String(http.StatusInternalServerError, "Render error: %v", err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, buf.String())
}

// LegacyEinkRedirectWebHandler serves the pre-rename GET /web/goals/eink URL
// with a permanent redirect to GET /web/achievements/eink, since a physical
// e-ink display is still configured with the old address.
func LegacyEinkRedirectWebHandler(c *gin.Context) {
	target := "/web/achievements/eink"
	if c.Request.URL.RawQuery != "" {
		target += "?" + c.Request.URL.RawQuery
	}
	c.Redirect(http.StatusMovedPermanently, target)
}
