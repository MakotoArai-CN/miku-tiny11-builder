package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"tiny11-builder/internal/config"
	"tiny11-builder/internal/logger"
	"tiny11-builder/internal/remover"
	"tiny11-builder/internal/utils"
)

type Tiny11NanoBuilder struct {
	*Tiny11CoreBuilder
	nanoRemover *remover.NanoRemover
}

func NewTiny11NanoBuilder(cfg *config.Config, log *logger.Logger) *Tiny11NanoBuilder {
	return &Tiny11NanoBuilder{
		Tiny11CoreBuilder: NewTiny11CoreBuilder(cfg, log),
		nanoRemover:       remover.NewNanoRemover(cfg, log),
	}
}

func (b *Tiny11NanoBuilder) Build() error {
	b.log.Header("Tiny11 Nano Builder - 终极精简版本")
	b.log.Warn("⚠️  警告：此版本将移除几乎所有可移除组件，仅用于极端测试场景！")

	var imageUnmounted = false

	// 步骤 1-2: 基础验证
	b.log.Step(1, "验证 ISO 镜像")
	if err := b.imgMgr.ValidateISO(); err != nil {
		return fmt.Errorf("ISO 验证失败: %w", err)
	}

	b.log.Step(2, "复制 Windows 镜像文件")
	if err := b.imgMgr.CopyImageFiles(); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}

	// 步骤 3: 获取镜像信息
	b.log.Step(3, "获取镜像信息")
	imageInfo, err := b.imgMgr.GetImageInfo()
	if err != nil {
		return fmt.Errorf("获取镜像信息失败: %w", err)
	}

	b.log.Info("架构: %s, 语言: %s, 索引: %d",
		imageInfo.Architecture, imageInfo.Language, imageInfo.Index)

	// 步骤 4: 挂载镜像
	b.log.Step(4, "挂载 install.wim")
	if err := b.imgMgr.MountInstallWim(imageInfo.Index); err != nil {
		return fmt.Errorf("挂载失败: %w", err)
	}

	defer func() {
		if !imageUnmounted {
			b.log.Info("执行紧急清理...")
			b.regMgr.UnloadHives()
			b.imgMgr.UnmountImage(false)
		}
	}()

	// 步骤 5: 主动获取文件夹所有权（预防性措施）
	b.log.Step(5, "预防性获取关键文件夹所有权")
	if err := b.proactivelyTakeOwnership(); err != nil {
		b.log.Warn("获取所有权失败（部分）: %v", err)
	}

	// 步骤 6: 移除预装应用
	b.log.Step(6, "移除预装应用")
	if err := b.remover.RemoveProvisionedApps(); err != nil {
		return fmt.Errorf("移除应用失败: %w", err)
	}

	// 步骤 7: 移除扩展应用 (Nano 特有)
	b.log.Step(7, "移除扩展应用列表 (Nano)")
	if err := b.nanoRemover.RemoveAggressiveApps(); err != nil {
		b.log.Warn("移除扩展应用失败: %v", err)
	}

	// 步骤 8: 移除系统包
	b.log.Step(8, "移除系统组件包 (Nano)")
	if err := b.nanoRemover.RemoveAggressivePackages(imageInfo.Language); err != nil {
		b.log.Warn("移除系统包失败: %v", err)
	}

	// 步骤 9: 移除 .NET Native Images
	b.log.Step(9, "移除预编译 .NET 程序集")
	if err := b.nanoRemover.RemoveNativeImages(); err != nil {
		b.log.Warn("移除 Native Images 失败: %v", err)
	}

	// 步骤 10: 精简 DriverStore
	b.log.Step(10, "精简驱动程序存储")
	if err := b.nanoRemover.SlimDriverStore(); err != nil {
		b.log.Warn("精简 DriverStore 失败: %v", err)
	}

	// 步骤 11: 精简字体
	b.log.Step(11, "精简系统字体")
	if err := b.nanoRemover.SlimFonts(); err != nil {
		b.log.Warn("精简字体失败: %v", err)
	}

	// 步骤 12: 移除系统文件夹
	b.log.Step(12, "移除非必需系统文件夹")
	if err := b.nanoRemover.RemoveSystemFolders(); err != nil {
		b.log.Warn("移除系统文件夹失败: %v", err)
	}

	// 步骤 13: 移除 Edge、OneDrive 和 WinRE
	b.log.Step(13, "移除 Edge、OneDrive 和 WinRE")
	b.remover.RemoveEdge()
	b.remover.RemoveOneDrive()
	b.coreRemover.RemoveWinRE()

	// 步骤 14: 组件清理
	b.log.Step(14, "清理镜像组件")
	if err := b.imgMgr.CleanupImage(); err != nil {
		b.log.Warn("清理失败（继续）: %v", err)
	}

	// 步骤 15: WinSxS 精简
	b.log.Step(15, "精简 WinSxS 组件存储")
	if err := b.coreRemover.RemoveWinSxS(); err != nil {
		return fmt.Errorf("精简 WinSxS 失败: %w", err)
	}

	// 步骤 16: 应用注册表优化
	b.log.Step(16, "应用注册表优化")
	if err := b.regMgr.LoadHives(); err != nil {
		return fmt.Errorf("加载注册表失败: %w", err)
	}

	b.regMgr.ApplyTweaks()
	b.regMgr.ApplyCoreTweaks()
	b.regMgr.ApplyNanoTweaks() // Nano 特有优化

	b.regMgr.UnloadHives()

	// 步骤 17: 移除系统服务
	b.log.Step(17, "移除非必需系统服务")
	if err := b.nanoRemover.RemoveSystemServices(); err != nil {
		b.log.Warn("移除服务失败: %v", err)
	}

	// 步骤 18: 复制 autounattend.xml
	b.copyAutounattend()

	// 步骤 19: 卸载镜像
	b.log.Step(19, "卸载并提交更改")
	if err := b.imgMgr.UnmountImage(true); err != nil {
		return fmt.Errorf("卸载失败: %w", err)
	}
	imageUnmounted = true

	// 步骤 20: 导出为 ESD 格式
	b.log.Step(20, "导出为 ESD 格式 (超高压缩)")
	if err := b.exportImageToESD(imageInfo.Index); err != nil {
		return fmt.Errorf("导出 ESD 失败: %w", err)
	}

	// 步骤 21: 处理 boot.wim
	b.log.Step(21, "精简 boot.wim")
	if err := b.processNanoBootWim(); err != nil {
		return fmt.Errorf("处理 boot.wim 失败: %w", err)
	}

	// 步骤 22: 清理 ISO 根目录
	b.log.Step(22, "清理 ISO 根目录")
	if err := b.cleanupISORoot(); err != nil {
		b.log.Warn("清理 ISO 根目录失败: %v", err)
	}

	// 步骤 23: 创建 ISO
	b.log.Step(23, "创建 ISO 镜像")
	isoPath, err := b.imgMgr.CreateISO()
	if err != nil {
		return fmt.Errorf("创建 ISO 失败: %w", err)
	}
	b.outputISO = isoPath

	// 步骤 24: 清理临时文件
	b.log.Step(24, "清理临时文件")
	b.imgMgr.Cleanup()

	// 强制 GC
	runtime.GC()

	return nil
}

