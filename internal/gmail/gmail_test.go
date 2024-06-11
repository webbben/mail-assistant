package gmail

import "testing"

func TestExtractEmailAddressFromHeader(t *testing.T) {
	tests := []struct {
		input string
		email string
		name  string
	}{
		{"First Last <email.addr@gmail.com>", "email.addr@gmail.com", "First Last"},
		{"email.addr@gmail.com", "email.addr@gmail.com", ""},
		{"No email here", "", ""},
		{"Another Name <another.email@example.com>", "another.email@example.com", "Another Name"},
	}

	for _, test := range tests {
		email, name := extractEmailAndName(test.input)
		if email != test.email {
			t.Errorf("expected email %q but got %q for input %q", test.email, email, test.input)
		}
		if name != test.name {
			t.Errorf("expected name %s but got %s for input %s", test.name, name, test.input)
		}
	}
}
