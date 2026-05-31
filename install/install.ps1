param([string]$Version = "latest")
$ErrorActionPreference = "Stop"

$Repo        = "kiiimatz/renode"
$BinName     = "renode.exe"
$ServiceName = "renode"
$InstallDir  = "C:\Program Files\renode"
$ScriptUrl   = "https://raw.githubusercontent.com/kiiimatz/renode/main/install/install.ps1"

# ── Self-elevate via UAC if not already admin ─────────────────────────────────
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
             [Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
  Write-Host "Requesting administrator privileges (UAC prompt will appear)..."
  $cmd = "iwr '$ScriptUrl' -UseBasicParsing | iex; Write-Host ''; Read-Host 'Done — press Enter to close'"
  Start-Process powershell.exe -ArgumentList "-NoProfile -ExecutionPolicy Bypass -Command `"$cmd`"" -Verb RunAs -Wait
  exit 0
}

# ── Resolve version ────────────────────────────────────────────────────────────
if ($Version -eq "latest") {
  $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" `
               -UseBasicParsing -Headers @{ "User-Agent" = "renode-installer" }
  $Version = $release.tag_name
}

Write-Host "Installing renode $Version for windows/amd64 ..."

# ── Windows Defender exclusion (must be done before download) ─────────────────
Write-Host "Adding Windows Defender exclusion for $InstallDir ..."
try {
  Add-MpPreference -ExclusionPath $InstallDir -ErrorAction Stop
} catch {
  Write-Host "Warning: could not add Defender exclusion: $_"
}

# ── Download binary ────────────────────────────────────────────────────────────
$Asset   = "renode_windows_amd64.exe"
$Url     = "https://github.com/$Repo/releases/download/$Version/$Asset"
$TmpPath = "$env:TEMP\renode_download.exe"

Write-Host "Downloading $Url ..."
Invoke-WebRequest -Uri $Url -OutFile $TmpPath -UseBasicParsing

# ── Install binary ─────────────────────────────────────────────────────────────
if (-not (Test-Path $InstallDir)) {
  New-Item -ItemType Directory -Path $InstallDir | Out-Null
}
Copy-Item -Path $TmpPath -Destination "$InstallDir\$BinName" -Force
Remove-Item $TmpPath -Force
Write-Host "Binary installed to $InstallDir\$BinName"

# ── Add to system PATH ────────────────────────────────────────────────────────
$machinePath = [System.Environment]::GetEnvironmentVariable("Path", "Machine")
if ($machinePath -notlike "*$InstallDir*") {
  [System.Environment]::SetEnvironmentVariable("Path", "$machinePath;$InstallDir", "Machine")
  Write-Host "Added $InstallDir to system PATH."
}

# ── Windows Service ────────────────────────────────────────────────────────────
$existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($existing) {
  Write-Host "Removing existing renode service ..."
  Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
  & sc.exe delete $ServiceName | Out-Null
  Start-Sleep 2
}

Write-Host "Creating Windows Service (auto-start daemon) ..."
& sc.exe create $ServiceName `
    binPath= "`"$InstallDir\$BinName`" up --foreground" `
    start= auto `
    DisplayName= "renode tunnel daemon" | Out-Null
& sc.exe description $ServiceName "renode open-source tunneling client" | Out-Null
& sc.exe failure $ServiceName reset= 60 actions= restart/5000/restart/10000/restart/30000 | Out-Null

Write-Host ""
Write-Host "renode $Version installed successfully."
Write-Host "Configure relay: edit %APPDATA%\renode\config.yaml"
Write-Host "Then start:      renode up"
Write-Host "(open a new terminal for PATH to take effect)"
