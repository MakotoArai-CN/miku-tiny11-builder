package utils

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ProgressBar 进度条结构
type ProgressBar struct {
	total       int64
	current     int64
	width       int
	prefix      string
	mu          sync.Mutex
	startTime   time.Time
	lastUpdate  time.Time
	isCompleted bool
	spinnerIdx  int
}

// NewProgressBar 创建新的进度条
func NewProgressBar(total int64, prefix string) *ProgressBar {
	width := GetConsoleWidth() - 50
	if width < 20 {
		width = 20
	}
	if width > 60 {
		width = 60
	}

	return &ProgressBar{
		total:      total,
		current:    0,
		width:      width,
		prefix:     prefix,
		startTime:  time.Now(),
		lastUpdate: time.Now(),
		spinnerIdx: 0,
	}
}

// Add 增加进度值
func (p *ProgressBar) Add(n int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current += n
	if p.current > p.total {
		p.current = p.total
	}

	// 限制更新频率（每100ms更新一次）避免闪烁
	now := time.Now()
	if now.Sub(p.lastUpdate) < 100*time.Millisecond && p.current < p.total {
		return
	}

	p.lastUpdate = now
	p.render()
}

// Set 设置当前进度值
func (p *ProgressBar) Set(current int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = current
	if p.current > p.total {
		p.current = p.total
	}

	p.lastUpdate = time.Now()
	p.render()
}

// Finish 完成进度条
func (p *ProgressBar) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = p.total
	p.isCompleted = true
	p.render()
	fmt.Println() // 换行
}

// render 渲染进度条（内部方法）
func (p *ProgressBar) render() {
	// 计算百分比
	percent := float64(0)
	if p.total > 0 {
		percent = float64(p.current) / float64(p.total) * 100
	} else {
		percent = 100
	}

	// 计算已填充的进度条长度
	filled := 0
	if p.total > 0 {
		filled = int(float64(p.width) * float64(p.current) / float64(p.total))
	}
	if filled > p.width {
		filled = p.width
	}
	if filled < 0 {
		filled = 0
	}

	// 构建进度条字符串
	filledBar := strings.Repeat("█", filled)
	emptyBar := strings.Repeat("░", p.width-filled)
	bar := filledBar + emptyBar

	// 计算速度和剩余时间
	elapsed := time.Since(p.startTime).Seconds()
	speed := float64(0)
	if elapsed > 0 {
		speed = float64(p.current) / elapsed
	}

	remaining := time.Duration(0)
	if speed > 0 && p.current < p.total {
		remainingSeconds := float64(p.total-p.current) / speed
		remaining = time.Duration(remainingSeconds) * time.Second
	}

	// 格式化输出
	var output string
	if p.isCompleted {
		// 完成状态
		output = fmt.Sprintf("\r%s [%s] %s %.1f%% %s | 用时: %s ",
			Colorize(p.prefix, MikuCyan),
			Colorize(bar, MikuGreen),
			Colorize("✓", MikuGreen),
			percent,
			formatBytes(p.current),
			formatDuration(time.Since(p.startTime)),
		)
	} else {
		// 进行中状态
		spinner := p.getSpinnerFrame()
		output = fmt.Sprintf("\r%s [%s] %s %.1f%% %s/%s | 速度: %s/s | 剩余: %s ",
			Colorize(p.prefix, MikuCyan),
			Colorize(bar, MikuPink),
			Colorize(spinner, MikuPink),
			percent,
			formatBytes(p.current),
			formatBytes(p.total),
			formatBytes(int64(speed)),
			formatDuration(remaining),
		)
	}

	// 清除行尾并输出
	fmt.Print(output)
}

// getSpinnerFrame 获取旋转动画帧
func (p *ProgressBar) getSpinnerFrame() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[p.spinnerIdx%len(frames)]
	p.spinnerIdx++
	return frame
}

// formatBytes 格式化字节大小
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []byte{'K', 'M', 'G', 'T', 'P', 'E'}
	if exp >= len(units) {
		exp = len(units) - 1
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), units[exp])
}

// formatDuration 格式化时间段
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}

	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// SimpleSpinner 简单的旋转动画
type SimpleSpinner struct {
	message   string
	stop      chan bool
	done      chan bool
	mu        sync.Mutex
	isRunning int32
}

// NewSpinner 创建旋转动画
func NewSpinner(message string) *SimpleSpinner {
	return &SimpleSpinner{
		message: message,
		stop:    make(chan bool, 1),
		done:    make(chan bool, 1),
	}
}

// Start 开始动画
func (s *SimpleSpinner) Start() {
	if !atomic.CompareAndSwapInt32(&s.isRunning, 0, 1) {
		return // 已经在运行
	}

	go func() {
		frames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
		i := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stop:
				s.done <- true
				return
			case <-ticker.C:
				s.mu.Lock()
				// 清除当前行并输出新内容
				fmt.Printf("\r%s %s %s",
					Colorize(frames[i], MikuPink),
					Colorize(s.message, MikuCyan),
					strings.Repeat(" ", 20), // 清除残留字符
				)
				s.mu.Unlock()
				i = (i + 1) % len(frames)
			}
		}
	}()
}

// Stop 停止动画
func (s *SimpleSpinner) Stop(success bool) {
	if !atomic.CompareAndSwapInt32(&s.isRunning, 1, 0) {
		return // 未在运行
	}

	// 发送停止信号
	select {
	case s.stop <- true:
	default:
	}

	// 等待goroutine结束
	select {
	case <-s.done:
	case <-time.After(500 * time.Millisecond):
	}

	time.Sleep(150 * time.Millisecond)

	s.mu.Lock()
	defer s.mu.Unlock()

	// 显示最终状态
	symbol := "✓"
	color := MikuGreen
	if !success {
		symbol = "✗"
		color = MikuRed
	}

	fmt.Printf("\r%s %s%s\n",
		Colorize(symbol, color),
		Colorize(s.message, MikuCyan),
		strings.Repeat(" ", 20),
	)
}

// UpdateMessage 更新消息
func (s *SimpleSpinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}