package image

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"tiny11-builder/internal/config"
	"tiny11-builder/internal/logger"
	"tiny11-builder/internal/types"
	"tiny11-builder/internal/utils"
)

type Manager struct {
	config *config.Config
	log    *logger.Logger
	info   *ImageInfo
}

type ImageInfo struct {
	Index        int
	Name         string
	Architecture string
	Language     string
	Build        string
	Description  string
	Size         int64
}

func NewManager(cfg *config.Config, log *logger.Logger) *Manager {
	return &Manager{
		config: cfg,
		log:    log,
	}
}

// ValidateISO 验证ISO镜像完整性
func (m *Manager) ValidateISO() error {
	bootWim := filepath.Join(m.config.ISODrive, "sources", "boot.wim")
	installWim := filepath.Join(m.config.ISODrive, "sources", "install.wim")
	installEsd := filepath.Join(m.config.ISODrive, "sources", "install.esd")

	spinner := utils.NewSpinner("验证ISO镜像完整性...")
	spinner.Start()

	if !utils.FileExists(bootWim) {
		spinner.Stop(false)
		return types.NewError(types.ErrCodeNotFound, "未找到boot.wim", nil).
			WithContext("path", bootWim)
	}

	hasWim := utils.FileExists(installWim)
	hasEsd := utils.FileExists(installEsd)

	if !hasWim && !hasEsd {
		spinner.Stop(false)
		return types.NewError(types.ErrCodeNotFound, "未找到install.wim或install.esd", nil)
	}

	spinner.Stop(true)

	// 如果是ESD格式，需要转换
	if !hasWim && hasEsd {
		m.log.Info("检测到install.esd，需要转换为install.wim")
		if err := m.convertEsdToWim(installEsd); err != nil {
			return types.NewError(types.ErrCodeDISM, "转换ESD失败", err)
		}
	}

	return nil
}

// convertEsdToWim 转换ESD镜像为WIM格式
func (m *Manager) convertEsdToWim(esdPath string) error {
	m.log.Section("转换ESD镜像格式")

	// 获取ESD信息
	output, err := utils.RunCommand("dism", "/English", "/Get-WimInfo",
		fmt.Sprintf("/WimFile:%s", esdPath))
	if err != nil {
		return fmt.Errorf("获取ESD信息失败: %w", err)
	}

	// 显示可用镜像
	fmt.Println()
	lines := m.parseImageList(output)
	for _, line := range lines {
		if strings.HasPrefix(line, "Index :") {
			fmt.Println(utils.Colorize(line, utils.MikuCyan))
		} else {
			fmt.Println(utils.Colorize(line, utils.MikuWhite))
		}
	}
	fmt.Println()

	// 选择索引
	index := m.config.ImageIndex
	if index == 0 {
		fmt.Print(utils.Colorize("请输入要转换的镜像索引: ", utils.MikuPink))
		fmt.Scanln(&index)
	}

	// 转换
	destWim := filepath.Join(m.config.Tiny11Dir, "sources", "install.wim")
	os.MkdirAll(filepath.Dir(destWim), 0755)

	m.log.Info("正在转换镜像，这可能需要10-30分钟...")
	spinner := utils.NewSpinner("转换install.esd到install.wim")
	spinner.Start()

	_, err = utils.RunCommand("dism", "/English",
		"/Export-Image",
		fmt.Sprintf("/SourceImageFile:%s", esdPath),
		fmt.Sprintf("/SourceIndex:%d", index),
		fmt.Sprintf("/DestinationImageFile:%s", destWim),
		"/Compress:max",
		"/CheckIntegrity")

	spinner.Stop(err == nil)

	if err != nil {
		return fmt.Errorf("转换失败: %w", err)
	}

	m.log.Success("ESD转换完成")
	return nil
}

