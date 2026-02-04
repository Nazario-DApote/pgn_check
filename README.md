# PGN Check - Validatore di File PGN

[![Build Linux](https://github.com/YOUR_USERNAME/pgn_check/workflows/Build%20and%20Test%20-%20Linux/badge.svg)](https://github.com/YOUR_USERNAME/pgn_check/actions)
[![Build Windows](https://github.com/YOUR_USERNAME/pgn_check/workflows/Build%20and%20Test%20-%20Windows/badge.svg)](https://github.com/YOUR_USERNAME/pgn_check/actions)

Un tool da linea di comando scritto in Go per validare file PGN (Portable Game Notation) con particolare attenzione al formato delle date.

## Caratteristiche

- ✅ Valida la struttura dei file PGN (singole partite o file multipli)
- 📅 Controlla il formato delle date nei campi `[Date]` e `[EventDate]` (richiesto: `YYYY.MM.DD`)
- 🔧 Tenta di correggere automaticamente date mal formattate
- 📍 Indica il numero di linea esatto degli errori
- 🎯 Supporta formati di data comuni: ISO 8601, DD/MM/YYYY, MM/DD/YYYY, etc.
- 💾 Salva file corretti con il flag `-o`
- 📊 Progress bar per file grandi (> 1MB) per monitorare l'avanzamento

## Installazione

### Da Release Binarie (Consigliato)

Scarica l'ultima versione pre-compilata dalla [pagina Releases](https://github.com/YOUR_USERNAME/pgn_check/releases):
- **Windows**: `pgn_check-windows-vX.X.X.zip`
- **Linux**: `pgn_check-linux-vX.X.X.tar.gz`

Estrai l'archivio e il binario è pronto all'uso!

### Da Sorgente

```bash
# Mostra versione
pgn_check.exe --version

# Clona il repository
cd pgn_check

# Compila il progetto
go build -o pgn_check.exe

# Oppure con versione incorporata
VERSION=$(cat VERSION)
go build -ldflags="-X main.Version=$VERSION" -o pgn_check.exe
```

## Utilizzo

```bash
# Valida un file PGN
pgn_check.exe test_files\example_valid.pgn

# Output per file valido:
# ✓ Il file PGN è valido!

# Output per file con errori:
# ✗ Trovati 1 errori nel file PGN:
#
# Linea 3: Data corretta automaticamente: '2024-01-15' → '2024.01.15'

# Valida e salva una versione corretta del file
pgn_check.exe -o output.pgn test_files\example_invalid_date.pgn

# Output:
# ✓ File corretto salvato in: output.pgn
# ✗ Trovati 1 errori nel file PGN:
#
# Linea 3: Data corretta automaticamente: '2024-01-15' → '2024.01.15'
```

## Opzioni

- `-o <file>` : Specifica un file di output dove salvare la versione corretta del PGN

## Formato Data Richiesto

Il formato corretto per il tag Date è: `YYYY.MM.DD`

Esempi:
- ✅ `[Date "2024.01.05"]` - Formato corretto
- ✅ `[Date "????.??.??"]` - Formato wildcard (data sconosciuta)
- ❌ `[Date "2024-01-05"]` - Formato ISO 8601 (viene corretto automaticamente)
- ❌ `[Date "05/01/2024"]` - Formato europeo (viene corretto se possibile)

## Formati Data Supportati per Correzione Automatica

Il tool tenta di correggere automaticamente questi formati:
- `YYYY-MM-DD` (ISO 8601)
- `DD/MM/YYYY` (formato europeo)
- `MM/DD/YYYY` (formato americano)
- `YYYY/MM/DD`
- `YYYYMMDD` (senza separatori)

## Esempio di File PGN Valido

```pgn
[Event "Example"]
[Site "?"]
[Date "2024.01.05"]
[Round "?"]
[White "?"]
[Black "?"]
[Result "*"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6
```

## Validazioni Implementate

1. **Tag PGN**: Verifica che i tag siano nel formato `[TagName "Value"]`
2. **Date**: Controlla e corregge il formato delle date nei campi `[Date]` e `[EventDate]`
3. **Result**: Valida i risultati ammessi: `1-0`, `0-1`, `1/2-1/2`, `*`
4. **Mosse**: Validazione completa della notazione delle mosse PGN
   - Verifica sequenza dei numeri di mossa (1., 2., 3., ecc.)
   - Valida notazione dei pezzi: K (Re), Q (Regina), R (Torre), B (Alfiere), N (Cavallo)
   - Valida notazione dei pedoni (solo casella destinazione)
   - Valida coordinate della scacchiera (a-h per colonne, 1-8 per righe)
   - Supporta arrocchi: O-O (corto) e O-O-O (lungo)
   - Supporta promozione pedoni: e8=Q
   - Supporta scacco (+) e scacco matto (#)
   - Supporta disambiguazione: Nbd7, N1c3, Raxb1
   - Supporta annotazioni: !, ?, !!, ??, !?, ?!
5. **Parentesi e Variazioni**: Controlla bilanciamento di parentesi e parentesi graffe
6. **File Multipli**: Gestisce correttamente file con centinaia di partite

### Esempi di Validazione Mosse

✅ **Mosse valide:**
- `e4`, `d5` - mosse di pedone
- `Nf3`, `Nc6` - mosse di cavallo
- `O-O`, `O-O-O` - arrocchi
- `e8=Q` - promozione pedone
- `Qh5+` - scacco
- `Qh4#` - scacco matto
- `Nbd7` - disambiguazione (cavallo da b)
- `R1c3` - disambiguazione (torre da riga 1)
- `exd5` - cattura con pedone

❌ **Mosse invalide (vengono segnalate):**
- `Xe1` - X non è un pezzo valido
- `b9` - 9 non è una riga valida (solo 1-8)
- `Qj5` - j non è una colonna valida (solo a-h)
- `3. Nf3` dopo `1. e4` - numero di mossa non sequenziale

## Performance

Il tool è ottimizzato per gestire file PGN molto grandi:
- **Velocità**: ~2.4 MB/s (validazione e correzione)
- **File da 100 MB**: ~42 secondi
- **File da 1 GB**: ~7 minuti
- **File da 8 GB**: ~57 minuti

### Benchmark

Per misurare le performance sul tuo sistema, usa gli script di benchmark inclusi:

```bash
# Windows (PowerShell)
.\benchmark.ps1                    # Testa i file grandi
.\benchmark.ps1 -All               # Testa tutti i file in test_files
.\benchmark.ps1 file.pgn           # Testa un file specifico

# Linux/Mac (Bash)
./benchmark.sh                     # Testa i file grandi
./benchmark.sh --all               # Testa tutti i file in test_files
./benchmark.sh file.pgn            # Testa un file specifico
```

Gli script di benchmark mostrano:
- Tempo di validazione e correzione per ogni file
- Velocità in MB/s
- Proiezioni per file molto grandi (100MB, 500MB, 1GB, 8GB)
- Statistiche aggregate

### Ottimizzazioni Implementate

- Buffer di lettura/scrittura da 1MB per I/O efficiente
- Regex pre-compilate per evitare ricompilazioni
- Progress bar aggiornata ogni 1000 righe per ridurre overhead
- Parsing ottimizzato delle mosse e date

## Validazione Batch

Per validare molteplici file PGN in una directory:

```bash
# Windows (PowerShell)
.\validate_all.ps1 .\test_files                    # Solo validazione
.\validate_all.ps1 .\test_files -OutputDir .\fixed  # Valida e correggi

# Linux/Mac (Bash)
./validate_all.sh ./test_files                     # Solo validazione
./validate_all.sh ./test_files -o ./fixed          # Valida e correggi
```

Gli script mostrano:
- Progresso per ogni file
- Elenco degli errori e warning trovati
- Riepilogo finale con conteggio file validi/invalidi
### Test
```bash
# Esegui tutti i test
go test -v ./...

# Test con coverage
go test -cover ./...
```

### Build
```bash
# Esegui il tool in modalità sviluppo
go run . test_files\example_valid.pgn

# Build standard
go build -o pgn_check.exe

# Build ottimizzata con versione
VERSION=$(cat VERSION)
go build -ldflags="-X main.Version=$VERSION -s -w" -o pgn_check.exe
```

### CI/CD Workflows

Il progetto include GitHub Actions workflows per build e test automatici:

- **build-linux.yml**: Compila, testa e crea artefatti per Linux
- **build-windows.yml**: Compila, testa e crea artefatti per Windows

Ogni workflow:
1. Legge la versione dal file `VERSION`
2. Compila il binario con la versione incorporata
3. Esegue tutti i test Go
4. Esegue i benchmark di performance
5. Crea un artefatto con binario, versione e risultati benchmark

Per aggiornare la versione, modifica semplicemente il file `VERSION`.
# Build ottimizzata
go build -ldflags="-s -w" -o pgn_check.exe
```

## File di Esempio

Il repository include file di esempio nella cartella `test_files/`:
- `example_valid.pgn` - File PGN valido
- `example_invalid_date.pgn` - File con data in formato non corretto
- `multiple_games_test.pgn` - File con più partite
- `test_eventdate.pgn` - File con EventDate malformattati
- `twic1617.pgn` - File reale con centinaia di partite

## Requisiti

- Go 1.21 o superiore

## Licenza

MIT
