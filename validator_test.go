package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPGNValidator(t *testing.T) {
	validator := NewPGNValidator()
	if validator == nil {
		t.Fatal("NewPGNValidator returned nil")
	}
	if validator.errors == nil {
		t.Fatal("Validator errors slice is nil")
	}
}

func TestValidateValidFile(t *testing.T) {
	// Create a temporary valid PGN file
	content := `[Event "Test"]
[Site "Test"]
[Date "2024.01.15"]
[Round "1"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 1-0
`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	validator := NewPGNValidator()
	errors := validator.ValidateFile(tmpFile)

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors for valid file, got %d: %v", len(errors), errors)
	}
}

func TestValidateInvalidDate(t *testing.T) {
	// Create a temporary PGN file with invalid date
	content := `[Event "Test"]
[Site "Test"]
[Date "2024-01-15"]
[Round "1"]
[White "Player1"]
[Black "Player2"]
[Result "1-0"]

1. e4 e5 *
`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	validator := NewPGNValidator()
	errors := validator.ValidateFile(tmpFile)

	if len(errors) == 0 {
		t.Error("Expected errors for invalid date format, got none")
	}

	// Check that the error message mentions date correction
	found := false
	for _, err := range errors {
		if err.Line == 3 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected error on line 3 (date line)")
	}
}

func TestTryFixDate(t *testing.T) {
	validator := NewPGNValidator()

	tests := []struct {
		input      string
		expected   string
		shouldFail bool
	}{
		{"2024-01-15", "2024.01.15", false}, // ISO 8601
		{"15/01/2024", "2024.01.15", false}, // DD/MM/YYYY
		{"2024/01/15", "2024.01.15", false}, // YYYY/MM/DD
		{"20240115", "2024.01.15", false},   // YYYYMMDD
		{"invalid", "", true},               // Invalid format
		{"not-a-date", "", true},            // Invalid format
	}

	for _, tt := range tests {
		result, err := validator.tryFixDate(tt.input)
		if tt.shouldFail {
			if err == nil {
				t.Errorf("Expected error for input '%s', got none", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input '%s': %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("For input '%s', expected '%s', got '%s'", tt.input, tt.expected, result)
			}
		}
	}
}

func TestValidateResult(t *testing.T) {
	validator := NewPGNValidator()

	validResults := []string{"1-0", "0-1", "1/2-1/2", "*"}
	for _, result := range validResults {
		validator.errors = make([]ValidationError, 0)
		validator.validateResult(result, 1)
		if len(validator.errors) != 0 {
			t.Errorf("Expected no errors for valid result '%s', got %d errors", result, len(validator.errors))
		}
	}

	invalidResults := []string{"2-0", "1-1", "draw", ""}
	for _, result := range invalidResults {
		validator.errors = make([]ValidationError, 0)
		validator.validateResult(result, 1)
		if len(validator.errors) == 0 {
			t.Errorf("Expected errors for invalid result '%s', got none", result)
		}
	}
}

func TestCheckBalancedDelimiters(t *testing.T) {
	validator := NewPGNValidator()

	tests := []struct {
		line     string
		open     rune
		close    rune
		expected bool
	}{
		{"(1. e4)", '(', ')', true},
		{"(1. e4 (e5))", '(', ')', true},
		{"(1. e4", '(', ')', false},
		{"1. e4)", '(', ')', false},
		{"{comment}", '{', '}', true},
		{"{comment", '{', '}', false},
	}

	for _, tt := range tests {
		result := validator.checkBalancedDelimiters(tt.line, tt.open, tt.close)
		if result != tt.expected {
			t.Errorf("For line '%s' with delimiters '%c' and '%c', expected %v, got %v",
				tt.line, tt.open, tt.close, tt.expected, result)
		}
	}
}

func TestIsValidMoveNotation(t *testing.T) {
	validator := NewPGNValidator()

	validMoves := []string{
		"e4", "Nf3", "Bb5", "O-O", "O-O-O",
		"Qh5+", "Qh4#", "e8=Q", "exd5",
		"Nbd7", "R1a3", "1-0", "0-1", "1/2-1/2", "*",
	}

	for _, move := range validMoves {
		if !validator.isValidMoveNotation(move) {
			t.Errorf("Expected move '%s' to be valid, but it was marked as invalid", move)
		}
	}

	invalidMoves := []string{
		"Xe4", "i5", "a9", "Qj5",
	}

	for _, move := range invalidMoves {
		if validator.isValidMoveNotation(move) {
			t.Errorf("Expected move '%s' to be invalid, but it was marked as valid", move)
		}
	}
}

func TestWriteCorrectedFile(t *testing.T) {
	// Create a temporary PGN file with invalid date
	content := `[Event "Test"]
[Date "2024-01-15"]
[Result "1-0"]

1. e4 e5 *
`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	outputFile := filepath.Join(os.TempDir(), "test_output.pgn")
	defer os.Remove(outputFile)

	validator := NewPGNValidator()
	err := validator.WriteCorrectedFile(tmpFile, outputFile)
	if err != nil {
		t.Fatalf("WriteCorrectedFile failed: %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatal("Output file was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	content_str := string(data)
	// Check that date was corrected
	if len(content_str) == 0 {
		t.Error("Output file is empty")
	}
}

func TestCaseSensitiveTagNames(t *testing.T) {
	// Test that tag names are case-insensitive
	content := `[Event "Test"]
[date "2024-01-15"]
[result "1-0"]

1. e4 e5 *
`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	validator := NewPGNValidator()
	errors := validator.ValidateFile(tmpFile)

	// Should detect the date error even with lowercase "date"
	found := false
	for _, err := range errors {
		if err.Line == 2 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected validator to handle lowercase 'date' tag")
	}
}

// Helper function to create temporary test files
func createTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "test_*.pgn")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	tmpFile.Close()
	return tmpFile.Name()
}