// CopyImageFiles 复制镜像文件
func (m *Manager) CopyImageFiles() error {
	m.log.Info("正在分析ISO镜像结构...")

	spinner := utils.NewSpinner("计算文件总大小...")
	spinner.Start()

	totalSize, fileCount, err := m.getDirSizeAndCount(m.config.ISODrive)
	if err != nil {
		spinner.Stop(false)
		return types.NewError(types.ErrCodeGeneral, "计算文件大小失败", err)
	}

	spinner.Stop(true)

	m.log.Info("ISO镜像大小: %s, 文件数: %d", utils.FormatBytes(totalSize), fileCount)

	// 创建目标目录
	if err := os.MkdirAll(m.config.Tiny11Dir, 0755); err != nil {
		return types.NewError(types.ErrCodePermission, "创建目标目录失败", err)
	}

	// 使用并发复制
	progress := utils.NewProgressBar(totalSize, "复制镜像文件")
	err = utils.CopyDirConcurrent(m.config.ISODrive, m.config.Tiny11Dir, progress)
	progress.Finish()

	if err != nil {
		return types.NewError(types.ErrCodeGeneral, "复制文件失败", err)
	}

	// 删除install.esd（如果存在）
	esdPath := filepath.Join(m.config.Tiny11Dir, "sources", "install.esd")
	if utils.FileExists(esdPath) {
		m.log.Info("移除install.esd...")
		os.Chmod(esdPath, 0666)
		if err := os.Remove(esdPath); err != nil {
			m.log.Warn("无法删除install.esd: %v", err)
		}
	}

	// 强制GC
	runtime.GC()

	m.log.Success("镜像文件复制完成")
	return nil
}

// getDirSizeAndCount 计算目录大小和文件数量
func (m *Manager) getDirSizeAndCount(path string) (int64, int, error) {
	var size int64
	var count int

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过错误
		}
		if !info.IsDir() {
			size += info.Size()
			count++
		}
		return nil
	})

	if err != nil {
		return 0, 0, err
	}

	return size, count, nil
}

// GetImageInfo 获取镜像信息
func (m *Manager) GetImageInfo() (*ImageInfo, error) {
	wimPath := filepath.Join(m.config.Tiny11Dir, "sources", "install.wim")

	if !utils.FileExists(wimPath) {
		return nil, types.NewError(types.ErrCodeNotFound, "install.wim不存在", nil).
			WithContext("path", wimPath)
	}

	m.log.Section("获取镜像信息")

	output, err := utils.RunCommand("dism", "/English", "/Get-WimInfo",
		fmt.Sprintf("/WimFile:%s", wimPath))
	if err != nil {
		return nil, types.NewError(types.ErrCodeDISM, "获取镜像信息失败", err)
	}

	// 显示镜像列表
	fmt.Println()
	lines := m.parseImageList(output)
	for _, line := range lines {
		if strings.HasPrefix(line, "Index :") {
			fmt.Println(utils.Colorize(line, utils.MikuCyan))
		} else if strings.Contains(line, "Name :") {
			fmt.Println(utils.Colorize(line, utils.MikuPink))
		} else {
			fmt.Println(utils.Colorize(line, utils.MikuWhite))
		}
	}
	fmt.Println()

	// 选择索引
	index := m.config.ImageIndex
	if index == 0 {
		fmt.Print(utils.Colorize("请输入镜像索引: ", utils.MikuPink))
		fmt.Scanln(&index)
	}

	// 验证索引
	availableIndices := m.extractAvailableIndices(output)
	if !m.isValidIndex(index, availableIndices) {
		return nil, types.NewError(types.ErrCodeInvalidInput, "无效的镜像索引", nil).
			WithContext("index", index)
	}

	// 获取详细信息
	spinner := utils.NewSpinner("读取镜像详细信息...")
	spinner.Start()

	output, err = utils.RunCommand("dism", "/English", "/Get-WimInfo",
		fmt.Sprintf("/WimFile:%s", wimPath),
		fmt.Sprintf("/Index:%d", index))

	spinner.Stop(err == nil)

	if err != nil {
		return nil, types.NewError(types.ErrCodeDISM, "获取详细信息失败", err)
	}

	// 解析信息
	info := &ImageInfo{Index: index}
	info.Name = utils.ExtractField(output, "Name")
	info.Description = utils.ExtractField(output, "Description")
	info.Architecture = utils.ExtractField(output, "Architecture")

	if info.Architecture == "x64" {
		info.Architecture = "amd64"
	}

	sizeStr := utils.ExtractField(output, "Size")
	info.Size = m.parseSizeString(sizeStr)

	// 检测语言
	info.Language = m.detectLanguage(wimPath, index)

	m.info = info

	// 显示详情
	fmt.Println()
	m.log.Info("镜像详情:")
	fmt.Printf("  %s %s\n", utils.Colorize("名称:", utils.MikuCyan),
		utils.Colorize(info.Name, utils.MikuWhite))
	fmt.Printf("  %s %s\n", utils.Colorize("架构:", utils.MikuCyan),
		utils.Colorize(info.Architecture, utils.MikuWhite))
	fmt.Printf("  %s %s\n", utils.Colorize("语言:", utils.MikuCyan),
		utils.Colorize(info.Language, utils.MikuWhite))
	fmt.Printf("  %s %s\n", utils.Colorize("大小:", utils.MikuCyan),
		utils.Colorize(utils.FormatBytes(info.Size), utils.MikuWhite))
	fmt.Println()

	return info, nil
}

