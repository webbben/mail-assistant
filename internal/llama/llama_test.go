package llama

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"testing"
)

type IsSpamTestCase struct {
	email  string
	isSpam bool
}

func getTestCase(filename string) *IsSpamTestCase {
	bytes, err := os.ReadFile("tests/spam/" + filename)
	if err != nil {
		log.Fatal("failed to load test case:", err)
	}
	sections := strings.Split(string(bytes), "<<TESTCASE>>")
	testCase := new(IsSpamTestCase)
	testCase.isSpam = strings.TrimSpace(sections[0]) == "TRUE"
	testCase.email = strings.TrimSpace(sections[1])
	return testCase
}

func loadTestCases() []*IsSpamTestCase {
	numTests := 9
	testCases := make([]*IsSpamTestCase, 0)
	for i := 0; i < numTests; i++ {
		testCases = append(testCases, getTestCase(fmt.Sprintf("%v.txt", i)))
	}
	return testCases
}

// run in terminal:
// go test -run ^TestIsEmailSpam$ github.com/webbben/mail-assistant/internal/llama
// by default VS code uses a timeout of 30s, but I removed it since the LLM startup takes a bit, and each call to the LLM takes a few seconds on average
func TestIsEmailSpam(t *testing.T) {
	// ollama
	cmd, err := StartServer()
	if err != nil {
		log.Fatal("failed to start ollama server:", err)
	}
	defer StopServer(cmd)

	client, err := GetClient()
	if err != nil {
		log.Fatal("failed to get ollama client:", err)
	}

	tests := loadTestCases()
	pass := 0
	falsePositive := 0
	invalid := 0

	for i, test := range tests {
		isSpam, raw := IsEmailSpam(client, test.email)
		if isSpam != test.isSpam {
			if isSpam {
				falsePositive++
			} else {
				// wrong format?
				if !strings.Contains(raw, ";") {
					invalid++
				}
			}
			t.Errorf("case: %v, spam detection incorrect. expected: %v, output: %v", i, test.isSpam, isSpam)
			t.Log(raw)
		} else {
			pass++
		}
	}
	t.Logf("Pass: %v/%v (%v%%)", pass, len(tests), math.Round(float64(pass)/float64(len(tests))*100))
	t.Logf("False positives: %v/%v (%v%%)", falsePositive, len(tests), math.Round(float64(falsePositive)/float64(len(tests))*100))
	t.Logf("Wrong format: %v/%v (%v%%)", invalid, len(tests), math.Round(float64(invalid)/float64(len(tests))*100))
}
