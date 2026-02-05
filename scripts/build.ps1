# ChromeGo 构建脚本
# 支持版本注入、图标嵌入和 UPX 压缩

param(
    [switch]$NoUPX,        # 禁用 UPX 压缩
    [switch]$Debug,        # 调试模式（保留符号信息）
    [switch]$CI            # CI 模式（GitHub Actions）
)

$ErrorActionPreference = "Stop"

# 确定项目根目录
if ($CI) {
    # CI 环境下，假设在项目根目录执行
    $projectRoot = Get-Location
} else {
    # 本地环境，脚本在 scripts 目录下
    $projectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
    if ($PSScriptRoot -like "*scripts*") {
        $projectRoot = Split-Path -Parent $PSScriptRoot
    }
}
Push-Location $projectRoot

try {
    # 清理并创建 dist 目录（仅本地模式）
    if (-not $CI) {
        $distDir = Join-Path $projectRoot "dist"
        if (Test-Path $distDir) {
            Remove-Item -Recurse -Force $distDir
        }
        New-Item -ItemType Directory -Path $distDir | Out-Null
    }

    # 获取版本信息
    $commit = git rev-parse --short HEAD 2>$null
    if (-not $commit) { $commit = "unknown" }
    
    $buildTime = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $version = "dev"
    $versionMajor = 0
    $versionMinor = 0
    $versionPatch = 0
    $versionBuild = 0

    # 如果有 tag，使用 tag 作为版本
    $tag = git describe --tags --exact-match 2>$null
    if ($tag) { 
        $version = $tag
        # 解析语义化版本号 (v0.1.0 格式)
        if ($tag -match '^v?(\d+)\.(\d+)\.(\d+)') {
            $versionMajor = [int]$Matches[1]
            $versionMinor = [int]$Matches[2]
            $versionPatch = [int]$Matches[3]
        }
    }

    # 格式化用于显示的版本字符串
    $displayVersion = "$versionMajor.$versionMinor.$versionPatch.$versionBuild"

    Write-Host "正在编译 ChromeGo..." -ForegroundColor Cyan
    Write-Host "  版本: $version" -ForegroundColor Gray
    Write-Host "  Commit: $commit" -ForegroundColor Gray
    Write-Host "  构建时间: $buildTime" -ForegroundColor Gray
    if ($CI) {
        Write-Host "  模式: CI (GitHub Actions)" -ForegroundColor Gray
    }

    # 检查并安装 goversioninfo
    $goversioninfo = Get-Command goversioninfo -ErrorAction SilentlyContinue
    if (-not $goversioninfo) {
        Write-Host "正在安装 goversioninfo..." -ForegroundColor Yellow
        go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
        if ($LASTEXITCODE -ne 0) {
            Write-Host "安装 goversioninfo 失败" -ForegroundColor Red
            exit 1
        }
    }

    # 更新 versioninfo.json 中的版本号
    $versionInfoPath = Join-Path $projectRoot "cmd\chromego\versioninfo.json"
    $versionInfo = Get-Content $versionInfoPath -Encoding UTF8 | ConvertFrom-Json
    
    # 更新文件版本
    $versionInfo.FixedFileInfo.FileVersion.Major = $versionMajor
    $versionInfo.FixedFileInfo.FileVersion.Minor = $versionMinor
    $versionInfo.FixedFileInfo.FileVersion.Patch = $versionPatch
    $versionInfo.FixedFileInfo.FileVersion.Build = $versionBuild
    
    # 更新产品版本
    $versionInfo.FixedFileInfo.ProductVersion.Major = $versionMajor
    $versionInfo.FixedFileInfo.ProductVersion.Minor = $versionMinor
    $versionInfo.FixedFileInfo.ProductVersion.Patch = $versionPatch
    $versionInfo.FixedFileInfo.ProductVersion.Build = $versionBuild
    
    # 更新字符串版本
    $versionInfo.StringFileInfo.FileVersion = $displayVersion
    $versionInfo.StringFileInfo.ProductVersion = $displayVersion
    
    # 保存更新后的 versioninfo.json
    $versionInfo | ConvertTo-Json -Depth 10 | Set-Content $versionInfoPath -Encoding UTF8
    
    # 生成资源文件
    Write-Host "正在生成资源文件..." -ForegroundColor Cyan
    Push-Location (Join-Path $projectRoot "cmd\chromego")
    try {
        & goversioninfo -64
        if ($LASTEXITCODE -ne 0) {
            Write-Host "生成资源文件失败" -ForegroundColor Red
            exit 1
        }
        Write-Host "资源文件生成成功" -ForegroundColor Green
    } finally {
        Pop-Location
    }

    # 构建 ldflags
    $ldflags = "-H=windowsgui"
    if (-not $Debug) {
        $ldflags += " -s -w"
    }
    $ldflags += " -X 'github.com/Virace/chrome-go/internal.Version=$version'"
    $ldflags += " -X 'github.com/Virace/chrome-go/internal.Commit=$commit'"
    $ldflags += " -X 'github.com/Virace/chrome-go/internal.BuildTime=$buildTime'"

    # 确定输出路径
    if ($CI) {
        $outputPath = Join-Path $projectRoot "ChromeGo.exe"
    } else {
        $outputPath = Join-Path $distDir "ChromeGo.exe"
    }

    # 编译
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

    # 清理生成的资源文件（仅本地模式，CI 模式下保留以确保构建正确）
    if (-not $CI) {
        $sysoFile = Join-Path $projectRoot "cmd\chromego\resource_windows.syso"
        if (Test-Path $sysoFile) {
            Remove-Item $sysoFile -Force
        }
    }

    Write-Host "`n构建完成!" -ForegroundColor Green
    Write-Host "输出: $outputPath" -ForegroundColor Cyan

} finally {
    Pop-Location
}

