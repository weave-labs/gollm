// File: examples/chatbot.go

package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"strings"

	"github.com/teilomillet/gollm"
)

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}

	// Create a new LLM instance with memory
	llm, err := gollm.NewLLM(
		gollm.SetProvider("openai"),
		gollm.SetModel("gpt-4o-mini"),
		gollm.SetAPIKey(apiKey),
		gollm.SetMemory(4000), // Enable memory with a 4000 token limit
		gollm.SetLogLevel(gollm.LogLevelInfo),
	)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	log.Println("Welcome to the Memory-Enabled Chatbot!")
	log.Println("Type 'exit' to quit, or 'clear memory' to reset the conversation.")

	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	for {
		log.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading input: %v\n", err)
			continue
		}
		input = strings.TrimSpace(input)

		if input == "exit" {
			break
		}

		prompt := gollm.NewPrompt(input)
		response, err := llm.Generate(ctx, prompt)
		if err != nil {
			log.Printf("Error generating response: %v", err)
			continue
		}

		log.Printf("Chatbot: %s\n", response.AsText())
	}

	log.Println("Thank you for chatting!")
}
