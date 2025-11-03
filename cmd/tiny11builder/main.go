package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"tiny11-builder/internal/api"
	"tiny11-builder/internal/app"
	"tiny11-builder/internal/cli"
	"tiny11-builder/internal/config"
	"tiny11-builder/internal/logger"
	"tiny11-builder/internal/utils"
)

func main() {
	// 初始化控制台
	if err := utils.InitConsole(); err != nil {
		fmt.Printf("警告: 初始化控制台失败: %v\n", err)
	}
	utils.SetConsoleTitle("Tiny11 Builder - Miku Edition 🎀")

	//  手动检测 API 模式 
	apiMode := false
	apiPort := 8080

	for i, arg := range os.Args[1:] {
		if arg == "-api" || arg == "--api" {
			apiMode = true
		}
		if (arg == "-port" || arg == "--port") && i+1 < len(os.Args)-1 {
			fmt.Sscanf(os.Args[i+2], "%d", &apiPort)
		}
	}

	//  API 模式 
	if apiMode {
		runAPIMode(apiPort)
		return
	}

	//  判断是否有其他命令行参数 
	hasArgs := len(os.Args) > 1

	if hasArgs {
		// 命令行模式 - 使用 cli.ParseArgsUnified()
		runCommandLineMode()
	} else {
		// 交互模式
		runInteractiveMode()
	}
}

// 命令行模式
func runCommandLineMode() {
	log := logger.NewLogger("tiny11builder")
	defer log.Close()

	// 验证管理员权限
	if !cli.IsAdmin() {
		log.Error("需要管理员权限运行此程序")
		fmt.Println()
		fmt.Println(utils.Colorize("请以管理员身份运行此程序:", utils.MikuYellow))
		fmt.Println(utils.Colorize("  1. 右键点击程序", utils.MikuWhite))
		fmt.Println(utils.Colorize("  2. 选择\"以管理员身份运行\"", utils.MikuWhite))
		fmt.Println()
		os.Exit(1)
	}

	// 使用 cli.ParseArgsUnified() 解析参数
	cfg, buildMode, themeName, err := cli.ParseArgsUnified(os.Args[1:])
	if err != nil {
		log.Error("参数解析错误: %v", err)
		cli.PrintUsageUnified()
		os.Exit(1)
	}

	// 清理旧目录
	cleanupOldBuild(cfg, log)

	// 创建目录
	if err := cfg.EnsureDirectories(); err != nil {
		log.Error("创建工作目录失败: %v", err)
		os.Exit(1)
	}

	// 应用主题名称
	if themeName != "" && themeName != "default" {
		cfg.ThemeName = themeName
	} else {
		cfg.ThemeName = ""
	}

	// 确定构建模式（如果未指定，默认 standard）
	if buildMode == "" {
		buildMode = "standard"
	}

	//  预装软件选择 
	selectPreinstallApps(cfg, log)

	runtime.GOMAXPROCS(runtime.NumCPU())

	// 执行构建
	executeBuild(cfg, buildMode, log)
}

// 交互模式
func runInteractiveMode() {
	showMainUI()
	log := logger.NewLogger("tiny11builder")
	defer log.Close()

	// 检查管理员权限
	if !cli.IsAdmin() {
		log.Error("需要管理员权限运行此程序")
		fmt.Println()
		fmt.Println(utils.Colorize("请以管理员身份运行此程序:", utils.MikuYellow))
		fmt.Println(utils.Colorize("  1. 右键点击程序", utils.MikuWhite))
		fmt.Println(utils.Colorize("  2. 选择\"以管理员身份运行\"", utils.MikuWhite))
		fmt.Println()
		fmt.Print(utils.Colorize("按Enter键退出...", utils.MikuGray))
		fmt.Scanln()
		os.Exit(1)
	}

	// 清理旧构建
	prelimCfg := config.NewConfig()
	cleanupOldBuild(prelimCfg, log)

	// 解析参数（交互式输入）
	cfg, buildMode, themeName, err := cli.ParseArgsUnified([]string{})
	if err != nil {
		log.Error("参数解析错误: %v", err)
		os.Exit(1)
	}

	if err := cfg.EnsureDirectories(); err != nil {
		log.Error("创建工作目录失败: %v", err)
		os.Exit(1)
	}

	// 选择模式
	if buildMode == "" {
		buildMode = showModeSelection()
	}

	// 选择主题
	if themeName != "" && themeName != "default" {
		cfg.ThemeName = themeName
	} else if themeName == "default" {
		cfg.ThemeName = ""
	} else {
		if showThemeSelection() {
			cfg.ThemeName = "miku"
		} else {
			cfg.ThemeName = ""
		}
	}

	// 预装软件选择
	selectPreinstallApps(cfg, log)

	runtime.GOMAXPROCS(runtime.NumCPU())

	// 执行构建
	executeBuild(cfg, buildMode, log)
}

