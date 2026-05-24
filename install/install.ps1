param(
  [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"

$Repo        = "kiiimatz/taildog"
$BinName     = "taildog.exe"
$ServiceName = "taildog"

# ── Admin check ────────────────────────────────────────────────────────────────
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
             [Security.Principal.WindowsBuiltInRole]::Administrator)

if ($isAdmin) {
  $InstallDir = "C:\Program Files\taildog"
  $PathScope  = "Machine"
} else {
  $InstallDir = "$env:LOCALAPPDATA\taildog"
  $PathScope  = "User"
  Write-Host "Note: not running as Administrator."
  Write-Host "      Installing to $InstallDir (user PATH, no Windows Service)."
  Write-Host "      Re-run from an elevated prompt to install system-wide."
  Write-Host ""
}

# ── Resolve version ────────────────────────────────────────────────────────────
if ($Version -eq "latest") {
  $releaseUrl = "https://api.github.com/repos/$Repo/releases/latest"
  $release    = Invoke-RestMethod -Uri $releaseUrl -UseBasicParsing `
                  -Headers @{ "User-Agent" = "taildog-installer" }
  $Version    = $release.tag_name
}

Write-Host "Installing taildog $Version for windows/amd64 ..."

# ── Download binary ────────────────────────────────────────────────────────────
$Asset   = "taildog_windows_amd64.exe"
$Url     = "https://github.com/$Repo/releases/download/$Version/$Asset"
$TmpPath = "$env:TEMP\taildog_download.exe"

Write-Host "Downloading $Url ..."
Invoke-WebRequest -Uri $Url -OutFile $TmpPath -UseBasicParsing

# ── Install binary ─────────────────────────────────────────────────────────────
if (-not (Test-Path $InstallDir)) {
  New-Item -ItemType Directory -Path $InstallDir | Out-Null
}
Copy-Item -Path $TmpPath -Destination "$InstallDir\$BinName" -Force
Remove-Item $TmpPath -Force

Write-Host "Binary installed to $InstallDir\$BinName"

# ── Add to PATH ────────────────────────────────────────────────────────────────
$currentPath = [System.Environment]::GetEnvironmentVariable("Path", $PathScope)
if ($currentPath -notlike "*$InstallDir*") {
  [System.Environment]::SetEnvironmentVariable(
    "Path",
    "$currentPath;$InstallDir",
    $PathScope
  )
  Write-Host "Added $InstallDir to $PathScope PATH."
}

# ── Windows Service (admin only) ───────────────────────────────────────────────
if ($isAdmin) {
  $existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
  if ($existing) {
    Write-Host "Removing existing taildog service ..."
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    & sc.exe delete $ServiceName | Out-Null
    Start-Sleep 2
  }

  Write-Host "Creating Windows Service ..."
  & sc.exe create $ServiceName `
    binPath= "`"$InstallDir\$BinName`" up --foreground" `
    start= auto `
    DisplayName= "taildog tunnel daemon" | Out-Null

  & sc.exe description $ServiceName "taildog open-source tunneling client" | Out-Null
  & sc.exe failure $ServiceName reset= 60 actions= restart/5000/restart/10000/restart/30000 | Out-Null

  Start-Service -Name $ServiceName
  Write-Host "Service '$ServiceName' created and started."
} else {
  Write-Host ""
  Write-Host "Skipped Windows Service (requires Administrator)."
  Write-Host "Start the daemon manually with: taildog up"
}

Write-Host ""
Write-Host "taildog $Version installed successfully."
Write-Host "Open a new terminal and run: taildog up"
