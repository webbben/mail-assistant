package assistant

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/webbben/mail-assistant/internal/llama"
	"github.com/webbben/mail-assistant/internal/personality"
	"github.com/webbben/mail-assistant/internal/types"
)

type AutoReplyTestCase struct {
	email       types.Email
	shouldReply bool
}

func getTestCase(filename string) *AutoReplyTestCase {
	bytes, err := os.ReadFile("tests/autoreply/" + filename)
	if err != nil {
		log.Fatal("failed to load test case:", err)
	}
	sections := strings.Split(string(bytes), "<<TESTCASE>>")
	testCase := new(AutoReplyTestCase)
	testCase.email = types.Email{
		From:    sections[0],
		Subject: sections[1],
		Body:    sections[2],
	}
	testCase.shouldReply = strings.TrimSpace(sections[3]) == "TRUE"
	return testCase
}

func loadTestCases() []*AutoReplyTestCase {
	numTests := 2
	testCases := make([]*AutoReplyTestCase, 0)
	for i := 0; i < numTests; i++ {
		testCases = append(testCases, getTestCase(fmt.Sprintf("%v.txt", i)))
	}
	return testCases
}

var test_p personality.Personality = personality.Personality{
	Name:            "Planchet",
	BasePersonality: "You are a servant from 18th century France, serving a great lord. You speak in the language of the era (in English though), and assume a lofty, sometimes pretentious air.",
	InsertDict: map[string]string{
		"user-name-last": "Webb",
	},
}

// go test -run ^TestAutoReply$ github.com/webbben/mail-assistant/internal/assistant
func TestAutoReply(t *testing.T) {
	// ollama
	cmd, err := llama.StartServer()
	if err != nil {
		log.Fatal("failed to start ollama server:", err)
	}
	defer llama.StopServer(cmd)

	client, err := llama.GetClient()
	if err != nil {
		log.Fatal("failed to get ollama client:", err)
	}

	categories := []string{
		"Work at Epic; The \"hamu\" project",
		"Mapo tofu",
		"Anything related to debts or unpaid expenses",
	}
	instructions := [][]string{
		{
			"Respond that I'm out of office until July 15th",
			"If they mention the hamu project, refer them to reach out to Dave, who is my back-up for this project.",
		},
		{
			"If the message is about setting up a date or appointment to eat mapo tofu, tell them I accept",
			"If the message is instead saying something negative about mapo tofu, tell them they are filthy heretics and should feel ashamed.",
		},
		{
			"If the message is about unpaid debt to a loanshark named \"Balboni\", tell him that we will get him the money next Tuesday",
			"If the message is about a dodged bill at Applebees, tell them they surely have the wrong guy",
		},
	}

	tests := loadTestCases()
	pass := 0
	falsePositive := 0
	invalid := 0
	outputLogs := make([]string, 0)

	for i, test := range tests {
		logString := fmt.Sprintf("==========\n   CASE %v\n==========\nEMAIL:\n%s", i, test.email)
		out, err := AutoReply(client, test.email, "Ben Webb", test_p, categories, instructions)
		if err != nil {
			t.Errorf("case: %v, error occurred: %s", i, err)
		}
		logString += fmt.Sprintf("\n----------\nREPLY:\n%s\n==========", out)
		outputLogs = append(outputLogs, logString)
		if out != "" {
			if test.shouldReply {
				pass++
			} else {
				falsePositive++
				t.Errorf("case: %v, output response when not supposed to.", i)
				t.Log(out)
			}
			if !strings.Contains(out, "~~~") {
				t.Logf("case: %v, output is invalid format (missing ~~~)", i)
				t.Log(out)
				invalid++
			}
		} else {
			if test.shouldReply {
				t.Errorf("case: %v, expected output response, but none given.", i)
			} else {
				pass++
			}
		}
	}
	t.Logf("Pass: %v/%v (%v%%)", pass, len(tests), math.Round(float64(pass)/float64(len(tests))*100))
	t.Logf("False positives: %v/%v (%v%%)", falsePositive, len(tests), math.Round(float64(falsePositive)/float64(len(tests))*100))
	t.Logf("Wrong format: %v/%v (%v%%)", invalid, len(tests), math.Round(float64(invalid)/float64(len(tests))*100))
	WriteLogs("tests/autoreply/log.txt", outputLogs, t)
}

