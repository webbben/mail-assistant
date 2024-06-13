package emailparse

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"testing"
)

type ParseEmailTestCase struct {
	raw        string
	parsedText string
	headers    map[string]string
}

func getTestCase(id string) *ParseEmailTestCase {
	bytes, err := os.ReadFile("tests/" + id + ".txt")
	if err != nil {
		log.Fatal("failed to load test case:", err)
	}
	sections := strings.Split(string(bytes), "<<TESTCASE>>")
	testCase := new(ParseEmailTestCase)
	testCase.raw = sections[0]
	testCase.parsedText = strings.TrimSpace(sections[1])

	testCase.headers = make(map[string]string)
	for _, headerLine := range strings.Split(sections[2], "///") {
		headerPieces := strings.Split(headerLine, ":")
		key := strings.TrimSpace(headerPieces[0])
		val := strings.TrimSpace(strings.Join(headerPieces[1:], ":"))
		testCase.headers[key] = val
	}
	return testCase
}

func loadTestCases() []*ParseEmailTestCase {
	numTests := 3
	testCases := make([]*ParseEmailTestCase, 0)
	for i := 0; i < numTests; i++ {
		testCases = append(testCases, getTestCase(fmt.Sprintf("email_%v", i)))
	}
	return testCases
}

func TestParseEmail(t *testing.T) {
	tests := loadTestCases()
	for i, test := range tests {
		parsedText, headers, err := ParseEmail(test.raw)
		if err != nil {
			t.Errorf("case: %v, an error occurred: %s", i, err.Error())
			continue
		}
		if parsedText != test.parsedText {
			t.Errorf("case: %v, wrong parsed text. expected: %q\noutput: %q", i, test.parsedText, parsedText)
		}
		match, n := 0, 0
		for k, v := range test.headers {
			if !slices.Contains(headers[k], v) {
				t.Errorf("case: %v, Expected header %s: %s\nGot: %s", i, k, v, headers[k])
			} else {
				match++
			}
			n++
		}
		t.Log("case:", i, ", header matches:", fmt.Sprintf("%v/%v", match, n))
	}
}
