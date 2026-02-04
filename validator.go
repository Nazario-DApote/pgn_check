package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/schollz/progressbar/v3"
)

// Pre-compiled regex patterns for better performance
var (
	// tagPattern matches PGN tags in format [TagName "Value"]
	// Groups: (1) tag name (word chars), (2) tag value (any chars)
	tagPattern = regexp.MustCompile(`^\[(\w+)\s+"(.*)"\]$`)

	// correctDatePattern matches dates in correct PGN format: YYYY.MM.DD
	// Matches exactly 4 digits, dot, 2 digits, dot, 2 digits
	correctDatePattern = regexp.MustCompile(`^\d{4}\.\d{2}\.\d{2}$`)

	// wildcardDatePattern matches unknown dates in PGN format: ????.??.??
	// Matches exactly 4 question marks, dot, 2 question marks, dot, 2 question marks
	wildcardDatePattern = regexp.MustCompile(`^\?{4}\.\?{2}\.\?{2}$`)

	// validMovePattern checks if line contains only valid PGN move characters
	// Allows: letters, numbers, spaces, +#=-!?().*/{} (standard PGN notation)
	validMovePattern = regexp.MustCompile(`^[a-zA-Z0-9\s\+\#\=\-\!\?\(\)\.\*\/\{\}]+$`)

	// movePattern extracts move numbers and moves from PGN notation
	// Groups: (1) move number, (2) white's move, (3) black's move (optional)
	// Matches: "1. e4 e5" or "23. Nf3"
	movePattern = regexp.MustCompile(`(\d+)\.\s*([^\s]+)(?:\s+([^\s]+))?`)

	// promotionPattern matches pawn promotion moves
	// Groups: (1) source file (optional for capture), (2) destination square, (3) promoted piece (Q/R/B/N)
	// Matches: "e8=Q" or "exd8=R"
	promotionPattern = regexp.MustCompile(`^([a-h])?([a-h][1-8])=([QRBN])$`)

	// piecePattern matches piece moves with optional disambiguation
	// Groups: (1) piece (K/Q/R/B/N), (2) source file (optional), (3) source rank (optional), (4) capture 'x' (optional), (5) destination
	// Matches: "Nf3", "Nbd7", "R1a3", "Qh4e1", "Bxe5"
	piecePattern = regexp.MustCompile(`^([KQRBN])([a-h])?([1-8])?(x)?([a-h][1-8])$`)

	// pawnPattern matches pawn moves with captures
	// Groups: (1) source file, (2) capture 'x' (optional), (3) destination square
	// Matches: "e4", "exd5"
	pawnPattern = regexp.MustCompile(`^([a-h])(x)?([a-h][1-8])$`)

	// simplePawnPattern matches simple pawn moves (destination only)
	// Matches: "e4", "d5", "a6" (file a-h, rank 1-8)
	simplePawnPattern = regexp.MustCompile(`^[a-h][1-8]$`)

	// Date fixing patterns - used to auto-correct common date formats to PGN standard

	// datePatternISO matches ISO 8601 date format: YYYY-MM-DD
	// Groups: (1) year (4 digits), (2) month (2 digits), (3) day (2 digits)
	datePatternISO = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)

	// datePatternDDMMYYYY matches European date format: DD/MM/YYYY
	// Groups: (1) day (2 digits), (2) month (2 digits), (3) year (4 digits)
	datePatternDDMMYYYY = regexp.MustCompile(`^(\d{2})/(\d{2})/(\d{4})$`)

	// datePatternYYYYMMDD matches slash-separated date: YYYY/MM/DD
	// Groups: (1) year (4 digits), (2) month (2 digits), (3) day (2 digits)
	datePatternYYYYMMDD = regexp.MustCompile(`^(\d{4})/(\d{2})/(\d{2})$`)

	// datePatternNoSep matches date without separators: YYYYMMDD
	// Groups: (1) year (4 digits), (2) month (2 digits), (3) day (2 digits)
	datePatternNoSep = regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})$`)
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
	// Increase buffer size to 1MB for better performance
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNumber := 0
	inHeader := true
	bytesRead := int64(0)

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		bytesRead += int64(len(line)) + 2 // +2 per newline (\r\n su Windows)

		// Update progress bar every 1000 lines for better performance
		if bar != nil && lineNumber%1000 == 0 {
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

	// Specific validation for Date and EventDate tags (case-insensitive)
	tagNameLower := strings.ToLower(tagName)
	if tagNameLower == "date" || tagNameLower == "eventdate" {
		v.validateDate(tagValue, lineNumber, line)
	}

	// Specific validation for Result tag (case-insensitive)
	if tagNameLower == "result" {
		v.validateResult(tagValue, lineNumber)
	}
}

// validateDate validates and attempts to correct date format
func (v *PGNValidator) validateDate(dateValue string, lineNumber int, originalLine string) {
	// Correct format: YYYY.MM.DD
	// Acceptable format with wildcards: ????.??.??
	// If format is already correct, do nothing
	if correctDatePattern.MatchString(dateValue) || wildcardDatePattern.MatchString(dateValue) {
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

	// YYYY-MM-DD (ISO 8601)
	if matches := datePatternISO.FindStringSubmatch(dateValue); matches != nil {
		return fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3]), nil
	}

	// DD/MM/YYYY or MM/DD/YYYY - assume DD/MM/YYYY for European format
	if matches := datePatternDDMMYYYY.FindStringSubmatch(dateValue); matches != nil {
		return fmt.Sprintf("%s.%s.%s", matches[3], matches[2], matches[1]), nil
	}

	// YYYY/MM/DD
	if matches := datePatternYYYYMMDD.FindStringSubmatch(dateValue); matches != nil {
		return fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3]), nil
	}

	// YYYYMMDD (no separators)
	if matches := datePatternNoSep.FindStringSubmatch(dateValue); matches != nil {
		return fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3]), nil
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
	if !v.checkProperNesting(line) {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: "Warning: Improper nesting of parentheses and braces",
		})
	}

	// Validate move notation and move numbers
	v.validateMoveNotation(line, lineNumber)
}

// checkBalancedDelimiters checks if opening and closing delimiters are balanced
func (v *PGNValidator) checkBalancedDelimiters(line string, open, close rune) bool {
	count := 0
	for _, char := range line {
		switch char {
		case open:
			count++
		case close:
			count--
			if count < 0 {
				return false // Closing delimiter before opening
			}
		}
	}
	return count == 0 // All delimiters must be closed
}

// validateMoveNotation validates individual move notation and move numbers
func (v *PGNValidator) validateMoveNotation(line string, lineNumber int) {
	// Remove comments in curly braces
	cleanLine := v.removeComments(line)

	// Remove variations in parentheses
	cleanLine = v.removeVariations(cleanLine)

	// Extract moves and move numbers using regex
	// Pattern per trovare numeri di mossa e le mosse stesse
	matches := movePattern.FindAllStringSubmatch(cleanLine, -1)

	expectedMoveNumber := 0

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		moveNumberStr := match[1]
		whiteMove := match[2]
		blackMove := ""
		if len(match) > 3 && match[3] != "" {
			blackMove = match[3]
		}

		// Parse move number
		var moveNumber int
		fmt.Sscanf(moveNumberStr, "%d", &moveNumber)

		// Check sequential move numbers
		if expectedMoveNumber == 0 {
			expectedMoveNumber = moveNumber
		} else {
			expectedMoveNumber++
			if moveNumber != expectedMoveNumber {
				v.errors = append(v.errors, ValidationError{
					Line:    lineNumber,
					Message: fmt.Sprintf("Warning: Move number out of sequence. Expected %d, found %d", expectedMoveNumber, moveNumber),
				})
				expectedMoveNumber = moveNumber
			}
		}

		// Validate white's move
		if !v.isValidMoveNotation(whiteMove) {
			v.errors = append(v.errors, ValidationError{
				Line:    lineNumber,
				Message: fmt.Sprintf("Warning: Invalid move notation '%s' at move %d", whiteMove, moveNumber),
			})
		}

		// Validate black's move if present
		if blackMove != "" && !v.isValidMoveNotation(blackMove) {
			v.errors = append(v.errors, ValidationError{
				Line:    lineNumber,
				Message: fmt.Sprintf("Warning: Invalid move notation '%s' at move %d", blackMove, moveNumber),
			})
		}
	}
}

// removeComments removes text in curly braces (comments)
func (v *PGNValidator) removeComments(line string) string {
	result := []rune{}
	inComment := false

	for _, char := range line {
		if char == '{' {
			inComment = true
		} else if char == '}' {
			inComment = false
		} else if !inComment {
			result = append(result, char)
		}
	}

	return string(result)
}

// removeVariations removes text in parentheses (variations)
func (v *PGNValidator) removeVariations(line string) string {
	result := []rune{}
	depth := 0

	for _, char := range line {
		if char == '(' {
			depth++
		} else if char == ')' {
			if depth > 0 {
				depth--
			}
		} else if depth == 0 {
			result = append(result, char)
		}
	}

	return string(result)
}

// isValidMoveNotation checks if a move follows correct PGN notation
func (v *PGNValidator) isValidMoveNotation(move string) bool {
	// Trim annotations like !, ?, !!, ??, !?, ?!
	move = strings.TrimRight(move, "!?")

	// Check for game result markers
	if move == "1-0" || move == "0-1" || move == "1/2-1/2" || move == "*" {
		return true
	}

	// Rimuove scacco e scacco matto prima di controllare castling
	moveWithoutCheck := strings.TrimRight(move, "+#")

	// Check for castling (with or without check/checkmate)
	if moveWithoutCheck == "O-O" || moveWithoutCheck == "O-O-O" ||
		moveWithoutCheck == "0-0" || moveWithoutCheck == "0-0-0" {
		return true
	}

	// Pattern for valid moves:
	// - Pieces: K, Q, R, B, N followed by coordinates
	// - Pawns: only coordinates
	// - Can contain: x (capture), = (promotion), + (check), # (checkmate)
	// - Coordinates: a-h for files, 1-8 for ranks
	// Remove check and checkmate at the end
	move = strings.TrimRight(move, "+#")

	// Pattern for promotion (ex: e8=Q)
	if promotionPattern.MatchString(move) {
		return true
	}

	// Pattern for moves of pieces with disambiguation
	// Es: Nbd7, N1c3, Qh4e1, Raxb1
	if piecePattern.MatchString(move) {
		matches := piecePattern.FindStringSubmatch(move)
		if len(matches) > 1 {
			piece := matches[1]
			// Verify that the piece is valid
			if piece == "K" || piece == "Q" || piece == "R" || piece == "B" || piece == "N" {
				return true
			}
		}
	}

	// Pawn move patterns
	// Ex: e4, exd5, e8
	if pawnPattern.MatchString(move) {
		return true
	}

	// Simple pawn move patterns (only destination square)
	// Ex: e4, d5, a6
	if simplePawnPattern.MatchString(move) {
		return true
	}

	// If the move does not match any valid pattern
	return false
}

// checkProperNesting verifies that parentheses and braces are properly nested
func (v *PGNValidator) checkProperNesting(line string) bool {
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

// fixBalancedDelimiters attempts to fix unbalanced parentheses and braces in a line
func (v *PGNValidator) fixBalancedDelimiters(line string) string {
	result := []rune{}
	stack := []rune{}

	for _, char := range line {
		switch char {
		case '(', '{':
			// Opening delimiter: add to result and push to stack
			result = append(result, char)
			stack = append(stack, char)
		case ')':
			// Closing parenthesis: check if there's a matching opening
			if len(stack) > 0 && stack[len(stack)-1] == '(' {
				result = append(result, char)
				stack = stack[:len(stack)-1]
			}
			// If no matching opening, skip this closing parenthesis
		case '}':
			// Closing brace: check if there's a matching opening
			if len(stack) > 0 && stack[len(stack)-1] == '{' {
				result = append(result, char)
				stack = stack[:len(stack)-1]
			}
			// If no matching opening, skip this closing brace
		default:
			result = append(result, char)
		}
	}

	// Add missing closing delimiters at the end of the line
	for i := len(stack) - 1; i >= 0; i-- {
		switch stack[i] {
		case '(':
			result = append(result, ')')
		case '{':
			result = append(result, '}')
		}
	}

	return string(result)
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
	// Increase buffer size to 1MB for better performance
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	// Increase writer buffer size to 1MB
	writer := bufio.NewWriterSize(outFile, 1024*1024)
	defer writer.Flush()
	bytesRead := int64(0)
	lineCount := 0
	inHeader := true

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		bytesRead += int64(len(line)) + 2 // +2 for newline (\r\n on Windows)

		// Update progress bar every 1000 lines for better performance
		if bar != nil && lineCount%1000 == 0 {
			bar.Set64(bytesRead)
		}

		correctedLine := line

		// If it's a tag, check if corrections are needed
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			inHeader = true
			matches := tagPattern.FindStringSubmatch(strings.TrimSpace(line))
			if matches != nil {
				tagName := matches[1]
				tagValue := matches[2]

				// Correct Date and EventDate tags if necessary (case-insensitive)
				tagNameLower := strings.ToLower(tagName)
				if tagNameLower == "date" || tagNameLower == "eventdate" {
					correctedDate, err := v.tryFixDate(tagValue)
					if err == nil {
						// Replace with corrected date
						correctedLine = fmt.Sprintf("[%s \"%s\"]", tagName, correctedDate)
					}
				}
			}
		} else if strings.TrimSpace(line) != "" && inHeader {
			// First non-empty, non-tag line marks start of moves
			inHeader = false
		}

		// Fix unbalanced parentheses and braces in move lines
		if !inHeader && strings.TrimSpace(line) != "" {
			correctedLine = v.fixBalancedDelimiters(correctedLine)
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
