# Script to validate all PGN files in a directory
# Usage: .\validate_all.ps1 <directory> [-OutputDir output_directory]

param(
    [Parameter(Mandatory=$true, Position=0)]
    [string]$InputDir,
    
    [Parameter(Mandatory=$false)]
    [string]$OutputDir
)

# Check if directory exists
if (-not (Test-Path $InputDir)) {
    Write-Host "Error: Directory '$InputDir' not found" -ForegroundColor Red
    exit 1
}

# Create output directory if specified
$autoCorrect = $false
if ($OutputDir) {
    $autoCorrect = $true
    New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
}

# Find all .pgn files
$pgnFiles = Get-ChildItem -Path $InputDir -Filter "*.pgn" -File

if ($pgnFiles.Count -eq 0) {
    Write-Host "No PGN files found in '$InputDir'" -ForegroundColor Yellow
    exit 0
}

$totalFiles = $pgnFiles.Count
$current = 0
$validCount = 0
$invalidCount = 0

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "PGN Validator - Batch Processing" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Directory: $InputDir"
Write-Host "Total files: $totalFiles"
if ($autoCorrect) {
    Write-Host "Output directory: $OutputDir"
}
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# Process each file
foreach ($file in $pgnFiles) {
    $current++
    $filename = $file.Name
    
    Write-Host "[$current/$totalFiles] Processing: $filename" -ForegroundColor White
    
    if ($autoCorrect) {
        # Validate and correct
        $outputFile = Join-Path $OutputDir $filename
        $result = & .\pgn_check.exe -o $outputFile $file.FullName 2>&1
        $exitCode = $LASTEXITCODE
    } else {
        # Validate only
        $result = & .\pgn_check.exe $file.FullName 2>&1
        $exitCode = $LASTEXITCODE
    }
    
    if ($exitCode -eq 0) {
        $validCount++
        Write-Host "  ✓ Valid" -ForegroundColor Green
    } else {
        $invalidCount++
        Write-Host "  ✗ Errors found" -ForegroundColor Yellow
        # Display errors and warnings if available
        if ($result) {
            $errorLines = $result | Where-Object { $_ -match "(Error:|Warning:|Line \d+:)" }
            foreach ($errLine in $errorLines) {
                if ($errLine -match "Warning:") {
                    Write-Host "    $errLine" -ForegroundColor Yellow
                } else {
                    Write-Host "    $errLine" -ForegroundColor Red
                }
            }
        }
    }
    Write-Host ""
}

# Summary
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Summary" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "Total processed: $totalFiles"
Write-Host "Valid files: " -NoNewline
Write-Host "$validCount" -ForegroundColor Green
Write-Host "Files with errors: " -NoNewline
Write-Host "$invalidCount" -ForegroundColor Yellow
if ($autoCorrect) {
    Write-Host "Corrected files saved to: $OutputDir"
}
Write-Host "=========================================" -ForegroundColor Cyan
