package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Flag definitions
	outputFile := flag.String("o", "", "Output file with corrections applied")
	flag.Parse()

	// Check arguments
	if flag.NArg() < 1 {
		fmt.Println("Usage: pgn_check [-o output.pgn] <file.pgn>")
		fmt.Println("Example: pgn_check game.pgn")
		fmt.Println("         pgn_check -o corrected.pgn game.pgn")
		os.Exit(1)
	}

	filename := flag.Arg(0)

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Printf("Error: file '%s' not found\n", filename)
		os.Exit(1)
	}

	// Validate PGN file
	validator := NewPGNValidator()
	errors := validator.ValidateFile(filename)

	// If -o specified, save corrected file
	if *outputFile != "" {
		if err := validator.WriteCorrectedFile(filename, *outputFile); err != nil {
			fmt.Printf("Error writing corrected file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Corrected file saved to: %s\n", *outputFile)
	}

	if len(errors) == 0 {
		fmt.Println("✓ PGN file is valid!")
		os.Exit(0)
	}

	// Print found errors
	fmt.Printf("✗ Found %d errors in PGN file:\n\n", len(errors))
	for _, err := range errors {
		fmt.Println(err)
	}
	os.Exit(1)
}
