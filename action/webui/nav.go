package webui

// NavSection identifies which top-level nav item BuildNav renders active.
type NavSection string

const (
	NavMoney        NavSection = "money"
	NavProgress     NavSection = "progress"
	NavWorkouts     NavSection = "workouts"
	NavAchievements NavSection = "achievements"
	NavIdeas        NavSection = "ideas"
	NavDocs         NavSection = "docs"
	NavDesignSystem NavSection = "design-system"
)

// BuildNav builds the shared top-level nav, highlighting active as the
// current page.
func BuildNav(active NavSection) []NavItem {
	return []NavItem{
		{Label: "Money", URL: "/web/money", Active: active == NavMoney},
		{Label: "Progress", URL: "/web/progress/browse", Active: active == NavProgress},
		{Label: "Workouts", URL: "/web/workouts", Active: active == NavWorkouts},
		{Label: "Achievements", URL: "/web/achievements", Active: active == NavAchievements},
		{Label: "Ideas", URL: "/web/ideas", Active: active == NavIdeas},
		{Label: "Docs", URL: "/web/docs", Active: active == NavDocs},
		{Label: "Design System", URL: "/web/design-system", Active: active == NavDesignSystem},
	}
}
