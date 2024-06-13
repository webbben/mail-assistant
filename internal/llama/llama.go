package llama

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/ollama/ollama/api"
)

const (
	model   = "llama3"
	baseURL = "http://localhost:11434/"
)

// starts the ollama server, and returns its Cmd reference so the process can be managed later
func StartServer() (*exec.Cmd, error) {
	cmd := exec.Command("ollama", "serve")
	err := cmd.Start()
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

// stops the ollama server, killing the process
func StopServer(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}

func GetClient() (*api.Client, error) {
	return api.ClientFromEnvironment()
}

// Call the Chat Completion API. Meant for conversations where past message context is needed.
func ChatCompletion(client *api.Client, messages []api.Message) ([]api.Message, error) {
	stream := false
	ctx := context.Background()
	req := &api.ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   &stream,
	}

	err := client.Chat(ctx, req, func(cr api.ChatResponse) error {
		messages = append(messages, cr.Message)
		return nil
	})
	if err != nil {
		return []api.Message{}, err
	}
	return messages, nil
}

// Generates a completion using the given system prompt to set the context and AI behavior/personality, and based on the given prompt.
//
// Use ChatCompletion for conversations and memory based generation
func GenerateCompletion(client *api.Client, systemPrompt string, prompt string) (string, error) {
	stream := false
	req := &api.GenerateRequest{
		Model:  model,
		Prompt: prompt,
		System: systemPrompt,
		Stream: &stream,
	}
	ctx := context.Background()
	output := ""
	err := client.Generate(ctx, req, func(gr api.GenerateResponse) error {
		output = gr.Response
		return nil
	})
	if err != nil {
		return "", err
	}
	return output, nil
}

func IsEmailSpam(client *api.Client, email string) bool {
	prompt := fmt.Sprintf("Email:\n\n%s", email)
	systemPrompt := "You are an assistant that categorizes the content of emails. If an email looks to be some kind of automated newsletter or advertisement from a company, return \"<<<SPAM>>>\". Otherwise, return \"<<<PASS>>>\"."
	out, err := GenerateCompletion(client, systemPrompt, prompt)
	if err != nil {
		log.Println("failed to do spam detection completion:", err)
		return false
	}
	return strings.Contains(out, "<<<SPAM>>>")
}
