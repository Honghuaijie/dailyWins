param([string]$Config = './configs', [string]$Environment = 'local')
$ErrorActionPreference = 'Stop'
Push-Location (Join-Path $PSScriptRoot '..')
try {
    $env:INIT_USERNAME = Read-Host '个人账号名'
    $secret = Read-Host '密码（至少 6 个字符，最多 72 字节）' -AsSecureString
    $pointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secret)
    try {
        $env:INIT_PASSWORD = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($pointer)
        & go run ./cmd/dailyWins-service -conf $Config -env $Environment -command init-account
        if ($LASTEXITCODE -ne 0) { throw '账号初始化未完成；不会覆盖已有账号' }
    } finally {
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pointer)
        Remove-Item Env:INIT_PASSWORD -ErrorAction SilentlyContinue
        Remove-Item Env:INIT_USERNAME -ErrorAction SilentlyContinue
    }
} finally { Pop-Location }
