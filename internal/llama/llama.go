package llama

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ollama/ollama/api"
)

/*
Documentation

API: https://github.com/ollama/ollama/blob/main/docs/api.md

Parameters: https://github.com/ollama/ollama/blob/main/docs/modelfile.md#valid-parameters-and-values
*/

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

// Generate a completion using custom options. Below are some common options, but find more information about options params here:
//
// https://github.com/ollama/ollama/blob/main/docs/modelfile.md#valid-parameters-and-values
//
// "temperature": float (default: 0.8) - increasing this will make the model answer more creatively
func GenerateCompletionWithOpts(client *api.Client, systemPrompt string, prompt string, opts map[string]interface{}) (string, error) {
	stream := false
	req := &api.GenerateRequest{
		Model:   model,
		Prompt:  prompt,
		System:  systemPrompt,
		Stream:  &stream,
		Options: opts,
	}
	return generateCompletion(client, req)
}

// Generates a completion using the given system prompt to set the context and AI behavior/personality, and based on the given prompt.
//
// Use ChatCompletion for conversations and memory based generation.
//
// Use GenerateCompletionWithOpts to customize options such as temperature.
func GenerateCompletion(client *api.Client, systemPrompt string, prompt string) (string, error) {
	stream := false
	req := &api.GenerateRequest{
		Model:  model,
		Prompt: prompt,
		System: systemPrompt,
		Stream: &stream,
	}
	return generateCompletion(client, req)
}

func generateCompletion(client *api.Client, req *api.GenerateRequest) (string, error) {
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

// Pass: 5/9            (56%)
// False positives: 1/9 (11%)
// Wrong format: 1/9    (11%)
const spam_prompt string = `
You are a robot that only outputs a number between 0 and 100.
You are given an email body, and you detect if it is junk or not. 
Output your confidence level about if the email is junk, where 0 means zero confidence (it is definitely not junk) and 100 is complete confidence (it is surely junk).
After the confidence level, also output a semi-colon and a short reason.

Sample outputs:
15;The email doesn't look like junk because the sender seems to know the recipient personally.
87;The email looks like junk because it's a newsletter from an online shop.
50;It's unclear if this is junk or not, since it has traits of both junk and regular emails.

An email is junk if:
* is a newsletter.
* it is a notification talking about inboxes, unread messages, etc.
* is doesn't refer to the recipient by their name.

An email is not junk if:
* it's from a person who seems to know the recipient personally.
* it includes odd, suspicious, or humorous content, since I want to see those messages personally.
`

// TODO - this doesn't work well enough so far, since it gives false positives a lot.
// Apparently it's kind of hard to prompt Llama3 to identify junk mail.
func IsEmailSpam(client *api.Client, email string) (bool, string) {
	prompt := fmt.Sprintf("Email:\n\n%s", email)
	system := strings.TrimSpace(spam_prompt)
	out, err := GenerateCompletionWithOpts(client, system, prompt, map[string]interface{}{"temperature": 0.0})
	if err != nil {
		log.Println("failed to do spam detection completion:", err)
		return false, ""
	}
	split := strings.Split(out, ";")
	confidence, err := strconv.Atoi(split[0])
	if err != nil {
		log.Println("failed to parse number from completion output:", err)
		return false, ""
	}

	return confidence > 50, out
}
