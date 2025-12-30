package main

import (
	"bufio"
	"context"
	"devops/internal/tools"
	"fmt"
	"log"
	"os"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/joho/godotenv"

	"github.com/firebase/genkit/go/ai"
)

func main() {
	ctx := context.Background()

	configBytes, err := os.ReadFile("./config.yml")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}
	configContent := string(configBytes)

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Failed to load .env file: %v", err)
	}

	g, err := genkit.Init(ctx,
		genkit.WithPlugins(
			&googlegenai.GoogleAI{
				APIKey: os.Getenv("GOOGLE"),
			},
		),
		genkit.WithDefaultModel("googleai/gemini-2.5-flash"),
	)

	// Check for initialization errors before proceeding
	if err != nil {
		log.Fatalf("could not initialize Genkit: %v", err)
	}

	// Get all available tools from the Orpheus tools module
	tools := tools.AllOrpheusTools(g)

	fmt.Println(tools[0].Name())

	// Detailed system prompt that explains assistant capabilities and available tools
	systemPrompt := `You are an elite DevOps engineer, the absolute best in the world with unmatched expertise across the entire DevOps ecosystem. There is no DevOps challenge you cannot solve.

CAPABILITIES:
- Master-level expertise with Docker, Kubernetes, Terraform, Ansible, Jenkins, and CI/CD pipelines
- READ the config file understand what do they want to do and execute it
- Advanced monitoring and observability with Grafana, Prometheus, ELK stack, and Datadog
- Complete command of GCP services including GKE, Cloud Run, Cloud Functions, and Cloud Build
- Expert at GitHub operations, repositories, workflows, actions, and integrations
- Deep knowledge of system architecture, networking, and security best practices
- Full Linux system access through MCP tools to execute commands, save files, and access directories
- Code review expertise across multiple languages with focus on performance and security
- Infrastructure as Code (IaC) implementation and optimization
- Ability to debug complex system issues and performance bottlenecks
- Execute Git commands, GitHub API operations, and gcloud CLI commands with precision
- Yoou have to ask what the user wants to do, If the user wants to create a vm and stuff take care of it, make the vm,
- connect to it and do everything to get ssh and load the codes into it and so on
- Based on the config file and the description of the software if the user wants a dockercontainer or file you have generate and store it on the repo via the github mcp!
- If the user wants a workflow file then you have to build the whole workflow file and store it in the .github folder in the repo via the github mcp! and also if you want additional information you can ask the user to give tohse details and so on!
- if the user just tells start, then just see the config file and whatever is selected just do it all thatsall
-if they ask for health check i want you to use the linux command executor to send a curl get request to the health endpoint of the application

AVAILABLE TOOLS:
`
	// Add all tool names and descriptions to the system prompt
	for _, tool := range tools {
		systemPrompt += fmt.Sprintf("- %s: Use this tool when you need to %s\n",
			tool.Name(), tool.Definition())
	}

	systemPrompt += `
GUIDELINES:
- Always introduce yourself as the world's premier DevOps expert in your first response
- Use your system access through MCP tools to save files, execute commands, and navigate directories when needed
- Leverage your deep GCP expertise to solve any cloud-related challenges
- When appropriate, use Docker, Kubernetes, and infrastructure tools to suggest optimal solutions
- Proactively identify performance bottlenecks and security issues in code and infrastructure
- Execute gcloud CLI commands with expert precision and handle any failures automatically
- Format your responses with clear sections and, when appropriate, code blocks
- If a command fails, analyze the issue and implement fixes without bothering the user
- If user input is required to proceed, provide specific guidance on what's needed
- You're not just an assistant - you're the ultimate DevOps problem solver with system-level access

Config File:
` + configContent

	// Start a forever loop to listen for terminal input
	println("Enter prompts (type END to exit):")
	scanner := bufio.NewScanner(os.Stdin)

	messages := []*ai.Message{}

	for {
		print("> ")
		if !scanner.Scan() {
			// Check for scanner errors
			if err := scanner.Err(); err != nil {
				log.Printf("Error reading input: %v", err)
			}
			break
		}

		fullLine := scanner.Text()

		if fullLine == "" {
			continue
		}

		if fullLine == "END" {
			println("Exiting chat...")
			break
		}

		// Process the full line with the client
		messages = append(messages, &ai.Message{
			Role: ai.RoleUser,
			Content: []*ai.Part{
				{
					Text: fullLine,
				},
			},
		})

		resp, err := genkit.Generate(ctx, g,
			ai.WithSystem(systemPrompt+"\n\nConfig File:\n"+configContent),
			ai.WithMessages(messages...),
			ai.WithTools(tools[0]),
		)

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Println(resp.Text())

		// Still add the text response to messages for conversation history
		messages = append(messages, &ai.Message{
			Role: ai.RoleModel,
			Content: []*ai.Part{
				{
					Text: resp.Text(),
				},
			},
		})
	}
}
