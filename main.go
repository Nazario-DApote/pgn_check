package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Definizione flag
	outputFile := flag.String("o", "", "File di output con le correzioni applicate")
	flag.Parse()

	// Verifica argomenti
	if flag.NArg() < 1 {
		fmt.Println("Uso: pgn_check [-o output.pgn] <file.pgn>")
		fmt.Println("Esempio: pgn_check game.pgn")
		fmt.Println("         pgn_check -o corrected.pgn game.pgn")
		os.Exit(1)
	}

	filename := flag.Arg(0)

	// Controlla se il file esiste
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Printf("Errore: file '%s' non trovato\n", filename)
		os.Exit(1)
	}

	// Valida il file PGN
	validator := NewPGNValidator()
	errors := validator.ValidateFile(filename)

	// Se specificato -o, salva il file corretto
	if *outputFile != "" {
		if err := validator.WriteCorrectedFile(filename, *outputFile); err != nil {
			fmt.Printf("Errore durante la scrittura del file corretto: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ File corretto salvato in: %s\n", *outputFile)
	}

	if len(errors) == 0 {
		fmt.Println("✓ Il file PGN è valido!")
		os.Exit(0)
	}

	// Stampa gli errori trovati
	fmt.Printf("✗ Trovati %d errori nel file PGN:\n\n", len(errors))
	for _, err := range errors {
		fmt.Println(err)
	}
	os.Exit(1)
}
