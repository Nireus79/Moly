# Moly CORS Proxy - Windows Installation Script
# Installs moly-proxy and configures Task Scheduler auto-start

# Requires admin privileges
if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "❌ This script requires administrator privileges"
    Write-Host "Please run PowerShell as Administrator"
    exit 1
}

Write-Host "╔════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║ Moly CORS Proxy - Windows Installation ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# Check Node.js
$nodeVersion = node -v 2>$null
if (-not $nodeVersion) {
    Write-Host "❌ Node.js not found. Please install Node.js 16+" -ForegroundColor Red
    Write-Host "   Download from: https://nodejs.org/"
    Write-Host "   Or via Chocolatey: choco install nodejs"
    exit 1
}

Write-Host "✓ Node.js found: $nodeVersion" -ForegroundColor Green

# Check npm
$npmVersion = npm -v 2>$null
if (-not $npmVersion) {
    Write-Host "❌ npm not found. Please install npm." -ForegroundColor Red
    exit 1
}

Write-Host "✓ npm found: $npmVersion" -ForegroundColor Green
Write-Host ""

# Install globally
Write-Host "Installing moly-proxy globally..."
npm install -g moly-proxy

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to install moly-proxy" -ForegroundColor Red
    exit 1
}

Write-Host "✓ moly-proxy installed successfully" -ForegroundColor Green
Write-Host ""

# Create Task Scheduler task
Write-Host "Configuring Task Scheduler auto-start..."

$proxyPath = (npm list -g moly-proxy --prefix | Select-Object -First 1).Replace("npm:", "").Trim() + "\node_modules\moly-proxy\src\cli.js"

# Use the npm command directly
$action = New-ScheduledTaskAction -Execute "node" -Argument "`"$proxyPath`""
$trigger = New-ScheduledTaskTrigger -AtLogOn
$principal = New-ScheduledTaskPrincipal -UserId "$env:USERNAME" -LogonType Interactive -RunLevel Highest
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 10)

# Remove existing task if it exists
$existingTask = Get-ScheduledTask -TaskName "Moly Proxy" -ErrorAction SilentlyContinue
if ($existingTask) {
    Unregister-ScheduledTask -TaskName "Moly Proxy" -Confirm:$false
}

# Register new task
Register-ScheduledTask -TaskName "Moly Proxy" `
    -Action $action `
    -Trigger $trigger `
    -Principal $principal `
    -Settings $settings `
    -Description "Moly CORS Proxy for Ollama" | Out-Null

Write-Host "✓ Task Scheduler task created" -ForegroundColor Green

# Offer to start task
$choice = Read-Host "Start Moly Proxy now? (y/n)"
if ($choice -eq "y" -or $choice -eq "Y") {
    Start-ScheduledTask -TaskName "Moly Proxy"
    Start-Sleep -Seconds 2

    $taskInfo = Get-ScheduledTaskInfo -TaskName "Moly Proxy"
    if ($taskInfo.LastTaskResult -eq 0) {
        Write-Host "✓ Service started successfully" -ForegroundColor Green
    } else {
        Write-Host "⚠️  Service may have issues. Check Task Scheduler." -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "╔════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║       Installation Complete!           ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""
Write-Host "Quick start:"
Write-Host "  1. Make sure Ollama is running"
Write-Host "  2. The proxy will start at next login (or manually start via Task Scheduler)"
Write-Host "  3. Install Moly extension"
Write-Host "  4. Moly will auto-detect the proxy"
Write-Host ""
Write-Host "Manage the task:"
Write-Host "  • Open Task Scheduler and search for 'Moly Proxy'"
Write-Host "  • Or use: tasklist /fi 'IMAGENAME eq node.exe'"
Write-Host ""
Write-Host "Stop the service:"
Write-Host "  Stop-ScheduledTask -TaskName 'Moly Proxy'"
Write-Host ""
