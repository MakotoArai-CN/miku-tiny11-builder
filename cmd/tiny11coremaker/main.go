package main

import (
	"fmt"
	"os"
	"tiny11-builder/internal/app"
	"tiny11-builder/internal/cli"
	"tiny11-builder/internal/logger"
	"tiny11-builder/internal/utils"
)

func main() {
	// 初始化控制台（UTF-8 + 颜色支持）
	if err := utils.InitConsole(); err != nil {
		fmt.Printf("警告: 初始化控制台失败: %v\n", err)
	}

	// 设置控制台标题
	utils.SetConsoleTitle("Tiny11 Core Builder - Miku Edition 🎀")

	// 显示Miku Banner
	utils.MikuBanner()

	// 显示Core版本警告
	fmt.Println()
	fmt.Println(utils.Colorize("╔════════════════════════════════════════════════════════════════════════╗", utils.MikuRed))
	fmt.Println(utils.Colorize("║                           ⚠️  重要警告  ⚠️                              ║", utils.MikuRed))
	fmt.Println(utils.Colorize("╠════════════════════════════════════════════════════════════════════════╣", utils.MikuRed))
	fmt.Println(utils.Colorize("║  Tiny11 Core 是一个高度精简的版本，仅用于测试和开发环境！             ║", utils.MikuYellow))
	fmt.Println(utils.Colorize("║                                                                        ║", utils.MikuYellow))
	fmt.Println(utils.Colorize("║  生成的镜像将：                                                         ║", utils.MikuYellow))
	fmt.Println(utils.Colorize("║    • 移除大部分系统组件                                                 ║", utils.MikuWhite))
	fmt.Println(utils.Colorize("║    • 无法安装Windows更新                                                ║", utils.MikuWhite))
	fmt.Println(utils.Colorize("║    • 无法添加语言包和功能                                               ║", utils.MikuWhite))
	fmt.Println(utils.Colorize("║    • 禁用Windows Defender和系统恢复                                     ║", utils.MikuWhite))
	fmt.Println(utils.Colorize("║                                                                        ║", utils.MikuYellow))
	fmt.Println(utils.Colorize("║  ⚠️  不建议用于日常使用！仅适合虚拟机测试环境！                        ║", utils.MikuRed+utils.Bold))
	fmt.Println(utils.Colorize("╚════════════════════════════════════════════════════════════════════════╝", utils.MikuRed))
	fmt.Println()

	fmt.Print(utils.Colorize("是否继续? (yes/no): ", utils.MikuPink))
	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "yes" && confirm != "YES" && confirm != "y" && confirm != "Y" {
		fmt.Println(utils.Colorize("\n操作已取消。", utils.MikuCyan))
		os.Exit(0)
	}

	fmt.Println()

	// 初始化日志
	log := logger.NewLogger("tiny11coremaker")
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

	// 解析命令行参数
	config, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		log.Error("参数解析错误: %v", err)
		cli.PrintUsage()
		os.Exit(1)
	}

	config.CoreMode = true

	// 创建应用实例
	builder := app.NewTiny11CoreBuilder(config, log)

	// 执行构建
	if err := builder.Build(); err != nil {
		log.Error("构建失败: %v", err)
		fmt.Println()
		fmt.Print(utils.Colorize("按Enter键退出...", utils.MikuGray))
		fmt.Scanln()
		os.Exit(1)
	}

	// 成功完成
	fmt.Println()
	log.Header("✨ Core版本构建完成 ✨")
	log.Success("Tiny11 Core镜像已成功创建!")
	log.Info("输出文件: %s", utils.Colorize(builder.GetOutputISO(), utils.MikuCyan))
	fmt.Println()

	// 最终警告
	fmt.Println(utils.Colorize("╔════════════════════════════════════════════════════════════════════════╗", utils.MikuYellow))
	fmt.Println(utils.Colorize("║  ⚠️  使用提醒:                                                          ║", utils.MikuYellow))
	fmt.Println(utils.Colorize("║    • 此镜像不可服务，无法接收更新                                       ║", utils.MikuWhite))
	fmt.Println(utils.Colorize("║    • 仅建议在隔离的测试环境中使用                                       ║", utils.MikuWhite))
	fmt.Println(utils.Colorize("║    • 不要用于生产环境或日常使用                                         ║", utils.MikuWhite))
	fmt.Println(utils.Colorize("╚════════════════════════════════════════════════════════════════════════╝", utils.MikuYellow))
	fmt.Println()

	fmt.Println(utils.Colorize("        ♪┏(・o･)┛♪┗ ( ･o･) ┓♪", utils.MikuPink))
	fmt.Println(utils.Colorize("          感谢使用 Miku Tiny11 Core!", utils.MikuCyan))
	fmt.Println()

	fmt.Print(utils.Colorize("按Enter键退出...", utils.MikuGray))
	fmt.Scanln()
}