// detectLanguage 检测系统语言
func (m *Manager) detectLanguage(wimPath string, index int) string {
	mountPath := m.config.ScratchDir
	os.MkdirAll(mountPath, 0755)

	spinner := utils.NewSpinner("检测系统语言...")
	spinner.Start()

	// 临时挂载（只读）
	_, err := utils.RunCommand("dism", "/English",
		"/Mount-Image",
		fmt.Sprintf("/ImageFile:%s", wimPath),
		fmt.Sprintf("/Index:%d", index),
		fmt.Sprintf("/MountDir:%s", mountPath),
		"/ReadOnly")

	language := "en-US"

	if err == nil {
		// 获取语言信息
		langOutput, langErr := utils.RunCommand("dism", "/English",
			"/Get-Intl",
			fmt.Sprintf("/Image:%s", mountPath))

		if langErr == nil {
			language = utils.ExtractLanguage(langOutput)
		}

		// 卸载
		utils.RunCommand("dism", "/English",
			"/Unmount-Image",
			fmt.Sprintf("/MountDir:%s", mountPath),
			"/Discard")
	}

	spinner.Stop(err == nil)

	return language
}

// MountInstallWim 挂载install.wim
func (m *Manager) MountInstallWim(index int) error {
	wimPath := filepath.Join(m.config.Tiny11Dir, "sources", "install.wim")
	mountPath := m.config.ScratchDir

	wimPath, _ = filepath.Abs(wimPath)
	mountPath, _ = filepath.Abs(mountPath)

	if !utils.FileExists(wimPath) {
		return types.NewError(types.ErrCodeNotFound, "install.wim不存在", nil).
			WithContext("path", wimPath)
	}

	m.log.Info("准备挂载install.wim (索引: %d)...", index)
	m.log.Info("WIM路径: %s", wimPath)
	m.log.Info("挂载点: %s", mountPath)

	// 获取文件权限
	if err := utils.Takeown(wimPath); err != nil {
		m.log.Warn("获取文件所有权失败: %v", err)
	}
	if err := utils.GrantPermission(wimPath); err != nil {
		m.log.Warn("设置文件权限失败: %v", err)
	}
	os.Chmod(wimPath, 0666)

	// 清理现有挂载
	if utils.DirExists(mountPath) {
		m.log.Info("清理现有挂载目录...")
		utils.RunCommand("dism", "/English",
			"/Unmount-Image",
			fmt.Sprintf("/MountDir:%s", mountPath),
			"/Discard")

		time.Sleep(2 * time.Second)

		if err := os.RemoveAll(mountPath); err != nil {
			m.log.Warn("删除挂载目录失败，尝试获取权限...")
			utils.TakeownRecursive(mountPath)
			utils.GrantPermissionRecursive(mountPath)
			time.Sleep(1 * time.Second)

			if err := os.RemoveAll(mountPath); err != nil {
				return types.NewError(types.ErrCodePermission, "无法清理挂载目录", err)
			}
		}

		m.log.Success("挂载目录已清理")
	}

	// 创建挂载目录
	if err := os.MkdirAll(mountPath, 0755); err != nil {
		return types.NewError(types.ErrCodePermission, "创建挂载目录失败", err)
	}

	// 验证目录为空
	entries, err := os.ReadDir(mountPath)
	if err != nil {
		return types.NewError(types.ErrCodeGeneral, "读取挂载目录失败", err)
	}

	if len(entries) > 0 {
		return types.NewError(types.ErrCodeGeneral, "挂载目录不为空", nil).
			WithContext("count", len(entries))
	}

	m.log.Success("挂载目录准备完成")

	// 挂载镜像
	spinner := utils.NewSpinner(fmt.Sprintf("挂载install.wim (索引 %d)", index))
	spinner.Start()

	output, err := utils.RunCommand("dism", "/English",
		"/Mount-Image",
		fmt.Sprintf("/ImageFile:%s", wimPath),
		fmt.Sprintf("/Index:%d", index),
		fmt.Sprintf("/MountDir:%s", mountPath))

	spinner.Stop(err == nil)

	if err != nil {
		m.log.Error("DISM输出: %s", output)
		return types.NewError(types.ErrCodeDISM, "挂载失败", err)
	}

	m.log.Success("镜像挂载成功")
	return nil
}

