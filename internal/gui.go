package internal

import (
	"syscall"
	"unsafe"
)

var (
	user32      = syscall.NewLazyDLL("user32.dll")
	kernel32    = syscall.NewLazyDLL("kernel32.dll")
	shcore      = syscall.NewLazyDLL("shcore.dll")
	messageBoxW = user32.NewProc("MessageBoxW")

	getConsoleWindow   = kernel32.NewProc("GetConsoleWindow")
	showWindowProc     = user32.NewProc("ShowWindow")
	allocConsole       = kernel32.NewProc("AllocConsole")
	setStdHandle       = kernel32.NewProc("SetStdHandle")
	createFileW        = kernel32.NewProc("CreateFileW")
	setConsoleOutputCP = kernel32.NewProc("SetConsoleOutputCP")
	setConsoleCP       = kernel32.NewProc("SetConsoleCP")

	// 高 DPI 支持
	setProcessDpiAwareness = shcore.NewProc("SetProcessDpiAwareness")
)

const (
	MB_OK              = 0x00000000
	MB_OKCANCEL        = 0x00000001
	MB_YESNO           = 0x00000004
	MB_ICONINFORMATION = 0x00000040
	MB_ICONQUESTION    = 0x00000020
	MB_ICONWARNING     = 0x00000030

	IDOK     = 1
	IDCANCEL = 2
	IDYES    = 6
	IDNO     = 7

	SW_HIDE = 0
	SW_SHOW = 5

	STD_OUTPUT_HANDLE = ^uintptr(0) - 11 + 1 // -11
	STD_ERROR_HANDLE  = ^uintptr(0) - 12 + 1 // -12

	GENERIC_WRITE         = 0x40000000
	FILE_SHARE_WRITE      = 0x00000002
	OPEN_EXISTING         = 3
	FILE_ATTRIBUTE_NORMAL = 0x80

	CP_UTF8 = 65001

	// DPI Awareness
	PROCESS_DPI_UNAWARE           = 0
	PROCESS_SYSTEM_DPI_AWARE      = 1
	PROCESS_PER_MONITOR_DPI_AWARE = 2
)

func init() {
	// 设置高 DPI 感知，解决高分屏模糊问题
	EnableHighDPI()
}

// EnableHighDPI 启用高 DPI 支持
func EnableHighDPI() {
	// Windows 8.1+ 使用 SetProcessDpiAwareness
	if setProcessDpiAwareness.Find() == nil {
		setProcessDpiAwareness.Call(PROCESS_PER_MONITOR_DPI_AWARE)
	}
}

// HideConsole 隐藏控制台窗口
func HideConsole() {
	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd != 0 {
		showWindowProc.Call(hwnd, SW_HIDE)
	}
}

// ShowConsole 显示控制台窗口并重定向输出
func ShowConsole() *ConsoleHandle {
	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd == 0 {
		// 没有控制台窗口，分配一个新的
		allocConsole.Call()

		// 设置控制台代码页为 UTF-8
		setConsoleOutputCP.Call(CP_UTF8)
		setConsoleCP.Call(CP_UTF8)

		// 重新打开 stdout/stderr 到新控制台
		conout, _ := syscall.UTF16PtrFromString("CONOUT$")
		handle, _, _ := createFileW.Call(
			uintptr(unsafe.Pointer(conout)),
			GENERIC_WRITE,
			FILE_SHARE_WRITE,
			0,
			OPEN_EXISTING,
			FILE_ATTRIBUTE_NORMAL,
			0,
		)

		if handle != 0 && handle != ^uintptr(0) {
			setStdHandle.Call(STD_OUTPUT_HANDLE, handle)
			setStdHandle.Call(STD_ERROR_HANDLE, handle)
			return &ConsoleHandle{handle: handle}
		}
	} else {
		// 设置控制台代码页为 UTF-8
		setConsoleOutputCP.Call(CP_UTF8)
		setConsoleCP.Call(CP_UTF8)
		showWindowProc.Call(hwnd, SW_SHOW)
	}
	return nil
}

// ConsoleHandle 控制台句柄
type ConsoleHandle struct {
	handle uintptr
}

// GetHandle 获取句柄
func (c *ConsoleHandle) GetHandle() uintptr {
	if c == nil {
		return 0
	}
	return c.handle
}

// ShowConfirm 显示确认对话框
func ShowConfirm(title, message string) bool {
	ret, _, _ := messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(message))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))),
		MB_YESNO|MB_ICONQUESTION,
	)
	return ret == IDYES
}

// ShowInfo 显示信息对话框
func ShowInfo(title, message string) {
	messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(message))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))),
		MB_OK|MB_ICONINFORMATION,
	)
}

// ShowError 显示错误对话框
func ShowError(message string) {
	messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(message))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("错误"))),
		MB_OK|MB_ICONWARNING,
	)
}
