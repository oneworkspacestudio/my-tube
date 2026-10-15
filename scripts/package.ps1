$ErrorActionPreference='Stop'
$root=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$exe=Join-Path $root 'dist/MyTube.exe'
$version=(& $exe version).Trim()
if($LASTEXITCODE -or $version -notmatch '^[0-9A-Za-z.\-]+$'){throw 'Invalid version'}
$hash=(Get-FileHash -LiteralPath $exe -Algorithm SHA256).Hash.ToLowerInvariant();$size=(Get-Item -LiteralPath $exe).Length
$manifest=@{schema=1;package=@{file='MyTube.exe';size=$size;sha256=$hash};app=@{id='my-tube';name='My Tube';description='Your videos, channels and community';version=$version;kind='process';port=8796;route='/my-tube';stripPrefix=$true;health='/healthz';depends=@('accounts');oidc=$true;callback='/my-tube/auth/callback';args=@('--no-browser','--port','${PORT}');env=@{MYTUBE_DATA_DIR='${DATA}/my-tube';MYTUBE_PUBLIC_URL='${ORIGIN}/my-tube';MYTUBE_DRIVE_URL='${ORIGIN}'};source=('https://github.com/oneworkspacestudio/my-tube/releases/tag/v'+$version)}}
$manifest|ConvertTo-Json -Depth 10|Set-Content -LiteralPath (Join-Path $root 'dist/my-tube.workspace.json') -Encoding utf8
$archive=Join-Path $root "dist/my-tube-$version-windows-amd64.zip"
Compress-Archive -LiteralPath @($exe,(Join-Path $root 'config.example.json'),(Join-Path $root 'environment.example'),(Join-Path $root 'Start-local.cmd'),(Join-Path $root 'README.md'),(Join-Path $root 'LICENSE'),(Join-Path $root 'NOTICE'),(Join-Path $root 'SECURITY.md'),(Join-Path $root 'docs'),(Join-Path $root 'dist/THIRD-PARTY-LICENSES.txt')) -DestinationPath $archive -Force
$checksums=@($exe,$archive,(Join-Path $root 'dist/my-tube.workspace.json'))|ForEach-Object {((Get-FileHash -LiteralPath $_ -Algorithm SHA256).Hash.ToLowerInvariant())+'  '+[IO.Path]::GetFileName($_)}
$checksums|Set-Content -LiteralPath (Join-Path $root 'dist/SHA256SUMS.txt') -Encoding ascii
Write-Output "Packaged My Tube $version"
