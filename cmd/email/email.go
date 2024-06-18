package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/webbben/mail-assistant/internal/auth"
	"github.com/webbben/mail-assistant/internal/config"
	"github.com/webbben/mail-assistant/internal/gmail"
	"github.com/webbben/mail-assistant/internal/util"
)

func main() {
	list := flag.Bool("list", false, "list the email IDs found in the inbox")
	id := flag.String("id", "", "specify a specific email ID to analyze the email")
	flag.Parse()

	srv := auth.GetGmailService()
	config, err := config.LoadConfig()
	if err != nil {
		fmt.Println("failed to load configuration:", err)
		os.Exit(1)
	}

	if *list == true {
		if *id != "" {
			fmt.Println("Can't specify id flag if using list flag")
			os.Exit(1)
		}
		messages, err := gmail.ListMessages(srv, config.GmailAddr)
		if err != nil {
			fmt.Println("failed to list mail from gmail inbox:", err)
			os.Exit(1)
		}
		for i, msg := range messages {
			fmt.Printf("%v: %s\n", i+1, msg.Id)
		}
		return
	}
	if *id != "" {
		email, err := gmail.ProcessEmail(srv, *id, config.GmailAddr)
		if err != nil {
			fmt.Println("failed to process email:", err)
			os.Exit(1)
		}
		from := email.From
		if email.SenderName != "" {
			from += fmt.Sprintf(" (%s)", email.SenderName)
		}
		fmt.Println("ID:", *id)
		fmt.Println("From:", from)
		fmt.Println("Subject:", email.Subject)
		fmt.Println("Content:")
		fmt.Println("~~~")
		fmt.Println(email.Body)
		fmt.Println("~~~")
		return
	}
	messages, err := gmail.ListMessages(srv, config.GmailAddr)
	if err != nil {
		fmt.Println("failed to list mail from gmail inbox:", err)
		os.Exit(1)
	}
	for _, msg := range messages {
		email, err := gmail.ProcessEmail(srv, msg.Id, config.GmailAddr)
		if err != nil {
			fmt.Println("failed to process email:", err)
			raw, err := gmail.GetRaw(srv, config.GmailAddr, msg.Id)
			fmt.Println("=====", msg.Id, "=====")
			if err != nil {
				fmt.Println("failed to load message:", err)
			} else {
				fmt.Println(raw)
			}
			util.Continue()
			continue
		}
		from := email.From
		if email.SenderName != "" {
			from += fmt.Sprintf(" (%s)", email.SenderName)
		}
		fmt.Println("ID:", msg.Id)
		fmt.Println("From:", from)
		fmt.Println("Subject:", email.Subject)
		fmt.Println("\nContent:")
		fmt.Println("~~~")
		fmt.Println(email.Body)
		fmt.Println("~~~")
		if !util.PromptYN("\nContinue?") {
			return
		}
	}
}
