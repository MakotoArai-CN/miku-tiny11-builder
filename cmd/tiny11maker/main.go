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
	utils.SetConsoleTitle("Tiny11 Builder - Miku Edition 🎀")

	// 显示Miku Banner
	utils.MikuBanner()

	// 初始化日志
	log := logger.NewLogger("tiny11maker")
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

	// 创建应用实例
	builder := app.NewTiny11Builder(config, log)

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
	log.Header("✨ 构建完成 ✨")
	log.Success("Tiny11镜像已成功创建!")
	log.Info("输出文件: %s", utils.Colorize(builder.GetOutputISO(), utils.MikuCyan))
	fmt.Println()
	
	fmt.Println(utils.Colorize("        ♪┏(・o･)┛♪┗ ( ･o･) ┓♪", utils.MikuPink))
	fmt.Println(utils.Colorize("          感谢使用 Miku Tiny11!", utils.MikuCyan))
	fmt.Println()
	
	fmt.Print(utils.Colorize("按Enter键退出...", utils.MikuGray))
	fmt.Scanln()
}