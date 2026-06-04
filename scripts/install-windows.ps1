#Requires -RunAsAdministrator

$ErrorActionPreference = "Stop"

$Repo = "ddc-111/115LocalNatManager"
$InstallDir = "$env:USERPROFILE\.115manager"
$BinName = "115manager.exe"
$ServiceName = "115Manager"

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Get-LatestVersion {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    return $release.tag_name
}

function Download-Binary {
    param([string]$Version)
    
    $filename = "$BinName-windows-amd64.zip"
    $url = "https://github.com/$Repo/releases/download/$Version/$filename"
    
    Write-Info "Downloading $filename..."
    
    if (!(Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    
    $zipPath = "$InstallDir\$filename"
    Invoke-WebRequest -Uri $url -OutFile $zipPath
    
    Write-Info "Extracting..."
    Expand-Archive -Path $zipPath -DestinationPath $InstallDir -Force
    Remove-Item $zipPath -Force
}

function Install-Service {
    $binaryPath = "$InstallDir\$BinName"
    
    if (Get-Service $ServiceName -ErrorAction SilentlyContinue) {
        Write-Warn "Service already exists, removing..."
        Stop-Service $ServiceName -Force -ErrorAction SilentlyContinue
        sc.exe delete $ServiceName | Out-Null
        Start-Sleep -Seconds 2
    }
    
    Write-Info "Creating Windows service..."
    
    $nssmPath = "$InstallDir\nssm.exe"
    if (!(Test-Path $nssmPath)) {
        Write-Info "Downloading NSSM..."
        $nssmUrl = "https://nssm.cc/release/nssm-2.24.zip"
        $nssmZip = "$InstallDir\nssm.zip"
        Invoke-WebRequest -Uri $nssmUrl -OutFile $nssmZip
        Expand-Archive -Path $nssmZip -DestinationPath $InstallDir -Force
        Copy-Item "$InstallDir\nssm-2.24\win64\nssm.exe" $nssmPath
        Remove-Item $nssmZip -Force
        Remove-Item "$InstallDir\nssm-2.24" -Recurse -Force
    }
    
    & $nssmPath install $ServiceName $binaryPath "-data" $InstallDir
    & $nssmPath set $ServiceName DisplayName "115 Local NAT Manager"
    & $nssmPath set $ServiceName Description "115 Cloud Download Manager Service"
    & $nssmPath set $ServiceName Start SERVICE_AUTO_START
    & $nssmPath set $ServiceName AppStdout "$InstallDir\stdout.log"
    & $nssmPath set $ServiceName AppStderr "$InstallDir\stderr.log"
    & $nssmPath set $ServiceName AppRestartDelay 5000
    
    Write-Info "Starting service..."
    Start-Service $ServiceName
}

function Add-FirewallRule {
    Write-Info "Adding firewall rule..."
    
    $ruleName = "115 Local NAT Manager"
    $existingRule = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
    
    if ($existingRule) {
        Remove-NetFirewallRule -DisplayName $ruleName
    }
    
    New-NetFirewallRule -DisplayName $ruleName `
        -Direction Inbound `
        -Protocol TCP `
        -LocalPort 11580 `
        -Action Allow `
        -Profile Private `
        -Description "Allow 115 Local NAT Manager" | Out-Null
}

function Print-Success {
    Write-Host ""
    Write-Host "==========================================" -ForegroundColor Green
    Write-Host "  Installation Complete!" -ForegroundColor Green
    Write-Host "==========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Binary installed to: $InstallDir\$BinName"
    Write-Host "Service name: $ServiceName"
    Write-Host "Server port: 11580"
    Write-Host ""
    Write-Host "Commands:"
    Write-Host "  Start:   Start-Service $ServiceName"
    Write-Host "  Stop:    Stop-Service $ServiceName"
    Write-Host "  Status:  Get-Service $ServiceName"
    Write-Host ""
    Write-Host "Chrome Extension:"
    Write-Host "  1. Open chrome://extensions"
    Write-Host "  2. Enable Developer mode"
    Write-Host "  3. Click 'Load unpacked'"
    Write-Host "  4. Select the extension folder"
    Write-Host ""
}

function Main {
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host "  115 Local NAT Manager Installer" -ForegroundColor Cyan
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host ""
    
    $version = Get-LatestVersion
    Write-Info "Latest version: $version"
    
    Download-Binary -Version $version
    Add-FirewallRule
    Install-Service
    Print-Success
}

Main
