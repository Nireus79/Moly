package main

import (
	"testing"
)

func TestValidateString(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name       string
		value      string
		required   bool
		minLen     int
		maxLen     int
		shouldFail bool
	}{
		{
			name:       "Empty optional string",
			value:      "",
			required:   false,
			shouldFail: false,
		},
		{
			name:       "Empty required string",
			value:      "",
			required:   true,
			shouldFail: true,
		},
		{
			name:       "Valid string",
			value:      "hello world",
			required:   true,
			minLen:     1,
			maxLen:     20,
			shouldFail: false,
		},
		{
			name:       "String too short",
			value:      "hi",
			required:   true,
			minLen:     5,
			maxLen:     20,
			shouldFail: true,
		},
		{
			name:       "String too long",
			value:      "this is a very long string that exceeds max length",
			required:   true,
			minLen:     1,
			maxLen:     10,
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateString("test_field", tc.value, tc.required, tc.minLen, tc.maxLen)

			if tc.shouldFail && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.shouldFail && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name       string
		email      string
		shouldFail bool
	}{
		{
			name:       "Valid email",
			email:      "user@example.com",
			shouldFail: false,
		},
		{
			name:       "Valid email with subdomain",
			email:      "user@mail.example.com",
			shouldFail: false,
		},
		{
			name:       "Invalid email no @",
			email:      "userexample.com",
			shouldFail: true,
		},
		{
			name:       "Invalid email no domain",
			email:      "user@.com",
			shouldFail: true,
		},
		{
			name:       "Empty email",
			email:      "",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateEmail(tc.email)

			if tc.shouldFail && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.shouldFail && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name       string
		url        string
		shouldFail bool
	}{
		{
			name:       "Valid https URL",
			url:        "https://example.com",
			shouldFail: false,
		},
		{
			name:       "Valid http URL",
			url:        "http://example.com",
			shouldFail: false,
		},
		{
			name:       "Valid chrome-extension URL",
			url:        "chrome-extension://abcd1234/sidebar.html",
			shouldFail: false,
		},
		{
			name:       "Invalid URL no protocol",
			url:        "example.com",
			shouldFail: true,
		},
		{
			name:       "Empty URL",
			url:        "",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateURL(tc.url)

			if tc.shouldFail && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.shouldFail && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateMessageContent(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name       string
		message    string
		shouldFail bool
	}{
		{
			name:       "Valid message",
			message:    "Hello, how are you?",
			shouldFail: false,
		},
		{
			name:       "Empty message",
			message:    "",
			shouldFail: true,
		},
		{
			name:       "Very long message",
			message:    "This is a very long message that " + "definitely exceeds " + "the maximum length " + "of 5000 characters " + "so it should fail " + "because the validator " + "should reject it.",
			shouldFail: false, // Still under 5000
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateMessageContent(tc.message)

			if tc.shouldFail && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.shouldFail && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateContactName(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name       string
		contact    string
		shouldFail bool
	}{
		{
			name:       "Valid name",
			contact:    "John Doe",
			shouldFail: false,
		},
		{
			name:       "Valid name with hyphen",
			contact:    "Mary-Jane",
			shouldFail: false,
		},
		{
			name:       "Valid name with apostrophe",
			contact:    "O'Brien",
			shouldFail: false,
		},
		{
			name:       "Invalid name with special chars",
			contact:    "John@Doe#",
			shouldFail: true,
		},
		{
			name:       "Empty name",
			contact:    "",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateContactName(tc.contact)

			if tc.shouldFail && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.shouldFail && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateMode(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name       string
		mode       string
		shouldFail bool
	}{
		{
			name:       "Valid socratic mode",
			mode:       "socratic",
			shouldFail: false,
		},
		{
			name:       "Valid direct mode",
			mode:       "direct",
			shouldFail: false,
		},
		{
			name:       "Invalid mode",
			mode:       "invalid",
			shouldFail: true,
		},
		{
			name:       "Empty mode",
			mode:       "",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateMode(tc.mode)

			if tc.shouldFail && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.shouldFail && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateContext(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name       string
		context    string
		shouldFail bool
	}{
		{
			name:       "Valid formal context",
			context:    "formal",
			shouldFail: false,
		},
		{
			name:       "Valid friendly context",
			context:    "friendly",
			shouldFail: false,
		},
		{
			name:       "Valid dating context",
			context:    "dating",
			shouldFail: false,
		},
		{
			name:       "Invalid context",
			context:    "casual",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateContext(tc.context)

			if tc.shouldFail && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.shouldFail && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateProvider(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name       string
		provider   string
		shouldFail bool
	}{
		{
			name:       "Valid claude provider",
			provider:   "claude",
			shouldFail: false,
		},
		{
			name:       "Valid openai provider",
			provider:   "openai",
			shouldFail: false,
		},
		{
			name:       "Valid ollama provider",
			provider:   "ollama",
			shouldFail: false,
		},
		{
			name:       "Invalid provider",
			provider:   "gemini",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateProvider(tc.provider)

			if tc.shouldFail && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.shouldFail && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateAPIKey(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name       string
		apiKey     string
		shouldFail bool
	}{
		{
			name:       "Valid API key",
			apiKey:     "sk-123456789abcdef",
			shouldFail: false,
		},
		{
			name:       "Valid API key with hyphens",
			apiKey:     "sk-abc-def-ghi-jkl",
			shouldFail: false,
		},
		{
			name:       "Valid API key with underscores",
			apiKey:     "sk_abc_def_ghi",
			shouldFail: false,
		},
		{
			name:       "Invalid API key with spaces",
			apiKey:     "sk 123456 abcdef",
			shouldFail: true,
		},
		{
			name:       "Too short API key",
			apiKey:     "short",
			shouldFail: true,
		},
		{
			name:       "Empty API key",
			apiKey:     "",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateAPIKey(tc.apiKey)

			if tc.shouldFail && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.shouldFail && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal string",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "String with leading/trailing spaces",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "String with null bytes",
			input:    "hello\x00world",
			expected: "helloworld",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := v.SanitizeString(tc.input)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestValidateInteger(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name       string
		value      int
		minVal     int
		maxVal     int
		shouldFail bool
	}{
		{
			name:       "Valid integer",
			value:      50,
			minVal:     0,
			maxVal:     100,
			shouldFail: false,
		},
		{
			name:       "Integer below minimum",
			value:      -5,
			minVal:     0,
			maxVal:     100,
			shouldFail: true,
		},
		{
			name:       "Integer above maximum",
			value:      150,
			minVal:     0,
			maxVal:     100,
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateInteger("test_field", tc.value, tc.minVal, tc.maxVal)

			if tc.shouldFail && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.shouldFail && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
