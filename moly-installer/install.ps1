# Moly Windows Installer
# Handles: Binary installation, native messaging registry setup, PATH configuration
# Supports: Chrome, Brave, and Chromium-based browsers

param(
    [string]$ExtensionId = "jkvuyxvgeivlakjahixagdztxvrcpzbc"
)

# Requires admin privileges for registry operations
$currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "Error: This installer requires administrator privileges" -ForegroundColor Red
    Write-Host "Please run PowerShell as Administrator and try again" -ForegroundColor Yellow
    exit 1
}

# Configuration
$MolyVersion = "1.0.0"
$InstallDir = "$env:APPDATA\Moly"
$ConfigDir = "$env:APPDATA\Moly"
$BinaryName = "moly.exe"
$BinaryPath = Join-Path $InstallDir $BinaryName
$ExtensionId = $ExtensionId -replace "[^a-z0-9]", ""

# Color helper functions
function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-Error-Custom {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
}

function Write-Warning-Custom {
    param([string]$Message)
    Write-Host "⚠ $Message" -ForegroundColor Yellow
}

function Write-Info {
    param([string]$Message)
    Write-Host "→ $Message" -ForegroundColor Cyan
}

# Main script
Write-Host ""
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "Moly Installer v$MolyVersion" -ForegroundColor Cyan
Write-Host "Windows Edition" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

# Check if binary exists
if (-not (Test-Path ".\$BinaryName")) {
    Write-Error-Custom "Binary not found: .\$BinaryName"
    Write-Host "Please run this installer from the directory containing $BinaryName"
    exit 1
}

Write-Info "Creating installation directory..."
try {
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        Write-Success "Directory created: $InstallDir"
    } else {
        Write-Host "Directory already exists: $InstallDir"
    }
} catch {
    Write-Error-Custom "Failed to create directory: $InstallDir"
    Write-Host $_.Exception.Message
    exit 1
}

Write-Info "Installing Moly binary..."
try {
    Copy-Item -Path ".\$BinaryName" -Destination $BinaryPath -Force
    Write-Success "Binary installed to $BinaryPath"
} catch {
    Write-Error-Custom "Failed to copy binary"
    Write-Host $_.Exception.Message
    exit 1
}

Write-Info "Setting up native messaging..."

# Function to setup native messaging registry entries
function Setup-NativeMessaging {
    param(
        [string]$BrowserName,
        [string]$RegistryPath
    )

    try {
        $FullRegistryPath = "HKCU:\$RegistryPath\com.moly.backend_host"

        # Create registry key if it doesn't exist
        if (-not (Test-Path $FullRegistryPath)) {
            New-Item -Path $FullRegistryPath -Force | Out-Null
        }

        # Create manifest content
        $manifest = @{
            name = "com.moly.backend_host"
            description = "Moly Backend Launcher"
            path = $BinaryPath
            type = "stdio"
            allowed_origins = @(
                "chrome-extension://$ExtensionId/"
            )
        }

        # Convert to JSON (PowerShell 3.0+)
        $manifestJson = $manifest | ConvertTo-Json

        # Save manifest to registry
        Set-ItemProperty -Path $FullRegistryPath -Name "(Default)" -Value $manifestJson -Force

        Write-Success "Native messaging configured for $BrowserName"
        Write-Host "  Registry: $FullRegistryPath"

        return $true
    } catch {
        Write-Warning-Custom "Failed to setup $BrowserName native messaging"
        Write-Host "  Error: $($_.Exception.Message)"
        return $false
    }
}

# Setup for Chrome
$chromeRegistryPath = "Software\Google\Chrome\NativeMessagingHosts"
Setup-NativeMessaging "Chrome" $chromeRegistryPath | Out-Null

# Setup for Brave
$braveRegistryPath = "Software\BraveSoftware\Brave-Browser\NativeMessagingHosts"
Setup-NativeMessaging "Brave" $braveRegistryPath | Out-Null

# Verify installation
Write-Host ""
Write-Info "Verifying installation..."

