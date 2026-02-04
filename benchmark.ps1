# PGN Check - Performance Benchmark Script
# Measures validation and correction performance on test files

param(
    [Parameter(Mandatory=$false)]
    [string]$TestFile = "",
    
    [Parameter(Mandatory=$false)]
    [switch]$All
)

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "PGN Check - Performance Benchmark" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host ""

# Check if executable exists
if (-not (Test-Path ".\pgn_check.exe")) {
    Write-Host "Error: pgn_check.exe not found. Building..." -ForegroundColor Yellow
    go build -o pgn_check.exe
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Build failed!" -ForegroundColor Red
        exit 1
    }
    Write-Host "Build completed successfully!" -ForegroundColor Green
    Write-Host ""
}

function Test-FilePerformance {
    param(
        [string]$FilePath
    )
    
    if (-not (Test-Path $FilePath)) {
        Write-Host "  File not found: $FilePath" -ForegroundColor Red
        return $null
    }
    
    $fileInfo = Get-Item $FilePath
    $fileSizeMB = [math]::Round($fileInfo.Length / 1MB, 2)
    $fileName = $fileInfo.Name
    
    Write-Host "Testing: $fileName ($fileSizeMB MB)" -ForegroundColor White
    
    # Test validation
    Write-Host "  Running validation..." -NoNewline
    $validationTime = Measure-Command { 
        .\pgn_check.exe $FilePath 2>&1 | Out-Null
    }
    $validationSeconds = [math]::Round($validationTime.TotalSeconds, 2)
    $validationSpeed = if ($validationSeconds -gt 0) { 
        [math]::Round($fileSizeMB / $validationSeconds, 2) 
    } else { 
        0 
    }
    Write-Host " $validationSeconds sec ($validationSpeed MB/s)" -ForegroundColor Green
    
    # Test correction
    $tempOutput = "temp_benchmark_output.pgn"
    Write-Host "  Running correction..." -NoNewline
    $correctionTime = Measure-Command { 
        .\pgn_check.exe -o $tempOutput $FilePath 2>&1 | Out-Null
    }
    $correctionSeconds = [math]::Round($correctionTime.TotalSeconds, 2)
    $correctionSpeed = if ($correctionSeconds -gt 0) { 
        [math]::Round($fileSizeMB / $correctionSeconds, 2) 
    } else { 
        0 
    }
    Write-Host " $correctionSeconds sec ($correctionSpeed MB/s)" -ForegroundColor Green
    
    # Cleanup
    if (Test-Path $tempOutput) {
        Remove-Item $tempOutput -Force
    }
    
    Write-Host ""
    
    return @{
        FileName = $fileName
        SizeMB = $fileSizeMB
        ValidationTime = $validationSeconds
        ValidationSpeed = $validationSpeed
        CorrectionTime = $correctionSeconds
        CorrectionSpeed = $correctionSpeed
    }
}

# Collect results
$results = @()

if ($TestFile -ne "") {
    # Test single file
    $result = Test-FilePerformance -FilePath $TestFile
    if ($result) {
        $results += $result
    }
} elseif ($All) {
    # Test all PGN files in test_files directory
    $pgnFiles = Get-ChildItem -Path "test_files" -Filter "*.pgn" | Sort-Object Length -Descending
    
    foreach ($file in $pgnFiles) {
        $result = Test-FilePerformance -FilePath $file.FullName
        if ($result) {
            $results += $result
        }
    }
} else {
    # Test default large files
    Write-Host "Testing large files (use -All to test all files)..." -ForegroundColor Yellow
    Write-Host ""
    
    $defaultFiles = @(
        "test_files\twic1617.pgn",
        "test_files\twic920.pgn"
    )
    
    foreach ($filePath in $defaultFiles) {
        $result = Test-FilePerformance -FilePath $filePath
        if ($result) {
            $results += $result
        }
    }
}

# Display summary
if ($results.Count -gt 0) {
    Write-Host "=========================================" -ForegroundColor Cyan
    Write-Host "Summary" -ForegroundColor Cyan
    Write-Host "=========================================" -ForegroundColor Cyan
    Write-Host ""
    
    # Calculate averages
    $totalSize = ($results | Measure-Object -Property SizeMB -Sum).Sum
    $avgValidationSpeed = ($results | Measure-Object -Property ValidationSpeed -Average).Average
    $avgCorrectionSpeed = ($results | Measure-Object -Property CorrectionSpeed -Average).Average
    
    Write-Host "Total data processed: $([math]::Round($totalSize, 2)) MB" -ForegroundColor White
    Write-Host "Average validation speed: $([math]::Round($avgValidationSpeed, 2)) MB/s" -ForegroundColor Green
    Write-Host "Average correction speed: $([math]::Round($avgCorrectionSpeed, 2)) MB/s" -ForegroundColor Green
    Write-Host ""
    
    # Projection for large files
    Write-Host "=========================================" -ForegroundColor Cyan
    Write-Host "Projections for Large Files" -ForegroundColor Cyan
    Write-Host "=========================================" -ForegroundColor Cyan
    Write-Host ""
    
    $fileSizes = @(100, 500, 1000, 8000)
    foreach ($size in $fileSizes) {
        $validationMinutes = [math]::Round(($size / $avgValidationSpeed) / 60, 1)
        $correctionMinutes = [math]::Round(($size / $avgCorrectionSpeed) / 60, 1)
        
        $sizeLabel = if ($size -ge 1000) { "$($size/1000) GB" } else { "$size MB" }
        Write-Host "${sizeLabel}:"
        Write-Host "  Validation: $validationMinutes min" -ForegroundColor Yellow
        Write-Host "  Correction: $correctionMinutes min" -ForegroundColor Yellow
    }
    Write-Host ""
    
    # Detailed results table
    Write-Host "=========================================" -ForegroundColor Cyan
    Write-Host "Detailed Results" -ForegroundColor Cyan
    Write-Host "=========================================" -ForegroundColor Cyan
    Write-Host ""
    
    $results | Format-Table -Property `
        @{Label="File"; Expression={$_.FileName}; Width=30}, `
        @{Label="Size(MB)"; Expression={$_.SizeMB}; Width=10}, `
        @{Label="Valid(s)"; Expression={$_.ValidationTime}; Width=10}, `
        @{Label="Valid(MB/s)"; Expression={$_.ValidationSpeed}; Width=12}, `
        @{Label="Corr(s)"; Expression={$_.CorrectionTime}; Width=10}, `
        @{Label="Corr(MB/s)"; Expression={$_.CorrectionSpeed}; Width=12}
}

Write-Host "Benchmark completed!" -ForegroundColor Green
