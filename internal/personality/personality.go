package personality

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/webbben/valet-de-chambre/internal/openai"
	"github.com/webbben/valet-de-chambre/internal/util"
)

type Personality struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`             // name the AI assumes in interactions.
	BasePersonality string        `json:"base_personality"` // description of the overall personality and behavior of this AI assistant, used as context for the other prompts.
	Phrases         Phrases       `json:"phrases"`          // Generated phrases this AI uses in certain points of interactions
	PhraseBuilder   PhraseBuilder `json:"phrase_builder"`   // Prompts used for generating phrases
	Prompts         Prompts       `json:"prompts"`          // Prompts for main workflows (e.g. handling incoming emails)
}

type Phrases struct {
	Dismiss  []string `json:"dismiss"`  // phrases for dismissal; what the AI says when an interaction has concluded
	Greeting []string `json:"greeting"` // phrases for greeting; what the AI says when an interaction begins
}

type Prompts struct {
	EmailWorkflow string `json:"email_workflow"` // prompt for how the email handling workflow and interaction should work.
}

type PhraseBuilder struct {
	Dismiss  string `json:"dismiss"`
	Greeting string `json:"greeting"`
}

// use openAI's API to generate lists of phrases that this AI personality will use
func (p *Personality) BuildPhrases(apiKey string) {
	pb := p.PhraseBuilder
	p.Phrases.Dismiss = buildPhrases(pb.Dismiss, 10, apiKey)
	p.Phrases.Greeting = buildPhrases(pb.Greeting, 10, apiKey)
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

// generates a list of phrases based on the given phrase prompt. meant for building the AI personality's phrase set, which are used outside of the dynamic conversation time.
func buildPhrases(phrasePrompt string, count int, apiKey string) []string {
	phrases := make([]string, 0)
	promptBase := "You are an assistant that generates phrases. Generate a phrase based on the following prompt and context, and return just the phrase - nothing else.\n"
	prompt := promptBase + phrasePrompt

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

func NewPersonalitySetup(apiKey string) {
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
			fmt.Print("Confirm? ")
			if !util.PromptYN() {
				p.BasePersonality = ""
			}
		}
	}

	fmt.Println("\n\nNext, enter prompts for generating phrases. Each prompt should describe how the AI should say the corresponding phrase.")
	fmt.Println("For example, a prompt for the Greeting phrase:")
	fmt.Println("\"Greet me as a butler would greet the person he serves, and tell me that mail has arrived that you will relay for me\"")
	fmt.Println("\n\nGreeting - What the AI assistant will say when starting an interaction with you.")
	for p.PhraseBuilder.Greeting == "" {
		fmt.Print("Greeting:")
		input, q = util.Input()
		if q {
			return
		}
		p.PhraseBuilder.Greeting = input
	}

	fmt.Println("\n\nDismiss - What the AI assistant will say when they are dismissed/are done relaying messages to you.")
	for p.PhraseBuilder.Dismiss == "" {
		fmt.Print("Dismiss:")
		input, q = util.Input()
		if q {
			return
		}
		p.PhraseBuilder.Dismiss = input
	}

	util.PrintlnColor(util.Hi_blue, "=== Review ===")
	util.PrintlnColor(util.Hi_blue, "ID:", p.ID)
	util.PrintlnColor(util.Hi_blue, "Name:", p.Name)
	util.PrintlnColor(util.Hi_blue, "Description:", p.BasePersonality)
	util.PrintlnColor(util.Hi_blue, "Greeting:", p.PhraseBuilder.Greeting)
	util.PrintlnColor(util.Hi_blue, "Dismiss:", p.PhraseBuilder.Dismiss)

	fmt.Print("Confirm? ")
	if !util.PromptYN() {
		return
	}
	util.PrintlnColor(util.Gray, "Building phrase sets. This may take a minute...")
	p.BuildPhrases(apiKey)

	util.PrintlnColor(util.Gray, "Saving personality JSON...")
	err := p.SaveToDisk()
	if err != nil {
		log.Println("failed to save personality json:", err)
		return
	}
	util.PrintlnColor(util.Green, "Saved to /personality/"+p.ID+".json!")

	fmt.Println("Next steps: find your personality JSON and set the paths to the prompt text file you want to use. Then, to use this personality, set it in your config.json.")
}
