package say_hi

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var MCPDefinition = mcp.Tool{Name: "greet", Description: "say hi"}

type Input struct {
	Name string `json:"name" jsonschema:"the name of the person to greet"`
}

type Output struct {
	Greeting string `json:"greeting" jsonschema:"the greeting to tell to the user"`
}

func SayHi(_ context.Context, _ *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, Output, error) {
	return nil, Output{Greeting: "Hi " + input.Name}, nil
}
