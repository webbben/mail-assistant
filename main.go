package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	auth "github.com/webbben/valet-de-chambre/internal/auth"
	"github.com/webbben/valet-de-chambre/internal/openai"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type Config struct {
	UserName       string `json:"user_name"`        // name of the human using this application
	PromptID       string `json:"prompt_id"`        // id of the prompt text file to use when starting AI interaction
	GmailAddr      string `json:"gmail_address"`    // gmail address to use
	AIName         string `json:"ai_name"`          // Name of the AI email assistant
	InboxCheckFreq int    `json:"inbox_check_freq"` // frequency in minutes in which the gmail inbox is checked for mail
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

	// Gmail client setup
	ctx := context.Background()
	b, err := os.ReadFile("cred/credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	googleConfig, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	fmt.Println("getting client...")
	client := auth.GetClient(googleConfig)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	// start loop listening for emails
	for {
		// check gmail inbox
		emails := GetEmails(srv, appConfig.GmailAddr)
		emails = append(emails, "Hi Ben,\nThis is James from Blacsand. Just reaching out to see if you are coming to the meeting next week. We want to discuss the Carity project and what the roadmap looks like.\n\nThanks,\nJames")

		if len(emails) > 0 {
			fmt.Printf("%s enters the room, approaching to convey a message for you.\n", appConfig.AIName)
			fmt.Printf("(To dismiss %s at any time, enter 'q' in the prompt)\n\n", appConfig.AIName)
			for i, email := range emails {
				if i > 0 {
					fmt.Println(appConfig.AIName+":", "Ah, and do you have time to hear another message? Or should I come back later?")

				}
				emailReply := GetResponseInteractive(email, apiKey, appConfig)
				if emailReply == "" {
					continue
				}
				fmt.Println("Sending response:\n", emailReply)
			}
		}
		time.Sleep(time.Minute * 60)
	}
}

func GetEmails(srv *gmail.Service, emailAddr string) []string {
	emails := []string{}
	r, err := srv.Users.Messages.List(emailAddr).Do()
	if err != nil {
		log.Println("failed to list emails:", err)
		return emails
	}
	if len(r.Messages) == 0 {
		log.Println("No emails found.")
		return emails
	}
	for _, msg := range r.Messages {
		if len(emails) >= 3 {
			break
		}
		msgBody, err, add := ProcessEmail(srv, msg.Id, emailAddr)
		if err != nil {
			log.Println("failed to process email:", err)
		}
		if add && len(msgBody) != 0 {
			emails = append(emails, msgBody)
		}
	}
	return emails
}

func ProcessEmail(srv *gmail.Service, messageID string, emailAddr string) (string, error, bool) {
	msg, err := srv.Users.Messages.Get(emailAddr, messageID).Do()
	if err != nil {
		return "", err, false
	}
	msgDate := time.Unix(msg.InternalDate, 0)

	// if the message more than a day old, ignore it
	if msgDate.Before(time.Now().Add(-24 * time.Hour)) {
		return "", nil, false
	}
	decoded, err := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
	if err != nil {
		return "", err, false
	}
	return string(decoded), nil, true
}

func GetResponseInteractive(message string, apiKey string, appConfig Config) string {
	prompt := openai.LoadPrompt(appConfig.PromptID, "Benjamin", message)
	if prompt == "" {
		fmt.Println("no prompt data.")
		return ""
	}
	fmt.Println(appConfig.PromptID)
	fmt.Println("prompt:\n", prompt)
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
			reply = strings.Split(content, "~~~")[1]
			fmt.Println("Shall I send this response?")
			fmt.Print("[Y/N]:")
			if yesOrNo() {
				break
			}
		}
	}
	fmt.Printf("\n%s: Very well. I await your summons, Monsieur.\n", appConfig.AIName)
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
