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
	emailcache "github.com/webbben/valet-de-chambre/internal/email_cache"
	"github.com/webbben/valet-de-chambre/internal/gmail"
	"github.com/webbben/valet-de-chambre/internal/openai"
	t "github.com/webbben/valet-de-chambre/internal/types"
)

func loadConfig() (t.Config, error) {
	bytes, err := os.ReadFile("config.json")
	if err != nil {
		return t.Config{}, err
	}
	var c t.Config
	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return t.Config{}, err
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

	// Load email cache
	if err := emailcache.LoadCacheFromDisk(); err != nil {
		log.Println("failed to load cache:", err)
	}

	// Get the standard greetings that will be used
	dismiss := openai.GetCustomPromptOutput(apiKey, "You've just finished relaying all the messages you had for your lord, and are formally dismissing yourself until more messages arrive for him. Phrase it as a statement, not a question.", "You are a noble's valet-de-chambre from 18th century France, who takes care of various duties for him such as relaying messages.")

	// start loop listening for emails
	for {
		// check gmail inbox
		emails := gmail.GetEmails(srv, apiKey, appConfig)
		emails = append(emails, t.Email{From: "james@blacsand.io", Body: "Hi Ben,\nThis is James from Blacsand. Just reaching out to see if you are coming to the meeting next week. We want to discuss the Carity project and what the roadmap looks like.\n\nThanks,\nJames"})

		fmt.Println("system: emails found:")
		for _, email := range emails {
			fmt.Println("From:", email.From)
			fmt.Println("Date:", email.Date)
			fmt.Println("Snippet:", email.Snippet)
			fmt.Println("Email length:", len(email.Body))
			fmt.Println("--------------\n", "--------------")
		}
		fmt.Print("process?")
		if !yesOrNo() {
			break
		}

		if len(emails) > 0 {
			fmt.Printf("%s enters the room, approaching to convey a message for you.\n", appConfig.AIName)
			fmt.Printf("(To dismiss %s at any time, enter 'q' in the prompt)\n\n", appConfig.AIName)
			for _, email := range emails {
				emailReply := GetResponseInteractive(email.Body, apiKey, appConfig)
				if emailReply == "<<SKIP>>" {
					emailcache.AddToCache(email, emailcache.IGNORE)
					continue
				}
				if emailReply == "" {
					log.Println("email reply unexpectedly empty.")
					continue
				}
				if emailReply == "<<QUIT>>" {
					break
				}
				if err := gmail.SendReply(srv, appConfig.GmailAddr, email, emailReply); err != nil {
					log.Println("failed to send reply:", err)
				} else {
					emailcache.AddToCache(email, emailcache.REPLY)
					someoneTalks("SYS", "email successfully sent to "+email.From)
				}
			}
		}
		someoneTalks(appConfig.AIName, dismiss)
		if err := emailcache.WriteCacheToDisk(); err != nil {
			log.Println("failed to write cache:", err)
		}
		time.Sleep(time.Minute * 60)
	}
}

func GetResponseInteractive(message string, apiKey string, appConfig t.Config) string {
	prompt := openai.LoadPrompt(appConfig.PromptID, appConfig.AIName, "Benjamin", message)
	if prompt == "" {
		log.Println("no prompt data.")
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
		log.Println("unexpected empty output from AI API")
		return ""
	}
	someoneTalks(appConfig.AIName, output[len(output)-1].Content)

	reply := ""
	for {
		response := getUserInput()
		if isQuit(response) {
			return "<<QUIT>>"
		}
		output = append(messages, openai.Message{
			Role:    "user",
			Content: response,
		})
		output = openai.MakeAPICall(apiKey, output)
		content := output[len(output)-1].Content

		if strings.TrimSpace(content) == "<<<IGNORE>>>" {
			someoneTalks(appConfig.AIName, "Very well, Monsieur, I will not respond to the message.")
			return "<<SKIP>>"
		}

		fmt.Printf("\n%s: %s\n", appConfig.AIName, content)

		if strings.Contains(content, "~~~") {
			// A reply draft is in the output
			reply := parseResponseFromAIOutput(content)
			if reply == "" {
				log.Println("No response parsed; exiting dialog.")
				break
			}
			someoneTalks(appConfig.AIName, "Shall I send this message?")
			if yesOrNo() {
				return reply
			}
			someoneTalks(appConfig.AIName, "Ah, how should I reply then, Monsieur?")
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

func someoneTalks(name string, statement string) {
	fmt.Printf("\n%s: %s\n", name, statement)
}