// proactivelyTakeOwnership 主动获取关键文件夹所有权
func (b *Tiny11NanoBuilder) proactivelyTakeOwnership() error {
	scratchDir := b.config.ScratchDir

	// 基于 nano11builder.ps1 的文件夹列表
	foldersToOwn := []string{
		filepath.Join(scratchDir, "Windows", "System32", "DriverStore", "FileRepository"),
		filepath.Join(scratchDir, "Windows", "Fonts"),
		filepath.Join(scratchDir, "Windows", "Web"),
		filepath.Join(scratchDir, "Windows", "Help"),
		filepath.Join(scratchDir, "Windows", "Cursors"),
		filepath.Join(scratchDir, "Program Files (x86)", "Microsoft"),
		filepath.Join(scratchDir, "Program Files", "WindowsApps"),
		filepath.Join(scratchDir, "Windows", "System32", "Microsoft-Edge-Webview"),
		filepath.Join(scratchDir, "Windows", "System32", "Recovery"),
		filepath.Join(scratchDir, "Windows", "WinSxS"),
		filepath.Join(scratchDir, "Windows", "assembly"),
		filepath.Join(scratchDir, "ProgramData", "Microsoft", "Windows Defender"),
		filepath.Join(scratchDir, "Windows", "System32", "InputMethod"),
		filepath.Join(scratchDir, "Windows", "Speech"),
		filepath.Join(scratchDir, "Windows", "Temp"),
	}

	filesToOwn := []string{
		filepath.Join(scratchDir, "Windows", "System32", "OneDriveSetup.exe"),
	}

	b.log.Info("主动获取关键文件夹所有权...")

	ownedFolders := 0
	for _, folder := range foldersToOwn {
		if !utils.DirExists(folder) {
			continue
		}

		b.log.Info("  获取所有权: %s", filepath.Base(folder))

		// 递归获取所有权
		if err := utils.TakeownRecursive(folder); err != nil {
			b.log.Warn("  ✗ takeown 失败: %v", err)
			continue
		}

		// 授予完全控制权限
		if err := utils.GrantPermissionRecursive(folder); err != nil {
			b.log.Warn("  ✗ icacls 失败: %v", err)
			continue
		}

		ownedFolders++
	}

	ownedFiles := 0
	for _, file := range filesToOwn {
		if !utils.FileExists(file) {
			continue
		}

		b.log.Info("  获取文件所有权: %s", filepath.Base(file))

		if err := utils.Takeown(file); err != nil {
			b.log.Warn("  ✗ takeown 失败: %v", err)
			continue
		}

		if err := utils.GrantPermission(file); err != nil {
			b.log.Warn("  ✗ icacls 失败: %v", err)
			continue
		}

		ownedFiles++
	}

	b.log.Success("获取了 %d 个文件夹和 %d 个文件的所有权", ownedFolders, ownedFiles)
	return nil
}

