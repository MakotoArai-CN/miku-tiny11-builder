package remover

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"tiny11-builder/internal/config"
	"tiny11-builder/internal/logger"
	"tiny11-builder/internal/utils"
)

// CoreRemover Core版移除器
type CoreRemover struct {
	config *config.Config
	log    *logger.Logger
}

// NewCoreRemover 创建Core版移除器
func NewCoreRemover(cfg *config.Config, log *logger.Logger) *CoreRemover {
	return &CoreRemover{
		config: cfg,
		log:    log,
	}
}

// RemoveWinSxS 移除WinSxS (保留必要组件)
func (r *CoreRemover) RemoveWinSxS() error {
	mountPath := r.config.ScratchDir
	winsxsPath := filepath.Join(mountPath, "Windows", "WinSxS")
	winsxsEditPath := filepath.Join(mountPath, "Windows", "WinSxS_edit")

	r.log.Section("处理WinSxS组件存储")
	r.log.Warn("这将移除大部分WinSxS内容，可能需要30-60分钟")

	// 检查WinSxS是否存在
	if !utils.DirExists(winsxsPath) {
		return fmt.Errorf("WinSxS目录不存在: %s", winsxsPath)
	}

	// 获取WinSxS大小
	spinner := utils.NewSpinner("计算WinSxS大小...")
	spinner.Start()
	originalSize, _ := r.getDirSize(winsxsPath)
	spinner.Stop(true)

	r.log.Info("原始WinSxS大小: %s", utils.FormatBytes(originalSize))

	// 创建编辑目录
	r.log.Info("创建临时工作目录...")
	if err := os.MkdirAll(winsxsEditPath, 0755); err != nil {
		return fmt.Errorf("创建工作目录失败: %w", err)
	}

	// 根据架构获取要保留的目录
	keepDirs := r.getKeepDirs()
	r.log.Info("将保留 %d 类必要组件", len(keepDirs))

	// 复制必要的目录
	totalCopied := 0
	totalFailed := 0

	for i, pattern := range keepDirs {
		r.log.Info("[%d/%d] 保留组件: %s",
			i+1, len(keepDirs),
			utils.Colorize(pattern, utils.MikuYellow))

		matches, err := filepath.Glob(filepath.Join(winsxsPath, pattern))
		if err != nil {
			r.log.Warn("  扫描失败: %v", err)
			continue
		}

		if len(matches) == 0 {
			r.log.Info("  未找到匹配项")
			continue
		}

		for _, match := range matches {
			relPath, _ := filepath.Rel(winsxsPath, match)
			destPath := filepath.Join(winsxsEditPath, relPath)

			info, err := os.Stat(match)
			if err != nil {
				continue
			}

			if info.IsDir() {
				err = r.copyDirQuiet(match, destPath)
			} else {
				err = r.copyFileQuiet(match, destPath)
			}

			if err != nil {
				totalFailed++
			} else {
				totalCopied++
			}
		}
	}

	r.log.Success("  ✓ 复制完成: 成功 %d, 失败 %d", totalCopied, totalFailed)

	// 获取所有权
	r.log.Info("获取WinSxS所有权...")
	spinner = utils.NewSpinner("获取目录所有权 (这可能需要几分钟)...")
	spinner.Start()

	if err := utils.TakeownRecursive(winsxsPath); err != nil {
		spinner.Stop(false)
		r.log.Warn("获取所有权失败: %v", err)
	} else {
		spinner.Stop(true)
	}

	// 设置权限
	r.log.Info("设置目录权限...")
	spinner = utils.NewSpinner("设置完全控制权限...")
	spinner.Start()

	if err := utils.GrantPermissionRecursive(winsxsPath); err != nil {
		spinner.Stop(false)
		r.log.Warn("设置权限失败: %v", err)
	} else {
		spinner.Stop(true)
	}

	// 删除原WinSxS
	r.log.Info("删除原始WinSxS目录...")
	spinner = utils.NewSpinner("删除WinSxS (这可能需要10-20分钟)...")
	spinner.Start()

	if err := os.RemoveAll(winsxsPath); err != nil {
		spinner.Stop(false)
		return fmt.Errorf("删除WinSxS失败: %w", err)
	}

	spinner.Stop(true)

	// 重命名编辑版本
	r.log.Info("应用精简后的WinSxS...")
	if err := os.Rename(winsxsEditPath, winsxsPath); err != nil {
		return fmt.Errorf("重命名失败: %w", err)
	}

	// 计算新大小
	newSize, _ := r.getDirSize(winsxsPath)
	saved := originalSize - newSize

	r.log.Success("WinSxS精简完成")
	r.log.Info("  原始大小: %s", utils.FormatBytes(originalSize))
	r.log.Info("  新大小:   %s", utils.FormatBytes(newSize))
	r.log.Info("  节省:     %s (%.1f%%)",
		utils.FormatBytes(saved),
		float64(saved)/float64(originalSize)*100)

	return nil
}

