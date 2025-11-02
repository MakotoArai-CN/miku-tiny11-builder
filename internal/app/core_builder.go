package app

import (
	"fmt"
	"tiny11-builder/internal/config"
	"tiny11-builder/internal/logger"
	"tiny11-builder/internal/remover"
	"tiny11-builder/internal/utils"
)

// Tiny11CoreBuilder Core版构建器
type Tiny11CoreBuilder struct {
	*Tiny11Builder
	coreRemover *remover.CoreRemover
}

// NewTiny11CoreBuilder 创建Core版构建器
func NewTiny11CoreBuilder(cfg *config.Config, log *logger.Logger) *Tiny11CoreBuilder {
	return &Tiny11CoreBuilder{
		Tiny11Builder: NewTiny11Builder(cfg, log),
		coreRemover:   remover.NewCoreRemover(cfg, log),
	}
}

// Build 执行Core版构建流程
func (b *Tiny11CoreBuilder) Build() error {
	b.log.Header("Tiny11 Core Builder - 不可服务版本")

	// 执行基础构建流程（步骤1-8）
	// 需要重写以插入Core特定步骤
	var imageUnmounted = false

	// 步骤 1-4: 与标准版相同
	b.log.Step(1, "验证ISO镜像")
	if err := b.imgMgr.ValidateISO(); err != nil {
		return fmt.Errorf("ISO验证失败: %w", err)
	}

	b.log.Step(2, "复制Windows镜像文件")
	if err := b.imgMgr.CopyImageFiles(); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}

	b.log.Step(3, "获取镜像信息")
	imageInfo, err := b.imgMgr.GetImageInfo()
	if err != nil {
		return fmt.Errorf("获取镜像信息失败: %w", err)
	}

	b.log.Step(4, "挂载install.wim")
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

	// 步骤 5-6: 移除应用
	b.log.Step(5, "移除预装应用")
	if err := b.remover.RemoveProvisionedApps(); err != nil {
		return fmt.Errorf("移除应用失败: %w", err)
	}

	b.log.Step(6, "移除系统组件")
	if err := b.remover.RemoveSystemPackages(imageInfo.Language); err != nil {
		b.log.Warn("移除系统包失败: %v", err)
	}

	// 询问.NET 3.5
	b.log.Step(7, "配置.NET Framework 3.5")
	if err := b.configureNET35(); err != nil {
		b.log.Warn(".NET 3.5配置失败: %v", err)
	}

	// 移除Edge和OneDrive
	b.log.Step(8, "移除Edge和OneDrive")
	b.remover.RemoveEdge()
	b.remover.RemoveOneDrive()

	// Core特有: 移除WinSxS
	b.log.Step(9, "移除WinSxS组件存储 (保留必要组件)")
	if err := b.coreRemover.RemoveWinSxS(); err != nil {
		return fmt.Errorf("移除WinSxS失败: %w", err)
	}

	// Core特有: 移除WinRE
	b.log.Step(10, "移除WinRE恢复环境")
	if err := b.coreRemover.RemoveWinRE(); err != nil {
		b.log.Warn("移除WinRE失败: %v", err)
	}

	// 移除计划任务
	b.log.Step(11, "移除遥测计划任务")
	b.remover.RemoveScheduledTasks()

	// 注册表优化
	b.log.Step(12, "应用注册表优化")
	if err := b.regMgr.LoadHives(); err != nil {
		return fmt.Errorf("加载注册表失败: %w", err)
	}

	b.regMgr.ApplyTweaks()
	b.regMgr.ApplyCoreTweaks()
	b.regMgr.UnloadHives()

	// 复制autounattend
	b.copyAutounattend()

	// 步骤 13+: 导出、Boot、ISO
	b.log.Step(13, "导出优化后的镜像")
	if err := b.imgMgr.UnmountImage(true); err != nil {
		return fmt.Errorf("卸载失败: %w", err)
	}
	imageUnmounted = true

	if err := b.imgMgr.ExportImage(imageInfo.Index); err != nil {
		return fmt.Errorf("导出失败: %w", err)
	}

	b.log.Step(14, "处理boot.wim")
	if err := b.processBootWim(); err != nil {
		return fmt.Errorf("处理boot.wim失败: %w", err)
	}

	b.log.Step(15, "创建ISO镜像")
	isoPath, err := b.imgMgr.CreateISO()
	if err != nil {
		return fmt.Errorf("创建ISO失败: %w", err)
	}
	b.outputISO = isoPath

	b.log.Step(16, "清理临时文件")
	b.imgMgr.Cleanup()

	return nil
}

// configureNET35 配置.NET 3.5
func (b *Tiny11CoreBuilder) configureNET35() error {
	fmt.Println()
	fmt.Print("是否启用.NET Framework 3.5? (安装后无法添加) [y/N]: ")

	var response string
	fmt.Scanln(&response)

	if response == "y" || response == "Y" {
		b.log.Info("启用.NET Framework 3.5...")

		mountPath := b.config.ScratchDir
		sxsPath := b.config.ISODrive + "\\sources\\sxs"

		spinner := utils.NewSpinner("安装.NET Framework 3.5 (这可能需要几分钟)...")
		spinner.Start()

		_, err := utils.RunCommand("dism",
			fmt.Sprintf("/Image:%s", mountPath),
			"/Enable-Feature",
			"/FeatureName:NetFX3",
			"/All",
			fmt.Sprintf("/Source:%s", sxsPath))

		spinner.Stop(err == nil)

		if err != nil {
			return fmt.Errorf("启用.NET 3.5失败: %w", err)
		}

		b.log.Success(".NET Framework 3.5已启用")
	} else {
		b.log.Info("跳过.NET Framework 3.5安装")
	}

	return nil
}