// MountBootWim 挂载boot.wim
func (m *Manager) MountBootWim() error {
	wimPath := filepath.Join(m.config.Tiny11Dir, "sources", "boot.wim")
	mountPath := m.config.ScratchDir

	wimPath, _ = filepath.Abs(wimPath)
	mountPath, _ = filepath.Abs(mountPath)

	if !utils.FileExists(wimPath) {
		return types.NewError(types.ErrCodeNotFound, "boot.wim不存在", nil).
			WithContext("path", wimPath)
	}

	m.log.Info("准备挂载boot.wim...")

	// 获取权限
	utils.Takeown(wimPath)
	utils.GrantPermission(wimPath)
	os.Chmod(wimPath, 0666)

	// 清理现有挂载
	if utils.DirExists(mountPath) {
		m.log.Info("清理现有挂载目录...")
		utils.RunCommand("dism", "/English",
			"/Unmount-Image",
			fmt.Sprintf("/MountDir:%s", mountPath),
			"/Discard")

		time.Sleep(2 * time.Second)

		if err := os.RemoveAll(mountPath); err != nil {
			utils.TakeownRecursive(mountPath)
			utils.GrantPermissionRecursive(mountPath)
			time.Sleep(1 * time.Second)
			os.RemoveAll(mountPath)
		}
	}

	os.MkdirAll(mountPath, 0755)

	// 挂载boot.wim的索引2
	spinner := utils.NewSpinner("挂载boot.wim (索引 2)")
	spinner.Start()

	_, err := utils.RunCommand("dism", "/English",
		"/Mount-Image",
		fmt.Sprintf("/ImageFile:%s", wimPath),
		"/Index:2",
		fmt.Sprintf("/MountDir:%s", mountPath))

	spinner.Stop(err == nil)

	if err != nil {
		return types.NewError(types.ErrCodeDISM, "挂载boot.wim失败", err)
	}

	m.log.Success("boot.wim挂载成功")
	return nil
}

