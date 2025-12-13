# NVS PowerShell Installer for Windows
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

function Write-ErrorMessage {
    param([string]$Message)
    Write-Host "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') [ERROR] $Message" -ForegroundColor Red
}

function Write-WarningMessage {
    param([string]$Message)
    Write-Host "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') [WARNING] $Message" -ForegroundColor Yellow
}

# Header banner
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "           NVS Installer              " -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan

# Initialize temp file variables
$tempChecksum = $null
$tempBinary = $null

try {
    # Detect architecture
    $arch = $env:PROCESSOR_ARCHITECTURE
    Write-Info "Detected architecture: $arch"

    # Set download URLs based on architecture
    $repo = "y3owk1n/nvs"
    $version = "1.12.1"
    $binaryName = "nvs.exe"
    $installDir = "$env:LOCALAPPDATA\Programs\nvs"
    $installPath = "$installDir\$binaryName"

    if ($arch -eq "AMD64") {
        $assetUrl = "https://github.com/$repo/releases/download/v$version/nvs-windows-amd64.exe"
        $checksumUrl = "https://github.com/$repo/releases/download/v$version/nvs-windows-amd64.exe.sha256"
    } elseif ($arch -eq "ARM64") {
        $assetUrl = "https://github.com/$repo/releases/download/v$version/nvs-windows-arm64.exe"
        $checksumUrl = "https://github.com/$repo/releases/download/v$version/nvs-windows-arm64.exe.sha256"
    } else {
        Write-ErrorMessage "Unsupported architecture: $arch"
        exit 1
    }

    Write-Info "Download URL: $assetUrl"
    Write-Info "Checksum URL: $checksumUrl"
    Write-Info "Install path: $installPath"

    # Create install directory if it doesn't exist
    if (!(Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        Write-Info "Created install directory: $installDir"
    }

    # Download checksum file
    $tempChecksum = [System.IO.Path]::GetTempFileName()
    Write-Info "Downloading checksum..."
    try {
        Invoke-WebRequest -Uri $checksumUrl -OutFile $tempChecksum -UseBasicParsing -ErrorAction Stop
    } catch {
        Write-ErrorMessage "Failed to download checksum file: $($_.Exception.Message)"
        exit 1
    }

    # Extract expected checksum (first field, handles both "checksum" and "checksum filename" formats)
    $checksumContent = Get-Content $tempChecksum -ErrorAction SilentlyContinue
    if ($checksumContent -and $checksumContent.Trim()) {
        $expectedChecksum = $checksumContent.Trim().Split()[0].Trim().ToLower()
        if ([string]::IsNullOrEmpty($expectedChecksum)) {
            Write-ErrorMessage "Invalid checksum file format"
            Remove-Item $tempChecksum -Force
            exit 1
        }
    } else {
        Write-ErrorMessage "Checksum file is empty or invalid"
        Remove-Item $tempChecksum -Force
        exit 1
    }
    Write-Info "Expected checksum: $expectedChecksum"

    # Download binary
    $tempBinary = [System.IO.Path]::GetTempFileName()
    Write-Info "Downloading binary..."
    try {
        Invoke-WebRequest -Uri $assetUrl -OutFile $tempBinary -UseBasicParsing -ErrorAction Stop
    } catch {
        Write-ErrorMessage "Failed to download binary: $($_.Exception.Message)"
        Remove-Item $tempChecksum -Force -ErrorAction SilentlyContinue
        exit 1
    }

    # Verify binary was downloaded
    if (!(Test-Path $tempBinary) -or (Get-Item $tempBinary).Length -eq 0) {
        Write-ErrorMessage "Downloaded binary file is empty or missing"
        Remove-Item $tempBinary -Force -ErrorAction SilentlyContinue
        Remove-Item $tempChecksum -Force -ErrorAction SilentlyContinue
        exit 1
    }

    # Compute checksum
    $computedChecksum = (Get-FileHash -Path $tempBinary -Algorithm SHA256).Hash.ToLower()
    Write-Info "Computed checksum: $computedChecksum"

    # Verify checksum
    if ($expectedChecksum -ne $computedChecksum) {
        Write-ErrorMessage "Checksum verification failed! The downloaded file may be corrupted."
        Remove-Item $tempBinary -Force
        Remove-Item $tempChecksum -Force
        exit 1
    } else {
        Write-Success "Checksum verification passed."
    }

    # Move binary to install location
    Move-Item -Path $tempBinary -Destination $installPath -Force
    Write-Success "Binary installed to: $installPath"

    # Clean up temp checksum file
    Remove-Item $tempChecksum -Force

    # Add to PATH if not already present
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$installDir*") {
        if ([string]::IsNullOrEmpty($userPath)) {
            $newPath = $installDir
        } else {
            $newPath = "$userPath;$installDir"
        }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-Success "Added $installDir to user PATH. Restart your terminal for changes to take effect."
    } else {
        Write-Info "Install directory already in PATH."
    }

    Write-Success "Installation complete!"
    Write-Host "You can now run: nvs help" -ForegroundColor Cyan

} catch {
    Write-ErrorMessage "Installation failed: $($_.Exception.Message)"
    # Clean up temp files
    if (Test-Path $tempBinary -ErrorAction SilentlyContinue) {
        Remove-Item $tempBinary -Force -ErrorAction SilentlyContinue
    }
    if (Test-Path $tempChecksum -ErrorAction SilentlyContinue) {
        Remove-Item $tempChecksum -Force -ErrorAction SilentlyContinue
    }
    exit 1
}
