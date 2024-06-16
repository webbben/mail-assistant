package util

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	Yellow  = color.New(color.FgYellow)
	Green   = color.New(color.FgGreen)
	Hi_blue = color.New(color.FgHiBlue)
	Gray    = color.New(color.FgHiBlack)
)

// get user input in a conversation
func GetUserInput() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nUser: ")
	resp, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("failed to read user input")
		return ""
	}
	return strings.TrimSpace(resp)
}

// get user input, and also returns if the input is a quit input
func Input() (string, bool) {
	reader := bufio.NewReader(os.Stdin)
	resp, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("failed to read user input")
		return "", false
	}
	resp = strings.TrimSpace(resp)
	return resp, IsQuit(resp)
}

func IsQuit(s string) bool {
	s = strings.ToLower(s)
	return s == "q" || s == "quit" || s == "exit"
}

// returns true if the user answers Yes
func PromptYN() bool {
	fmt.Print("[Y/N]:")
	return IsYes(GetUserInput())
}

func IsYes(s string) bool {
	s = strings.ToLower(s)
	return s == "y" || s == "yes"
}

func SomeoneTalks(name string, statement string, c *color.Color) {
	if c == nil {
		c = color.New(color.Reset)
	}
	sf := c.SprintFunc()
	fmt.Printf(sf("\n%s: %s\n"), name, statement)
}

func PrintlnColor(c *color.Color, a ...any) {
	pf := c.PrintlnFunc()
	pf(a...)
}

func PrintfColor(c *color.Color, s string, a ...any) {
	pf := c.PrintfFunc()
	pf(s, a...)
}

func Continue() {
	fmt.Print("Press Enter to continue...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}
