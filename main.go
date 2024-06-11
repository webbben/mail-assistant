package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
	auth "github.com/webbben/mail-assistant/internal/auth"
	"github.com/webbben/mail-assistant/internal/debug"
	emailcache "github.com/webbben/mail-assistant/internal/email_cache"
	"github.com/webbben/mail-assistant/internal/gmail"
	"github.com/webbben/mail-assistant/internal/llama"
	"github.com/webbben/mail-assistant/internal/openai"
	"github.com/webbben/mail-assistant/internal/personality"
	t "github.com/webbben/mail-assistant/internal/types"
	"github.com/webbben/mail-assistant/internal/util"
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
	// start loop listening for emails
	for {
		// check gmail inbox
		emails := gmail.GetEmails(srv, ollamaClient, appConfig)
		util.SomeoneTalks("SYS", "Emails found:", util.Gray)
		for _, email := range emails {
			fmt.Println("From:", email.From)
			fmt.Println("Date:", email.Date)
			fmt.Println("Snippet:", email.Snippet)
			fmt.Println("Email length:", len(email.Body))
			fmt.Println("--------------\n", "--------------")
		}
		fmt.Print("process?")
		if !util.PromptYN() {
			break
		}

		if len(emails) > 0 {
			fmt.Printf("%s enters the room, approaching to convey a message for you.\n", p.Name)
			util.SomeoneTalks(p.Name, p.Phrases.Get("greeting"), util.Hi_blue)
			fmt.Printf("(To dismiss %s at any time, enter 'q' in the prompt)\n\n", p.Name)
			for _, email := range emails {
				emailReply := GetResponseInteractive(email.Body, emailReplyPrompt, ollamaClient, appConfig, p)
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
		}
		util.SomeoneTalks(p.Name, p.Phrases.Get("dismiss"), util.Hi_blue)

		emailcache.RemoveOldEntries(appConfig.LookbackDays)
		if err := emailcache.WriteCacheToDisk(); err != nil {
			log.Println("failed to write cache:", err)
		}
		time.Sleep(time.Minute * 60)
	}
}

func GetResponseInteractive(message string, basePrompt string, ollamaClient *api.Client, appConfig t.Config, p *personality.Personality) string {
	prompt := openai.LoadPrompt(p.Prompts.EmailWorkflow, p.Name, appConfig.UserName, message)
	prompt = p.FormatPrompt(appConfig.UserName, basePrompt, message)
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

		if strings.TrimSpace(content) == "<<<IGNORE>>>" {
			util.SomeoneTalks(p.Name, p.Phrases.Get("ignore"), util.Hi_blue)
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
			//util.SomeoneTalks(p.Name, "Shall I send this message?", util.Hi_blue)
			if util.PromptYN() {
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
