package internal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"
)

// Extract7z 使用纯 Go 解压 7z 文件到指定目录
// 如果纯 Go 库失败，会尝试使用外部 7z 程序
func Extract7z(archivePath, destDir string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 首先尝试纯 Go 解压
	err := extract7zPureGo(archivePath, destDir)
	if err == nil {
		return nil
	}

	// 纯 Go 失败，提示并尝试外部 7z 程序
	fmt.Printf("内置解压器不支持此压缩格式，尝试使用外部 7-Zip...\n")

	extErr := extract7zExternal(archivePath, destDir)
	if extErr == nil {
		return nil
	}

	// 两种方式都失败
	if strings.Contains(extErr.Error(), "未找到 7z 程序") {
		return fmt.Errorf("解压失败：内置解压器不支持此格式，且未找到外部 7-Zip 程序。\n\n请安装 7-Zip: https://www.7-zip.org/")
	}

	return fmt.Errorf("解压失败: %v", extErr)
}

// extract7zPureGo 使用纯 Go 库解压
func extract7zPureGo(archivePath, destDir string) error {
	r, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("打开压缩包失败: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// 清理路径，防止路径遍历攻击
		name := filepath.Clean(f.Name)
		if strings.HasPrefix(name, "..") {
			continue
		}
		destPath := filepath.Join(destDir, name)

		// 处理目录
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
			continue
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("创建父目录失败: %w", err)
		}

		// 解压文件
		if err := extractFilePureGo(f, destPath); err != nil {
			return err
		}
	}

	return nil
}

// extractFilePureGo 解压单个文件
func extractFilePureGo(f *sevenzip.File, destPath string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("打开压缩文件失败: %w", err)
	}
	defer rc.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, rc); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// extract7zExternal 使用外部 7z 程序解压
func extract7zExternal(archivePath, destDir string) error {
	// 尝试多个可能的 7z 路径
	sevenZipPaths := []string{
		"7z",
		"7za",
		filepath.Join(os.Getenv("ProgramFiles"), "7-Zip", "7z.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "7-Zip", "7z.exe"),
	}

	var cmd *exec.Cmd
	for _, p := range sevenZipPaths {
		if _, err := exec.LookPath(p); err == nil {
			cmd = exec.Command(p, "x", archivePath, "-o"+destDir, "-y")
			break
		}
		// 检查绝对路径
		if _, err := os.Stat(p); err == nil {
			cmd = exec.Command(p, "x", archivePath, "-o"+destDir, "-y")
			break
		}
	}

	if cmd == nil {
		return fmt.Errorf("未找到 7z 程序")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("7z 解压失败: %v, 输出: %s", err, string(output))
	}

	return nil
}
