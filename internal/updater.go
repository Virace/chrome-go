package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// Chrome 官方数据源 (包含直接下载链接)
	chromeDataURL = "https://raw.githubusercontent.com/Bush2021/chrome_installer/main/data.json"
	// Chrome++ GitHub API
	chromePlusAPI = "https://api.github.com/repos/Bush2021/chrome_plus/releases/latest"
)

// ChromeData data.json 结构
type ChromeData map[string]ChromeChannel

// ChromeChannel 单个通道的数据
type ChromeChannel struct {
	Version string   `json:"version"`
	Size    int64    `json:"size"`
	SHA1    string   `json:"sha1"`
	SHA256  string   `json:"sha256"`
	URLs    []string `json:"urls"`
}

// GitHubRelease GitHub Release API 响应结构
type GitHubRelease struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset Release 资源
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// VersionInfo 版本信息
type VersionInfo struct {
	ChromeVersion     string   // Chrome 版本
	ChromeURLs        []string // Chrome 安装包下载地址列表（多源）
	ChromePlusVersion string   // Chrome++ 版本
	ChromePlusURL     string   // Chrome++ 下载地址
}

// GetLatestVersion 获取最新版本信息
func GetLatestVersion(channel string) (*VersionInfo, error) {
	// 从 data.json 获取 Chrome 信息
	chromeData, err := fetchChromeData()
	if err != nil {
		return nil, fmt.Errorf("获取 Chrome 版本失败: %w", err)
	}

	// 根据通道选择对应的 key
	key := getChromeDataKey(channel)
	channelData, ok := chromeData[key]
	if !ok {
		return nil, fmt.Errorf("未找到通道 %s 的数据", channel)
	}

	// 排序 URL：优先 dl.google.com 和 www.google.com
	chromeURLs := sortURLsByPriority(channelData.URLs)

	// 获取 Chrome++ 信息
	plusRelease, err := fetchRelease(chromePlusAPI)
	if err != nil {
		return nil, fmt.Errorf("获取 Chrome++ 版本失败: %w", err)
	}

	// 查找 chrome_plus 压缩包
	plusURL := ""
	for _, asset := range plusRelease.Assets {
		if strings.HasSuffix(asset.Name, ".7z") {
			plusURL = asset.BrowserDownloadURL
			break
		}
	}

	return &VersionInfo{
		ChromeVersion:     channelData.Version,
		ChromeURLs:        chromeURLs,
		ChromePlusVersion: plusRelease.TagName,
		ChromePlusURL:     plusURL,
	}, nil
}

// getChromeDataKey 根据通道返回 data.json 中的 key
func getChromeDataKey(channel string) string {
	switch strings.ToLower(channel) {
	case "beta":
		return "win_beta_x64"
	case "dev":
		return "win_dev_x64"
	case "canary":
		return "win_canary_x64"
	default:
		return "win_stable_x64"
	}
}

