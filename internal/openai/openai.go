package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	gpt3 string = "gpt-3.5-turbo"
	gpt4 string = "gpt-4o"
)

func LoadAPIKey() string {
	path := "cred/openai.txt"
	bytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("error reading openai secret key file:", err)
		return ""
	}
	return strings.TrimSpace(string(bytes))
}

func LoadPrompt(id string, name string, message string) string {
	path := "prompts/" + id + ".txt"
	bytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("error reading prompt file:", err)
		return ""
	}
	s := strings.TrimSpace(string(bytes))
	s = fmt.Sprintf(s, name, message)
	return s

}

type APIResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	SystemFingerprint string   `json:"system_fingerprint"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      Message     `json:"message"`
	Logprobs     interface{} `json:"logprobs"` // Logprobs can be null or an object, using interface{} to handle both
	FinishReason string      `json:"finish_reason"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Request struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	FreqPenalty float64   `json:"frequency_penalty"` // penalize repeated tokens by giving a higher value. -2.0 to 2.0. defaults to 0.
	Temperature float64   `json:"temperature"`       // 0 - 2; higher values like 0.8 make the output more random, while lower values like 0.2 make it more focused and deterministic. defaults to 1.
}

func MakeAPICall(apiKey string, requestMessages []Message) []Message {
	endpoint := "https://api.openai.com/v1/chat/completions"
	requestData := Request{
		Model:       gpt3,
		Temperature: 0.2,
		FreqPenalty: 1,
		Messages:    requestMessages,
	}

	requestDataBytes, err := json.Marshal(requestData)
	if err != nil {
		log.Fatalf("Failed to marshal request data: %v", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(requestDataBytes))
	if err != nil {
		log.Fatalf("Failed to create HTTP request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	var response APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Fatalf("Failed to decode response body: %v", err)
	}

	if len(response.Choices) == 0 {
		fmt.Println(response)
		return []Message{}
	}

	outputMessages := append(requestMessages, response.Choices[0].Message)
	return outputMessages
}
