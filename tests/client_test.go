package tests

import (
	"context"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/transport"
)

func (s *IntegrationTestSuite) TestClient() {
	ctx := context.Background()

	// Connect to a server over stdin/stdout
	t1, t2 := mcp.NewInMemoryTransports()

	go func() {
		server := transport.MCPServer(s.repo)
		s.Require().NoError(server.Run(ctx, t1))
	}()

	time.Sleep(1 * time.Second)

	// Create a new client, with no features.
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)

	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	// Call a tool on the server.
	params := &mcp.CallToolParams{
		Name: "add_food",
		Arguments: map[string]any{
			"name":      "Яйцо куриное / Chicken egg",
			"food_type": "component",
			"nutrients": map[string]any{
				"calories":        157,
				"protein_g":       12.7,
				"total_fat_g":     10.9,
				"cholesterol_mg":  372,
				"carbohydrates_g": 0.7,
			},
			"description":    "",
			"serving_name":   "",
			"serving_size_g": 50,
		},
	}
	res, err := session.CallTool(ctx, params)
	if err != nil {
		log.Fatalf("CallTool failed: %v", err)
	}
	if res.IsError {
		log.Fatal("tool failed")
	}
	for _, c := range res.Content {
		log.Print(c.(*mcp.TextContent).Text)
	}
}
