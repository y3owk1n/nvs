# NVS PowerShell Uninstaller for Windows
# Requires PowerShell 5.1+

# Function to write colored output
function Write-Info {
    param([string]$Message)
    Write-Host "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') [INFO] $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') [SUCCESS] $Message" -ForegroundColor Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') [ERROR] $Message" -ForegroundColor Red
}

function Write-Warning {
    param([string]$Message)
    Write-Host "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') [WARNING] $Message" -ForegroundColor Yellow
}

# Header banner
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "            NVS Uninstaller           " -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan

try {
    $binaryName = "nvs.exe"
    $installDir = "$env:LOCALAPPDATA\Programs\nvs"
    $installPath = "$installDir\$binaryName"

    Write-Info "Install path: $installPath"

    # Check if binary exists
    if (!(Test-Path $installPath)) {
        Write-Info "No installed binary found at $installPath."
        exit 0
    }

    # Confirm uninstallation
    $confirmation = Read-Host "Are you sure you want to uninstall NVS? (y/N)"
    if ($confirmation -notmatch "^[Yy]$") {
        Write-Info "Uninstallation cancelled."
        exit 0
    }

    # Remove binary
    Remove-Item $installPath -Force
    Write-Success "Removed binary: $installPath"

    # Remove directory if empty
    if ((Get-ChildItem $installDir -Force | Measure-Object).Count -eq 0) {
        Remove-Item $installDir -Force
        Write-Info "Removed empty install directory: $installDir"
    } else {
        Write-Warning "Install directory not empty, keeping: $installDir"
    }

    # Remove from PATH
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = $userPath -split ";" | Where-Object { $_.Trim() -ne "" }
    $newPathEntries = $pathEntries | Where-Object { $_ -ne $installDir }
    $newPath = $newPathEntries -join ";"

    if ($newPath -ne $userPath) {
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-Success "Removed $installDir from user PATH. Restart your terminal for changes to take effect."
    } else {
        Write-Info "Install directory not found in PATH."
    }

    Write-Success "Uninstallation complete."

} catch {
    Write-Error "Uninstallation failed: $($_.Exception.Message)"
    exit 1
}