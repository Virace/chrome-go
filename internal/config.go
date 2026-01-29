package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 程序配置
type Config struct {
	ChromePath        string `json:"chrome_path"`         // Chrome 主程序目录，默认 "App"
	Channel           string `json:"channel"`             // 更新通道: stable/beta/dev/canary
	Version           string `json:"version"`             // 当前已安装的 Chrome 版本
	ChromePlusVersion string `json:"chrome_plus_version"` // 当前已安装的 Chrome++ 版本
	Threads           int    `json:"threads"`             // 下载线程数，默认 16
	KeepVersions      int    `json:"keep_versions"`       // 保留旧版本数量，默认 3
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		ChromePath:        "App",
		Channel:           "stable",
		Version:           "",
		ChromePlusVersion: "",
		Threads:           16,
		KeepVersions:      3,
	}
}

// GetThreads 获取下载线程数（确保有效值）
func (c *Config) GetThreads() int {
	if c.Threads <= 0 {
		return 16
	}
	if c.Threads > 64 {
		return 64
	}
	return c.Threads
}

// GetKeepVersions 获取保留旧版本数量
func (c *Config) GetKeepVersions() int {
	if c.KeepVersions <= 0 {
		return 3
	}
	return c.KeepVersions
}

// ConfigPath 返回配置文件路径
func ConfigPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "config.json")
}

// LoadConfig 加载配置文件
func LoadConfig() (*Config, error) {
	configPath := ConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，创建默认配置
			cfg := DefaultConfig()
			if saveErr := cfg.Save(); saveErr != nil {
				return nil, saveErr
			}
			return cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save 保存配置到文件
func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0644)
}

// GetChromePath 获取 Chrome 可执行文件的绝对路径
func (c *Config) GetChromePath() string {
	exe, _ := os.Executable()
	baseDir := filepath.Dir(exe)
	return filepath.Join(baseDir, c.ChromePath, "chrome.exe")
}

// GetChromePlusDllPath 获取 Chrome++ version.dll 的绝对路径
func (c *Config) GetChromePlusDllPath() string {
	exe, _ := os.Executable()
	baseDir := filepath.Dir(exe)
	return filepath.Join(baseDir, c.ChromePath, "version.dll")
}

// GetChromePlusIniPath 获取 Chrome++ INI 配置文件的绝对路径
func (c *Config) GetChromePlusIniPath() string {
	exe, _ := os.Executable()
	baseDir := filepath.Dir(exe)
	return filepath.Join(baseDir, c.ChromePath, "chrome++.ini")
}

// GetAppDir 获取 App 目录的绝对路径
func (c *Config) GetAppDir() string {
	exe, _ := os.Executable()
	baseDir := filepath.Dir(exe)
	return filepath.Join(baseDir, c.ChromePath)
}
