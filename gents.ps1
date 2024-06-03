#!/usr/bin/env pwsh
param (
    [string]$inputFile,
    [string]$outputFile
)

# Exit on error
$ErrorActionPreference = "Stop"

# Generate new TypeScript client
go run vendor/github.com/mjl-/sherpats/cmd/sherpats/main.go -bytes-to-string -slices-nullable -maps-nullable -nullable-optional -namespace api api < $inputFile > "$outputFile.tmp"

# Compare the new output with the existing one and update if different
if (Compare-Object (Get-Content $outputFile) (Get-Content "$outputFile.tmp") -SyncWindow 0) {
    Move-Item -Force "$outputFile.tmp" $outputFile
} else {
    Remove-Item "$outputFile.tmp"
}
