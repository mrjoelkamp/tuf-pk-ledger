package ledger

import (
	"testing"
)

func TestParseProviderURI(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		scheme   string
		host     string
		path     string
	}{
		{"no trailing slash", "https://test.com", "https://test.com", "https", "test.com", ""},
		{"trailing slash", "https://test.com/", "https://test.com/", "https", "test.com", "/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseProviderURI(tc.input)
			if err != nil {
				t.Fatalf(err.Error())
			}

			if result.String() != tc.expected {
				t.Errorf("For input %s, expected %s, but got %s", tc.input, tc.expected, result)
			}
			if result.Scheme != tc.scheme {
				t.Errorf("For input %s, expected %s, but got %s", tc.input, tc.scheme, result.Scheme)
			}
			if result.Host != tc.host {
				t.Errorf("For input %s, expected %s, but got %s", tc.input, tc.host, result.Host)
			}
			if result.Path != tc.path {
				t.Errorf("For input %s, expected %s, but got %s", tc.input, tc.path, result.Path)
			}
		})
	}
}