// API 模式
func runAPIMode(port int) {
	log := logger.NewLogger("api-server")
	defer log.Close()

	log.Info("启动 API 服务器模式 (端口: %d)", port)
	server := api.NewServer(port, log)
	if err := server.Start(); err != nil {
		log.Error("API服务器启动失败: %v", err)
		os.Exit(1)
	}
}

// 清理旧构建目录
func cleanupOldBuild(cfg *config.Config, log *logger.Logger) {
	buildDir := filepath.Join(cfg.WorkDir, "build")
	if utils.DirExists(buildDir) {
		log.Warn("检测到旧的构建目录，将进行清理...")
		spinner := utils.NewSpinner("正在清理残留文件...")
		spinner.Start()

		// 先尝试卸载可能挂载的镜像
		utils.RunCommand("dism", "/English", "/Unmount-Image",
			fmt.Sprintf("/MountDir:%s", cfg.ScratchDir), "/Discard")
		time.Sleep(1 * time.Second)

		err := os.RemoveAll(buildDir)
		spinner.Stop(err == nil)

		if err != nil {
			log.Error("清理旧目录失败: %v", err)
			log.Warn("请手动删除 %s 目录或重启电脑后再试。", buildDir)
			fmt.Print(utils.Colorize("按Enter键退出...", utils.MikuGray))
			fmt.Scanln()
			os.Exit(1)
		}
		log.Success("清理完成！")
		fmt.Println()
	}
}

// 执行构建
func executeBuild(cfg *config.Config, buildMode string, log *logger.Logger) {
	var builder app.Builder

	switch buildMode {
	case "standard":
		cfg.CoreMode = false
		builder = app.NewTiny11Builder(cfg, log)
	case "core":
		cfg.CoreMode = true
		if !showCoreWarning() {
			fmt.Println(utils.Colorize("\n操作已取消。", utils.MikuCyan))
			os.Exit(0)
		}
		builder = app.NewTiny11CoreBuilder(cfg, log)
	case "nano":
		cfg.CoreMode = true
		if !showNanoWarning() {
			fmt.Println(utils.Colorize("\n操作已取消。", utils.MikuCyan))
			os.Exit(0)
		}
		builder = app.NewTiny11NanoBuilder(cfg, log)
	default:
		log.Error("无效的构建模式: %s", buildMode)
		os.Exit(1)
	}

	log.Info("工作目录: %s", cfg.WorkDir)
	log.Info("输出路径: %s", cfg.OutputISO)

	if err := builder.Build(); err != nil {
		log.Error("构建失败: %v", err)
		fmt.Println()
		fmt.Print(utils.Colorize("按Enter键退出...", utils.MikuGray))
		fmt.Scanln()
		os.Exit(1)
	}

	showSuccessInfo(builder, log)
}

