package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/schollz/progressbar/v3"
)

// ValidationError rappresenta un errore di validazione PGN
type ValidationError struct {
	Line    int
	Message string
}

func (e ValidationError) String() string {
	return fmt.Sprintf("Linea %d: %s", e.Line, e.Message)
}

// PGNValidator gestisce la validazione dei file PGN
type PGNValidator struct {
	errors []ValidationError
}

// NewPGNValidator crea una nuova istanza del validatore
func NewPGNValidator() *PGNValidator {
	return &PGNValidator{
		errors: make([]ValidationError, 0),
	}
}

// ValidateFile valida un file PGN e ritorna una lista di errori
func (v *PGNValidator) ValidateFile(filename string) []ValidationError {
	v.errors = make([]ValidationError, 0)

	file, err := os.Open(filename)
	if err != nil {
		v.errors = append(v.errors, ValidationError{
			Line:    0,
			Message: fmt.Sprintf("Impossibile aprire il file: %v", err),
		})
		return v.errors
	}
	defer file.Close()

	// Ottieni la dimensione del file per la progress bar
	fileInfo, err := file.Stat()
	if err != nil {
		v.errors = append(v.errors, ValidationError{
			Line:    0,
			Message: fmt.Sprintf("Impossibile ottenere info sul file: %v", err),
		})
		return v.errors
	}
	fileSize := fileInfo.Size()

	// Crea progress bar solo per file grandi (> 1MB)
	var bar *progressbar.ProgressBar
	if fileSize > 1024*1024 {
		bar = progressbar.NewOptions64(
			fileSize,
			progressbar.OptionSetDescription("Validazione in corso"),
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

	// Pattern per i tag PGN: [TagName "Value"]
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

		// Salta le linee vuote
		if line == "" {
			continue
		}

		// Controlla se siamo ancora negli header (tag tra parentesi quadre)
		if strings.HasPrefix(line, "[") {
			inHeader = true
			v.validateTag(line, lineNumber, tagPattern)
		} else if inHeader && !strings.HasPrefix(line, "[") {
			// Prima linea delle mosse, usciamo dagli header
			inHeader = false
			v.validateMoves(line, lineNumber)
		} else if !inHeader {
			// Linee successive delle mosse
			v.validateMoves(line, lineNumber)
		}
	}

	if err := scanner.Err(); err != nil {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: fmt.Sprintf("Errore nella lettura del file: %v", err),
		})
	}

	// Completa la progress bar al 100%
	if bar != nil {
		bar.Set64(fileSize)
		bar.Finish()
		fmt.Println()
	}

	return v.errors
}

// validateTag valida un singolo tag PGN
func (v *PGNValidator) validateTag(line string, lineNumber int, pattern *regexp.Regexp) {
	matches := pattern.FindStringSubmatch(line)

	if matches == nil {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: fmt.Sprintf("Tag PGN malformato: %s", line),
		})
		return
	}

	tagName := matches[1]
	tagValue := matches[2]

	// Validazione specifica per i tag Date e EventDate
	if tagName == "Date" || tagName == "EventDate" {
		v.validateDate(tagValue, lineNumber, line)
	}

	// Validazione specifica per il tag Result
	if tagName == "Result" {
		v.validateResult(tagValue, lineNumber)
	}
}

// validateDate valida e tenta di correggere il formato della data
func (v *PGNValidator) validateDate(dateValue string, lineNumber int, originalLine string) {
	// Formato corretto: YYYY.MM.DD
	// Formato accettabile con wildcard: ????.??.??
	correctPattern := regexp.MustCompile(`^\d{4}\.\d{2}\.\d{2}$`)
	wildcardPattern := regexp.MustCompile(`^\?{4}\.\?{2}\.\?{2}$`)

	// Se il formato è già corretto, non fare nulla
	if correctPattern.MatchString(dateValue) || wildcardPattern.MatchString(dateValue) {
		return
	}

	// Tenta di correggere il formato
	correctedDate, err := v.tryFixDate(dateValue)

	if err != nil {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: fmt.Sprintf("Formato data non valido: '%s'. Formato richiesto: YYYY.MM.DD (esempio: 2024.01.05)", dateValue),
		})
	} else {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: fmt.Sprintf("Data corretta automaticamente: '%s' → '%s'", dateValue, correctedDate),
		})
	}
}