// UnmountImage 卸载镜像
func (m *Manager) UnmountImage(commit bool) error {
	mountPath := m.config.ScratchDir

	if !m.isMounted(mountPath) {
		m.log.Skip("镜像未挂载，无需卸载")
		return nil
	}

	action := "/Discard"
	actionDesc := "放弃更改"
	if commit {
		action = "/Commit"
		actionDesc = "保存更改"
	}

	m.log.Info("卸载镜像 (%s)...", actionDesc)

	spinner := utils.NewSpinner(fmt.Sprintf("卸载镜像 (%s)", actionDesc))
	spinner.Start()

	_, err := utils.RunCommand("dism", "/English",
		"/Unmount-Image",
		fmt.Sprintf("/MountDir:%s", mountPath),
		action)

	spinner.Stop(err == nil)

	if err != nil {
		return types.NewError(types.ErrCodeDISM, "卸载失败", err)
	}

	// 强制GC
	runtime.GC()

	m.log.Success("镜像卸载成功")
	return nil
}

// isMounted 检查镜像是否已挂载
func (m *Manager) isMounted(mountPath string) bool {
	if !utils.DirExists(mountPath) {
		return false
	}

	entries, err := os.ReadDir(mountPath)
	if err != nil || len(entries) == 0 {
		return false
	}

	// 检查Windows目录
	windowsDir := filepath.Join(mountPath, "Windows")
	if !utils.DirExists(windowsDir) {
		return false
	}

	// 查询DISM挂载状态
	output, err := utils.RunCommand("dism", "/English", "/Get-MountedImageInfo")
	if err != nil {
		return false
	}

	return strings.Contains(output, mountPath)
}

// CleanupImage 清理镜像
func (m *Manager) CleanupImage() error {
	mountPath := m.config.ScratchDir

	m.log.Info("清理镜像组件存储...")

	spinner := utils.NewSpinner("执行组件清理 (这可能需要几分钟)")
	spinner.Start()

	output, err := utils.RunCommand("dism", "/English",
		fmt.Sprintf("/Image:%s", mountPath),
		"/Cleanup-Image",
		"/StartComponentCleanup",
		"/ResetBase")

	spinner.Stop(err == nil)

	if err != nil {
		m.log.Warn("组件清理失败，将使用延迟清理")

		_, err2 := utils.RunCommand("dism", "/English",
			fmt.Sprintf("/Image:%s", mountPath),
			"/Cleanup-Image",
			"/StartComponentCleanup")

		if err2 != nil {
			return types.NewError(types.ErrCodeDISM, "清理失败", err).
				WithContext("output", output)
		}

		m.log.Info("已设置延迟清理")
		return nil
	}

	// 强制GC
	runtime.GC()

	m.log.Success("镜像清理完成")
	return nil
}

