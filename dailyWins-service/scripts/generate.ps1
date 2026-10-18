$ErrorActionPreference = 'Stop'
Push-Location (Join-Path $PSScriptRoot '..')
try {
    $goBin = Join-Path (go env GOPATH) 'bin'
    $env:PATH = "$goBin;$env:PATH"
    & protoc --proto_path=./internal --proto_path=./third_party --go_out=paths=source_relative:./internal conf/conf.proto
    if ($LASTEXITCODE -ne 0) { throw '配置 proto 生成失败' }
    & protoc --proto_path=./api --proto_path=./third_party --go_out=. --go_opt=module=dailyWins-service --go-http_out=. --go-http_opt=module=dailyWins-service --go-grpc_out=. --go-grpc_opt=module=dailyWins-service user/v1/user.proto dailywins/v1/dailywins.proto
    if ($LASTEXITCODE -ne 0) { throw 'API proto 生成失败' }
    & wire ./cmd/dailyWins-service
    if ($LASTEXITCODE -ne 0) { throw 'Wire 生成失败' }
} finally { Pop-Location }
