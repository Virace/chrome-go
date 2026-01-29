package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Run 主应用入口
func Run() {
	// 默认隐藏控制台窗口
	HideConsole()

	// 加载配置
	cfg, err := LoadConfig()
	if err != nil {
		ShowError("加载配置失败: " + err.Error())
		return
	}

	// 获取基础路径
	exe, _ := os.Executable()
	baseDir := filepath.Dir(exe)

	// 检查 Chrome 和 Chrome++ 是否已安装
	chromePath := cfg.GetChromePath()
	chromePlusDllPath := cfg.GetChromePlusDllPath()
	chromePlusIniPath := cfg.GetChromePlusIniPath()

	chromeExists := fileExists(chromePath)
	chromePlusExists := fileExists(chromePlusDllPath)

	// 启动 Chrome（如果存在）
	if chromeExists {
		startChrome(chromePath)
	}

	// 后台检测更新
	latestVersion, err := GetLatestVersion(cfg.Channel)
	if err != nil {
		if !chromeExists {
			// Chrome 不存在且无法获取版本，显示错误
			ShowError("无法获取更新信息: " + err.Error())
		}
		return
	}

	// 判断 Chrome 是否需要更新
	needChromeUpdate := false
	if !chromeExists {
		needChromeUpdate = true
	} else if cfg.Version == "" {
		needChromeUpdate = true
	} else if CompareVersion(cfg.Version, latestVersion.ChromeVersion) {
		needChromeUpdate = true
	}

	// 判断 Chrome++ 是否需要更新
	needChromePlusUpdate := false
	if !chromePlusExists {
		needChromePlusUpdate = true
	} else if cfg.ChromePlusVersion == "" {
		needChromePlusUpdate = true
	} else if cfg.ChromePlusVersion != latestVersion.ChromePlusVersion {
		needChromePlusUpdate = true
	}

	// 都不需要更新，静默退出
	if !needChromeUpdate && !needChromePlusUpdate {
		return
	}

	// 构建提示消息
	var message string
	if !chromeExists {
		message = fmt.Sprintf("未检测到 Chrome，是否下载安装？\n\n"+
			"Chrome 版本: %s\n"+
			"Chrome++ 版本: %s",
			latestVersion.ChromeVersion, latestVersion.ChromePlusVersion)
	} else {
		var updates []string
		if needChromeUpdate {
			updates = append(updates, fmt.Sprintf("Chrome: %s → %s", cfg.Version, latestVersion.ChromeVersion))
		}
		if needChromePlusUpdate {
			oldVer := cfg.ChromePlusVersion
			if oldVer == "" {
				oldVer = "未安装"
			}
			updates = append(updates, fmt.Sprintf("Chrome++: %s → %s", oldVer, latestVersion.ChromePlusVersion))
		}

		message = "发现以下更新:\n\n"
		for _, u := range updates {
			message += "• " + u + "\n"
		}
		message += "\n请手动关闭浏览器后点击\"是\"开始更新"
	}

	if !ShowConfirm("ChromeGo 更新", message) {
		return
	}

	// 用户确认更新，显示控制台窗口
	consoleHandle := ShowConsole()
	if consoleHandle != nil && consoleHandle.GetHandle() != 0 {
		os.Stdout = os.NewFile(consoleHandle.GetHandle(), "stdout")
		os.Stderr = os.NewFile(consoleHandle.GetHandle(), "stderr")
	}

	// 执行更新
	if err := doUpdate(cfg, latestVersion, needChromeUpdate, needChromePlusUpdate); err != nil {
		ShowError("更新失败: " + err.Error())
		return
	}

	// 更新版本号并保存配置
	if needChromeUpdate {
		cfg.Version = latestVersion.ChromeVersion
	}
	if needChromePlusUpdate {
		cfg.ChromePlusVersion = latestVersion.ChromePlusVersion
	}
	if err := cfg.Save(); err != nil {
		ShowError("保存配置失败: " + err.Error())
		return
	}

	// 创建 Chrome++ 配置快捷方式
	if needChromePlusUpdate && fileExists(chromePlusIniPath) {
		shortcutPath := filepath.Join(baseDir, "Chrome++配置.lnk")
		if !fileExists(shortcutPath) {
			CreateShortcut(chromePlusIniPath, shortcutPath, "Chrome++ 配置文件")
		}
	}

	// 显示完成信息
	var completeMsg string
	if needChromeUpdate && needChromePlusUpdate {
		completeMsg = fmt.Sprintf("Chrome 已更新到 %s\nChrome++ 已更新到 %s",
			latestVersion.ChromeVersion, latestVersion.ChromePlusVersion)
	} else if needChromeUpdate {
		completeMsg = fmt.Sprintf("Chrome 已更新到 %s", latestVersion.ChromeVersion)
	} else {
		completeMsg = fmt.Sprintf("Chrome++ 已更新到 %s", latestVersion.ChromePlusVersion)
	}
	ShowInfo("更新完成", completeMsg)

	// 启动 Chrome
	startChrome(chromePath)
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// doUpdate 执行更新流程
func doUpdate(cfg *Config, version *VersionInfo, updateChrome, updateChromePlus bool) error {
	exe, _ := os.Executable()
	baseDir := filepath.Dir(exe)
	appDir := filepath.Join(baseDir, cfg.ChromePath)
	tempDir := filepath.Join(baseDir, "temp")
	threads := cfg.GetThreads()

	// 确保临时目录存在
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// 确保 Data 和 Cache 目录存在
	os.MkdirAll(filepath.Join(baseDir, "Data"), 0755)
	os.MkdirAll(filepath.Join(baseDir, "Cache"), 0755)

	// 更新 Chrome
	if updateChrome {
		chromePkg := filepath.Join(tempDir, "chrome_installer.exe")
		fmt.Printf("使用 %d 线程下载...\n", threads)
		if err := DownloadChromeWithProgress(version.ChromeURLs, chromePkg, threads); err != nil {
			return fmt.Errorf("下载 Chrome 失败: %w", err)
		}
		fmt.Println()

		fmt.Println("正在解压 Chrome...")
		if err := ExtractChrome(chromePkg, appDir); err != nil {
			return fmt.Errorf("解压 Chrome 失败: %w", err)
		}
		fmt.Println("Chrome 解压完成")
	}

	// 更新 Chrome++
	if updateChromePlus && version.ChromePlusURL != "" {
		plusPkg := filepath.Join(tempDir, "chrome_plus.7z")
		if err := DownloadFileWithProgress(version.ChromePlusURL, plusPkg, "Chrome++", threads); err != nil {
			return fmt.Errorf("下载 Chrome++ 失败: %w", err)
		}
		fmt.Println()

		fmt.Println("正在解压 Chrome++...")
		if err := ExtractChromePlus(plusPkg, appDir); err != nil {
			return fmt.Errorf("解压 Chrome++ 失败: %w", err)
		}
		fmt.Println("Chrome++ 解压完成")
	}

	return nil
}

// startChrome 启动 Chrome 浏览器
func startChrome(chromePath string) {
	cmd := exec.Command(chromePath)
	cmd.Start()
}
