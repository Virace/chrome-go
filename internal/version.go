package internal

import "fmt"

var (
	// 编译时通过 ldflags 注入
	Version   = "dev"     // 版本号 (从 tag 获取)
	Commit    = "unknown" // Git commit hash
	BuildTime = "unknown" // 构建时间
)

// VersionString 返回格式化的版本字符串
func VersionString() string {
	if len(Commit) > 7 {
		return fmt.Sprintf("%s (%s)", Version, Commit[:7])
	}
	return fmt.Sprintf("%s (%s)", Version, Commit)
}

// FullVersionString 返回完整版本信息
func FullVersionString() string {
	return fmt.Sprintf("ChromeGo %s\nCommit: %s\nBuild Time: %s", Version, Commit, BuildTime)
}