// tryFixDate tenta di correggere vari formati di data
func (v *PGNValidator) tryFixDate(dateValue string) (string, error) {
	// Rimuovi spazi
	dateValue = strings.TrimSpace(dateValue)

	// Prova diversi formati comuni
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
		// MM/DD/YYYY (formato americano)
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
		// YYYYMMDD (senza separatori)
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

	return "", fmt.Errorf("impossibile correggere il formato della data")
}

// validateResult valida il tag Result
func (v *PGNValidator) validateResult(resultValue string, lineNumber int) {
	validResults := map[string]bool{
		"1-0":     true, // Vittoria del Bianco
		"0-1":     true, // Vittoria del Nero
		"1/2-1/2": true, // Patta
		"*":       true, // Partita in corso o risultato sconosciuto
	}

	if !validResults[resultValue] {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: fmt.Sprintf("Risultato non valido: '%s'. Valori ammessi: 1-0, 0-1, 1/2-1/2, *", resultValue),
		})
	}
}

// validateMoves valida le mosse della partita
func (v *PGNValidator) validateMoves(line string, lineNumber int) {
	// Validazione base: controlla che la linea contenga caratteri validi per le mosse
	// Le mosse possono contenere: numeri, lettere, +, #, =, -, !, ?, spazi, parentesi
	validMovePattern := regexp.MustCompile(`^[\w\s\.\+\#\=\-\!\?\(\)\*\/]+$`)

	if !validMovePattern.MatchString(line) {
		v.errors = append(v.errors, ValidationError{
			Line:    lineNumber,
			Message: "Formato mosse non valido: caratteri non ammessi trovati",
		})
	}
}

// WriteCorrectedFile legge il file PGN, applica le correzioni e lo scrive nel file di output
func (v *PGNValidator) WriteCorrectedFile(inputFile, outputFile string) error {
	// Apri il file di input
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("impossibile aprire il file di input: %v", err)
	}
	defer file.Close()

	// Ottieni la dimensione del file per la progress bar
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("impossibile ottenere info sul file: %v", err)
	}
	fileSize := fileInfo.Size()

	// Crea il file di output
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("impossibile creare il file di output: %v", err)
	}
	defer outFile.Close()

	// Crea progress bar solo per file grandi (> 1MB)
	var bar *progressbar.ProgressBar
	if fileSize > 1024*1024 {
		bar = progressbar.NewOptions64(
			fileSize,
			progressbar.OptionSetDescription("Correzione in corso"),
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

		// Se è un tag, controlla se necessita correzioni
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			matches := tagPattern.FindStringSubmatch(strings.TrimSpace(line))
			if matches != nil {
				tagName := matches[1]
				tagValue := matches[2]

				// Correggi i tag Date e EventDate se necessario
				if tagName == "Date" || tagName == "EventDate" {
					correctedDate, err := v.tryFixDate(tagValue)
					if err == nil {
						// Sostituisci con la data corretta
						correctedLine = fmt.Sprintf("[%s \"%s\"]", tagName, correctedDate)
					}
				}
			}
		}

		// Scrivi la linea (corretta o originale)
		if _, err := writer.WriteString(correctedLine + "\n"); err != nil {
			return fmt.Errorf("errore durante la scrittura: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("errore durante la lettura: %v", err)
	}

	// Completa la progress bar al 100%
	if bar != nil {
		bar.Set64(fileSize)
		bar.Finish()
		fmt.Println()
	}

	return nil
}
