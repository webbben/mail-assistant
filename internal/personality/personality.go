package personality

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ollama/ollama/api"
	"github.com/webbben/mail-assistant/internal/llama"
	"github.com/webbben/mail-assistant/internal/openai"
	"github.com/webbben/mail-assistant/internal/util"
)

type Personality struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`             // name the AI assumes in interactions.
	BasePersonality string            `json:"base_personality"` // description of the overall personality and behavior of this AI assistant, used as context for the other prompts.
	PhrasePrompts   map[string]string `json:"phrase_prompts"`   // Prompts used for generating phrases
	Prompts         Prompts           `json:"prompts"`          // Prompts for main workflows (e.g. handling incoming emails)
	InsertDict      map[string]string `json:"insert_dict"`      // dictionary of terms to insert into the prompts
}

type Prompts struct {
	EmailWorkflow string `json:"email_workflow"` // prompt for how the email handling workflow and interaction should work.
	AutoReply     string `json:"autoreply"`      // prompt for handling autoreply messages
}

func (p Personality) GenPhrase(client *api.Client, phraseKey string) string {
	if p.PhrasePrompts == nil {
		log.Println("failed to generate phrase: no phrase prompts defined")
		return ""
	}
	prompt := p.PhrasePrompts[phraseKey]
	if prompt == "" {
		log.Println("failed to generate phrase: given phraseKey not mapped")
		return ""
	}
	system := p.BasePersonality + ". Return a phrase for the following prompt, but keep it brief - no longer than one or two sentences."
	out, err := llama.GenerateCompletion(client, system, prompt)
	if err != nil {
		log.Println("failed to generate completion:", err)
		return ""
	}
	return strings.Trim(out, "\"")
}

func (p *Personality) SaveToDisk() error {
	bytes, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.Create("personality/" + p.ID + ".json")
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}

func Load(id string) (*Personality, error) {
	bytes, err := os.ReadFile("personality/" + id + ".json")
	if err != nil {
		return nil, err
	}
	var p Personality
	if err := json.Unmarshal(bytes, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// generates a list of phrases based on the given phrase prompt. meant for building the AI personality's phrase set, which are used outside of the dynamic conversation time.
func buildPhrases(phrasePrompt string, basePersonality string, count int, apiKey string) []string {
	phrases := make([]string, 0)
	instructions := "\nUsing this personality, generate a phrase based on the following prompt and context, and return just the phrase, without quotes or anything else.\n"
	prompt := basePersonality + instructions + phrasePrompt

	input := []openai.Message{
		{
			Role:    "system",
			Content: prompt,
		},
	}
	messages := openai.MakeAPICall(apiKey, input)
	phrases = append(phrases, messages[len(messages)-1].Content)

	for i := 1; i < count; i++ {
		messages = append(messages, openai.Message{
			Role:    "user",
			Content: "Generate another phrase, please",
		})
		messages = openai.MakeAPICall(apiKey, messages)
		phrase := messages[len(messages)-1].Content
		util.PrintlnColor(util.Gray, phrase)
		phrases = append(phrases, phrase)
	}
	return phrases
}

func NewPersonalitySetup() {
	p := Personality{}
	q := false
	defer func() {
		if q {
			util.PrintlnColor(util.Gray, "New personality setup aborted.")
		}
	}()
	input := ""
	util.PrintlnColor(util.Hi_blue, "==============\n", "New Personality Setup\n", "==============")
	util.PrintlnColor(util.Gray, "(To cancel, enter 'q' at any time)")
	for p.ID == "" {
		fmt.Print("\nPersonality ID:")
		input, q = util.Input()
		if q {
			return
		}
		p.ID = strings.ReplaceAll(strings.TrimSpace(input), " ", "_")
	}
	for p.Name == "" {
		fmt.Print("\nAI Name:")
		input, q = util.Input()
		if q {
			return
		}
		p.Name = input
	}

	fmt.Println("\n\nEnter a brief description of the AI's personality. This will be used as context for all prompts that are given to the OpenAI APIs, to represent how this AI should speak and behave in general.")
	fmt.Println("For example:")
	fmt.Println("\"You are a butler from Victorian era England, who speaks in polite language of the time period\"")
	for p.BasePersonality == "" {
		fmt.Print("\nDescription:")
		input, q = util.Input()
		if q {
			return
		}
		p.BasePersonality = input
		if p.BasePersonality != "" {
			util.PrintlnColor(util.Gray, "You entered:", p.BasePersonality)
			if !util.PromptYN("Confirm?") {
				p.BasePersonality = ""
			}
		}
	}

	fmt.Println("\n\nNext, enter prompts for generating phrases. Each prompt should describe how the AI should say the corresponding phrase.")
	fmt.Println("For example, a prompt for the Greeting phrase:")
	fmt.Println("\"Greet me as a butler would greet the person he serves, and tell me that mail has arrived that you will relay for me\"")
	fmt.Println("\n\nGreeting - What the AI assistant will say when starting an interaction with you.")
	p.PhrasePrompts = make(map[string]string)
	for p.PhrasePrompts["greeting"] == "" {
		fmt.Print("Greeting:")
		input, q = util.Input()
		if q {
			return
		}
		p.PhrasePrompts["greeting"] = input
	}

	fmt.Println("\n\nDismiss - What the AI assistant will say when they are dismissed/are done relaying messages to you.")
	for p.PhrasePrompts["dismiss"] == "" {
		fmt.Print("Dismiss:")
		input, q = util.Input()
		if q {
			return
		}
		p.PhrasePrompts["dismiss"] = input
	}

	util.PrintlnColor(util.Hi_blue, "=== Review ===")
	util.PrintlnColor(util.Hi_blue, "ID:", p.ID)
	util.PrintlnColor(util.Hi_blue, "Name:", p.Name)
	util.PrintlnColor(util.Hi_blue, "Description:", p.BasePersonality)
	util.PrintlnColor(util.Hi_blue, "Greeting:", p.PhrasePrompts["greeting"])
	util.PrintlnColor(util.Hi_blue, "Dismiss:", p.PhrasePrompts["dismiss"])

	if !util.PromptYN("Confirm?") {
		return
	}

	util.PrintlnColor(util.Gray, "Saving personality JSON...")
	err := p.SaveToDisk()
	if err != nil {
		log.Println("failed to save personality json:", err)
		return
	}
	util.PrintlnColor(util.Green, "Saved to /personality/"+p.ID+".json!")

	fmt.Println("Next steps: find your personality JSON and set the paths to the prompt text file you want to use. Then, to use this personality, set it in your config.json.")
}

func (p Personality) FormatPrompt(Username, prompt, messageToReply, from, senderName, subject string) string {
	output := prompt
	if p.InsertDict == nil {
		p.InsertDict = make(map[string]string)
	}
	p.InsertDict["user-name"] = Username
	p.InsertDict["ai-name"] = p.Name
	p.InsertDict["from"] = from
	if senderName != "" {
		p.InsertDict["from"] += "(" + senderName + ")"
	}
	p.InsertDict["subject"] = subject
	for key, val := range p.InsertDict {
		key = "<<" + strings.ToUpper(key) + ">>"
		output = strings.ReplaceAll(output, key, val)
	}
	if messageToReply != "" {
		output = fmt.Sprintf(output, messageToReply)
	}
	return output
}
