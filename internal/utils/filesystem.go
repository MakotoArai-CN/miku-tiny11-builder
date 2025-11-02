package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	"strings"
)

const (
	MaxConcurrentCopy = 8 // 最大并发复制数
)

// CopyTask 复制任务结构
type CopyTask struct {
	Src  string
	Dst  string
	Size int64
}

// CopyDirConcurrent 并发复制目录 - 大幅提升速度
func CopyDirConcurrent(src, dst string, progress *ProgressBar) error {
	// 收集所有文件
	var tasks []CopyTask
	var totalSize int64

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过错误文件
		}

		relPath, _ := filepath.Rel(src, path)
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		tasks = append(tasks, CopyTask{
			Src:  path,
			Dst:  targetPath,
			Size: info.Size(),
		})
		totalSize += info.Size()
		return nil
	})

	if err != nil {
		return err
	}

	// 如果没有进度条，创建一个
	if progress == nil {
		progress = NewProgressBar(totalSize, "复制文件")
		defer progress.Finish()
	}

	// 使用 worker pool 并发复制
	taskChan := make(chan CopyTask, len(tasks))
	errChan := make(chan error, len(tasks))
	var wg sync.WaitGroup

	// 启动 workers
	workers := MaxConcurrentCopy
	if workers > runtime.NumCPU() {
		workers = runtime.NumCPU()
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				if err := copyFileOptimized(task.Src, task.Dst, task.Size, progress); err != nil {
					// 记录错误但继续
					select {
					case errChan <- fmt.Errorf("复制失败 %s: %w", task.Src, err):
					default:
					}
				}
			}
		}()
	}

	// 分发任务
	for _, task := range tasks {
		taskChan <- task
	}
	close(taskChan)

	// 等待所有任务完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误
	if len(errChan) > 0 {
		return <-errChan
	}

	// 强制垃圾回收
	runtime.GC()

	return nil
}

// copyFileOptimized 优化的文件复制 - 使用内存池
func copyFileOptimized(src, dst string, fileSize int64, progress *ProgressBar) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		// 无法打开源文件，更新进度并跳过
		if progress != nil {
			progress.Add(fileSize)
		}
		return nil
	}
	defer sourceFile.Close()

	// 确保目标目录存在
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		if progress != nil {
			progress.Add(fileSize)
		}
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		if progress != nil {
			progress.Add(fileSize)
		}
		return nil // 跳过无法创建的文件
	}
	defer destFile.Close()

	// 根据文件大小选择缓冲区
	var buf *[]byte
	if fileSize < 1024*1024 { // < 1MB 使用小缓冲区
		buf = GetSmallBuffer()
		defer PutSmallBuffer(buf)
	} else {
		buf = GetLargeBuffer()
		defer PutLargeBuffer(buf)
	}

	// 流式复制
	for {
		n, readErr := sourceFile.Read(*buf)
		if n > 0 {
			if _, writeErr := destFile.Write((*buf)[:n]); writeErr != nil {
				return writeErr
			}
			if progress != nil {
				progress.Add(int64(n))
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	return nil
}

// CopyFile 简单文件复制
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("获取源文件信息失败: %w", err)
	}

	// 确保目标目录存在
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer destFile.Close()

	// 使用缓冲区复制
	buf := GetLargeBuffer()
	defer PutLargeBuffer(buf)

	_, err = io.CopyBuffer(destFile, sourceFile, *buf)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 同步到磁盘
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("同步文件失败: %w", err)
	}

	// 设置文件权限
	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("设置文件权限失败: %w", err)
	}

	return nil
}

// CopyDir 递归复制目录
func CopyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("获取源目录信息失败: %w", err)
	}

	// 创建目标目录
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("读取目录内容失败: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := CopyFile(srcPath, dstPath); err != nil {
				// 记录警告但继续
				fmt.Printf("警告: 复制文件失败 %s: %v\n", srcPath, err)
			}
		}
	}

	return nil
}

// DownloadFile 下载文件，带重试机制
func DownloadFile(url, filepath string) error {
	tmpFile := filepath + ".tmp"
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		err := downloadFileAttempt(url, tmpFile)
		if err == nil {
			// 下载成功，重命名文件
			if err := os.Rename(tmpFile, filepath); err != nil {
				os.Remove(tmpFile)
				return fmt.Errorf("重命名临时文件失败: %w", err)
			}
			return nil
		}

		lastErr = err
	}

	// 清理临时文件
	os.Remove(tmpFile)
	return fmt.Errorf("下载失败(尝试%d次): %w", maxRetries, lastErr)
}

