package assistant

import (
	"strings"

	"github.com/ollama/ollama/api"
	"github.com/webbben/mail-assistant/internal/llama"
	"github.com/webbben/mail-assistant/internal/types"
)

func AutoReply(client *api.Client, email types.Email, prompt string) (string, bool, error) {
	s, err := llama.GenerateCompletion(client, prompt, email.String())
	if err != nil {
		return "", false, err
	}
	if strings.Contains(s, "<<<IGNORE>>>") {
		return "", false, nil
	}
	return s, true, nil
}
