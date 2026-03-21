$ErrorActionPreference = "Stop"

$baseUrl = "http://localhost:8080"
$stamp = Get-Date -Format "yyyyMMddHHmmss"
$username = "alice_$stamp"
$email = "alice_$stamp@example.com"
$password = "StrongPass123!"

function Assert-Status {
    param(
        [string]$Name,
        [int]$Expected,
        [scriptblock]$Action
    )

    try {
        & $Action | Out-Null
        $actual = 200
    }
    catch {
        if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
            $actual = [int]$_.Exception.Response.StatusCode
        }
        else {
            throw
        }
    }

    if ($actual -ne $Expected) {
        throw "$Name failed. Expected status $Expected but got $actual"
    }

    Write-Host "[PASS] $Name -> $actual"
}

Write-Host "Running auth smoke tests against $baseUrl"

$health = Invoke-RestMethod -Method Get -Uri "$baseUrl/health"
if ($health.status -ne "ok") {
    throw "Health check failed"
}
Write-Host "[PASS] health -> ok"

$registerBody = @{ username = $username; email = $email; password = $password } | ConvertTo-Json
$registerRes = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/v1/auth/register" -ContentType "application/json" -Body $registerBody
if (-not $registerRes.token) {
    throw "Register did not return token"
}
Write-Host "[PASS] register -> token returned"

$loginBody = @{ email = $email; password = $password } | ConvertTo-Json
$loginRes = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/v1/auth/login" -ContentType "application/json" -Body $loginBody
if (-not $loginRes.token -or -not $loginRes.refresh_token) {
    throw "Login did not return token"
}
Write-Host "[PASS] login -> token returned"

$refreshBody = @{ refresh_token = $loginRes.refresh_token } | ConvertTo-Json
$refreshRes = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/v1/auth/refresh" -ContentType "application/json" -Body $refreshBody
if (-not $refreshRes.access_token -or -not $refreshRes.refresh_token) {
    throw "Refresh did not return updated token pair"
}
Write-Host "[PASS] refresh -> rotated token pair returned"

Assert-Status -Name "login wrong password" -Expected 401 -Action {
    $badBody = @{ email = $email; password = "WrongPass123!" } | ConvertTo-Json
    Invoke-WebRequest -Method Post -Uri "$baseUrl/api/v1/auth/login" -ContentType "application/json" -Body $badBody -UseBasicParsing
}

Assert-Status -Name "register duplicate email" -Expected 409 -Action {
    $dupBody = @{ username = "dup_$stamp"; email = $email; password = "AnotherStrong123!" } | ConvertTo-Json
    Invoke-WebRequest -Method Post -Uri "$baseUrl/api/v1/auth/register" -ContentType "application/json" -Body $dupBody -UseBasicParsing
}

Assert-Status -Name "me without token" -Expected 401 -Action {
    Invoke-WebRequest -Method Get -Uri "$baseUrl/api/v1/auth/me" -UseBasicParsing
}

$meRes = Invoke-RestMethod -Method Get -Uri "$baseUrl/api/v1/auth/me" -Headers @{ Authorization = "Bearer $($refreshRes.access_token)" }
if ($meRes.email -ne $email) {
    throw "me endpoint returned unexpected email"
}
Write-Host "[PASS] me with token -> user claims returned"

$logoutBody = @{ refresh_token = $refreshRes.refresh_token } | ConvertTo-Json
Invoke-WebRequest -Method Post -Uri "$baseUrl/api/v1/auth/logout" -ContentType "application/json" -Body $logoutBody -UseBasicParsing | Out-Null
Write-Host "[PASS] logout -> refresh token revoked"

Assert-Status -Name "refresh after logout" -Expected 401 -Action {
    Invoke-WebRequest -Method Post -Uri "$baseUrl/api/v1/auth/refresh" -ContentType "application/json" -Body $logoutBody -UseBasicParsing
}

Write-Host "All auth smoke tests passed."