// fetchChromeData 获取 Chrome 数据
func fetchChromeData() (ChromeData, error) {
	resp, err := http.Get(chromeDataURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var data ChromeData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}

// sortURLsByPriority 按优先级排序 URL
// 优先级: dl.google.com > www.google.com > 其他
func sortURLsByPriority(urls []string) []string {
	var priority1 []string // dl.google.com
	var priority2 []string // www.google.com
	var priority3 []string // 其他

	for _, url := range urls {
		if !strings.HasPrefix(url, "https://") {
			continue // 只使用 HTTPS
		}
		if strings.Contains(url, "dl.google.com") {
			priority1 = append(priority1, url)
		} else if strings.Contains(url, "www.google.com") {
			priority2 = append(priority2, url)
		} else {
			priority3 = append(priority3, url)
		}
	}

	result := make([]string, 0, len(urls))
	result = append(result, priority1...)
	result = append(result, priority2...)
	result = append(result, priority3...)
	return result
}

// fetchRelease 调用 GitHub API 获取 Release 信息
func fetchRelease(url string) (*GitHubRelease, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "ChromeGo-Updater")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// DownloadChromeWithProgress 多源多线程下载 Chrome 并显示进度
func DownloadChromeWithProgress(urls []string, destPath string, threads int) error {
	return MultiSourceDownload(urls, destPath, threads, func(downloaded, total int64) {
		if total > 0 {
			percent := float64(downloaded) / float64(total) * 100
			fmt.Printf("\r正在下载 Chrome: %s / %s (%.1f%%)    ", FormatBytes(downloaded), FormatBytes(total), percent)
		}
	})
}

// DownloadFileWithProgress 下载文件并显示进度（单源）
func DownloadFileWithProgress(url, destPath, name string, threads int) error {
	return MultiThreadDownload(url, destPath, threads, func(downloaded, total int64) {
		if total > 0 {
			percent := float64(downloaded) / float64(total) * 100
			fmt.Printf("\r正在下载 %s: %s / %s (%.1f%%)    ", name, FormatBytes(downloaded), FormatBytes(total), percent)
		}
	})
}

// ExtractChrome 解压 Chrome 安装包
// 安装包结构: Chrome-bin\chrome.exe -> 需要移动到 App\chrome.exe
func ExtractChrome(archivePath, destDir string) error {
	tempDir := destDir + "_temp"

	// 使用内置 7z 解压
	if err := Extract7z(archivePath, tempDir); err != nil {
		os.RemoveAll(tempDir)
		return err
	}

	// 移动 Chrome-bin 内容到目标目录
	chromeBinDir := filepath.Join(tempDir, "Chrome-bin")
	if _, err := os.Stat(chromeBinDir); os.IsNotExist(err) {
		// 如果没有 Chrome-bin 目录，可能直接就是内容
		chromeBinDir = tempDir
	}

	// 确保目标目录存在
	if err := os.MkdirAll(destDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		return err
	}

	// 复制文件
	err := copyDir(chromeBinDir, destDir)
	os.RemoveAll(tempDir) // 确保清理临时目录
	return err
}

// ExtractChromePlus 解压 Chrome++ 增强包
// 包结构: x64\App\version.dll, x64\App\chrome++.ini
func ExtractChromePlus(archivePath, destDir string) error {
	tempDir := destDir + "_plus_temp"

	// 使用内置 7z 解压
	if err := Extract7z(archivePath, tempDir); err != nil {
		os.RemoveAll(tempDir)
		return err
	}

	// 源文件路径
	srcDir := filepath.Join(tempDir, "x64", "App")
	versionDll := filepath.Join(srcDir, "version.dll")
	chromePlusIni := filepath.Join(srcDir, "chrome++.ini")

	// 目标路径
	destVersionDll := filepath.Join(destDir, "version.dll")
	destIni := filepath.Join(destDir, "chrome++.ini")

	// 复制 version.dll
	if err := copyFile(versionDll, destVersionDll); err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("复制 version.dll 失败: %w", err)
	}

	// 处理 chrome++.ini (合并或创建)
	if _, err := os.Stat(destIni); os.IsNotExist(err) {
		// 本地不存在，直接复制并设置默认路径
		if err := copyFile(chromePlusIni, destIni); err != nil {
			os.RemoveAll(tempDir)
			return fmt.Errorf("复制 chrome++.ini 失败: %w", err)
		}
		// 设置默认的数据目录和缓存目录
		os.RemoveAll(tempDir)
		return updateIniPaths(destIni)
	}

	// 本地存在，执行合并
	err := MergeIni(chromePlusIni, destIni)
	os.RemoveAll(tempDir) // 确保清理临时目录
	return err
}

// updateIniPaths 更新 INI 文件中的路径配置
func updateIniPaths(iniPath string) error {
	content, err := os.ReadFile(iniPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var result []string

	dataSet := false
	cacheSet := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "data_dir=") {
			result = append(result, "data_dir=%app%\\..\\Data")
			dataSet = true
		} else if strings.HasPrefix(trimmed, "cache_dir=") {
			result = append(result, "cache_dir=%app%\\..\\Cache")
			cacheSet = true
		} else {
			result = append(result, line)
		}
	}

	// 如果没有这些配置项，添加到末尾
	if !dataSet {
		result = append(result, "data_dir=%app%\\..\\Data")
	}
	if !cacheSet {
		result = append(result, "cache_dir=%app%\\..\\Cache")
	}

	return os.WriteFile(iniPath, []byte(strings.Join(result, "\n")), 0644)
}

// copyDir 递归复制目录
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// copyFile 复制单个文件
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// CompareVersion 比较版本号，返回 true 如果 remote > local
func CompareVersion(local, remote string) bool {
	// 提取版本号数字部分
	re := regexp.MustCompile(`[\d.]+`)
	localVer := re.FindString(local)
	remoteVer := re.FindString(remote)

	localParts := strings.Split(localVer, ".")
	remoteParts := strings.Split(remoteVer, ".")

	for i := 0; i < len(remoteParts) && i < len(localParts); i++ {
		var l, r int
		fmt.Sscanf(localParts[i], "%d", &l)
		fmt.Sscanf(remoteParts[i], "%d", &r)
		if r > l {
			return true
		}
		if r < l {
			return false
		}
	}

	return len(remoteParts) > len(localParts)
}

// CreateShortcut 创建快捷方式
func CreateShortcut(targetPath, shortcutPath, description string) error {
	// 使用 PowerShell 创建 .lnk 快捷方式
	script := fmt.Sprintf(`
$WshShell = New-Object -ComObject WScript.Shell
$Shortcut = $WshShell.CreateShortcut('%s')
$Shortcut.TargetPath = '%s'
$Shortcut.Description = '%s'
$Shortcut.Save()
`, shortcutPath, targetPath, description)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	return cmd.Run()
}