$verificationPassed = $true

if (-not (Test-Path $BinaryPath)) {
    Write-Error-Custom "Binary verification failed: $BinaryPath not found"
    $verificationPassed = $false
} else {
    Write-Success "Binary found at $BinaryPath"
}

if (-not (Test-Path $ConfigDir)) {
    Write-Error-Custom "Config directory verification failed"
    $verificationPassed = $false
} else {
    Write-Success "Config directory ready: $ConfigDir"
}

# Check if path is in system PATH (optional check)
$pathEnv = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($pathEnv -notlike "*$InstallDir*") {
    Write-Warning-Custom "Installation directory not in PATH"
    Write-Host "You can add it manually or run 'Add-MolyToPath' after installation"
} else {
    Write-Success "Installation directory is in PATH"
}

if (-not $verificationPassed) {
    Write-Host ""
    Write-Error-Custom "Installation verification failed"
    exit 1
}

Write-Success "Installation verification passed"

# Final summary
Write-Host ""
Write-Host "============================================" -ForegroundColor Green
Write-Host "Installation Complete!" -ForegroundColor Green
Write-Host "============================================" -ForegroundColor Green
Write-Host ""

Write-Host "Moly is now ready to use:" -ForegroundColor Cyan
Write-Host ""

Write-Host "1. Start Moly backend (in PowerShell or Command Prompt):" -ForegroundColor White
Write-Host "   & '$BinaryPath'" -ForegroundColor Gray
Write-Host "   or simply:" -ForegroundColor Gray
Write-Host "   moly" -ForegroundColor Gray
Write-Host ""

Write-Host "2. Load extension in browser:" -ForegroundColor White
Write-Host "   Chrome:  chrome://extensions" -ForegroundColor Gray
Write-Host "   Brave:   brave://extensions" -ForegroundColor Gray
Write-Host ""

Write-Host "3. Install the extension:" -ForegroundColor White
Write-Host "   - Click 'Load unpacked'" -ForegroundColor Gray
Write-Host "   - Select: <path-to>\moly-extension\dist" -ForegroundColor Gray
Write-Host ""

Write-Host "Configuration:" -ForegroundColor Cyan
Write-Host "  Install location: $InstallDir" -ForegroundColor Gray
Write-Host "  Database path:    $ConfigDir\moly.db" -ForegroundColor Gray
Write-Host "  Config file:      $ConfigDir\moly.config.json" -ForegroundColor Gray
Write-Host ""

Write-Host "Environment Variables (optional):" -ForegroundColor Cyan
Write-Host "  MOLY_PORT=:11436 (server port)" -ForegroundColor Gray
Write-Host "  MOLY_HOST=127.0.0.1 (bind address)" -ForegroundColor Gray
Write-Host "  MOLY_LOG_LEVEL=info (log level)" -ForegroundColor Gray
Write-Host "  MOLY_CORS_PROXY_PORT=:11435 (proxy port)" -ForegroundColor Gray
Write-Host ""

Write-Host "Troubleshooting:" -ForegroundColor Cyan
Write-Host "  If 'moly' command not found, open a new PowerShell window" -ForegroundColor Gray
Write-Host "  or manually run: & '$BinaryPath'" -ForegroundColor Gray
Write-Host ""

Write-Host "To uninstall:" -ForegroundColor Cyan
Write-Host "  Remove-Item -Path '$InstallDir' -Recurse -Force" -ForegroundColor Gray
Write-Host "  Remove-ItemProperty -Path 'HKCU:\Software\Google\Chrome\NativeMessagingHosts' -Name 'com.moly.backend_host' -ErrorAction SilentlyContinue" -ForegroundColor Gray
Write-Host "  Remove-ItemProperty -Path 'HKCU:\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts' -Name 'com.moly.backend_host' -ErrorAction SilentlyContinue" -ForegroundColor Gray
Write-Host ""

Write-Host "For more help, see: moly-installer\README.md" -ForegroundColor Cyan
Write-Host ""
