package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/schollz/progressbar/v3"
)

// ValidationError represents a PGN validation error
type ValidationError struct {
	Line    int
	Message string
}

func (e ValidationError) String() string {
	return fmt.Sprintf("Line %d: %s", e.Line, e.Message)
}

// PGNValidator handles PGN file validation
type PGNValidator struct {
	errors []ValidationError
}

// NewPGNValidator creates a new validator instance
func NewPGNValidator() *PGNValidator {
	return &PGNValidator{
		errors: make([]ValidationError, 0),
	}
}

// ValidateFile validates a PGN file and returns a list of errors
func (v *PGNValidator) ValidateFile(filename string) []ValidationError {
	v.errors = make([]ValidationError, 0)

	file, err := os.Open(filename)
	if err != nil {
		v.errors = append(v.errors, ValidationError{
			Line:    0,
			Message: fmt.Sprintf("Cannot open file: %v", err),
		})
		return v.errors
	}
	defer file.Close()

	// Get file size for progress bar
	fileInfo, err := file.Stat()
	if err != nil {
		v.errors = append(v.errors, ValidationError{
			Line:    0,
			Message: fmt.Sprintf("Cannot get file info: %v", err),
		})
		return v.errors
	}
	fileSize := fileInfo.Size()

	// Create progress bar only for large files (> 1MB)
	var bar *progressbar.ProgressBar
	if fileSize > 1024*1024 {
		bar = progressbar.NewOptions64(
			fileSize,
			progressbar.OptionSetDescription("Validating"),
			progressbar.OptionSetWidth(40),
			progressbar.OptionShowBytes(true),
			progressbar.OptionUseIECUnits(false),
			progressbar.OptionSetPredictTime(true),
			progressbar.OptionShowCount(),
		)
	}

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	inHeader := true
	bytesRead := int64(0)

	// Pattern for PGN tags: [TagName "Value"]
	tagPattern := regexp.MustCompile(`^\[(\w+)\s+"(.*)"\]$`)

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		bytesRead += int64(len(line)) + 2 // +2 per newline (\r\n su Windows)

		// Aggiorna progress bar ogni 100 linee per migliori performance
		if bar != nil && lineNumber%100 == 0 {
			bar.Set64(bytesRead)
		}

		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check if we're still in the header (tags in square brackets)
		if strings.HasPrefix(line, "[") {
			inHeader = true
			v.validateTag(line, lineNumber, tagPattern)
		} else if inHeader && !strings.HasPrefix(line, "[") {
			// First move line, exiting headers
			inHeader = false
			v.validateMoves(line, lineNumber)
		} else if !inHeader {
			// Subsequent move lines
			v.validateMoves(line, lineNumber)
		}
	}

	if err := scanner.Err(); err != nil {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: fmt.Sprintf("Error reading file: %v", err),
		})
	}

	// Complete progress bar to 100%
	if bar != nil {
		bar.Set64(fileSize)
		bar.Finish()
		fmt.Println()
	}

	return v.errors
}

// validateTag validates a single PGN tag
func (v *PGNValidator) validateTag(line string, lineNumber int, pattern *regexp.Regexp) {
	matches := pattern.FindStringSubmatch(line)

	if matches == nil {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: fmt.Sprintf("Malformed PGN tag: %s", line),
		})
		return
	}

	tagName := matches[1]
	tagValue := matches[2]

	// Specific validation for Date and EventDate tags
	if tagName == "Date" || tagName == "EventDate" {
		v.validateDate(tagValue, lineNumber, line)
	}

	// Specific validation for Result tag
	if tagName == "Result" {
		v.validateResult(tagValue, lineNumber)
	}
}

// validateDate validates and attempts to correct date format
func (v *PGNValidator) validateDate(dateValue string, lineNumber int, originalLine string) {
	// Correct format: YYYY.MM.DD
	// Acceptable format with wildcards: ????.??.??
	correctPattern := regexp.MustCompile(`^\d{4}\.\d{2}\.\d{2}$`)
	wildcardPattern := regexp.MustCompile(`^\?{4}\.\?{2}\.\?{2}$`)

	// If format is already correct, do nothing
	if correctPattern.MatchString(dateValue) || wildcardPattern.MatchString(dateValue) {
		return
	}

	// Attempt to correct the format
	correctedDate, err := v.tryFixDate(dateValue)

	if err != nil {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: fmt.Sprintf("Invalid date format: '%s'. Required format: YYYY.MM.DD (example: 2024.01.05)", dateValue),
		})
	} else {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: fmt.Sprintf("Date auto-corrected: '%s' → '%s'", dateValue, correctedDate),
		})
	}
}

// tryFixDate attempts to correct various date formats
func (v *PGNValidator) tryFixDate(dateValue string) (string, error) {
	// Remove spaces
	dateValue = strings.TrimSpace(dateValue)

	// Try different common formats
	patterns := []struct {
		regex   *regexp.Regexp
		convert func([]string) string
	}{
		// YYYY-MM-DD (ISO 8601)
		{
			regex: regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`),
			convert: func(m []string) string {
				return fmt.Sprintf("%s.%s.%s", m[1], m[2], m[3])
			},
		},
		// DD/MM/YYYY
		{
			regex: regexp.MustCompile(`^(\d{2})/(\d{2})/(\d{4})$`),
			convert: func(m []string) string {
				return fmt.Sprintf("%s.%s.%s", m[3], m[2], m[1])
			},
		},
		// MM/DD/YYYY (American format)
		{
			regex: regexp.MustCompile(`^(\d{2})/(\d{2})/(\d{4})$`),
			convert: func(m []string) string {
				return fmt.Sprintf("%s.%s.%s", m[3], m[1], m[2])
			},
		},
		// YYYY/MM/DD
		{
			regex: regexp.MustCompile(`^(\d{4})/(\d{2})/(\d{2})$`),
			convert: func(m []string) string {
				return fmt.Sprintf("%s.%s.%s", m[1], m[2], m[3])
			},
		},
		// YYYYMMDD (no separators)
		{
			regex: regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})$`),
			convert: func(m []string) string {
				return fmt.Sprintf("%s.%s.%s", m[1], m[2], m[3])
			},
		},
	}

	for _, p := range patterns {
		matches := p.regex.FindStringSubmatch(dateValue)
		if matches != nil {
			return p.convert(matches), nil
		}
	}

	return "", fmt.Errorf("cannot correct date format")
}

