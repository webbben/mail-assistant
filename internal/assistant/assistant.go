package assistant

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/webbben/mail-assistant/internal/config"
	"github.com/webbben/mail-assistant/internal/debug"
	emailcache "github.com/webbben/mail-assistant/internal/email_cache"
	"github.com/webbben/mail-assistant/internal/gmail"
	"github.com/webbben/mail-assistant/internal/llama"
	"github.com/webbben/mail-assistant/internal/personality"
	"github.com/webbben/mail-assistant/internal/types"
	t "github.com/webbben/mail-assistant/internal/types"
	"github.com/webbben/mail-assistant/internal/util"
	g "google.golang.org/api/gmail/v1"
)

func GetResponseInteractive(message t.Email, basePrompt string, ollamaClient *api.Client, appConfig config.Config, p *personality.Personality) string {
	prompt := p.FormatPrompt(appConfig.UserName, basePrompt, message)
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
func WaitForNextSummon(srv *g.Service, client *api.Client, config config.Config, p personality.Personality) {
	input := make(chan string)
	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		util.SomeoneTalks("SYS", "Press the enter key to summon", util.Gray)
		s, _ := reader.ReadString('\n')
		input <- s
	}()

	newMailCount := 0

	util.ClearScreen()

	for {
		select {
		case <-ticker.C:
			// check for new emails
			newMail, err := checkForNewMail(srv, config.GmailAddr)
			if err != nil {
				log.Println("error checking for new mail:", err)
				continue
			}
			// use auto-reply if enabled and applicable to the new emails
			if len(newMail) > 0 && config.AutoReply.Enabled {
				// only auto reply one email every tick, just so not too many emails are sent out at once
				// just a random mitigation measure against unexpected bugs or bad behavior, since one bad auto-reply email is better than 100.
				msgID := newMail[0]
				err := autoReplyMessage(client, srv, msgID, config, p)
				if err != nil {
					log.Println("failed to autoreply:", err)
				}
			}
			if len(newMail) > newMailCount {
				newMailCount = len(newMail)
				util.SomeoneTalks("SYS", fmt.Sprintf("%v new email(s) waiting to be received.", len(newMail)), util.Gray)
			}
		case <-input:
			return
		}
	}
}

func autoReplyMessage(client *api.Client, srv *g.Service, msgID string, config config.Config, p personality.Personality) error {
	// load the email content
	email, err := gmail.ProcessEmail(srv, msgID, config.GmailAddr)
	if err != nil {
		return err
	}
	reply, err, _ := AutoReply(client, email, config.UserName, p, config.AutoReply.Categories, config.AutoReply.Instructions)
	if err != nil {
		return err
	}
	if reply == "" {
		return nil
	}
	if !util.PromptYN("Do you want to autoreply to " + email.From + "?") {
		return nil
	}
	emailcache.AddToCache(email, emailcache.REPLY)
	util.SomeoneTalks("SYS", fmt.Sprintf("(%s) Auto reply sent to %s", util.CurrentTime(), email.From), util.Gray)
	return gmail.SendReply(srv, config.GmailAddr, email, reply)
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

func AutoReply(client *api.Client, email types.Email, username string, p personality.Personality, categories []string, instructions [][]string) (string, error, []int) {
	// first, detect if the given email is related to the auto reply categories, and output which one it is related to.
	cats := getEmailCategories(client, email, categories)
	if cats[0] == 0 {
		return "", nil, cats
	}
	instr := make([]string, 0)
	for _, cat := range cats {
		instr = append(instr, instructions[cat-1]...)
	}
	// prompt for a response based on the related categories and their instructions
	prompt := formatARReplyPrompt(p.BasePersonality, username, p.Name, instr)
	s, err := llama.GenerateCompletion(client, prompt, email.String())
	if err != nil {
		return "", err, cats
	}
	return s, nil, cats
}

const ARReplyForCategory = `
<<BASE-PERSONALITY>>
Your name is <<AI-NAME>> and your duty is to write responses to letters on behalf of your lord <<USER-NAME>>.
You will be given a message, and you will write a response to it based on the given instructions below. Surround the response in "~~~" so it is easily parsed.

Instructions:
<<INSTRUCTIONS>>
`

func formatARReplyPrompt(basePersonality string, username string, aiName string, instructions []string) string {
	instr := ""
	for _, line := range instructions {
		instr += fmt.Sprintf("* %s\n", line)
	}
	d := make(map[string]string)
	d["base-personality"] = basePersonality
	d["instructions"] = instr
	d["user-name"] = username
	d["ai-name"] = aiName
	return util.InsertMappedValues(ARReplyForCategory, d)
}

const ARCategoryPrompt = `
You are a robot that only outputs numbers from 0 to <<N>>.
You will be given a message, and you will determine if it falls into any of the following categories. Output the number of the category it falls into, or 0 if none apply.
If more than one apply, you can output more than one number, separated by commas. Only choose a category if you are 100 percent sure about it.

Sample outputs:
0
1
2
1,2

Categories:

<<CATEGORIES>>
`

func formatARCategoryPrompt(categories []string) string {
	catString := ""
	for i, cat := range categories {
		catString += fmt.Sprintf("%v) %s\n", i+1, cat)
	}
	d := map[string]string{
		"n":          fmt.Sprintf("%v", len(categories)),
		"categories": catString,
	}
	return util.InsertMappedValues(ARCategoryPrompt, d)
}

func getEmailCategories(client *api.Client, email types.Email, categories []string) []int {
	prompt := formatARCategoryPrompt(categories)
	out, err := llama.GenerateCompletionWithOpts(client, prompt, email.String(), map[string]interface{}{"temperature": 0.0})
	if err != nil {
		log.Println("failed to get email categories:", err)
		return []int{0}
	}
	nums := strings.Split(out, ",")
	if len(nums) == 0 {
		log.Println("invalid output for email category detection:", out)
		return []int{0}
	}
	parsedInts := make([]int, 0)
	for _, num := range nums {
		v, err := strconv.Atoi(strings.TrimSpace(num))
		if err != nil {
			log.Println("error parsing int:", err)
			return []int{0}
		}
		parsedInts = append(parsedInts, v)
	}
	return parsedInts
}