// getKeepDirs 获取要保留的目录列表
func (r *CoreRemover) getKeepDirs() []string {
	arch := r.config.GetArchitecture()

	// 通用目录
	common := []string{
		"Catalogs",
		"FileMaps",
		"Fusion",
		"InstallTemp",
		"Manifests",
	}

	// 架构特定目录
	var archSpecific []string

	if arch == "amd64" {
		archSpecific = []string{
			// x86组件
			"x86_microsoft.windows.common-controls_6595b64144ccf1df_*",
			"x86_microsoft.windows.gdiplus_6595b64144ccf1df_*",
			"x86_microsoft.windows.i..utomation.proxystub_6595b64144ccf1df_*",
			"x86_microsoft.windows.isolationautomation_6595b64144ccf1df_*",
			"x86_microsoft-windows-s..ngstack-onecorebase_31bf3856ad364e35_*",
			"x86_microsoft-windows-s..stack-termsrv-extra_31bf3856ad364e35_*",
			"x86_microsoft-windows-servicingstack_31bf3856ad364e35_*",
			"x86_microsoft-windows-servicingstack-inetsrv_*",
			"x86_microsoft-windows-servicingstack-onecore_*",
			"x86_microsoft.vc80.crt_1fc8b3b9a1e18e3b_*",
			"x86_microsoft.vc90.crt_1fc8b3b9a1e18e3b_*",
			"x86_microsoft.windows.c..-controls.resources_6595b64144ccf1df_*",
			// amd64组件
			"amd64_microsoft.vc80.crt_1fc8b3b9a1e18e3b_*",
			"amd64_microsoft.vc90.crt_1fc8b3b9a1e18e3b_*",
			"amd64_microsoft.windows.c..-controls.resources_6595b64144ccf1df_*",
			"amd64_microsoft.windows.common-controls_6595b64144ccf1df_*",
			"amd64_microsoft.windows.gdiplus_6595b64144ccf1df_*",
			"amd64_microsoft.windows.i..utomation.proxystub_6595b64144ccf1df_*",
			"amd64_microsoft.windows.isolationautomation_6595b64144ccf1df_*",
			"amd64_microsoft-windows-s..stack-inetsrv-extra_31bf3856ad364e35_*",
			"amd64_microsoft-windows-s..stack-msg.resources_31bf3856ad364e35_*",
			"amd64_microsoft-windows-s..stack-termsrv-extra_31bf3856ad364e35_*",
			"amd64_microsoft-windows-servicingstack_31bf3856ad364e35_*",
			"amd64_microsoft-windows-servicingstack-inetsrv_31bf3856ad364e35_*",
			"amd64_microsoft-windows-servicingstack-msg_31bf3856ad364e35_*",
			"amd64_microsoft-windows-servicingstack-onecore_31bf3856ad364e35_*",
		}
	} else if arch == "arm64" {
		archSpecific = []string{
			// x86组件
			"x86_microsoft.vc80.crt_1fc8b3b9a1e18e3b_*",
			"x86_microsoft.vc90.crt_1fc8b3b9a1e18e3b_*",
			"x86_microsoft.windows.c..-controls.resources_6595b64144ccf1df_*",
			"x86_microsoft.windows.common-controls_6595b64144ccf1df_*",
			"x86_microsoft.windows.gdiplus_6595b64144ccf1df_*",
			"x86_microsoft.windows.i..utomation.proxystub_6595b64144ccf1df_*",
			"x86_microsoft.windows.isolationautomation_6595b64144ccf1df_*",
			// arm组件
			"arm_microsoft.windows.c..-controls.resources_6595b64144ccf1df_*",
			"arm_microsoft.windows.common-controls_6595b64144ccf1df_*",
			"arm_microsoft.windows.gdiplus_6595b64144ccf1df_*",
			"arm_microsoft.windows.i..utomation.proxystub_6595b64144ccf1df_*",
			"arm_microsoft.windows.isolationautomation_6595b64144ccf1df_*",
			// arm64组件
			"arm64_microsoft.vc80.crt_1fc8b3b9a1e18e3b_*",
			"arm64_microsoft.vc90.crt_1fc8b3b9a1e18e3b_*",
			"arm64_microsoft.windows.c..-controls.resources_6595b64144ccf1df_*",
			"arm64_microsoft.windows.common-controls_6595b64144ccf1df_*",
			"arm64_microsoft.windows.gdiplus_6595b64144ccf1df_*",
			"arm64_microsoft.windows.i..utomation.proxystub_6595b64144ccf1df_*",
			"arm64_microsoft.windows.isolationautomation_6595b64144ccf1df_*",
			"arm64_microsoft-windows-servicing-adm_31bf3856ad364e35_*",
			"arm64_microsoft-windows-servicingcommon_31bf3856ad364e35_*",
			"arm64_microsoft-windows-servicing-onecore-uapi_31bf3856ad364e35_*",
			"arm64_microsoft-windows-servicingstack_31bf3856ad364e35_*",
			"arm64_microsoft-windows-servicingstack-inetsrv_31bf3856ad364e35_*",
			"arm64_microsoft-windows-servicingstack-msg_31bf3856ad364e35_*",
			"arm64_microsoft-windows-servicingstack-onecore_31bf3856ad364e35_*",
		}
	}

	return append(common, archSpecific...)
}

