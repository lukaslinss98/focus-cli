package domain

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"youtube.com", "youtube.com"},
		{"www.youtube.com", "youtube.com"},
		{"HTTPS://WWW.Reddit.COM/r/go", "reddit.com"},
		{"news.ycombinator.com.", "news.ycombinator.com"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := Normalize(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Errorf("Normalize(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestNormalizeRejectsInvalidWebsites(t *testing.T) {
	for _, input := range []string{"", "localhost", "127.0.0.1", "youtube.com/path", "-youtube.com", "youtube..com"} {
		t.Run(input, func(t *testing.T) {
			if _, err := Normalize(input); err == nil {
				t.Errorf("Normalize(%q) succeeded", input)
			}
		})
	}
}
