package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ollama/ollama/api"
	"github.com/webbben/mail-assistant/internal/llama"
	"github.com/webbben/mail-assistant/internal/util"
)

func main() {
	system := flag.String("system", "", "provide a system prompt to define the context of the chat. e.g. -system=\"You are an old timey detective investigating a murder case, and you are subtly suspicious of me\"")
	flag.Parse()

	cmd, err := llama.StartServer()
	if err != nil {
		fmt.Println("Failed to start ollama server:", err)
		os.Exit(1)
	}
	defer llama.StopServer(cmd)

	client, err := llama.GetClient()
	if err != nil {
		fmt.Println("Failed to get ollama client:", err)
		os.Exit(1)
	}

	messages := []api.Message{}
	if *system != "" {
		messages = append(messages, api.Message{
			Role:    "system",
			Content: *system,
		})
	}
	util.SomeoneTalks("SYS", "(Enter 'q' to quit at any time)", util.Gray)
	for {
		input := util.GetUserInput()
		if util.IsQuit(input) {
			return
		}
		messages = append(messages, api.Message{
			Role:    "user",
			Content: input,
		})
		messages, err = llama.ChatCompletion(client, messages)
		if err != nil {
			fmt.Println("error doing chat completion:", err)
			return
		}
		util.SomeoneTalks("Bot", messages[len(messages)-1].Content, util.Hi_blue)
	}
}
