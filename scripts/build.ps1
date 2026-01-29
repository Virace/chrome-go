# ChromeGo 构建脚本
# 支持版本注入和 UPX 压缩

param(
    [switch]$NoUPX,        # 禁用 UPX 压缩
    [switch]$Debug         # 调试模式（保留符号信息）
)

$ErrorActionPreference = "Stop"

# 切换到项目根目录
$projectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
if ($PSScriptRoot -like "*scripts*") {
    $projectRoot = Split-Path -Parent $PSScriptRoot
}
Push-Location $projectRoot

try {
    # 清理并创建 dist 目录
    $distDir = Join-Path $projectRoot "dist"
    if (Test-Path $distDir) {
        Remove-Item -Recurse -Force $distDir
    }
    New-Item -ItemType Directory -Path $distDir | Out-Null

    # 获取版本信息
    $commit = git rev-parse --short HEAD 2>$null
    if (-not $commit) { $commit = "unknown" }
    
    $buildTime = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $version = "dev"

    # 如果有 tag，使用 tag 作为版本
    $tag = git describe --tags --exact-match 2>$null
    if ($tag) { 
        $version = $tag 
    }

    Write-Host "正在编译 ChromeGo..." -ForegroundColor Cyan
    Write-Host "  版本: $version" -ForegroundColor Gray
    Write-Host "  Commit: $commit" -ForegroundColor Gray
    Write-Host "  构建时间: $buildTime" -ForegroundColor Gray

    # 构建 ldflags
    $ldflags = "-H=windowsgui"
    if (-not $Debug) {
        $ldflags += " -s -w"
    }
    $ldflags += " -X 'github.com/Virace/chrome-go/internal.Version=$version'"
    $ldflags += " -X 'github.com/Virace/chrome-go/internal.Commit=$commit'"
    $ldflags += " -X 'github.com/Virace/chrome-go/internal.BuildTime=$buildTime'"

    # 编译
    $outputPath = Join-Path $distDir "ChromeGo.exe"
    go build -ldflags $ldflags -o $outputPath ./cmd/chromego

    if ($LASTEXITCODE -ne 0) {
        Write-Host "编译失败" -ForegroundColor Red
        exit 1
    }

    $fileInfo = Get-Item $outputPath
    $sizeMB = [math]::Round($fileInfo.Length / 1MB, 2)
    Write-Host "编译成功: $outputPath ($sizeMB MB)" -ForegroundColor Green

    # UPX 压缩
    if (-not $NoUPX) {
        $upx = Get-Command upx -ErrorAction SilentlyContinue
        if ($upx) {
            Write-Host "正在使用 UPX 压缩..." -ForegroundColor Cyan
            & upx --best --lzma $outputPath 2>&1 | Out-Null
            
            if ($LASTEXITCODE -eq 0) {
                $newInfo = Get-Item $outputPath
                $newSizeMB = [math]::Round($newInfo.Length / 1MB, 2)
                $ratio = [math]::Round(($newInfo.Length / $fileInfo.Length) * 100, 1)
                Write-Host "UPX 压缩完成: $newSizeMB MB ($ratio%)" -ForegroundColor Green
            } else {
                Write-Host "UPX 压缩失败，使用未压缩版本" -ForegroundColor Yellow
            }
        } else {
            Write-Host "未找到 UPX，跳过压缩。安装: choco install upx 或 scoop install upx" -ForegroundColor Yellow
        }
    }

    Write-Host "`n构建完成!" -ForegroundColor Green
    Write-Host "输出: $outputPath" -ForegroundColor Cyan

} finally {
    Pop-Location
}
