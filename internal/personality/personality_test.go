package personality

import "testing"

func TestFormatPrompt(t *testing.T) {
	tests := []struct {
		p              Personality
		prompt         string
		username       string
		messageToReply string
		expOut         string
	}{
		{
			Personality{
				Name: "AI",
			},
			"Your name is <<AI-NAME>> and you are the assistant of <<USER-NAME>>. %s",
			"Userdude",
			"Here's a message for you to read.",
			"Your name is AI and you are the assistant of Userdude. Here's a message for you to read.",
		},
		{
			Personality{
				Name: "Jimmy John",
			},
			"Hello! My name is <<USER-NAME>> and you are <<AI-NAME>>, an assistant who reads me emails.\nHere's an email for you to read:\n%s",
			"James Jameson",
			"Hi,\nmy name is Dave and I like pickles.\nRegards,\nDave!",
			"Hello! My name is James Jameson and you are Jimmy John, an assistant who reads me emails.\nHere's an email for you to read:\nHi,\nmy name is Dave and I like pickles.\nRegards,\nDave!",
		},
	}

	for _, test := range tests {
		out := test.p.FormatPrompt(test.username, test.prompt, test.messageToReply)
		if out != test.expOut {
			t.Errorf("wrong output. expected: %q\noutput: %q", test.expOut, out)
		}
	}
}