func WriteLogs(path string, content []string, t *testing.T) {
	// writes the current data in the in-memory cache to the disk, overwriting any previous data in the file
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Println("error writing logs:", err)
		t.Error("failed to write logs...")
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, s := range content {
		_, err := writer.WriteString(s + "\n")
		if err != nil {
			log.Println("error writing string to log:", err)
			t.Error("failed to write logs...")
			return
		}
	}
	if err = writer.Flush(); err != nil {
		log.Println("error flushing log writer:", err)
		t.Error("failed to write logs...")
	}

}

func TestGetEmailCategories(t *testing.T) {
	cat1 := []string{
		"Software engineering, Epic, or the \"hamu\" project",
		"Mapo tofu",
		"Anything about debts, asking for money, or unpaid expenses",
	}
	tests := []struct {
		categories    []string
		email         types.Email
		expCategories []int
	}{
		{
			cat1,
			types.Email{
				From:    "somedude@gmail.com",
				Subject: "Mapo Tofu",
				Body:    "Hey man!\nDo you like mapo tofu? I ate some for the first time the other day and thought it wasn't that good, honestly. Do you think I just had a bad one?\n\n-Dave",
			},
			[]int{2},
		},
		{
			cat1,
			types.Email{
				From:    "somedude@gmail.com",
				Subject: "Lunch on Saturday",
				Body:    "How's it going?\nAre you free on Saturday? I was thinking of trying this new Chinese restaurant. I heard it has some pretty good mapo tofu too.\n\n-Dave",
			},
			[]int{2},
		},
		{
			cat1,
			types.Email{
				From:    "amy@epic.com",
				Subject: "Meeting next Monday",
				Body:    "Hi John,\nDo you think you could join a meeting we're having on Monday about the new software update? I had some questions that I thought you might know the answer to.\n\nThanks,\nAmy\nEpic Sales Team",
			},
			[]int{1},
		},
		{
			cat1,
			types.Email{
				From:    "tommytuffknuckles@gmail.com",
				Subject: "Pay up or else!",
				Body:    "Is this Phil? It's Tommy - I work for Balboni and you know he's pretty pissed right now. You still owe him that $20 he lent you, and it's time to pay up.",
			},
			[]int{3},
		},
		{
			cat1,
			types.Email{
				From:    "randomguy@gmail.com",
				Subject: "Italian food",
				Body:    "Hey dude, it's Jack.\nWhat was that Italian restaurant you were telling me about the other day? I wanna try it out!\nLater.",
			},
			[]int{0},
		},
		{
			cat1,
			types.Email{
				From:    "tim@epic.com",
				Subject: "Lunch tomorrow",
				Body:    "Hey man, are you free tomorrow for lunch? I was thinking after the meeting for the hamu project, we could leave the office and go check out that mapo tofu restaurant.\n-Tim",
			},
			[]int{1, 2},
		},
		{
			cat1,
			types.Email{
				From:    "tim@epic.com",
				Subject: "Lunch tomorrow",
				Body:    "Hey man, are you free tomorrow for lunch? I was thinking after the meeting for the hamu project, we could leave the office and go check out that mapo tofu restaurant.\nAlso, would you mind paying me back for the taxi last weekend? I think you owe me $25 for your share of the ride.\n-Tim",
			},
			[]int{1, 2, 3},
		},
	}

	// ollama
	cmd, err := llama.StartServer()
	if err != nil {
		log.Fatal("failed to start ollama server:", err)
	}
	defer llama.StopServer(cmd)

	client, err := llama.GetClient()
	if err != nil {
		log.Fatal("failed to get ollama client:", err)
	}
	pass := 0
	for i, test := range tests {
		cats := getEmailCategories(client, test.email, test.categories)
		if len(cats) != len(test.expCategories) {
			t.Errorf("case %v, output: %v, exp: %v", i, cats, test.expCategories)
			continue
		}
		fail := false
		for _, c := range test.expCategories {
			if !slices.Contains(cats, c) {
				t.Errorf("case %v, output: %v, exp: %v", i, cats, test.expCategories)
				fail = true
				break
			}
		}
		if !fail {
			pass++
		}
	}
	t.Logf("Pass: %v/%v (%v%%)", pass, len(tests), math.Round(float64(pass)/float64(len(tests))*100))
}
