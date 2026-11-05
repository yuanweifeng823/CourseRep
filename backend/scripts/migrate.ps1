param(
    [ValidateSet('up', 'status')]
    [string]$Action = 'up'
)

$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($env:DATABASE_URL)) {
    throw '请先设置 DATABASE_URL，例如本地 Compose 数据库连接串。'
}

go run -tags='no_mysql no_sqlite3 no_ydb no_azuresql no_clickhouse no_libsql no_mssql no_vertica' `
    github.com/pressly/goose/v3/cmd/goose@v3.29.0 `
    -dir (Join-Path $PSScriptRoot '..\migrations') `
    postgres $env:DATABASE_URL $Action

if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