// exportImageToESD 导出为 ESD 格式
func (b *Tiny11NanoBuilder) exportImageToESD(index int) error {
	sourceWim := filepath.Join(b.config.Tiny11Dir, "sources", "install.wim")
	destEsd := filepath.Join(b.config.Tiny11Dir, "sources", "install.esd")

	b.log.Info("导出为 ESD 格式 (recovery 压缩)...")

	if !utils.FileExists(sourceWim) {
		return fmt.Errorf("源 WIM 文件不存在: %s", sourceWim)
	}

	// 删除旧的 ESD 文件
	if utils.FileExists(destEsd) {
		os.Remove(destEsd)
	}

	spinner := utils.NewSpinner("导出为 ESD 格式 (这将花费较长时间但文件更小)")
	spinner.Start()

	_, err := utils.RunCommand("dism", "/English",
		"/Export-Image",
		fmt.Sprintf("/SourceImageFile:%s", sourceWim),
		"/SourceIndex:1",
		fmt.Sprintf("/DestinationImageFile:%s", destEsd),
		"/Compress:recovery")

	spinner.Stop(err == nil)

	if err != nil {
		return fmt.Errorf("导出 ESD 失败: %w", err)
	}

	// 验证 ESD 文件
	if !utils.FileExists(destEsd) {
		return fmt.Errorf("ESD 文件创建失败")
	}

	esdInfo, err := os.Stat(destEsd)
	if err != nil {
		return fmt.Errorf("无法获取 ESD 文件信息: %w", err)
	}

	wimInfo, _ := os.Stat(sourceWim)

	b.log.Success("已导出为 ESD 格式")
	if wimInfo != nil {
		b.log.Info("  WIM 大小: %s", utils.FormatBytes(wimInfo.Size()))
	}
	b.log.Info("  ESD 大小: %s", utils.FormatBytes(esdInfo.Size()))

	if wimInfo != nil {
		saved := wimInfo.Size() - esdInfo.Size()
		ratio := float64(saved) / float64(wimInfo.Size()) * 100
		b.log.Info("  节省空间: %s (%.1f%%)", utils.FormatBytes(saved), ratio)
	}

	// 删除 WIM 文件
	os.Remove(sourceWim)

	return nil
}

