package progress

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var GetProgressTypeExamplesMCPDefinition = mcp.Tool{
	Name: "get_progress_type_examples",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint: true,
		Title:        "Get progress type natural language examples",
	},
	Description: `Get natural language mappings to help convert user descriptions into numeric progress values (-2 to +2).

IMPORTANT: Call this tool at the START of every reflection session before asking progress questions.

Use this tool to:
- Learn how to interpret user's natural language responses ("sunny", "stuck", "forgot", etc.)
- Get emoji mappings to make conversations more engaging
- Understand different metaphors available for each progress type

Returns mappings for all 4 progress types:
1. MOOD: weather metaphors (sunny ☀️ to stormy ⛈️), light metaphors (bright ✨ to dark 🌑), color metaphors (green 💚 to red ❤️‍🔥)
2. HABIT_PROGRESS: consistency levels (crushing it 💪 to not doing ❌), garden metaphors (blooming 🌸 to withered 🍂)
3. PROJECT_PROGRESS: momentum (breakthrough 🚀 to changed plans 🔄), journey metaphors (sprinting 🏃 to lost 🗺️)
4. PROMISE_STATE: awareness levels (did something ✅ to forgot 🤷), flame metaphors (burning 🔥 to extinguished 💨)

How to use mappings:
- When user says "I'm feeling sunny today" → mood type, "sunny" = +2
- When user says "barely trying" → habit_progress type, "trying" = 0
- When user says "we're stuck" → project_progress type, "stuck" = 0
- When user says "I forgot" → promise_state type, "forgot" = -1

Offer metaphor choices when user is uncertain: "Would you describe your mood like weather (sunny to stormy) or colors (green to red)?"`,
}

type ProgressTypeExamplesInput struct {
	// No input parameters
}

type MappingValue struct {
	Word  string `json:"word" jsonschema:"Natural language word or phrase"`
	Value int    `json:"value" jsonschema:"Progress value from -2 to +2"`
	Emoji string `json:"emoji" jsonschema:"Associated emoji"`
}

type MappingSet struct {
	MappingName string         `json:"mapping_name" jsonschema:"Name of mapping metaphor"`
	Values      []MappingValue `json:"values" jsonschema:"Natural language mappings for each value"`
}

type ProgressTypeMapping struct {
	ProgressType string       `json:"progress_type" jsonschema:"Progress type"`
	Mappings     []MappingSet `json:"mappings" jsonschema:"Different mapping metaphors for this progress type"`
}

type ProgressTypeExamplesOutput struct {
	Examples []ProgressTypeMapping `json:"examples" jsonschema:"Mapping examples for each progress type"`
}

func GetProgressTypeExamples(ctx context.Context, _ *mcp.CallToolRequest, _ ProgressTypeExamplesInput) (*mcp.CallToolResult, ProgressTypeExamplesOutput, error) {
	output := ProgressTypeExamplesOutput{
		Examples: []ProgressTypeMapping{
			{
				ProgressType: "mood",
				Mappings: []MappingSet{
					{
						MappingName: "mood as weather",
						Values: []MappingValue{
							{Word: "sunny", Value: 2, Emoji: "☀️"},
							{Word: "partly cloudy", Value: 1, Emoji: "⛅"},
							{Word: "overcast", Value: 0, Emoji: "☁️"},
							{Word: "rainy", Value: -1, Emoji: "🌧️"},
							{Word: "stormy", Value: -2, Emoji: "⛈️"},
						},
					},
					{
						MappingName: "mood as light",
						Values: []MappingValue{
							{Word: "bright", Value: 2, Emoji: "✨"},
							{Word: "light", Value: 1, Emoji: "💡"},
							{Word: "dim", Value: 0, Emoji: "🕯️"},
							{Word: "dark", Value: -1, Emoji: "🌑"},
							{Word: "pitch black", Value: -2, Emoji: "⚫"},
						},
					},
					{
						MappingName: "mood as colors",
						Values: []MappingValue{
							{Word: "green", Value: 2, Emoji: "💚"},
							{Word: "white", Value: 1, Emoji: "🤍"},
							{Word: "gray", Value: 0, Emoji: "🩶"},
							{Word: "black", Value: -1, Emoji: "🖤"},
							{Word: "red", Value: -2, Emoji: "❤️‍🔥"},
						},
					},
				},
			},
			{
				ProgressType: "habit_progress",
				Mappings: []MappingSet{
					{
						MappingName: "habit consistency",
						Values: []MappingValue{
							{Word: "crushing it", Value: 2, Emoji: "💪"},
							{Word: "mostly doing", Value: 1, Emoji: "👍"},
							{Word: "trying", Value: 0, Emoji: "🤔"},
							{Word: "rarely", Value: -1, Emoji: "😔"},
							{Word: "not doing", Value: -2, Emoji: "❌"},
						},
					},
					{
						MappingName: "habit as garden",
						Values: []MappingValue{
							{Word: "blooming", Value: 2, Emoji: "🌸"},
							{Word: "growing", Value: 1, Emoji: "🌱"},
							{Word: "planted", Value: 0, Emoji: "🌰"},
							{Word: "wilting", Value: -1, Emoji: "🥀"},
							{Word: "withered", Value: -2, Emoji: "🍂"},
						},
					},
				},
			},
			{
				ProgressType: "project_progress",
				Mappings: []MappingSet{
					{
						MappingName: "project momentum",
						Values: []MappingValue{
							{Word: "breakthrough", Value: 2, Emoji: "🚀"},
							{Word: "moving forward", Value: 1, Emoji: "➡️"},
							{Word: "stuck", Value: 0, Emoji: "⏸️"},
							{Word: "setback", Value: -1, Emoji: "↩️"},
							{Word: "changed plans", Value: -2, Emoji: "🔄"},
						},
					},
					{
						MappingName: "project as journey",
						Values: []MappingValue{
							{Word: "sprinting", Value: 2, Emoji: "🏃"},
							{Word: "walking", Value: 1, Emoji: "🚶"},
							{Word: "resting", Value: 0, Emoji: "🧘"},
							{Word: "backtracking", Value: -1, Emoji: "🔙"},
							{Word: "lost", Value: -2, Emoji: "🗺️"},
						},
					},
				},
			},
			{
				ProgressType: "promise_state",
				Mappings: []MappingSet{
					{
						MappingName: "promise awareness",
						Values: []MappingValue{
							{Word: "did something", Value: 1, Emoji: "✅"},
							{Word: "remember", Value: 0, Emoji: "💭"},
							{Word: "forgot", Value: -1, Emoji: "🤷"},
						},
					},
					{
						MappingName: "promise as flame",
						Values: []MappingValue{
							{Word: "burning", Value: 1, Emoji: "🔥"},
							{Word: "lit", Value: 0, Emoji: "🕯️"},
							{Word: "extinguished", Value: -1, Emoji: "💨"},
						},
					},
				},
			},
		},
	}

	return nil, output, nil
}