// ExportImage 导出镜像
func (m *Manager) ExportImage(index int) error {
	sourceWim := filepath.Join(m.config.Tiny11Dir, "sources", "install.wim")
	destWim := filepath.Join(m.config.Tiny11Dir, "sources", "install2.wim")

	m.log.Info("导出优化后的镜像...")

	if !utils.FileExists(sourceWim) {
		return types.NewError(types.ErrCodeNotFound, "源WIM文件不存在", nil).
			WithContext("path", sourceWim)
	}

	// 检查磁盘空间
	sourceInfo, err := os.Stat(sourceWim)
	if err != nil {
		return types.NewError(types.ErrCodeGeneral, "无法获取源文件信息", err)
	}

	drive := filepath.VolumeName(destWim)
	if drive == "" {
		drive = "C:"
	}

	freeSpace, err := m.getFreeDiskSpace(drive)
	if err != nil {
		m.log.Warn("无法获取磁盘空间信息: %v", err)
	} else {
		requiredSpace := sourceInfo.Size() + (2 * 1024 * 1024 * 1024) // 额外2GB缓冲

		m.log.Info("磁盘空间检查:")
		m.log.Info("  可用空间: %s", utils.FormatBytes(int64(freeSpace)))
		m.log.Info("  需要空间: %s", utils.FormatBytes(requiredSpace))

		if int64(freeSpace) < requiredSpace {
			return types.NewError(types.ErrCodeDiskSpace, "磁盘空间不足", nil).
				WithContext("available", freeSpace).
				WithContext("required", requiredSpace)
		}
	}

	// 删除旧文件
	if utils.FileExists(destWim) {
		m.log.Info("删除旧的导出文件...")
		utils.Takeown(destWim)
		utils.GrantPermission(destWim)
		os.Chmod(destWim, 0666)

		if err := os.Remove(destWim); err != nil {
			return types.NewError(types.ErrCodePermission, "无法删除旧文件", err).
				WithContext("path", destWim)
		}

		m.log.Success("旧文件已删除")
	}

	// 导出镜像（带重试）
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			m.log.Info("第 %d 次重试...", attempt)
			time.Sleep(2 * time.Second)
		}

		spinner := utils.NewSpinner(fmt.Sprintf("导出镜像 (recovery压缩) - 尝试 %d/%d",
			attempt, maxRetries))
		spinner.Start()

		output, err := utils.RunCommand("dism", "/English",
			"/Export-Image",
			fmt.Sprintf("/SourceImageFile:%s", sourceWim),
			fmt.Sprintf("/SourceIndex:%d", index),
			fmt.Sprintf("/DestinationImageFile:%s", destWim),
			"/Compress:recovery",
			"/CheckIntegrity")

		spinner.Stop(err == nil)

		if err == nil {
			m.log.Success("镜像导出成功")
			break
		}

		lastErr = err
		m.log.Warn("导出失败: %v", err)

		if output != "" {
			m.log.Info("DISM 输出: %s", output)
		}

		// 清理失败的文件
		if utils.FileExists(destWim) {
			os.Remove(destWim)
		}

		if attempt == maxRetries {
			return types.NewError(types.ErrCodeDISM, "导出失败", lastErr).
				WithContext("attempts", maxRetries).
				WithContext("output", output)
		}
	}

	// 验证导出的文件
	if !utils.FileExists(destWim) {
		return types.NewError(types.ErrCodeGeneral, "导出后文件不存在", nil)
	}

	destInfo, err := os.Stat(destWim)
	if err != nil {
		return types.NewError(types.ErrCodeGeneral, "无法获取导出文件信息", err)
	}

	if destInfo.Size() < 100*1024*1024 { // 小于100MB可能损坏
		return types.NewError(types.ErrCodeGeneral, "导出的文件太小，可能损坏", nil).
			WithContext("size", destInfo.Size())
	}

	m.log.Info("导出文件大小: %s", utils.FormatBytes(destInfo.Size()))

	// 替换原始文件
	m.log.Info("替换原始WIM文件...")

	utils.Takeown(sourceWim)
	utils.GrantPermission(sourceWim)
	os.Chmod(sourceWim, 0666)

	if err := os.Remove(sourceWim); err != nil {
		return types.NewError(types.ErrCodePermission, "删除原文件失败", err)
	}

	if err := os.Rename(destWim, sourceWim); err != nil {
		return types.NewError(types.ErrCodeGeneral, "重命名文件失败", err)
	}

	// 验证最终文件
	finalInfo, err := os.Stat(sourceWim)
	if err != nil {
		return types.NewError(types.ErrCodeGeneral, "验证最终文件失败", err)
	}

	compressionRatio := float64(sourceInfo.Size()-finalInfo.Size()) / float64(sourceInfo.Size()) * 100

	m.log.Success("镜像导出完成")
	m.log.Info("  原始大小: %s", utils.FormatBytes(sourceInfo.Size()))
	m.log.Info("  压缩后:   %s", utils.FormatBytes(finalInfo.Size()))
	m.log.Info("  压缩率:   %.1f%%", compressionRatio)

	// 强制GC
	runtime.GC()

	return nil
}

