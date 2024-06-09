package main

import (
	"fmt"

	"github.com/webbben/valet-de-chambre/internal/openai"
	"github.com/webbben/valet-de-chambre/internal/personality"
)

func main() {
	apiKey := openai.LoadAPIKey()
	if apiKey == "" {
		fmt.Println("failed to load apikey; aborting personality setup")
		return
	}
	personality.NewPersonalitySetup(apiKey)
}
