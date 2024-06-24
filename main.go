package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/webbben/mail-assistant/internal/assistant"
	auth "github.com/webbben/mail-assistant/internal/auth"
	"github.com/webbben/mail-assistant/internal/config"
	"github.com/webbben/mail-assistant/internal/debug"
	emailcache "github.com/webbben/mail-assistant/internal/email_cache"
	"github.com/webbben/mail-assistant/internal/gmail"
	"github.com/webbben/mail-assistant/internal/llama"
	"github.com/webbben/mail-assistant/internal/personality"
	"github.com/webbben/mail-assistant/internal/util"
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

	for {
		// check gmail inbox
		util.ClearScreen()
		util.SomeoneTalks("SYS", "Loading your emails from your inbox. This may take a minute...", util.Gray)
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
			util.ClearScreen()
			fmt.Printf("%s enters the room, approaching to convey a message for you.\n", p.Name)
			util.SomeoneTalks(p.Name, p.GenPhrase(ollamaClient, "greeting"), util.Hi_blue)
			fmt.Printf("(To dismiss %s at any time, enter 'q' in the prompt)\n\n", p.Name)
			for _, email := range emails {
				emailReply := assistant.GetResponseInteractive(email, emailReplyPrompt, ollamaClient, appConfig, p)
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
				util.ClearScreen()
			}
			util.SomeoneTalks(p.Name, p.GenPhrase(ollamaClient, "dismiss"), util.Hi_blue)
		}

		emailcache.RemoveOldEntries(appConfig.LookbackDays)
		if err := emailcache.WriteCacheToDisk(); err != nil {
			log.Println("failed to write cache:", err)
		}
		assistant.WaitForNextSummon(srv, ollamaClient, appConfig, *p)
	}
}
