# ChromeGo

> 🚀 一键部署便携版 Chrome，自动更新 Chrome 浏览器与 Chrome++ 增强组件

[![GitHub Release](https://img.shields.io/github/v/release/X-Item/chromego?style=flat-square&logo=github)](https://github.com/X-Item/chromego/releases/latest)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/X-Item/chromego?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows_x64-0078D6?style=flat-square&logo=windows)](https://www.microsoft.com/windows)
[![GitHub Downloads](https://img.shields.io/github/downloads/X-Item/chromego/total?style=flat-square&logo=github)](https://github.com/X-Item/chromego/releases)

## ✨ 功能特性

- **🔄 自动更新** - 后台检测 Chrome 和 Chrome++ 最新版本，一键更新
- **📦 便携部署** - 无需安装，解压即用，数据与程序分离
- **⚡ 多源加速** - 智能选择最优下载源，多线程并发下载
- **🔧 Chrome++ 集成** - 自动下载并配置 Chrome++ 增强组件
- **🎯 多通道支持** - 支持 Stable、Beta、Dev、Canary 等版本通道
- **📝 配置保留** - 更新时自动合并 Chrome++ 配置，不丢失个性化设置
- **🖥️ 高 DPI 支持** - 完美适配高分辨率屏幕

## 📥 快速开始

### 下载

从 [Releases](https://github.com/X-Item/chromego/releases/latest) 页面下载最新版本。

### 目录结构

```
ChromeGo/
├── ChromeGo.exe          # 主程序
├── config.json           # 配置文件（自动生成）
├── Chrome++配置.lnk      # Chrome++ 配置快捷方式
├── App/                  # Chrome 程序目录
│   ├── chrome.exe
│   ├── version.dll       # Chrome++ 核心
│   └── chrome++.ini      # Chrome++ 配置
├── Data/                 # 用户数据目录
└── Cache/                # 缓存目录
```

### 使用方法

1. 运行 `ChromeGo.exe`
2. 首次运行会自动下载并安装 Chrome 和 Chrome++
3. 之后每次运行会先启动浏览器，后台检测更新
4. 发现新版本时会弹窗询问是否更新

## ⚙️ 配置说明

配置文件 `config.json` 会在首次运行时自动创建：

```json
{
  "chrome_path": "App",
  "channel": "stable",
  "version": "",
  "chrome_plus_version": "",
  "threads": 16
}
```

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `chrome_path` | Chrome 程序目录 | `App` |
| `channel` | 更新通道 (stable/beta/dev/canary) | `stable` |
| `version` | 当前已安装 Chrome 版本（自动管理） | - |
| `chrome_plus_version` | 当前已安装 Chrome++ 版本（自动管理） | - |
| `threads` | 下载线程数 (1-64) | `16` |

## 🔨 从源码构建

### 前置要求

- Go 1.21 或更高版本
- [UPX](https://upx.github.io/) (可选，用于压缩可执行文件)
- [7-Zip](https://www.7-zip.org/) (可选，当内置解压器不支持某些格式时使用)

### 编译

```powershell
# 克隆仓库
git clone https://github.com/X-Item/chromego.git
cd chromego

# 使用构建脚本（推荐）
.\scripts\build.ps1

# 或手动构建
go build -ldflags="-H windowsgui -s -w" -o ChromeGo.exe ./cmd/chromego
```

构建脚本参数：
- `-NoUPX`: 禁用 UPX 压缩
- `-Debug`: 调试模式（保留符号信息）

## 📦 依赖项目

本项目集成了以下优秀的开源项目：

### Chrome 安装包来源

- **[Bush2021/chrome_installer](https://github.com/Bush2021/chrome_installer)** - 提供 Chrome 官方离线安装包的下载数据源

### Chrome++ 增强组件

- **[Bush2021/chrome_plus](https://github.com/Bush2021/chrome_plus)** - Chrome 便携化与标签页增强
  - 许可证：GPL-3.0 (v1.6.0+)
  - 原作者：[Shuax](https://github.com/shuax/)

Chrome++ 提供以下增强功能：
- 双击/右键关闭标签页
- 保留最后一个标签页
- 滚轮切换标签页
- 便携化设计（数据目录可配置）
- 老板键
- 更多功能请参考 [Chrome++ 配置文件](https://github.com/Bush2021/chrome_plus/blob/main/src/chrome++.ini)

## 📋 系统要求

- Windows 10/11 (x64)
- 网络连接

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

本项目采用 [MIT 许可证](LICENSE)。

**第三方组件许可证：**
- Chrome++ (v1.6.0+) 采用 [GPL-3.0](https://github.com/Bush2021/chrome_plus/blob/main/LICENSE) 许可证
- Chrome 浏览器为 Google 的商标和产品

## 💡 致谢

- [Bush2021](https://github.com/Bush2021/) - Chrome++ 维护者，Chrome 安装包数据源维护者
- [Shuax](https://github.com/shuax/) - Chrome++ 原作者
- 所有 [Chrome++ 贡献者](https://github.com/Bush2021/chrome_plus/graphs/contributors)

---

> ⚠️ **免责声明**：本项目仅提供 Chrome 浏览器的便携化部署工具，Chrome 浏览器本身为 Google 的产品。请遵守 Google Chrome 的使用条款。