// copyDirQuiet 静默复制目录
func (r *CoreRemover) copyDirQuiet(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略错误
		}

		relPath, _ := filepath.Rel(src, path)
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return r.copyFileQuiet(path, targetPath)
	})
}

// copyFileQuiet 静默复制文件
func (r *CoreRemover) copyFileQuiet(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	os.MkdirAll(filepath.Dir(dst), 0755)

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// getDirSize 计算目录大小
func (r *CoreRemover) getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// DisableDefender 禁用Windows Defender（通过注册表）
func (r *CoreRemover) DisableDefender() error {
	r.log.Section("禁用Windows Defender")
	r.log.Info("Defender将通过注册表禁用")
	// 实际实现在registry模块中
	return nil
}

// DisableWindowsUpdate 禁用Windows Update（通过注册表）
func (r *CoreRemover) DisableWindowsUpdate() error {
	r.log.Section("禁用Windows Update")
	r.log.Info("Windows Update将通过注册表禁用")
	// 实际实现在registry模块中
	return nil
}

// RemoveWinRE 移除WinRE恢复环境
func (r *CoreRemover) RemoveWinRE() error {
	mountPath := r.config.ScratchDir
	recoveryPath := filepath.Join(mountPath, "Windows", "System32", "Recovery")
	winrePath := filepath.Join(recoveryPath, "winre.wim")

	r.log.Section("移除WinRE恢复环境")

	if !utils.FileExists(winrePath) {
		r.log.Info("winre.wim 不存在，跳过")
		return nil
	}

	// 获取权限
	utils.TakeownRecursive(recoveryPath)
	utils.GrantPermissionRecursive(recoveryPath)

	// 删除WinRE
	if err := os.Remove(winrePath); err != nil {
		r.log.Warn("删除winre.wim失败: %v", err)
	} else {
		r.log.Success("✓ winre.wim已删除")
	}

	// 创建空占位文件
	f, err := os.Create(winrePath)
	if err == nil {
		f.Close()
		r.log.Info("已创建空占位文件")
	}

	r.log.Success("WinRE移除完成")
	return nil
}