// PGN Check - A command-line tool for validating PGN (Portable Game Notation) files
//
// Author: Nazario D'Apote <nazario.dapote@gmail.com>
// License: MIT
// Repository: https://github.com/nazariodapote/pgn_check

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// Version is set at build time using ldflags
var Version = "dev"

func main() {
	// Flag definitions
	outputFile := flag.String("o", "", "Output file with corrections applied")
	version := flag.Bool("version", false, "Show version information")
	versionShort := flag.Bool("v", false, "Show version information")
	flag.Parse()

	// Show version if requested
	if *version || *versionShort {
		fmt.Printf("pgn_check version %s\n", Version)
		fmt.Println("Author: Nazario D'Apote <nazario.dapote@gmail.com>")
		fmt.Println("License: MIT")
		os.Exit(0)
	}

	// Check arguments
	if flag.NArg() < 1 {
		fmt.Println("Usage: pgn_check [-o output.pgn] [-v|--version] <file.pgn>")
		fmt.Println("Example: pgn_check game.pgn")
		fmt.Println("         pgn_check -o corrected.pgn game.pgn")
		fmt.Println("         pgn_check --version")
		os.Exit(1)
	}

	filename := flag.Arg(0)

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Fatalf("Error: file '%s' not found\n", filename)
	}

	// Validate PGN file
	validator := NewPGNValidator()
	errors := validator.ValidateFile(filename)

	// If -o specified, save corrected file
	if *outputFile != "" {
		if err := validator.WriteCorrectedFile(filename, *outputFile); err != nil {
			log.Fatalf("Error writing corrected file: %v\n", err)
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