// 预装软件选择
func selectPreinstallApps(cfg *config.Config, log *logger.Logger) {
	preinstallDir := filepath.Join(cfg.WorkDir, "preinstall")
	configFile := filepath.Join(preinstallDir, "preinstall.json")

	if !utils.FileExists(configFile) {
		log.Info("未找到预装软件配置文件，跳过预装软件功能")
		return
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Warn("读取预装软件配置失败: %v，跳过预装软件功能", err)
		return
	}

	var preinstallConfig struct {
		Enabled bool `json:"enabled"`
		Apps    []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"apps"`
	}

	if err := json.Unmarshal(data, &preinstallConfig); err != nil {
		log.Warn("解析预装软件配置失败: %v，跳过预装软件功能", err)
		return
	}

	if !preinstallConfig.Enabled {
		log.Info("预装软件功能已禁用")
		return
	}

	if len(preinstallConfig.Apps) == 0 {
		log.Info("预装软件列表为空")
		return
	}

	fmt.Println()
	fmt.Println(utils.Colorize("┌────────────────────────────────────────────────────────────────────────┐", utils.MikuCyan))
	fmt.Println(utils.Colorize("│                         软件预装选项                                   │", utils.MikuPink+utils.Bold))
	fmt.Println(utils.Colorize("└────────────────────────────────────────────────────────────────────────┘", utils.MikuCyan))
	fmt.Println()
	fmt.Println(utils.Colorize("  检测到以下可预装软件:", utils.MikuCyan))
	fmt.Println()

	for i, app := range preinstallConfig.Apps {
		fmt.Printf(utils.Colorize("    [%d] %s", utils.MikuWhite), i+1, app.Name)
		if app.Description != "" {
			fmt.Printf(utils.Colorize(" - %s", utils.MikuGray), app.Description)
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println(utils.Colorize("  [A] 安装全部", utils.MikuGreen))
	fmt.Println(utils.Colorize("  [N] 不安装任何软件 (推荐)", utils.MikuGray))
	fmt.Println()
	fmt.Print(utils.Colorize("请选择 [编号/A/N]: ", utils.MikuPink))

	var choice string
	fmt.Scanln(&choice)
	choice = strings.TrimSpace(strings.ToUpper(choice))

	switch choice {
	case "A":
		for _, app := range preinstallConfig.Apps {
			cfg.PreinstallApps = append(cfg.PreinstallApps, app.ID)
		}
		fmt.Println(utils.Colorize("✓ 将预装全部软件", utils.MikuGreen))
	case "N", "":
		fmt.Println(utils.Colorize("✓ 不预装软件", utils.MikuCyan))
	default:
		var idx int
		if _, err := fmt.Sscanf(choice, "%d", &idx); err == nil {
			if idx >= 1 && idx <= len(preinstallConfig.Apps) {
				app := preinstallConfig.Apps[idx-1]
				cfg.PreinstallApps = []string{app.ID}
				fmt.Printf(utils.Colorize("✓ 将预装: %s\n", utils.MikuGreen), app.Name)
			} else {
				fmt.Println(utils.Colorize("✗ 无效选项，不预装软件", utils.MikuRed))
			}
		} else {
			fmt.Println(utils.Colorize("✗ 无效选项，不预装软件", utils.MikuRed))
		}
	}

	fmt.Println()
}

// UI 显示函数
func showMainUI() {
	utils.MikuBanner()
	fmt.Println(utils.Colorize("╔════════════════════════════════════════════════════════════════════════╗", utils.MikuCyan))
	fmt.Println(utils.Colorize("║                    Windows 11 精简镜像构建工具                          ║", utils.MikuCyan))
	fmt.Println(utils.Colorize("║                         Miku Edition v2.1                              ║", utils.MikuPink))
	fmt.Println(utils.Colorize("╚════════════════════════════════════════════════════════════════════════╝", utils.MikuCyan))
	fmt.Println()
}

func showModeSelection() string {
	fmt.Println(utils.Colorize("┌────────────────────────────────────────────────────────────────────────┐", utils.MikuCyan))
	fmt.Println(utils.Colorize("│                         请选择构建模式                                 │", utils.MikuPink+utils.Bold))
	fmt.Println(utils.Colorize("└────────────────────────────────────────────────────────────────────────┘", utils.MikuCyan))
	fmt.Println()
	fmt.Println(utils.Colorize("  [1] 标准版 (Standard)", utils.MikuCyan+utils.Bold))
	fmt.Println(utils.Colorize("      • 移除大部分预装应用和膨胀软件", utils.MikuWhite))
	fmt.Println(utils.Colorize("      • 保留系统可服务性", utils.MikuWhite))
	fmt.Println(utils.Colorize("      • 可安装更新、语言包和功能", utils.MikuWhite))
	fmt.Println(utils.Colorize("      • 适合日常使用", utils.MikuGreen))
	fmt.Println(utils.Colorize("      • 大小: ~5-6 GB", utils.MikuGray))
	fmt.Println()
	fmt.Println(utils.Colorize("  [2] Core版 (极限精简)", utils.MikuPink+utils.Bold))
	fmt.Println(utils.Colorize("      • 移除所有标准版内容", utils.MikuWhite))
	fmt.Println(utils.Colorize("      • 移除大部分 WinSxS 组件", utils.MikuWhite))
	fmt.Println(utils.Colorize("      • 禁用 Windows Update 和 Defender", utils.MikuWhite))
	fmt.Println(utils.Colorize("      • 不可服务，仅用于测试环境", utils.MikuYellow))
	fmt.Println(utils.Colorize("      • 大小: ~4-5 GB", utils.MikuGray))
	fmt.Println()
	fmt.Println(utils.Colorize("  [3] Nano版 (终极精简) ⚡", utils.MikuRed+utils.Bold))
	fmt.Println(utils.Colorize("      • 移除所有 Core 版内容", utils.MikuWhite))
	fmt.Println(utils.Colorize("      • 精简驱动、字体、系统文件夹", utils.MikuWhite))
	fmt.Println(utils.Colorize("      • 移除大量系统服务", utils.MikuWhite))
	fmt.Println(utils.Colorize("      • 使用 ESD 格式 (超高压缩)", utils.MikuWhite))
	fmt.Println(utils.Colorize("      • ⚠️  极端精简，仅用于特殊场景", utils.MikuRed))
	fmt.Println(utils.Colorize("      • 大小: ~2.5-3.5 GB", utils.MikuGray))
	fmt.Println()
	fmt.Println(utils.Colorize("  [Q] 退出程序", utils.MikuGray))
	fmt.Println()

	for {
		fmt.Print(utils.Colorize("请输入选项 [1/2/3/Q]: ", utils.MikuPink))
		var choice string
		fmt.Scanln(&choice)
		choice = strings.ToUpper(strings.TrimSpace(choice))

		switch choice {
		case "1":
			fmt.Println(utils.Colorize("✓ 已选择: 标准版", utils.MikuGreen))
			fmt.Println()
			return "standard"
		case "2":
			fmt.Println(utils.Colorize("✓ 已选择: Core版", utils.MikuPink))
			fmt.Println()
			return "core"
		case "3":
			fmt.Println(utils.Colorize("✓ 已选择: Nano版 (终极精简)", utils.MikuRed))
			fmt.Println()
			return "nano"
		case "Q", "QUIT", "EXIT":
			fmt.Println(utils.Colorize("\n再见！", utils.MikuCyan))
			os.Exit(0)
		default:
			fmt.Println(utils.Colorize("  ✗ 无效选项，请重新输入", utils.MikuRed))
		}
	}
}

func showThemeSelection() bool {
	fmt.Println(utils.Colorize("┌────────────────────────────────────────────────────────────────────────┐", utils.MikuCyan))
	fmt.Println(utils.Colorize("│                         主题选择                                       │", utils.MikuPink+utils.Bold))
	fmt.Println(utils.Colorize("└────────────────────────────────────────────────────────────────────────┘", utils.MikuCyan))
	fmt.Println()
	fmt.Println(utils.Colorize("  是否应用 Miku 主题?", utils.MikuCyan))
	fmt.Println()
	fmt.Println(utils.Colorize("  Miku主题包含:", utils.MikuWhite))
	fmt.Println(utils.Colorize("    • 系统名称显示为 'Miku Tiny11'", utils.MikuWhite))
	fmt.Println(utils.Colorize("    • 青色和粉色配色方案", utils.MikuWhite))
	fmt.Println(utils.Colorize("    • 优化的视觉效果", utils.MikuWhite))
	fmt.Println(utils.Colorize("    • 自定义壁纸和图标 (如果已配置)", utils.MikuGray))
	fmt.Println()
	fmt.Print(utils.Colorize("应用Miku主题? [y/N]: ", utils.MikuPink))

	var choice string
	fmt.Scanln(&choice)
	choice = strings.ToLower(strings.TrimSpace(choice))

	apply := choice == "y" || choice == "yes"
	if apply {
		fmt.Println(utils.Colorize("✓ 将应用Miku主题", utils.MikuGreen))
	} else {
		fmt.Println(utils.Colorize("✓ 使用默认主题", utils.MikuCyan))
	}
	fmt.Println()

	return apply
}

func showCoreWarning() bool {
	fmt.Println(utils.Colorize("╔════════════════════════════════════════════════════════════════════════╗", utils.MikuRed))
	fmt.Println(utils.Colorize("║                           ⚠️  重要警告  ⚠️                              ║", utils.MikuRed+utils.Bold))
	fmt.Println(utils.Colorize("╠════════════════════════════════════════════════════════════════════════╣", utils.MikuRed))
	fmt.Println(utils.Colorize("║  Tiny11 Core 是一个高度精简的版本，仅用于测试和开发环境！             ║", utils.MikuYellow))
	fmt.Println(utils.Colorize("║  ⚠️  不建议用于日常使用！仅适合虚拟机测试环境！                        ║", utils.MikuRed+utils.Bold))
	fmt.Println(utils.Colorize("╚════════════════════════════════════════════════════════════════════════╝", utils.MikuRed))
	fmt.Println()
	fmt.Print(utils.Colorize("确认继续? (yes/no): ", utils.MikuPink+utils.Bold))

	var confirm string
	fmt.Scanln(&confirm)
	confirm = strings.ToLower(strings.TrimSpace(confirm))

	return confirm == "yes" || confirm == "y"
}

func showNanoWarning() bool {
	fmt.Println(utils.Colorize("╔════════════════════════════════════════════════════════════════════════╗", utils.MikuRed))
	fmt.Println(utils.Colorize("║                      ⚠️  极端精简警告  ⚠️                               ║", utils.MikuRed+utils.Bold))
	fmt.Println(utils.Colorize("╠════════════════════════════════════════════════════════════════════════╣", utils.MikuRed))
	fmt.Println(utils.Colorize("║  Tiny11 Nano 是终极精简版本，仅用于极端测试场景！                     ║", utils.MikuYellow))
	fmt.Println(utils.Colorize("║                                                                        ║", utils.MikuYellow))
	fmt.Println(utils.Colorize("║  生成的镜像将：                                                         ║", utils.MikuYellow))
	fmt.Println(utils.Colorize("║    • 移除几乎所有可移除的系统组件                                       ║", utils.MikuWhite))
	fmt.Println(utils.Colorize("║    • 精简驱动、字体、系统文件夹                                         ║", utils.MikuWhite))
	fmt.Println(utils.Colorize("║    • 移除大量系统服务（打印、蓝牙、诊断等）                             ║", utils.MikuWhite))
	fmt.Println(utils.Colorize("║    • 完全禁用 Windows Update 和 Defender                                ║", utils.MikuWhite))
	fmt.Println(utils.Colorize("║    • 使用 ESD 格式导出（超高压缩但解压慢）                              ║", utils.MikuWhite))
	fmt.Println(utils.Colorize("║    • 可能导致某些软件无法运行                                           ║", utils.MikuWhite))
	fmt.Println(utils.Colorize("║                                                                        ║", utils.MikuYellow))
	fmt.Println(utils.Colorize("║  ⚠️  此版本可能无法正常启动！仅用于实验和特殊场景！                    ║", utils.MikuRed+utils.Bold))
	fmt.Println(utils.Colorize("╚════════════════════════════════════════════════════════════════════════╝", utils.MikuRed))
	fmt.Println()
	fmt.Print(utils.Colorize("确认继续? 请输入 'I UNDERSTAND' (大写): ", utils.MikuPink+utils.Bold))

	var confirm string
	fmt.Scanln(&confirm)

	return confirm == "I UNDERSTAND"
}

func showSuccessInfo(builder app.Builder, log *logger.Logger) {
	fmt.Println()
	log.Header("✨ 构建完成 ✨")
	log.Success("Tiny11镜像已成功创建!")

	isoPath := builder.GetOutputISO()
	isoInfo, err := os.Stat(isoPath)
	if err == nil {
		fmt.Println()
		fmt.Println(utils.Colorize("╔════════════════════════════════════════════════════════════════════════╗", utils.MikuCyan))
		fmt.Println(utils.Colorize("║                          📊 构建统计                                   ║", utils.MikuCyan+utils.Bold))
		fmt.Println(utils.Colorize("╚════════════════════════════════════════════════════════════════════════╝", utils.MikuCyan))
		fmt.Println()
		fmt.Printf("  %s %s\n",
			utils.Colorize("ISO大小:    ", utils.MikuCyan),
			utils.Colorize(utils.FormatBytes(isoInfo.Size()), utils.MikuGreen+utils.Bold))
		fmt.Printf("  %s %s\n",
			utils.Colorize("输出路径:   ", utils.MikuCyan),
			utils.Colorize(isoPath, utils.MikuWhite))
		fmt.Printf("  %s %s\n",
			utils.Colorize("创建时间:   ", utils.MikuCyan),
			utils.Colorize(isoInfo.ModTime().Format("2006-01-02 15:04:05"), utils.MikuGray))
	}

	fmt.Println()
	fmt.Println(utils.Colorize("╔════════════════════════════════════════════════════════════════════════╗", utils.MikuPink))
	fmt.Println(utils.Colorize("║              ♪┏(・o･)┛♪┗ ( ･o･) ┓♪                                   ║", utils.MikuPink+utils.Bold))
	fmt.Println(utils.Colorize("║                感谢使用 Miku Tiny11 Builder!                           ║", utils.MikuCyan))
	fmt.Println(utils.Colorize("╚════════════════════════════════════════════════════════════════════════╝", utils.MikuPink))
	fmt.Println()
	fmt.Print(utils.Colorize("按Enter键退出...", utils.MikuGray))
	fmt.Scanln()
}
