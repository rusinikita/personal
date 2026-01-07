package progress

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"personal/domain"
	"personal/gateways"
)

const (
	streakDaysCount    = 30 // количество дней для отображения в стрике
	topActivitiesCount = 5  // количество топ активностей для показа
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

        .emoji-spacer {
            font-size: 12px;
            color: #808080;
            padding: 0 4px;
            white-space: nowrap;
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
                {{- if and $day.IsSunday (ne (add $i 1) (len $.StreakDays)) -}}
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
                        {{if $p.IsSpacer -}}
                            <div class="emoji-spacer">←{{$p.DaysGap}}d→</div>
                        {{- else -}}
                            <div class="emoji-cell{{if $p.IsToday}} today{{end}}">{{$p.Emoji}}</div>
                        {{- end -}}
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
	Active   bool
	IsSunday bool // Флаг для быстрой проверки в шаблоне
}

type ProgressCell struct {
	Emoji    string
	IsToday  bool
	IsSpacer bool // true если это разделитель между замерами
	DaysGap  int  // количество дней в разделителе (если IsSpacer=true)
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

// DashboardWebHandler renders the progress dashboard HTML page.
func DashboardWebHandler(c *gin.Context) {
	ctx := c.Request.Context()

	// Получить DB из контекста
	db := gateways.DBFromContext(ctx)
	if db == nil {
		c.String(500, "Database not available")
		return
	}

	// Построить данные из БД
	data, err := buildDashboardDataFromDB(ctx, db)
	if err != nil {
		c.String(500, "Failed to build dashboard data: %v", err)
		return
	}

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}

	tmpl, err := template.New("dashboard").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		c.String(500, "Template error: %v", err)
		return
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		c.String(500, "Render error: %v", err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, buf.String())
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

// calculateAvgGap вычисляет средний промежуток между днями
func calculateAvgGap(gaps []int) string {
	if len(gaps) == 0 {
		return "0.0d"
	}
	sum := 0
	for _, g := range gaps {
		sum += g
	}
	avg := float64(sum) / float64(len(gaps))
	return fmt.Sprintf("%.1fd", avg)
}

// filterProgressByActivity фильтрует точки прогресса по activity_id
func filterProgressByActivity(points []domain.ActivityPoint, activityID int64) []domain.ActivityPoint {
	filtered := make([]domain.ActivityPoint, 0)
	for _, p := range points {
		if p.ActivityID == activityID {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// formatFrequency форматирует частоту (N -> "each Xd")
func formatFrequency(days int) string {
	return fmt.Sprintf("each %dd", days)
}

// formatTimeAgoPtr - перегрузка для *time.Time
func formatTimeAgoPtr(t *time.Time) string {
	if t == nil {
		return "never"
	}
	return formatTimeAgo(*t)
}

// getStalenessClassPtr - перегрузка для *time.Time
func getStalenessClassPtr(lastPointAt *time.Time, frequencyDays int) string {
	if lastPointAt == nil {
		return "warning"
	}
	diffDays := time.Since(*lastPointAt).Hours() / 24
	if diffDays > float64(frequencyDays*2) {
		return "warning"
	}
	return ""
}

// isSameDay проверяет, являются ли две даты одним и тем же днем
func isSameDay(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// buildActivityProgressCells строит компактные ячейки прогресса для активности
// Показывает только замеры (без пустых дней), вставляя разделители с количеством пропущенных дней между замерами
// ВАЖНО: ячейки строятся СЛЕВА НАПРАВО от самых старых к новым
func buildActivityProgressCells(activity domain.Activity, allProgress []domain.ActivityPoint, today time.Time) []ProgressCell {
	// Отфильтровать точки для этой активности
	activityProgress := filterProgressByActivity(allProgress, activity.ID)

	// Отсортировать по дате (от старых к новым)
	sort.Slice(activityProgress, func(i, j int) bool {
		return activityProgress[i].ProgressAt.Before(activityProgress[j].ProgressAt)
	})

	// Построить ячейки
	cells := []ProgressCell{}
	var prevDate *time.Time

	for _, point := range activityProgress {
		// Если есть предыдущая точка, проверить разрыв
		if prevDate != nil {
			daysGap := int(point.ProgressAt.Sub(*prevDate).Hours() / 24)
			if daysGap > 1 {
				// Добавить разделитель
				cells = append(cells, ProgressCell{
					IsSpacer: true,
					DaysGap:  daysGap,
				})
			}
		}

		// Добавить ячейку с замером
		isToday := isSameDay(point.ProgressAt, today)
		emoji := getEmoji(string(activity.ProgressType), &point.Value)

		cells = append(cells, ProgressCell{
			Emoji:   emoji,
			IsToday: isToday,
		})

		progressDate := point.ProgressAt
		prevDate = &progressDate
	}

	return cells
}

// buildDashboardDataFromDB - главная функция построения данных из БД
func buildDashboardDataFromDB(ctx context.Context, db gateways.DB) (DashboardData, error) {
	// Получить userID из контекста, если есть, иначе использовать 1 (для личного дашборда)
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		userID = 1 // fallback для личного дашборда
	}

	// Шаг 1: Получить топ-5 активностей
	activities, err := db.ListActivities(ctx, domain.ActivityFilter{
		UserID:     userID,
		ActiveOnly: true,
	})
	if err != nil {
		return DashboardData{}, fmt.Errorf("failed to list activities: %w", err)
	}

	// Stable sort: активность с TOP_ACTIVITY_ID всегда наверху
	if topActivityIDStr := os.Getenv("TOP_ACTIVITY_ID"); topActivityIDStr != "" {
		if topActivityID, err := strconv.ParseInt(topActivityIDStr, 10, 64); err == nil {
			sort.SliceStable(activities, func(i, j int) bool {
				// Если i имеет TOP_ACTIVITY_ID, он должен быть перед j
				if activities[i].ID == topActivityID {
					return true
				}
				// Если j имеет TOP_ACTIVITY_ID, он должен быть перед i
				if activities[j].ID == topActivityID {
					return false
				}
				// Иначе сохраняем исходный порядок (stable sort)
				return false
			})
		}
	}

	// Берем первые N активностей (ListActivities уже сортирует по срочности)
	if len(activities) > topActivitiesCount {
		activities = activities[:topActivitiesCount]
	}

	// Шаг 2: Получить все замеры за N дней
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -streakDaysCount)

	allProgress, err := db.ListProgress(ctx, domain.ProgressFilter{
		UserID: userID,
		From:   thirtyDaysAgo,
		To:     now,
	})
	if err != nil {
		return DashboardData{}, fmt.Errorf("failed to list progress: %w", err)
	}

	// Шаг 3: Вычислить общий стрик (N дней включая сегодня)
	dateMap := make(map[string]bool)
	for _, point := range allProgress {
		dateKey := point.ProgressAt.Format("2006-01-02")
		dateMap[dateKey] = true
	}

	// Построить массив из N дней (от старых к новым, включая сегодня)
	streakStartDate := now.AddDate(0, 0, -(streakDaysCount - 1))
	streakDays := make([]StreakDay, streakDaysCount)
	for i := 0; i < streakDaysCount; i++ {
		date := streakStartDate.AddDate(0, 0, i)
		dateKey := date.Format("2006-01-02")

		streakDays[i] = StreakDay{
			Active:   dateMap[dateKey],
			IsSunday: date.Weekday() == time.Sunday,
		}
	}

	// Подсчитать текущий стрик (с конца массива)
	currentStreak := 0
	for i := streakDaysCount - 1; i >= 0; i-- {
		if streakDays[i].Active {
			currentStreak++
		} else {
			break
		}
	}

	// Шаг 4: Вычислить средние промежутки (avg gap)
	activeDays := []int{}
	for i, day := range streakDays {
		if day.Active {
			activeDays = append(activeDays, i)
		}
	}

	gaps := []int{}
	for i := 1; i < len(activeDays); i++ {
		gap := activeDays[i] - activeDays[i-1] - 1
		if gap > 0 {
			gaps = append(gaps, gap)
		}
	}

	avgGapMonth := calculateAvgGap(gaps)
	lastSevenGaps := gaps
	if len(gaps) > 7 {
		lastSevenGaps = gaps[len(gaps)-7:]
	}
	avgGapWeek := calculateAvgGap(lastSevenGaps)

	// Шаг 5: Построить ячейки прогресса для каждой активности
	activityViews := make([]ActivityView, 0, len(activities))
	for _, activity := range activities {
		// Построить компактные ячейки прогресса (только замеры с разделителями)
		cells := buildActivityProgressCells(activity, allProgress, now)

		view := ActivityView{
			Name:           activity.Name,
			Frequency:      formatFrequency(activity.FrequencyDays),
			TimeAgo:        formatTimeAgoPtr(activity.LastPointAt),
			StalenessClass: getStalenessClassPtr(activity.LastPointAt, activity.FrequencyDays),
			ProgressCells:  cells,
		}
		activityViews = append(activityViews, view)

		// Шаг 6: Вызвать GetTrendStats для каждой активности (для будущего использования)
		_, err := db.GetTrendStats(ctx, activity.ID, userID, thirtyDaysAgo, now)
		if err != nil {
			// Логируем ошибку, но не прерываем рендеринг
			log.Printf("Failed to get trend stats for activity %d: %v", activity.ID, err)
		}
	}

	// Обработка edge case: нет активностей
	if len(activities) == 0 {
		return DashboardData{
			CurrentStreak: 0,
			AvgGapMonth:   "0.0d",
			AvgGapWeek:    "0.0d",
			StreakDays:    make([]StreakDay, streakDaysCount),
			Activities:    []ActivityView{},
		}, nil
	}

	return DashboardData{
		CurrentStreak: currentStreak,
		AvgGapMonth:   avgGapMonth,
		AvgGapWeek:    avgGapWeek,
		StreakDays:    streakDays,
		Activities:    activityViews,
	}, nil
}
