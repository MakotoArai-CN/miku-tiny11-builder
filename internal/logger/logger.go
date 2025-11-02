package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"tiny11-builder/internal/utils"
)

// Logger 日志记录器
type Logger struct {
	file    *os.File
	logger  *log.Logger
}

// NewLogger 创建日志记录器
func NewLogger(name string) *Logger {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.log", name, timestamp)
	
	// file, err := os.Create(filename)
	// 创建log文件夹，然后存入log文件
	logPath := "log"
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		os.Mkdir(logPath, 0777)
	}
	file, err := os.OpenFile(logPath+"/"+filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("警告: 无法创建日志文件: %v\n", err)
		return &Logger{
			logger: log.New(os.Stdout, "", log.LstdFlags),
		}
	}

	return &Logger{
		file:   file,
		logger: log.New(file, "", log.LstdFlags),
	}
}

// Info 记录信息
func (l *Logger) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	colorMsg := utils.Colorize(msg, utils.MikuWhite)
	fmt.Println(colorMsg)
	if l.logger != nil {
		l.logger.Println(msg)
	}
}

// Success 记录成功信息
func (l *Logger) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	colorMsg := fmt.Sprintf("%s %s",
		utils.Colorize("✓", utils.MikuGreen),
		utils.Colorize(msg, utils.MikuGreen),
	)
	fmt.Println(colorMsg)
	if l.logger != nil {
		l.logger.Println("[SUCCESS] " + msg)
	}
}

// Warn 记录警告
func (l *Logger) Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	colorMsg := fmt.Sprintf("%s %s",
		utils.Colorize("⚠", utils.MikuYellow),
		utils.Colorize(msg, utils.MikuYellow),
	)
	fmt.Println(colorMsg)
	if l.logger != nil {
		l.logger.Println("[WARN] " + msg)
	}
}

// Error 记录错误
func (l *Logger) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	colorMsg := fmt.Sprintf("%s %s",
		utils.Colorize("✗", utils.MikuRed),
		utils.Colorize(msg, utils.MikuRed),
	)
	fmt.Println(colorMsg)
	if l.logger != nil {
		l.logger.Println("[ERROR] " + msg)
	}
}

// Step 记录步骤
func (l *Logger) Step(num int, desc string) {
	width := utils.GetConsoleWidth()
	if width > 100 {
		width = 100
	}

	// 创建分隔线
	separator := strings.Repeat("─", width-4)
	
	fmt.Println()
	fmt.Println(utils.Colorize("┌"+separator+"┐", utils.MikuCyan))
	
	stepText := fmt.Sprintf(" [步骤 %d] %s ", num, desc)
	padding := width - len(stepText) - 4
	if padding < 0 {
		padding = 0
	}
	
	fmt.Printf("%s%s%s%s\n",
		utils.Colorize("│ ", utils.MikuCyan),
		utils.Colorize(fmt.Sprintf("[步骤 %d]", num), utils.MikuPink),
		utils.Colorize(" "+desc, utils.MikuCyan),
		utils.Colorize(strings.Repeat(" ", padding)+"│", utils.MikuCyan),
	)
	
	fmt.Println(utils.Colorize("└"+separator+"┘", utils.MikuCyan))
	
	if l.logger != nil {
		l.logger.Printf("[STEP %d] %s", num, desc)
	}
}

// Header 显示标题
func (l *Logger) Header(title string) {
	width := utils.GetConsoleWidth()
	if width > 100 {
		width = 100
	}

	separator := strings.Repeat("═", width)
	padding := (width - len(title)) / 2
	if padding < 0 {
		padding = 0
	}

	fmt.Println()
	fmt.Println(utils.Colorize(separator, utils.MikuCyan))
	fmt.Printf("%s%s%s\n",
		strings.Repeat(" ", padding),
		utils.Colorize(title, utils.MikuPink+utils.Bold),
		strings.Repeat(" ", padding),
	)
	fmt.Println(utils.Colorize(separator, utils.MikuCyan))
	fmt.Println()
}

// Section 显示区块
func (l *Logger) Section(title string) {
	fmt.Println()
	fmt.Printf("%s %s\n",
		utils.Colorize("▶", utils.MikuPink),
		utils.Colorize(title, utils.MikuCyan+utils.Bold),
	)
}

// Skip 记录跳过信息
func (l *Logger) Skip(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	colorMsg := fmt.Sprintf("%s %s",
		utils.Colorize("⊘", utils.MikuGray),
		utils.Colorize(msg, utils.MikuGray),
	)
	fmt.Println(colorMsg)
	if l.logger != nil {
		l.logger.Println("[SKIP] " + msg)
	}
}

// Close 关闭日志
func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}