// downloadFileAttempt 单次下载尝试
func downloadFileAttempt(url, filepath string) error {
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP错误: %d %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	// 使用缓冲区池
	buf := GetLargeBuffer()
	defer PutLargeBuffer(buf)

	_, err = io.CopyBuffer(out, resp.Body, *buf)
	if err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	if err := out.Sync(); err != nil {
		return fmt.Errorf("同步文件失败: %w", err)
	}

	return nil
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists 检查目录是否存在
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetFileSize 获取文件大小
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetDirSize 计算目录大小
func GetDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过错误
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// CountFiles 统计目录中的文件数量
func CountFiles(path string) (int, error) {
	count := 0
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count, err
}

// EnsureDir 确保目录存在
func EnsureDir(path string) error {
	if !DirExists(path) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// RemoveIfExists 删除文件或目录（如果存在）
func RemoveIfExists(path string) error {
	if FileExists(path) || DirExists(path) {
		return os.RemoveAll(path)
	}
	return nil
}

// FormatBytes 格式化字节大小
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}

	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}

// WriteFile 写入文件
func WriteFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// ReadFile 读取文件
func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	return data, nil
}

// IsEmpty 检查目录是否为空
func IsEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// CleanDir 清空目录内容
func CleanDir(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		if err := os.RemoveAll(fullPath); err != nil {
			return err
		}
	}

	return nil
}

// GetTempDir 获取临时目录
func GetTempDir() string {
	return os.TempDir()
}

// CreateTempFile 创建临时文件
func CreateTempFile(pattern string) (*os.File, error) {
	return os.CreateTemp("", pattern)
}

// CreateTempDir 创建临时目录
func CreateTempDir(pattern string) (string, error) {
	return os.MkdirTemp("", pattern)
}

// SafeRemove 安全删除文件或目录（忽略错误）
func SafeRemove(path string) {
	os.RemoveAll(path)
}

// SafeRemoveAll 批量安全删除
func SafeRemoveAll(paths ...string) {
	for _, path := range paths {
		os.RemoveAll(path)
	}
}

// CopyFileIfNotExists 仅在目标不存在时复制
func CopyFileIfNotExists(src, dst string) error {
	if FileExists(dst) {
		return nil
	}
	return CopyFile(src, dst)
}

// MoveFile 移动文件
func MoveFile(src, dst string) error {
	// 确保目标目录存在
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	// 先尝试重命名
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// 重命名失败则复制后删除
	if err := CopyFile(src, dst); err != nil {
		return err
	}

	return os.Remove(src)
}

// TouchFile 创建空文件或更新修改时间
func TouchFile(path string) error {
	if !FileExists(path) {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		return f.Close()
	}

	now := time.Now()
	return os.Chtimes(path, now, now)
}

// GetAbsPath 获取绝对路径
func GetAbsPath(path string) (string, error) {
	return filepath.Abs(path)
}

// IsSubPath 检查 child 是否是 parent 的子路径
func IsSubPath(parent, child string) (bool, error) {
	parentAbs, err := filepath.Abs(parent)
	if err != nil {
		return false, err
	}

	childAbs, err := filepath.Abs(child)
	if err != nil {
		return false, err
	}

	rel, err := filepath.Rel(parentAbs, childAbs)
	if err != nil {
		return false, err
	}

	return !strings.HasPrefix(rel, ".."), nil
}

// ListFiles 列出目录中的所有文件（递归）
func ListFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ListDirs 列出目录中的所有子目录
func ListDirs(dir string) ([]string, error) {
	var dirs []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(dir, entry.Name()))
		}
	}

	return dirs, nil
}

// GetFileExtension 获取文件扩展名
func GetFileExtension(path string) string {
	return filepath.Ext(path)
}

// GetFileName 获取文件名（不含扩展名）
func GetFileName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return base[:len(base)-len(ext)]
}

// JoinPath 连接路径
func JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}

// SplitPath 分割路径
func SplitPath(path string) (dir, file string) {
	return filepath.Split(path)
}