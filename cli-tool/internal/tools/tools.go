package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/mcp"
)

func AllOrpheusTools(g *genkit.Genkit) []ai.Tool {
	// Create an MCP client using HTTP transport for GitHub
	githubMCPClient, err := mcp.NewGenkitMCPClient(mcp.MCPClientOptions{
		Name: "github",
		StreamableHTTP: &mcp.StreamableHTTPConfig{
			BaseURL: "https://api.githubcopilot.com/mcp/",
			Headers: map[string]string{
				"Authorization": "Bearer " + os.Getenv("GITHUB"),
			},
		},
	})

	if err != nil {
		// Return empty slice if there's an error
		return []ai.Tool{}
	}

	// Get tools from the GitHub MCP client
	tools, err := githubMCPClient.GetActiveTools(context.Background(), g)
	if err != nil {
		return []ai.Tool{}
	}

	gcloudTool := genkit.DefineTool(
		g, "execute_gcloud_command", "Execute Google Cloud CLI commands",
		func(ctx *ai.ToolContext, input struct {
			Command string `json:"command"`
		}) (string, error) {
			// Split the command string into parts
			fmt.Println(input.Command)
			parts := strings.Fields(input.Command)
			if len(parts) == 0 {
				return "", fmt.Errorf("empty gcloud command")
			}

			// Ensure the command starts with "gcloud"
			if parts[0] != "gcloud" {
				return "", fmt.Errorf("command must start with 'gcloud'")
			}

			// Create command
			cmd := exec.Command(parts[0], parts[1:]...)

			// Capture command output
			output, err := cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("gcloud command error: %w\nOutput: %s", err, string(output))
			}

			return string(output), nil
		},
	)

	linuxTool := genkit.DefineTool(
		g, "execute_linux_command", "Execute Linux system commands",
		func(ctx *ai.ToolContext, input struct {
			Command string `json:"command"`
		}) (string, error) {
			fmt.Println(input.Command)
			// Split the command string into parts
			parts := strings.Fields(input.Command)
			if len(parts) == 0 {
				return "", fmt.Errorf("empty command")
			}

			// Create command
			cmd := exec.Command(parts[0], parts[1:]...)

			// Capture command output
			output, err := cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("linux command error: %w\nOutput: %s", err, string(output))
			}

			return string(output), nil
		},
	)

	tools = append(tools, gcloudTool)
	tools = append(tools, linuxTool)

	return tools
}
