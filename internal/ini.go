package internal

import (
	"bufio"
	"os"
	"strings"
)

// IniFile 表示一个 INI 文件
type IniFile struct {
	Sections map[string]map[string]string
	Order    []string // 保持 section 顺序
}

// ParseIni 解析 INI 文件
func ParseIni(path string) (*IniFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ini := &IniFile{
		Sections: make(map[string]map[string]string),
		Order:    []string{},
	}

	currentSection := ""
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		// Section 头
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			if _, exists := ini.Sections[currentSection]; !exists {
				ini.Sections[currentSection] = make(map[string]string)
				ini.Order = append(ini.Order, currentSection)
			}
			continue
		}

		// Key=Value
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			// 确保有默认 section
			if currentSection == "" {
				currentSection = ""
				if _, exists := ini.Sections[""]; !exists {
					ini.Sections[""] = make(map[string]string)
					ini.Order = append(ini.Order, "")
				}
			}

			ini.Sections[currentSection][key] = value
		}
	}

	return ini, scanner.Err()
}

// MergeIni 合并两个 INI 文件
// newIniPath: 新版本的 INI 文件
// localIniPath: 本地已有的 INI 文件
// 规则: 保留本地配置值，只添加新版本中新增的配置项
func MergeIni(newIniPath, localIniPath string) error {
	newIni, err := ParseIni(newIniPath)
	if err != nil {
		return err
	}

	localIni, err := ParseIni(localIniPath)
	if err != nil {
		return err
	}

	modified := false

	// 遍历新版本的所有 section 和 key
	for section, keys := range newIni.Sections {
		if _, exists := localIni.Sections[section]; !exists {
			// 本地没有这个 section，添加整个 section
			localIni.Sections[section] = keys
			localIni.Order = append(localIni.Order, section)
			modified = true
		} else {
			// section 存在，检查是否有新的 key
			for key, value := range keys {
				if _, exists := localIni.Sections[section][key]; !exists {
					// 本地没有这个 key，添加
					localIni.Sections[section][key] = value
					modified = true
				}
				// 如果 key 已存在，保留本地值
			}
		}
	}

	// 如果有修改，保存文件
	if modified {
		return SaveIni(localIni, localIniPath)
	}

	return nil
}

// SaveIni 保存 INI 文件
func SaveIni(ini *IniFile, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, section := range ini.Order {
		keys := ini.Sections[section]

		// 写入 section 头（除非是空 section）
		if section != "" {
			file.WriteString("[" + section + "]\n")
		}

		// 写入所有 key=value
		for key, value := range keys {
			file.WriteString(key + "=" + value + "\n")
		}

		file.WriteString("\n")
	}

	return nil
}

// HasNewKeys 检查新版本 INI 是否有新增的配置项
func HasNewKeys(newIniPath, localIniPath string) (bool, error) {
	newIni, err := ParseIni(newIniPath)
	if err != nil {
		return false, err
	}

	localIni, err := ParseIni(localIniPath)
	if err != nil {
		// 本地文件不存在，认为有新内容
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}

	for section, keys := range newIni.Sections {
		localSection, exists := localIni.Sections[section]
		if !exists {
			return true, nil
		}
		for key := range keys {
			if _, exists := localSection[key]; !exists {
				return true, nil
			}
		}
	}

	return false, nil
}
