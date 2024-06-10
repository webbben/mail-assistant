package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/webbben/valet-de-chambre/internal/openai"
	"github.com/webbben/valet-de-chambre/internal/personality"
)

func main() {
	rebuild := flag.String("rebuild", "", "rebuild the phrase sets for a personality file. the personality file should be specified just as the filename, without .json, and its assumed it is within the /personality directory from the project root.")
	phrases := flag.String("phrases", "", "phrases to rebuild. each phrase name should be lowercase and delimited by commas, e.g. -phrases=dismiss,ignore")
	flag.Parse()

	apiKey := openai.LoadAPIKey()
	if apiKey == "" {
		fmt.Println("failed to load apikey; aborting personality setup")
		return
	}

	if *rebuild == "" {
		personality.NewPersonalitySetup(apiKey)
		return
	}
	p, err := personality.Load(*rebuild)
	if err != nil {
		fmt.Println("Failed to load personality file. Are you sure you entered a valid personality ID?", err)
		os.Exit(1)
	}
	if *phrases == "" {
		p.BuildPhrases(apiKey)
	} else {
		p.RebuildPhrases(apiKey, *phrases)
	}
	p.SaveToDisk()
}
