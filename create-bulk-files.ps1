param(
    [Parameter(Mandatory=$true)]
    [string]$Source
)

# Baca setiap baris file path
Get-Content $Source | ForEach-Object {

    $root = Split-Path $Source -Parent
    $fullPath = Join-Path $root $_

    # Buat folder jika belum ada
    $dir = Split-Path $fullPath
    if (!(Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }

    # Buat file dengan encoding UTF-8
    Set-Content -Path $fullPath -Value "" -Encoding utf8 -Force
}
