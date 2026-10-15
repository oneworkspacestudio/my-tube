param([Parameter(Mandatory=$true)][string]$MyAccountsExe,[string]$TubeExe=(Join-Path $PSScriptRoot '../dist/MyTube.exe'))
$ErrorActionPreference='Stop'
$fixture=Join-Path ([IO.Path]::GetTempPath()) ('mytube-oidc-'+[Guid]::NewGuid().ToString('N'))
function FreePort { $listener=[Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback,0);$listener.Start();$port=$listener.LocalEndpoint.Port;$listener.Stop();return $port }
$accountsPort=FreePort;$mytubePort=FreePort
$issuer="http://localhost:$accountsPort";$origin="http://127.0.0.1:$mytubePort"
$password=[Guid]::NewGuid().ToString('N')+[Guid]::NewGuid().ToString('N')
New-Item -ItemType Directory -Path (Join-Path $fixture 'accounts/secrets'),(Join-Path $fixture 'mytube') -Force | Out-Null
$config=@{version=1;issuer=$issuer;port=$accountsPort;adminEmail='admin@example.com';adminName='Tube Fixture';tunnelMode='local';sharedEmail=$false}
$config | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $fixture 'accounts/config.json') -Encoding utf8
Add-Type -AssemblyName System.Security.Cryptography.ProtectedData
$bootstrap=[Text.Encoding]::UTF8.GetBytes((@{name='Tube Fixture';email='admin@example.com';password=$password}|ConvertTo-Json))
$protected=[Security.Cryptography.ProtectedData]::Protect($bootstrap,$null,[Security.Cryptography.DataProtectionScope]::LocalMachine)
[IO.File]::WriteAllBytes((Join-Path $fixture 'accounts/secrets/bootstrap.dpapi'),$protected)
$process=$null
try {
 $process=Start-Process -FilePath (Resolve-Path -LiteralPath $MyAccountsExe).Path -ArgumentList @('-data',('"'+(Join-Path $fixture 'accounts')+'"'),'-no-browser') -WindowStyle Hidden -PassThru
 $ready=$false
 for($n=0;$n -lt 100;$n++){if($process.HasExited){throw 'Disposable MyAccounts exited'};try{if((Invoke-RestMethod "$issuer/healthz").status -eq 'ok'){$ready=$true;break}}catch{};Start-Sleep -Milliseconds 300}
 if(!$ready){throw 'Disposable MyAccounts did not start'}
 $env:MYTUBE_TEST_ISSUER=$issuer;$env:MYTUBE_TEST_ORIGIN=$origin;$env:MYTUBE_TEST_PASSWORD=$password;$env:MYTUBE_TEST_DATA=(Join-Path $fixture 'mytube');$env:MYTUBE_TEST_EXE=(Resolve-Path -LiteralPath $TubeExe).Path
 Push-Location (Join-Path $PSScriptRoot '../web')
 try {node scripts/test-myaccounts.mjs;if($LASTEXITCODE){throw 'Real MyAccounts integration failed'}}finally{Pop-Location}
 Write-Output 'PASS: real MyAccounts discovery, code/PKCE callback, signed identity, stable subject mapping and private-video isolation'
} finally {
 if($process -and !$process.HasExited){& $MyAccountsExe -data (Join-Path $fixture 'accounts') stop | Out-Null;if(!$process.WaitForExit(10000)){Stop-Process -Id $process.Id -Force}}
 foreach($name in @('MYTUBE_TEST_ISSUER','MYTUBE_TEST_ORIGIN','MYTUBE_TEST_PASSWORD','MYTUBE_TEST_DATA','MYTUBE_TEST_EXE')){Remove-Item ('Env:'+ $name) -ErrorAction SilentlyContinue}
 $password=$null;$bootstrap=$null;$protected=$null
}
