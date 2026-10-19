param([string]$Go='go',[switch]$BrowserTests,[switch]$ReuseDependencies)
$ErrorActionPreference='Stop'
$root=[IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
Push-Location $root
try {
 if(!$ReuseDependencies){npm --prefix web ci;if($LASTEXITCODE){throw 'npm ci failed'}}
 npm --prefix web run build;if($LASTEXITCODE){throw 'Frontend build failed'}
 & $Go test -buildvcs=false -timeout 120s ./...;if($LASTEXITCODE){throw 'Go tests failed'}
 & $Go vet -buildvcs=false ./...;if($LASTEXITCODE){throw 'Go vet failed'}
 $env:CGO_ENABLED='0'
 & $Go build -buildvcs=false -trimpath -ldflags '-s -w' -o dist/MyTube.exe ./cmd/my-tube;if($LASTEXITCODE){throw 'Windows build failed'}
 if($BrowserTests){Push-Location web;try{npm test;if($LASTEXITCODE){throw 'Browser tests failed'};node scripts/test-portable.mjs;if($LASTEXITCODE){throw 'Portable tests failed'}}finally{Pop-Location}}
 node scripts/licenses.mjs $Go;if($LASTEXITCODE){throw 'License generation failed'}
 & (Join-Path $PSScriptRoot 'package.ps1')
}finally{Pop-Location}
