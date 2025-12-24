package progress

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Progress Dashboard</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Noto+Emoji&display=swap" rel="stylesheet">
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, monospace;
            background: #fff;
            margin: 0;
            padding: 0;
        }
        
        .dashboard {
            width: 100vw;
            height: 100vh;
            background: #fff;
            border: 2px solid #000;
            display: flex;
            flex-direction: column;
            overflow: hidden;
        }
        
        /* Overall Streak Section */
        .overall-streak {
            padding: 12px 16px;
            border-bottom: 2px solid #000;
        }
        
        .section-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 8px;
        }
        
        .section-title {
            font-size: 11px;
            font-weight: 600;
            letter-spacing: 0.5px;
            text-transform: uppercase;
            color: #000;
        }
        
        .streak-stats {
            display: flex;
            gap: 16px;
            font-size: 11px;
            color: #808080;
        }
        
        .streak-stats strong {
            color: #000;
        }
        
        .streak-grid {
            display: flex;
            gap: 3px;
        }
        
        .streak-cell {
            width: 18px;
            height: 18px;
            border: 1px solid #808080;
            background: #fff;
        }

        .streak-cell.active {
            background: #000;
            border-color: #000;
        }

        .week-separator {
            width: 2px;
            background: #808080;
            margin: 0 6px;
            position: relative;
        }
        
        /* Activities Section */
        .activities {
            flex: 1;
            display: flex;
            flex-direction: column;
        }
        
        .activity-row {
            padding: 10px 16px;
            border-bottom: 1px solid #808080;
            display: flex;
            flex-direction: column;
            gap: 5px;
        }
        
        .activity-row:last-child {
            border-bottom: none;
        }
        
        .activity-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        
        .activity-info {
            display: flex;
            align-items: baseline;
            gap: 6px;
        }
        
        .activity-name {
            font-size: 14px;
            font-weight: 500;
            color: #000;
        }
        
        .activity-freq {
            font-size: 11px;
            color: #808080;
        }

        .activity-ago {
            font-size: 11px;
            color: #808080;
        }

        .activity-ago.stale {
            color: #808080;
        }
        
        .activity-ago.warning {
            color: #000;
            font-weight: 700;
        }
        
        .emoji-grid {
            display: flex;
            gap: 2px;
        }
        
        .emoji-cell {
            width: 26px;
            height: 26px;
            border: 1px solid #808080;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 15px;
            background: #fff;
            font-family: 'Noto Emoji', sans-serif;
            color: #000;
            font-weight: 900;
            filter: contrast(2);
        }
        
        .emoji-cell.empty {
            background: #fff;
        }

        .emoji-cell.today {
            border: 2px solid #000;
        }
    </style>
</head>
<body>
    <div class="dashboard">
        <!-- Overall Streak Section -->
        <div class="overall-streak">
            <div class="section-header">
                <div class="section-title">Overall Streak · 30 days</div>
                <div class="streak-stats">
                    <span>streak: <strong>{{.CurrentStreak}}d</strong></span>
                    <span>avg gap: <strong>{{.AvgGapMonth}}</strong> (mo) · <strong>{{.AvgGapWeek}}</strong> (wk)</span>
                </div>
            </div>
            <div class="streak-grid">
                {{- range $i, $day := .StreakDays -}}
                <div class="streak-cell{{if $day.Active}} active{{end}}"></div>
                {{- if and (eq (mod (add $i 1) 7) 0) (ne (add $i 1) (len $.StreakDays)) -}}
                <div class="week-separator"></div>
                {{- end -}}
                {{- end -}}
            </div>
        </div>
        
        <!-- Activities Section -->
        <div class="activities">
            {{- range .Activities}}
            <div class="activity-row">
                <div class="activity-header">
                    <div class="activity-info">
                        <span class="activity-name">{{.Name}}</span>
                        <span class="activity-freq">· {{.Frequency}}</span>
                    </div>
                    <span class="activity-ago{{if .StalenessClass}} {{.StalenessClass}}{{end}}">{{.TimeAgo}}</span>
                </div>
                <div class="emoji-grid">
                    {{- range $i, $p := .ProgressCells -}}
                    <div class="emoji-cell{{if not $p.Emoji}} empty{{end}}{{if $p.IsToday}} today{{end}}">{{$p.Emoji}}</div>
                    {{- end -}}
                </div>
            </div>
            {{- end}}
        </div>
    </div>