// getFreeDiskSpace 获取磁盘剩余空间
func (m *Manager) getFreeDiskSpace(drive string) (uint64, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	if !strings.HasSuffix(drive, "\\") {
		drive = drive + "\\"
	}

	drivePtr, err := syscall.UTF16PtrFromString(drive)
	if err != nil {
		return 0, err
	}

	var freeBytesAvailable uint64
	var totalBytes uint64
	var totalFreeBytes uint64

	ret, _, _ := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(drivePtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)))

	if ret == 0 {
		return 0, fmt.Errorf("无法获取磁盘空间")
	}

	return freeBytesAvailable, nil
}

// CreateISO 创建ISO镜像
func (m *Manager) CreateISO() (string, error) {
	m.log.Section("创建ISO镜像")

	// 复制autounattend.xml
	autoUnattendSrc := filepath.Join(m.config.ResourcesDir, "autounattend.xml")
	autoUnattendDst := filepath.Join(m.config.Tiny11Dir, "autounattend.xml")

	if utils.FileExists(autoUnattendSrc) {
		m.log.Info("复制无人值守配置文件...")
		if err := utils.CopyFile(autoUnattendSrc, autoUnattendDst); err != nil {
			m.log.Warn("复制autounattend.xml失败: %v", err)
		}
	} else {
		m.log.Warn("未找到autounattend.xml")
	}

	// 查找oscdimg
	oscdimg, err := m.findOscdimg()
	if err != nil {
		return "", err
	}

	// 验证引导文件
	etfsboot := filepath.Join(m.config.Tiny11Dir, "boot", "etfsboot.com")
	efisys := filepath.Join(m.config.Tiny11Dir, "efi", "microsoft", "boot", "efisys.bin")

	if !utils.FileExists(etfsboot) {
		return "", types.NewError(types.ErrCodeNotFound, "未找到BIOS引导文件", nil).
			WithContext("path", etfsboot)
	}

	if !utils.FileExists(efisys) {
		return "", types.NewError(types.ErrCodeNotFound, "未找到UEFI引导文件", nil).
			WithContext("path", efisys)
	}

	m.log.Info("正在构建ISO镜像文件...")
	m.log.Info("输出路径: %s", m.config.OutputISO)

	bootData := fmt.Sprintf("2#p0,e,b%s#pEF,e,b%s", etfsboot, efisys)

	spinner := utils.NewSpinner("创建ISO镜像 (这可能需要几分钟)")
	spinner.Start()

	_, err = utils.RunCommand(oscdimg,
		"-m",
		"-o",
		"-u2",
		"-udfver102",
		fmt.Sprintf("-bootdata:%s", bootData),
		m.config.Tiny11Dir,
		m.config.OutputISO)

	spinner.Stop(err == nil)

	if err != nil {
		return "", types.NewError(types.ErrCodeGeneral, "创建ISO失败", err)
	}

	// 验证ISO文件
	if !utils.FileExists(m.config.OutputISO) {
		return "", types.NewError(types.ErrCodeGeneral, "ISO文件创建后未找到", nil)
	}

	fileInfo, _ := os.Stat(m.config.OutputISO)
	if fileInfo != nil {
		m.log.Success("ISO创建成功 (大小: %s)", utils.FormatBytes(fileInfo.Size()))
	} else {
		m.log.Success("ISO创建成功")
	}

	return m.config.OutputISO, nil
}

