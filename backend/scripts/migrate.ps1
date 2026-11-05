param(
    [ValidateSet('up', 'status')]
    [string]$Action = 'up'
)

if ([string]::IsNullOrWhiteSpace($env:DATABASE_URL)) {
    throw '请先设置 DATABASE_URL，例如本地 Compose 数据库连接串。'
}

# goose 把日志写入 stderr；合并到输出流再打印，避免 PowerShell 5.1 生成 NativeCommandError。
$output = & go run -tags='no_mysql no_sqlite3 no_ydb no_azuresql no_clickhouse no_libsql no_mssql no_vertica' `
    github.com/pressly/goose/v3/cmd/goose@v3.29.0 `
    -dir (Join-Path $PSScriptRoot '..\migrations') `
    postgres $env:DATABASE_URL $Action 2>&1
$code = $LASTEXITCODE
$output | ForEach-Object { Write-Host $_ }

if ($code -ne 0) {
    exit $code
}