</body>
</html>`

// Emoji mappings by progress type
var emojiMappings = map[string]map[int]string{
	"mood": {
		2:  "☀️",
		1:  "⛅",
		0:  "☁️",
		-1: "🌧️",
		-2: "⛈️",
	},
	"habit_progress": {
		2:  "💪",
		1:  "👍",
		0:  "🤔",
		-1: "😔",
		-2: "❌",
	},
	"project_progress": {
		2:  "🚀",
		1:  "➡️",
		0:  "⏸️",
		-1: "↩️",
		-2: "🔄",
	},
	"promise_state": {
		1:  "✅",
		0:  "💭",
		-1: "🤷",
	},
}

type StreakDay struct {
	Active bool
}

type ProgressCell struct {
	Emoji   string
	IsToday bool
}

type Activity struct {
	Name          string
	Frequency     string
	FrequencyDays int
	ProgressType  string
	LastCheckIn   time.Time
	Progress      []*int // nil means no check-in
}

type ActivityView struct {
	Name           string
	Frequency      string
	TimeAgo        string
	StalenessClass string
	ProgressCells  []ProgressCell
}

type DashboardData struct {
	CurrentStreak int
	AvgGapMonth   string
	AvgGapWeek    string
	StreakDays    []StreakDay
	Activities    []ActivityView
}

func intPtr(v int) *int {
	return &v
}

func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)
	hours := int(diff.Hours())
	days := hours / 24

	if hours < 1 {
		return "now"
	}
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dd", days)
}

func getStalenessClass(lastCheckIn time.Time, frequencyDays int) string {
	diffDays := time.Since(lastCheckIn).Hours() / 24
	if diffDays > float64(frequencyDays*2) {
		return "warning"
	}
	return ""
}

func getEmoji(progressType string, value *int) string {
	if value == nil {
		return ""
	}
	if mapping, ok := emojiMappings[progressType]; ok {
		if emoji, ok := mapping[*value]; ok {
			return emoji
		}
	}
	return ""
}

func buildActivityView(a Activity) ActivityView {
	cells := make([]ProgressCell, len(a.Progress))
	for i, v := range a.Progress {
		cells[i] = ProgressCell{
			Emoji:   getEmoji(a.ProgressType, v),
			IsToday: i == 0, // First cell (most recent) is marked as today
		}
	}

	return ActivityView{
		Name:           a.Name,
		Frequency:      a.Frequency,
		TimeAgo:        formatTimeAgo(a.LastCheckIn),
		StalenessClass: getStalenessClass(a.LastCheckIn, a.FrequencyDays),
		ProgressCells:  cells,
	}
}

func getDemoData() DashboardData {
	now := time.Now()

	// Demo activities
	activities := []Activity{
		{
			Name:          "Зарядка",
			Frequency:     "daily",
			FrequencyDays: 1,
			ProgressType:  "habit_progress",
			LastCheckIn:   now.Add(-2 * time.Hour),
			Progress:      []*int{intPtr(2), intPtr(1), intPtr(2), intPtr(1), nil, intPtr(2), intPtr(2), nil, nil, intPtr(1), intPtr(2), intPtr(2), intPtr(0), intPtr(1)},
		},
		{
			Name:          "Доволен прожитыми днями?",
			Frequency:     "weekly",
			FrequencyDays: 7,
			ProgressType:  "mood",
			LastCheckIn:   now.Add(-3 * 24 * time.Hour),
			Progress:      []*int{intPtr(2), intPtr(1), intPtr(2), intPtr(0), intPtr(1), intPtr(2), intPtr(1), intPtr(0), intPtr(2), intPtr(1)},
		},
		{
			Name:          "Trainer V2",
			Frequency:     "weekly",
			FrequencyDays: 7,
			ProgressType:  "project_progress",
			LastCheckIn:   now.Add(-5 * 24 * time.Hour),
			Progress:      []*int{intPtr(2), intPtr(1), intPtr(0), intPtr(1), intPtr(2), intPtr(1), intPtr(0), intPtr(1)},
		},
		{
			Name:          "Статья для inDrive",
			Frequency:     "5d",
			FrequencyDays: 5,
			ProgressType:  "promise_state",
			LastCheckIn:   now.Add(-8 * 24 * time.Hour),
			Progress:      []*int{intPtr(1), intPtr(0), intPtr(0), intPtr(-1), intPtr(1), intPtr(0), intPtr(1)},
		},
		{
			Name:          "Разговаривать с мамой",
			Frequency:     "4d",
			FrequencyDays: 4,
			ProgressType:  "habit_progress",
			LastCheckIn:   now.Add(-1 * 24 * time.Hour),
			Progress:      []*int{intPtr(2), intPtr(1), intPtr(2), intPtr(1), intPtr(2), intPtr(1), intPtr(2), intPtr(1)},
		},
	}

	// Build activity views
	activityViews := make([]ActivityView, len(activities))
	for i, a := range activities {
		activityViews[i] = buildActivityView(a)
	}

	// Demo streak pattern (30 days)
	streakPattern := []bool{false, false, true, true, true, false, false, true, true, true, true, true, false, true, true, true, true, false, false, true, true, true, true, true, true, false, true, true, true, true}
	streakDays := make([]StreakDay, len(streakPattern))
	for i, active := range streakPattern {
		streakDays[i] = StreakDay{Active: active}
	}

	// Calculate current streak (count from end)
	currentStreak := 0
	for i := len(streakPattern) - 1; i >= 0; i-- {
		if streakPattern[i] {
			currentStreak++
		} else {
			break
		}
	}

	return DashboardData{
		CurrentStreak: currentStreak,
		AvgGapMonth:   "1.2d",
		AvgGapWeek:    "0.8d",
		StreakDays:    streakDays,
		Activities:    activityViews,
	}
}

// DashboardWebHandler renders the progress dashboard HTML page.
func DashboardWebHandler(c *gin.Context) {
	funcMap := template.FuncMap{
		"mod": func(a, b int) int { return a % b },
		"add": func(a, b int) int { return a + b },
	}

	tmpl, err := template.New("dashboard").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		c.String(500, "Template error: %v", err)
		return
	}

	data := getDemoData()

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		c.String(500, "Render error: %v", err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, buf.String())
}