// findOscdimg 查找oscdimg.exe
func (m *Manager) findOscdimg() (string, error) {
	// 搜索路径优先级
	searchPaths := []string{
		// 1. 本地目录
		filepath.Join(m.config.WorkDir, "oscdimg.exe"),
		filepath.Join(m.config.TempDir, "oscdimg.exe"),

		// 2. ADK 标准路径 (amd64)
		`C:\Program Files (x86)\Windows Kits\10\Assessment and Deployment Kit\Deployment Tools\amd64\Oscdimg\oscdimg.exe`,

		// 3. ADK x86 路径
		`C:\Program Files (x86)\Windows Kits\10\Assessment and Deployment Kit\Deployment Tools\x86\Oscdimg\oscdimg.exe`,

		// 4. ADK ARM64 路径
		`C:\Program Files (x86)\Windows Kits\10\Assessment and Deployment Kit\Deployment Tools\arm64\Oscdimg\oscdimg.exe`,

		// 5. 其他可能的位置
		`C:\Windows\System32\oscdimg.exe`,
	}

	// 优先使用已存在的
	for _, path := range searchPaths {
		if utils.FileExists(path) {
			m.log.Info("找到 oscdimg: %s", path)
			return path, nil
		}
	}

	// 下载
	m.log.Info("未找到 oscdimg.exe，正在从Microsoft下载...")
	localPath := filepath.Join(m.config.TempDir, "oscdimg.exe")

	url := "https://msdl.microsoft.com/download/symbols/oscdimg.exe/3D44737265000/oscdimg.exe"

	spinner := utils.NewSpinner("下载 oscdimg.exe")
	spinner.Start()

	err := utils.DownloadFile(url, localPath)

	spinner.Stop(err == nil)

	if err != nil {
		return "", types.NewError(types.ErrCodeNetwork, "下载oscdimg失败", err)
	}

	m.log.Success("oscdimg.exe 下载完成")
	return localPath, nil
}

// Cleanup 清理临时文件
func (m *Manager) Cleanup() error {
	m.log.Info("清理临时文件...")

	cleaned := 0
	errors := 0

	// 清理build目录
	if utils.DirExists(m.config.Tiny11Dir) {
		if err := os.RemoveAll(m.config.Tiny11Dir); err != nil {
			m.log.Warn("无法删除tiny11目录: %v", err)
			errors++
		} else {
			cleaned++
		}
	}

	if utils.DirExists(m.config.ScratchDir) {
		if err := os.RemoveAll(m.config.ScratchDir); err != nil {
			m.log.Warn("无法删除scratchdir目录: %v", err)
			errors++
		} else {
			cleaned++
		}
	}

	if utils.DirExists(m.config.TempDir) {
		if err := os.RemoveAll(m.config.TempDir); err != nil {
			m.log.Warn("无法删除temp目录: %v", err)
			errors++
		} else {
			cleaned++
		}
	}

	// 强制GC
	runtime.GC()

	if errors > 0 {
		m.log.Warn("清理完成，但有%d个错误", errors)
	} else {
		m.log.Success("临时文件清理完成 (清理了%d个项目)", cleaned)
	}

	return nil
}

// 辅助方法

func (m *Manager) parseImageList(output string) []string {
	var lines []string
	outputLines := strings.Split(output, "\n")

	for _, line := range outputLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lines = append(lines, trimmed)
	}

	return lines
}

func (m *Manager) extractAvailableIndices(output string) []int {
	var indices []int
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Index :") {
			var index int
			if _, err := fmt.Sscanf(trimmed, "Index : %d", &index); err == nil {
				indices = append(indices, index)
			}
		}
	}

	return indices
}

func (m *Manager) isValidIndex(index int, availableIndices []int) bool {
	for _, availIdx := range availableIndices {
		if index == availIdx {
			return true
		}
	}
	return false
}

func (m *Manager) parseSizeString(sizeStr string) int64 {
	sizeStr = strings.ReplaceAll(sizeStr, ",", "")
	sizeStr = strings.ReplaceAll(sizeStr, " bytes", "")
	sizeStr = strings.TrimSpace(sizeStr)

	var size int64
	fmt.Sscanf(sizeStr, "%d", &size)

	return size
}