// validateResult validates the Result tag
func (v *PGNValidator) validateResult(resultValue string, lineNumber int) {
	validResults := map[string]bool{
		"1-0":     true, // White wins
		"0-1":     true, // Black wins
		"1/2-1/2": true, // Draw
		"*":       true, // Game in progress or unknown result
	}

	if !validResults[resultValue] {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: fmt.Sprintf("Invalid result: '%s'. Valid values: 1-0, 0-1, 1/2-1/2, *", resultValue),
		})
	}
}

// validateMoves validates game moves
func (v *PGNValidator) validateMoves(line string, lineNumber int) {
	// Basic validation: check that line contains valid characters for moves
	// Moves can contain: numbers, letters, +, #, =, -, !, ?, spaces, parentheses, braces
	validMovePattern := regexp.MustCompile(`^[a-zA-Z0-9\s\+\#\=\-\!\?\(\)\.\*\/\{\}]+$`)

	if !validMovePattern.MatchString(line) {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: "Invalid move format: disallowed characters found",
		})
	}

	// Validate balanced parentheses for variations
	if !v.checkBalancedDelimiters(line, '(', ')') {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: "Warning: Unbalanced parentheses in variations",
		})
	}

	// Validate balanced curly braces for comments
	if !v.checkBalancedDelimiters(line, '{', '}') {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: "Warning: Unbalanced curly braces in comments",
		})
	}

	// Check for proper nesting of parentheses and braces
	if !v.checkProperNesting(line, lineNumber) {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: "Warning: Improper nesting of parentheses and braces",
		})
	}
}

// checkBalancedDelimiters checks if opening and closing delimiters are balanced
func (v *PGNValidator) checkBalancedDelimiters(line string, open, close rune) bool {
	count := 0
	for _, char := range line {
		if char == open {
			count++
		} else if char == close {
			count--
			if count < 0 {
				return false // Closing delimiter before opening
			}
		}
	}
	return count == 0 // All delimiters must be closed
}

// checkProperNesting verifies that parentheses and braces are properly nested
func (v *PGNValidator) checkProperNesting(line string, lineNumber int) bool {
	stack := []rune{}

	for _, char := range line {
		switch char {
		case '(', '{':
			stack = append(stack, char)
		case ')':
			if len(stack) == 0 || stack[len(stack)-1] != '(' {
				return false
			}
			stack = stack[:len(stack)-1]
		case '}':
			if len(stack) == 0 || stack[len(stack)-1] != '{' {
				return false
			}
			stack = stack[:len(stack)-1]
		}
	}

	return len(stack) == 0
}

// WriteCorrectedFile reads the PGN file, applies corrections, and writes to output file
func (v *PGNValidator) WriteCorrectedFile(inputFile, outputFile string) error {
	// Open input file
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("cannot open input file: %v", err)
	}
	defer file.Close()

	// Get file size for progress bar
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("cannot get file info: %v", err)
	}
	fileSize := fileInfo.Size()

	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("cannot create output file: %v", err)
	}
	defer outFile.Close()

	// Create progress bar only for large files (> 1MB)
	var bar *progressbar.ProgressBar
	if fileSize > 1024*1024 {
		bar = progressbar.NewOptions64(
			fileSize,
			progressbar.OptionSetDescription("Correcting"),
			progressbar.OptionSetWidth(40),
			progressbar.OptionShowBytes(true),
			progressbar.OptionUseIECUnits(false),
			progressbar.OptionSetPredictTime(true),
			progressbar.OptionShowCount(),
		)
	}

	scanner := bufio.NewScanner(file)
	writer := bufio.NewWriter(outFile)
	defer writer.Flush()
	bytesRead := int64(0)
	lineCount := 0

	tagPattern := regexp.MustCompile(`^\[(\w+)\s+"(.*)"\]$`)

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		bytesRead += int64(len(line)) + 2 // +2 per newline (\r\n su Windows)

		// Aggiorna progress bar ogni 100 linee per migliori performance
		if bar != nil && lineCount%100 == 0 {
			bar.Set64(bytesRead)
		}

		correctedLine := line

		// If it's a tag, check if corrections are needed
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			matches := tagPattern.FindStringSubmatch(strings.TrimSpace(line))
			if matches != nil {
				tagName := matches[1]
				tagValue := matches[2]

				// Correct Date and EventDate tags if necessary
				if tagName == "Date" || tagName == "EventDate" {
					correctedDate, err := v.tryFixDate(tagValue)
					if err == nil {
						// Replace with corrected date
						correctedLine = fmt.Sprintf("[%s \"%s\"]", tagName, correctedDate)
					}
				}
			}
		}

		// Write line (corrected or original)
		if _, err := writer.WriteString(correctedLine + "\n"); err != nil {
			return fmt.Errorf("error writing: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading: %v", err)
	}

	// Complete progress bar to 100%
	if bar != nil {
		bar.Set64(fileSize)
		bar.Finish()
		fmt.Println()
	}

	return nil
}
