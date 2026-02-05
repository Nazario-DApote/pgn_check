# PGN Check - PGN File Validator

[![Build Linux](https://github.com/YOUR_USERNAME/pgn_check/workflows/Build%20and%20Test%20-%20Linux/badge.svg)](https://github.com/YOUR_USERNAME/pgn_check/actions)
[![Build Windows](https://github.com/YOUR_USERNAME/pgn_check/workflows/Build%20and%20Test%20-%20Windows/badge.svg)](https://github.com/YOUR_USERNAME/pgn_check/actions)

A command-line tool written in Go to validate PGN (Portable Game Notation) files with special attention to date format.

## Features

- ✅ Validates PGN file structure (single games or multiple game files)
- 📅 Checks date format in `[Date]` and `[EventDate]` fields (required: `YYYY.MM.DD`)
- 🔧 Attempts to automatically correct malformed dates
- 📍 Shows exact line number of errors
- 🎯 Supports common date formats: ISO 8601, DD/MM/YYYY, MM/DD/YYYY, etc.
- 💾 Saves corrected files with the `-o` flag
- 📊 Progress bar for large files (> 1MB) to monitor progress

## Installation

### From Binary Releases (Recommended)

Download the latest pre-compiled version from the [Releases page](https://github.com/YOUR_USERNAME/pgn_check/releases):
- **Windows**: `pgn_check-windows-vX.X.X.zip`
- **Linux**: `pgn_check-linux-vX.X.X.tar.gz`

Extract the archive and the binary is ready to use!

### From Source

```bash
# Show version
pgn_check.exe --version

# Clone the repository
cd pgn_check

# Build the project
go build -o pgn_check.exe

# Or with embedded version
VERSION=$(cat VERSION)
go build -ldflags="-X main.Version=$VERSION" -o pgn_check.exe
```

## Usage

```bash
# Validate a PGN file
pgn_check.exe test_files\example_valid.pgn

# Output for valid file:
# ✓ PGN file is valid!

# Output for file with errors:
# ✗ Found 1 errors in PGN file:
#
# Line 3: Date auto-corrected: '2024-01-15' → '2024.01.15'

# Validate and save a corrected version of the file
pgn_check.exe -o output.pgn test_files\example_invalid_date.pgn

# Output:
# ✓ Corrected file saved to: output.pgn
# ✗ Found 1 errors in PGN file:
#
# Line 3: Date auto-corrected: '2024-01-15' → '2024.01.15'
```

## Options

- `-o <file>` : Specify an output file where to save the corrected PGN version

## Required Date Format

The correct format for the Date tag is: `YYYY.MM.DD`

Examples:
- ✅ `[Date "2024.01.05"]` - Correct format
- ✅ `[Date "????.??.??"]` - Wildcard format (unknown date)
- ❌ `[Date "2024-01-05"]` - ISO 8601 format (automatically corrected)
- ❌ `[Date "05/01/2024"]` - European format (corrected if possible)

## Date Formats Supported for Automatic Correction

The tool attempts to automatically correct these formats:
- `YYYY-MM-DD` (ISO 8601)
- `DD/MM/YYYY` (European format)
- `MM/DD/YYYY` (American format)
- `YYYY/MM/DD`
- `YYYYMMDD` (no separators)

## Example of Valid PGN File

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

## Implemented Validations

1. **PGN Tags**: Verifies that tags are in the format `[TagName "Value"]`
2. **Dates**: Checks and corrects date format in `[Date]` and `[EventDate]` fields
3. **Result**: Validates allowed results: `1-0`, `0-1`, `1/2-1/2`, `*`
4. **Moves**: Complete validation of PGN move notation
   - Verifies move number sequence (1., 2., 3., etc.)
   - Validates piece notation: K (King), Q (Queen), R (Rook), B (Bishop), N (Knight)
   - Validates pawn notation (destination square only)
   - Validates board coordinates (a-h for files, 1-8 for ranks)
   - Supports castling: O-O (kingside) and O-O-O (queenside)
   - Supports pawn promotion: e8=Q
   - Supports check (+) and checkmate (#)
   - Supports disambiguation: Nbd7, N1c3, Raxb1
   - Supports annotations: !, ?, !!, ??, !?, ?!
5. **Parentheses and Variations**: Checks balance of parentheses and braces
6. **Multiple Files**: Correctly handles files with hundreds of games

### Move Validation Examples

✅ **Valid moves:**
- `e4`, `d5` - pawn moves
- `Nf3`, `Nc6` - knight moves
- `O-O`, `O-O-O` - castling
- `e8=Q` - pawn promotion
- `Qh5+` - check
- `Qh4#` - checkmate
- `Nbd7` - disambiguation (knight from b)
- `R1c3` - disambiguation (rook from rank 1)
- `exd5` - pawn capture

❌ **Invalid moves (will be reported):**
- `Xe1` - X is not a valid piece
- `b9` - 9 is not a valid rank (only 1-8)
- `Qj5` - j is not a valid file (only a-h)
- `3. Nf3` after `1. e4` - non-sequential move number

## Performance

The tool is optimized to handle very large PGN files:
- **Speed**: ~2.4 MB/s (validation and correction)
- **100 MB file**: ~42 seconds
- **1 GB file**: ~7 minutes
- **8 GB file**: ~57 minutes

### Benchmarks

To measure performance on your system, use the included benchmark scripts:

```bash
# Windows (PowerShell)
.\benchmark.ps1                    # Test large files
.\benchmark.ps1 -All               # Test all files in test_files
.\benchmark.ps1 file.pgn           # Test a specific file

# Linux/Mac (Bash)
./benchmark.sh                     # Test large files
./benchmark.sh --all               # Test all files in test_files
./benchmark.sh file.pgn            # Test a specific file
```

The benchmark scripts show:
- Validation and correction time for each file
- Speed in MB/s
- Projections for very large files (100MB, 500MB, 1GB, 8GB)
- Aggregate statistics

### Implemented Optimizations

- 1MB read/write buffers for efficient I/O
- Pre-compiled regex to avoid recompilations
- Progress bar updated every 1000 lines to reduce overhead
- Optimized parsing of moves and dates

## Batch Validation

To validate multiple PGN files in a directory:

```bash
# Windows (PowerShell)
.\validate_all.ps1 .\test_files                    # Validation only
.\validate_all.ps1 .\test_files -OutputDir .\fixed  # Validate and fix

# Linux/Mac (Bash)
./validate_all.sh ./test_files                     # Validation only
./validate_all.sh ./test_files -o ./fixed          # Validate and fix
```

The scripts show:
- Progress for each file
- List of errors and warnings found
- Final summary with valid/invalid file count

## Development

### Tests
```bash
# Run all tests
go test -v ./...

# Test with coverage
go test -cover ./...
```

### Build
```bash
# Run the tool in development mode
go run . test_files\example_valid.pgn

# Standard build
go build -o pgn_check.exe

# Optimized build with version
VERSION=$(cat VERSION)
go build -ldflags="-X main.Version=$VERSION -s -w" -o pgn_check.exe
```

### CI/CD Workflows

The project includes GitHub Actions workflows for automatic build and testing:

- **build-linux.yml**: Compiles, tests, and creates artifacts for Linux
- **build-windows.yml**: Compiles, tests, and creates artifacts for Windows

Each workflow:
1. Reads the version from the `VERSION` file
2. Compiles the binary with embedded version
3. Runs all Go tests
4. Runs performance benchmarks
5. Creates an artifact with binary, version, and benchmark results

To update the version, simply modify the `VERSION` file.

## Example Files

The repository includes example files in the `test_files/` folder:
- `example_valid.pgn` - Valid PGN file
- `example_invalid_date.pgn` - File with incorrectly formatted date
- `multiple_games_test.pgn` - File with multiple games
- `test_eventdate.pgn` - File with malformed EventDate
- `twic1617.pgn` - Real file with hundreds of games

## Requirements

- Go 1.21 or higher

## Author

**Nazario D'Apote**
- Email: nazario [dot] dapote [at] gmail [dot] com
- GitHub: [@nazariodapote](https://github.com/nazariodapote)

## License

MIT License - see the [LICENSE](LICENSE) file for details.

Copyright (c) 2026 Nazario D'Apote
