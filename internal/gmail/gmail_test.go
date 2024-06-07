package gmail

import "testing"

func TestExtractEmailAddressFromHeader(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"First Last <email.addr@gmail.com>", "email.addr@gmail.com"},
		{"email.addr@gmail.com", "email.addr@gmail.com"},
		{"No email here", ""},
		{"Another Name <another.email@example.com>", "another.email@example.com"},
	}

	for _, test := range tests {
		result := extractEmailAddr(test.input)
		if result != test.expected {
			t.Errorf("expected %q but got %q for input %q", test.expected, result, test.input)
		}
	}
}
