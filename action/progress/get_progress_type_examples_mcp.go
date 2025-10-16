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
	Description: `Returns natural language mapping examples for all progress types.

Provides structured mapping sets showing how to express progress values using natural language with emojis.
No input parameters required. Returns hardcoded examples for mood, habit_progress, project_progress, and promise_state.`,
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
