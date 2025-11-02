package utils

import (
	"fmt"
	"syscall"
	"unsafe"
	"os"

	"golang.org/x/sys/windows"
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procGetStdHandle               = kernel32.NewProc("GetStdHandle")
	procSetConsoleOutputCP         = kernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleCP               = kernel32.NewProc("SetConsoleCP")
)

const (
	ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	ENABLE_PROCESSED_OUTPUT           = 0x0001
	STD_OUTPUT_HANDLE                 = ^uintptr(10) + 1 // -11
	CP_UTF8                           = 65001
)

// InitConsole 初始化控制台（支持UTF-8和颜色）
func InitConsole() error {
	// 设置控制台代码页为 UTF-8
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setConsoleCP := kernel32.NewProc("SetConsoleCP")
	setConsoleOutputCP := kernel32.NewProc("SetConsoleOutputCP")
	
	setConsoleCP.Call(uintptr(65001))       // CP_UTF8
	setConsoleOutputCP.Call(uintptr(65001)) // CP_UTF8

	// 同时设置环境变量
	os.Setenv("PYTHONIOENCODING", "utf-8")
	os.Setenv("LANG", "en_US.UTF-8")

	// 设置UTF-8编码（原有代码）
	procSetConsoleOutputCP.Call(CP_UTF8)
	procSetConsoleCP.Call(CP_UTF8)

	// 获取标准输出句柄
	handle, _, _ := procGetStdHandle.Call(STD_OUTPUT_HANDLE)
	
	// 获取当前模式
	var mode uint32
	procGetConsoleMode.Call(handle, uintptr(unsafe.Pointer(&mode)))
	
	// 启用虚拟终端处理（支持ANSI颜色）
	mode |= ENABLE_VIRTUAL_TERMINAL_PROCESSING | ENABLE_PROCESSED_OUTPUT
	procSetConsoleMode.Call(handle, uintptr(mode))

	return nil
}

// SetConsoleTitle 设置控制台标题
func SetConsoleTitle(title string) {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleTitleW").Call(
		uintptr(unsafe.Pointer(titlePtr)),
	)
}

// ClearScreen 清屏
func ClearScreen() {
	cmd := windows.NewLazySystemDLL("kernel32.dll").NewProc("FillConsoleOutputCharacterW")
	var csbi windows.ConsoleScreenBufferInfo
	handle := windows.Handle(^uintptr(10) + 1)
	
	windows.GetConsoleScreenBufferInfo(handle, &csbi)
	
	var written uint32
	size := uint32(csbi.Size.X) * uint32(csbi.Size.Y)
	cmd.Call(
		uintptr(handle),
		uintptr(' '),
		uintptr(size),
		0,
		uintptr(unsafe.Pointer(&written)),
	)
	
	// 移动光标到左上角
	windows.SetConsoleCursorPosition(handle, windows.Coord{X: 0, Y: 0})
}

// GetConsoleWidth 获取控制台宽度
func GetConsoleWidth() int {
	var csbi windows.ConsoleScreenBufferInfo
	handle := windows.Handle(^uintptr(10) + 1)
	
	if err := windows.GetConsoleScreenBufferInfo(handle, &csbi); err != nil {
		return 80 // 默认宽度
	}
	
	return int(csbi.Size.X)
}

// ANSI颜色代码
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	
	// Miku主题配色
	MikuCyan    = "\033[38;2;57;197;187m"   // #39C5BB 初音未来青色
	MikuPink    = "\033[38;2;255;105;180m"  // #FF69B4 粉色
	MikuGreen   = "\033[38;2;0;255;127m"    // #00FF7F 春绿色
	MikuYellow  = "\033[38;2;255;215;0m"    // #FFD700 金色
	MikuRed     = "\033[38;2;255;107;107m"  // #FF6B6B 红色
	MikuPurple  = "\033[38;2;186;85;211m"   // #BA55D3 紫色
	MikuWhite   = "\033[38;2;255;255;255m"  // 白色
	MikuGray    = "\033[38;2;128;128;128m"  // 灰色
	
	// 背景色
	BgMikuCyan = "\033[48;2;57;197;187m"
	BgMikuPink = "\033[48;2;255;105;180m"
)

// Colorize 为文本添加颜色
func Colorize(text, color string) string {
	return color + text + Reset
}

// MikuBanner 显示Miku主题Banner
func MikuBanner() {
	banner := `
   ███╗   ███╗██╗██╗  ██╗██╗   ██╗    ████████╗██╗███╗   ██╗██╗   ██╗ ██╗ ██╗
   ████╗ ████║██║██║ ██╔╝██║   ██║    ╚══██╔══╝██║████╗  ██║╚██╗ ██╔╝███║███║
   ██╔████╔██║██║█████╔╝ ██║   ██║       ██║   ██║██╔██╗ ██║ ╚████╔╝ ╚██║╚██║
   ██║╚██╔╝██║██║██╔═██╗ ██║   ██║       ██║   ██║██║╚██╗██║  ╚██╔╝   ██║ ██║
   ██║ ╚═╝ ██║██║██║  ██╗╚██████╔╝       ██║   ██║██║ ╚████║   ██║    ██║ ██║
   ╚═╝     ╚═╝╚═╝╚═╝  ╚═╝ ╚═════╝        ╚═╝   ╚═╝╚═╝  ╚═══╝   ╚═╝    ╚═╝ ╚═╝
`
	fmt.Println(Colorize(banner, MikuCyan))
	fmt.Println(Colorize("                    Windows 11 精简镜像构建工具 - Miku Edition 🎀", MikuPink))
	fmt.Println(Colorize("                         Powered by Go | Made with ♥", MikuGray))
	fmt.Println()
}