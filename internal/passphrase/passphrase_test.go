package passphrase

import (
	"testing"
)

// Test basic validation functionality
func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		passphrase  string
		expectValid bool
	}{
		{"Valid passphrase", "ValidPassword123!", true},
		{"Too short", "short", false},
		{"Missing uppercase", "validpassword123!", false},
		{"Missing lowercase", "VALIDPASSWORD123!", false},
		{"Missing digit", "ValidPassword!", false},
		{"Missing special", "ValidPassword123", false},
		{"Valid minimum", "Password123!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Validate(tt.passphrase)
			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, result.Valid)
			}
		})
	}
}

// Test hybrid validation functionality
func TestValidateHybrid(t *testing.T) {
	tests := []struct {
		name        string
		passphrase  string
		expectValid bool
	}{
		{"Valid strong passphrase", "MyUniquePassword123!", true},
		{"Too short", "short", false},
		{"Missing requirements", "password", false},
		{"Valid but common", "Password123!", true}, // May have warnings but still valid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateHybrid(tt.passphrase)
			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v", tt.expectValid, result.Valid)
			}
			// Check that result has expected fields
			if result.Strength == "" {
				t.Error("Expected strength to be set")
			}
		})
	}
}

// Test error message formatting
func TestGetErrorMessage(t *testing.T) {
	result := ValidationResult{
		Valid:  false,
		Errors: []string{"test error"},
	}

	msg := GetErrorMessage(result)
	if msg == "" {
		t.Error("Expected error message, got empty string")
	}

	// Test valid result
	validResult := ValidationResult{Valid: true}
	validMsg := GetErrorMessage(validResult)
	if validMsg != "" {
		t.Error("Expected empty message for valid result")
	}
}

// Test hybrid error message formatting
func TestGetHybridErrorMessage(t *testing.T) {
	result := HybridValidationResult{
		Valid:  false,
		Errors: []string{"test error"},
	}

	msg := GetHybridErrorMessage(result)
	if msg == "" {
		t.Error("Expected error message, got empty string")
	}

	// Test valid result with no warnings
	validResult := HybridValidationResult{Valid: true}
	validMsg := GetHybridErrorMessage(validResult)
	if validMsg != "" {
		t.Error("Expected empty message for valid result with no warnings")
	}
}

// Test strength assessment
func TestGetStrength(t *testing.T) {
	tests := []struct {
		name       string
		passphrase string
		expected   string
	}{
		{"Very weak", "abc", "Very Weak"},
		{"Weak invalid", "password", "Weak"},
		{"Strong valid", "StrongPassword123!", "Strong"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strength := GetStrength(tt.passphrase)
			if strength != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, strength)
			}
		})
	}
}

// Test special character detection
func TestIsSpecialChar(t *testing.T) {
	if !isSpecialChar('!') {
		t.Error("Expected '!' to be special character")
	}
	if !isSpecialChar('@') {
		t.Error("Expected '@' to be special character")
	}
	if isSpecialChar('a') {
		t.Error("Expected 'a' to not be special character")
	}
	if isSpecialChar('1') {
		t.Error("Expected '1' to not be special character")
	}
}

// Test format crack time
func TestFormatCrackTime(t *testing.T) {
	result := formatCrackTime(nil)
	if result != "unknown" {
		t.Errorf("Expected 'unknown' for nil, got %s", result)
	}

	result = formatCrackTime("2 hours")
	if result != "2 hours" {
		t.Errorf("Expected '2 hours', got %s", result)
	}
}

// Test basic validation (internal function)
func TestValidateBasicFunction(t *testing.T) {
	result := validateBasic("ValidPassword123!")
	if !result.Valid {
		t.Error("Expected valid result for good password")
	}

	result = validateBasic("short")
	if result.Valid {
		t.Error("Expected invalid result for short password")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for invalid password")
	}
}
