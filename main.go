package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
	auth "github.com/webbben/mail-assistant/internal/auth"
	"github.com/webbben/mail-assistant/internal/config"
	"github.com/webbben/mail-assistant/internal/debug"
	emailcache "github.com/webbben/mail-assistant/internal/email_cache"
	"github.com/webbben/mail-assistant/internal/gmail"
	"github.com/webbben/mail-assistant/internal/llama"
	"github.com/webbben/mail-assistant/internal/personality"
	t "github.com/webbben/mail-assistant/internal/types"
	"github.com/webbben/mail-assistant/internal/util"
	g "google.golang.org/api/gmail/v1"
)

func loadConfig() (config.Config, error) {
	bytes, err := os.ReadFile("config.json")
	if err != nil {
		return config.Config{}, err
	}
	var c config.Config
	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return config.Config{}, err
	}
	return c, nil
}

func loadPrompt(id string) string {
	path := "prompts/" + id + ".txt"
	bytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("error reading prompt file:", err)
		return ""
	}
	return strings.TrimSpace(string(bytes))
}

func main() {
	// ollama
	cmd, err := llama.StartServer()
	if err != nil {
		log.Fatal("failed to start ollama server:", err)
	}
	defer llama.StopServer(cmd)
	ollamaClient, err := llama.GetClient()
	if err != nil {
		log.Fatal("failed to get ollama client:", err)
	}

	// app config
	appConfig, err := loadConfig()
	if err != nil {
		log.Fatal("failed to load config json:", err)
	}
	debug.SetDebugMode(appConfig.Debug)

	// Oauth + Gmail setup
	srv := auth.GetGmailService()

	// load personality file
	p, err := personality.Load(appConfig.PersonalityID)
	if err != nil {
		fmt.Println("failed to load personality file.")
		// TODO use default personality
	}
	emailReplyPrompt := loadPrompt(p.Prompts.EmailWorkflow)

	// Load email cache
	if err := emailcache.LoadCacheFromDisk(); err != nil {
		log.Println("failed to load cache:", err)
	}

	util.SomeoneTalks("SYS", "Loading your emails from your inbox. This may take a minute...", util.Gray)
	for {
		// check gmail inbox
		emails := gmail.GetEmails(srv, ollamaClient, appConfig)
		if len(emails) > 0 {
			util.SomeoneTalks("SYS", "Emails found:", util.Gray)
			for _, email := range emails {
				fmt.Println("From:", email.From)
				fmt.Println("Date:", email.Date)
				fmt.Println("Snippet:", email.Snippet)
				fmt.Println("Email length:", len(email.Body))
				fmt.Println("--------------")
			}
		} else {
			util.SomeoneTalks("SYS", "No emails found that need processing.", util.Gray)
		}

		if len(emails) > 0 && util.PromptYN("Go through these emails now?") {
			fmt.Printf("%s enters the room, approaching to convey a message for you.\n", p.Name)
			util.SomeoneTalks(p.Name, p.GenPhrase(ollamaClient, "greeting"), util.Hi_blue)
			fmt.Printf("(To dismiss %s at any time, enter 'q' in the prompt)\n\n", p.Name)
			for _, email := range emails {
				emailReply := GetResponseInteractive(email, emailReplyPrompt, ollamaClient, appConfig, p)
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
					util.SomeoneTalks("SYS", "email successfully sent to "+email.From, util.Gray)
				}
			}
			util.SomeoneTalks(p.Name, p.GenPhrase(ollamaClient, "dismiss"), util.Hi_blue)
		}

		emailcache.RemoveOldEntries(appConfig.LookbackDays)
		if err := emailcache.WriteCacheToDisk(); err != nil {
			log.Println("failed to write cache:", err)
		}
		waitForNextSummon(srv, appConfig.GmailAddr)
	}
}

func GetResponseInteractive(message t.Email, basePrompt string, ollamaClient *api.Client, appConfig config.Config, p *personality.Personality) string {
	prompt := p.FormatPrompt(appConfig.UserName, basePrompt, message.Body, message.From, message.SenderName, message.Subject)
	if prompt == "" {
		debug.Println("no prompt data.")
		return ""
	}
	messages := []api.Message{
		{
			Role:    "system",
			Content: prompt,
		},
	}
	messages, err := llama.ChatCompletion(ollamaClient, messages)
	if err != nil {
		log.Println("failed to generate chat completion:", err)
		return ""
	}
	if len(messages) == 0 {
		debug.Println("unexpected empty output from AI API")
		return ""
	}
	util.SomeoneTalks(p.Name, messages[len(messages)-1].Content, util.Hi_blue)

	reply := ""
	for {
		response := util.GetUserInput()
		if util.IsQuit(response) {
			return "<<QUIT>>"
		}
		messages = append(messages, api.Message{
			Role:    "user",
			Content: response,
		})
		messages, err = llama.ChatCompletion(ollamaClient, messages)
		if err != nil {
			log.Println("failed to generate chat completion:", err)
			return ""
		}
		content := messages[len(messages)-1].Content

		if strings.Contains(content, "<<<IGNORE>>>") {
			util.SomeoneTalks(p.Name, p.GenPhrase(ollamaClient, "ignore"), util.Hi_blue)
			return "<<SKIP>>"
		}
		util.SomeoneTalks(p.Name, content, util.Hi_blue)

		if strings.Contains(content, "~~~") {
			// A reply draft is in the output
			reply := parseReplyMessage(content)
			if reply == "" {
				log.Println("No response parsed; exiting dialog.")
				break
			}
			if util.PromptYN("Confirm reply?") {
				return reply
			}
			util.SomeoneTalks(p.Name, "Ah, how should I reply then, Monsieur?", util.Hi_blue)
		}
	}
	return reply
}

func parseReplyMessage(content string) string {
	replyParts := strings.Split(content, "~~~")
	reply := ""
	if len(replyParts) == 0 {
		debug.Println("No ~~~ found in content. Are you sure there should be a reply?")
		return ""
	}
	if len(replyParts) > 3 {
		debug.Println("too many ~~~ found... the content must be incorrectly formatted.")
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
		debug.Println("No reply parsed...")
	}
	return reply
}

// waits for the user to summon the assistant, and checks for new mail while waiting
func waitForNextSummon(srv *g.Service, gmailAddr string) {
	input := make(chan string)
	ticker := time.NewTicker(10 * time.Minute)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		util.SomeoneTalks("SYS", "Press the enter key to summon", util.Gray)
		s, _ := reader.ReadString(' ')
		input <- s
	}()

	for {
		select {
		case <-ticker.C:
			// check for new emails
			newMail, err := checkForNewMail(srv, gmailAddr)
			if err != nil {
				log.Fatal(err)
			}
			util.SomeoneTalks("SYS", fmt.Sprintf("%v new email(s) waiting to be received.", len(newMail)), util.Gray)
		case <-input:
			return
		}
	}
}

// checks if new, unprocessed emails are waiting, and returns their message IDs. If an error occurs while checking gmail API, the error is returned.
func checkForNewMail(srv *g.Service, addr string) ([]string, error) {
	newMail := make([]string, 0)
	list, err := gmail.ListMessages(srv, addr)
	if err != nil {
		return nil, err
	}
	for _, msg := range list {
		if _, cached := emailcache.IsCached(msg.Id); !cached {
			newMail = append(newMail, msg.Id)
		}
	}
	return newMail, nil
}
