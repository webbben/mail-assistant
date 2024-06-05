package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	auth "github.com/webbben/valet-de-chambre/internal/auth"
	"github.com/webbben/valet-de-chambre/internal/gmail"
	"github.com/webbben/valet-de-chambre/internal/openai"
)

type Config struct {
	UserName        string `json:"user_name"`         // name of the human using this application
	PromptID        string `json:"prompt_id"`         // id of the prompt text file to use when starting AI interaction
	GmailAddr       string `json:"gmail_address"`     // gmail address to use
	AIName          string `json:"ai_name"`           // Name of the AI email assistant
	InboxCheckFreq  int    `json:"inbox_check_freq"`  // frequency in minutes in which the gmail inbox is checked for mail
	EmailBatchLimit int    `json:"email_batch_limit"` // limit to the number of emails that will be processed in a single batch
}

func loadConfig() (Config, error) {
	bytes, err := os.ReadFile("config.json")
	if err != nil {
		return Config{}, err
	}
	var c Config
	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return Config{}, err
	}
	return c, nil
}

func main() {
	appConfig, err := loadConfig()
	if err != nil {
		log.Fatal("failed to load config json:", err)
	}

	// OpenAI setup
	apiKey := openai.LoadAPIKey()
	if apiKey == "" {
		fmt.Println("failed to load openai API key...")
		return
	}

	// Oauth + Gmail setup
	srv := auth.GetGmailService()

	// start loop listening for emails
	for {
		// check gmail inbox
		emails := gmail.GetEmails(srv, appConfig.GmailAddr, appConfig.EmailBatchLimit)
		emails = append(emails, gmail.Email{From: "james@blacsand.io", Body: "Hi Ben,\nThis is James from Blacsand. Just reaching out to see if you are coming to the meeting next week. We want to discuss the Carity project and what the roadmap looks like.\n\nThanks,\nJames"})

		fmt.Println("system: emails found:")
		for _, email := range emails {
			fmt.Println("From:", email.From)
			fmt.Println("Date:", email.Date)
			fmt.Println("Snippet:", email.Snippet)
			fmt.Println("Email length:", len(email.Body))
			fmt.Println("--------------\n", "--------------")
		}
		fmt.Print("process? [y/n]:")
		if !yesOrNo() {
			break
		}

		if len(emails) > 0 {
			fmt.Printf("%s enters the room, approaching to convey a message for you.\n", appConfig.AIName)
			fmt.Printf("(To dismiss %s at any time, enter 'q' in the prompt)\n\n", appConfig.AIName)
			for i, email := range emails {
				if i > 0 {
					fmt.Println(appConfig.AIName+":", "Ah, and do you have time to hear another message? Or should I come back later?")

				}
				emailReply := GetResponseInteractive(email.Body, apiKey, appConfig)
				if emailReply == "" {
					continue
				}
				fmt.Println("Sending response:\n", emailReply)
			}
		}
		time.Sleep(time.Minute * 60)
	}
	fmt.Printf("\n%s: Very well. I await your summons, Monsieur.\n", appConfig.AIName)
}

func GetResponseInteractive(message string, apiKey string, appConfig Config) string {
	prompt := openai.LoadPrompt(appConfig.PromptID, "Benjamin", message)
	if prompt == "" {
		fmt.Println("no prompt data.")
		return ""
	}
	messages := []openai.Message{
		{
			Role:    "system",
			Content: prompt,
		},
	}
	output := openai.MakeAPICall(apiKey, messages)
	if len(output) == 0 {
		fmt.Printf("\n%s: *muttering to self* (Hmm, nevermind...)\n", appConfig.AIName)
		return ""
	}
	fmt.Printf("\n%s: %s\n", appConfig.AIName, output[len(output)-1].Content)

	reply := ""
	for {
		response := getUserInput()
		if isQuit(response) {
			return ""
		}
		output = append(messages, openai.Message{
			Role:    "user",
			Content: response,
		})
		output = openai.MakeAPICall(apiKey, output)
		content := output[len(output)-1].Content

		fmt.Printf("\n%s: %s\n", appConfig.AIName, content)

		if strings.Contains(content, "~~~") {
			// A reply draft is in the output
			reply := parseResponseFromAIOutput(content)
			if reply == "" {
				log.Println("No response parsed; exiting dialog.")
				break
			}
			fmt.Println("Shall I send this response?")
			fmt.Print("[Y/N]:")
			if yesOrNo() {
				break
			}
		}
	}
	return reply
}

func parseResponseFromAIOutput(content string) string {
	replyParts := strings.Split(content, "~~~")
	reply := ""
	if len(replyParts) == 0 {
		log.Println("No ~~~ found in content. Are you sure there should be a reply?")
		return ""
	}
	if len(replyParts) > 3 {
		log.Println("too many ~~~ found... the content must be incorrectly formatted.")
	}
	// we expect the message to be wrapped inside ~~~
	if len(replyParts) >= 2 {
		reply = replyParts[1]
	}
	// if not found, look through all parts for the first appearance of text.
	if reply == "" {
		for _, part := range replyParts {
			if part != "" {
				return part
			}
		}
		log.Println("No reply parsed...")
	}
	return reply
}

func getUserInput() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nUser: ")
	resp, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("failed to read user input")
	}
	return strings.TrimSpace(resp)
}

func isQuit(s string) bool {
	s = strings.ToLower(s)
	return s == "q" || s == "quit" || s == "exit"
}

// returns true if the user answers Yes
func yesOrNo() bool {
	fmt.Print("[Y/N]:")
	return isYes(getUserInput())
}

func isYes(s string) bool {
	s = strings.ToLower(s)
	return s == "y" || s == "yes"
}