// processNanoBootWim 处理 boot.wim (Nano 版本)
func (b *Tiny11NanoBuilder) processNanoBootWim() error {
	bootWimPath := filepath.Join(b.config.Tiny11Dir, "sources", "boot.wim")
	newBootWimPath := filepath.Join(b.config.Tiny11Dir, "sources", "boot_new.wim")
	finalBootWimPath := filepath.Join(b.config.Tiny11Dir, "sources", "boot_final.wim")

	b.log.Info("获取 boot.wim 所有权...")
	utils.Takeown(bootWimPath)
	utils.GrantPermission(bootWimPath)
	os.Chmod(bootWimPath, 0666)

	// 导出索引 2 (Setup)
	b.log.Info("导出 boot.wim 索引 2...")
	spinner := utils.NewSpinner("导出 boot.wim")
	spinner.Start()

	_, err := utils.RunCommand("dism", "/English",
		"/Export-Image",
		fmt.Sprintf("/SourceImageFile:%s", bootWimPath),
		"/SourceIndex:2",
		fmt.Sprintf("/DestinationImageFile:%s", newBootWimPath))

	spinner.Stop(err == nil)

	if err != nil {
		return fmt.Errorf("导出 boot.wim 失败: %w", err)
	}

	// 挂载导出的镜像
	b.log.Info("挂载 boot.wim...")
	if err := b.imgMgr.MountBootWim(); err != nil {
		// 使用新导出的 WIM
		mountPath := b.config.ScratchDir
		_, err = utils.RunCommand("dism", "/English",
			"/Mount-Image",
			fmt.Sprintf("/ImageFile:%s", newBootWimPath),
			"/Index:1",
			fmt.Sprintf("/MountDir:%s", mountPath))

		if err != nil {
			return fmt.Errorf("挂载 boot.wim 失败: %w", err)
		}
	}

	// 应用注册表优化
	if err := b.regMgr.LoadHives(); err != nil {
		b.log.Warn("加载 boot.wim 注册表失败: %v", err)
	} else {
		b.regMgr.ApplyBootTweaks()
		b.regMgr.UnloadHives()
	}

	// 卸载
	b.log.Info("卸载 boot.wim...")
	if err := b.imgMgr.UnmountImage(true); err != nil {
		b.log.Warn("卸载 boot.wim 失败: %v", err)
	}

	// 等待系统释放文件
	runtime.GC()
	utils.Sleep(5)

	// 删除原始 boot.wim
	utils.Takeown(bootWimPath)
	utils.GrantPermission(bootWimPath)
	os.Chmod(bootWimPath, 0666)
	os.Remove(bootWimPath)

	// 压缩导出最终版本
	b.log.Info("压缩 boot.wim...")
	spinner = utils.NewSpinner("最终压缩 boot.wim")
	spinner.Start()

	_, err = utils.RunCommand("dism", "/English",
		"/Export-Image",
		fmt.Sprintf("/SourceImageFile:%s", newBootWimPath),
		"/SourceIndex:1",
		fmt.Sprintf("/DestinationImageFile:%s", finalBootWimPath),
		"/Compress:max")

	spinner.Stop(err == nil)

	if err != nil {
		return fmt.Errorf("压缩 boot.wim 失败: %w", err)
	}

	// 清理中间文件并重命名
	os.Remove(newBootWimPath)
	os.Rename(finalBootWimPath, bootWimPath)

	b.log.Success("boot.wim 处理完成")
	return nil
}

// cleanupISORoot 清理 ISO 根目录
func (b *Tiny11NanoBuilder) cleanupISORoot() error {
	isoRoot := b.config.Tiny11Dir

	// 保留的文件/文件夹（基于 nano11builder.ps1）
	keepList := map[string]bool{
		"boot":             true,
		"efi":              true,
		"sources":          true,
		"bootmgr":          true,
		"bootmgr.efi":      true,
		"setup.exe":        true,
		"autounattend.xml": true,
	}

	entries, err := os.ReadDir(isoRoot)
	if err != nil {
		return err
	}

	removed := 0
	for _, entry := range entries {
		entryNameLower := strings.ToLower(entry.Name())

		if !keepList[entryNameLower] {
			path := filepath.Join(isoRoot, entry.Name())
			b.log.Info("移除非必需项: %s", entry.Name())

			if err := os.RemoveAll(path); err != nil {
				b.log.Warn("  ✗ 失败: %v", err)
			} else {
				removed++
			}
		}
	}

	b.log.Success("清理了 %d 个非必需文件/文件夹", removed)
	return nil
}