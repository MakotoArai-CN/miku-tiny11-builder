package cli

import (
	"flag"
	"fmt"
	"strings"
	"tiny11-builder/internal/config"
)

// ParseArgsUnified 解析统一版本的命令行参数
func ParseArgsUnified(args []string) (*config.Config, string, string, error) {
	fs := flag.NewFlagSet("tiny11-builder", flag.ContinueOnError)

	iso := fs.String("iso", "", "ISO挂载的驱动器号 (例: E)")
	scratch := fs.String("scratch", "", "临时文件驱动器号 (例: D)")
	index := fs.Int("index", 0, "镜像索引 (0=自动选择)")
	output := fs.String("output", "", "输出ISO路径")
	mode := fs.String("mode", "", "构建模式: standard 或 core")
	theme := fs.String("theme", "default", "主题名称: default, miku 或自定义")
	verbose := fs.Bool("v", false, "详细日志")
	help := fs.Bool("h", false, "显示帮助")

	if err := fs.Parse(args); err != nil {
		return nil, "", "", err
	}

	if *help {
		PrintUsageUnified()
		return nil, "", "", fmt.Errorf("显示帮助")
	}

	cfg := config.NewConfig()
	cfg.Verbose = *verbose

	// 验证ISO驱动器
	if *iso != "" {
		*iso = strings.ToUpper(strings.TrimSuffix(*iso, ":"))
		if len(*iso) != 1 || (*iso)[0] < 'C' || (*iso)[0] > 'Z' {
			return nil, "", "", fmt.Errorf("无效的驱动器号: %s", *iso)
		}
		cfg.ISODrive = *iso + ":"
	} else {
		// 交互式输入
		fmt.Print("请输入Windows 11 ISO挂载的驱动器号: ")
		var drive string
		fmt.Scanln(&drive)
		drive = strings.ToUpper(strings.TrimSuffix(drive, ":"))
		if len(drive) != 1 || drive[0] < 'C' || drive[0] > 'Z' {
			return nil, "", "", fmt.Errorf("无效的驱动器号: %s", drive)
		}
		cfg.ISODrive = drive + ":"
	}

	// 设置临时目录
	if *scratch != "" {
		*scratch = strings.ToUpper(strings.TrimSuffix(*scratch, ":"))
		cfg.ScratchDrive = *scratch + ":"
	}

	cfg.ImageIndex = *index
	if *output != "" {
		cfg.OutputISO = *output
	}

	cfg.ThemeName = *theme

	// 验证模式参数
	buildMode := strings.ToLower(*mode)
	if buildMode != "" && buildMode != "standard" && buildMode != "core" {
		return nil, "", "", fmt.Errorf("无效的模式: %s (应为 standard 或 core)", *mode)
	}

	return cfg, buildMode, *theme, nil
}

// ParseArgs 保留兼容性（旧版）
func ParseArgs(args []string) (*config.Config, error) {
	cfg, _, _, err := ParseArgsUnified(args)
	return cfg, err
}

// PrintUsageUnified 打印统一版本使用说明
func PrintUsageUnified() {
	fmt.Print(`
Tiny11 Builder - Windows 11精简镜像构建工具 (统一版)

用法:
  tiny11builder.exe [选项]

选项:
  -iso <drive>      ISO挂载的驱动器号 (例: -iso E)
  -scratch <drive>  临时文件驱动器号 (例: -scratch D)
  -mode <mode>      构建模式: standard (标准版) 或 core (极限精简)
  -theme <name>     主题名称: default, miku 或自定义主题名
  -index <number>   镜像索引 (默认自动选择)
  -output <path>    输出ISO路径 (默认: ./tiny11.iso)
  -v                详细日志输出
  -h                显示此帮助

构建模式:
  standard          标准版 - 移除膨胀软件，保留可服务性
                    • 可安装更新和功能
                    • 适合日常使用
                    • 大小: 约5-6 GB

  core              Core版 - 极限精简，不可服务
                    • 移除WinSxS大部分内容
                    • 禁用更新和Defender
                    • 仅用于测试环境
                    • 大小: 约4-5 GB

主题:
  default           默认 - 保持Windows原样
  miku              Miku主题 - 青色和粉色配色，自定义品牌

示例:
  # 交互式模式（推荐）
  tiny11builder.exe

  # 使用命令行参数
  tiny11builder.exe -iso E -mode standard
  tiny11builder.exe -iso E -mode standard -theme miku
  tiny11builder.exe -iso E -scratch D -mode core -v

  # 自动化构建
  tiny11builder.exe -iso E -mode standard -theme miku -index 3 -output "D:\miku_tiny11.iso"
`)
}

// PrintUsage 保留兼容性
func PrintUsage() {
	PrintUsageUnified()
}