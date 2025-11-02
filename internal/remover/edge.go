package remover

import (
	"fmt"
	"os"
	"path/filepath"
	"tiny11-builder/internal/utils"
)

// RemoveEdge 移除Microsoft Edge
func (r *AppRemover) RemoveEdge() error {
	mountPath := r.config.ScratchDir
	r.log.Section("移除Microsoft Edge")

	removed := 0
	failed := 0

	// Edge程序文件路径列表
	edgePaths := []struct {
		path string
		desc string
	}{
		{filepath.Join(mountPath, "Program Files (x86)", "Microsoft", "Edge"), "Edge主程序"},
		{filepath.Join(mountPath, "Program Files (x86)", "Microsoft", "EdgeUpdate"), "Edge更新程序"},
		{filepath.Join(mountPath, "Program Files (x86)", "Microsoft", "EdgeCore"), "Edge核心组件"},
	}

	// 移除Edge程序文件
	for _, item := range edgePaths {
		if utils.DirExists(item.path) {
			r.log.Info("移除: %s", item.desc)
			if err := os.RemoveAll(item.path); err != nil {
				r.log.Warn("  ✗ 失败: %v", err)
				failed++
			} else {
				r.log.Success("  ✓ 成功")
				removed++
			}
		}
	}

	// 移除Edge WebView
	webviewPath := filepath.Join(mountPath, "Windows", "System32", "Microsoft-Edge-Webview")
	if utils.DirExists(webviewPath) {
		r.log.Info("移除Edge WebView...")

		// 获取所有权和权限
		if err := utils.TakeownRecursive(webviewPath); err != nil {
			r.log.Warn("  获取所有权失败: %v", err)
		}

		if err := utils.GrantPermissionRecursive(webviewPath); err != nil {
			r.log.Warn("  设置权限失败: %v", err)
		}

		// 删除目录
		if err := os.RemoveAll(webviewPath); err != nil {
			r.log.Warn("  ✗ 删除失败: %v", err)
			failed++
		} else {
			r.log.Success("  ✓ 删除成功")
			removed++
		}
	}

	// 移除WinSxS中的Edge WebView
	r.removeEdgeFromWinSxS(mountPath, &removed, &failed)

	r.log.Info("")
	r.log.Success("Edge移除完成: 成功 %d, 失败 %d", removed, failed)

	return nil
}

// removeEdgeFromWinSxS 从WinSxS中移除Edge组件
func (r *AppRemover) removeEdgeFromWinSxS(mountPath string, removed, failed *int) {
	winsxsPath := filepath.Join(mountPath, "Windows", "WinSxS")

	// 根据架构确定搜索模式
	arch := r.config.GetArchitecture()
	pattern := fmt.Sprintf("%s_microsoft-edge-webview_*", arch)

	r.log.Info("扫描WinSxS中的Edge组件 (模式: %s)...", pattern)

	// 查找匹配的目录
	matches, err := filepath.Glob(filepath.Join(winsxsPath, pattern))
	if err != nil {
		r.log.Warn("  扫描失败: %v", err)
		return
	}

	if len(matches) == 0 {
		r.log.Info("  未找到Edge组件")
		return
	}

	// 移除找到的目录
	for _, match := range matches {
		dirName := filepath.Base(match)
		r.log.Info("  移除WinSxS: %s", dirName)

		// 获取权限
		if err := utils.TakeownRecursive(match); err != nil {
			r.log.Warn("    获取所有权失败: %v", err)
		}

		if err := utils.GrantPermissionRecursive(match); err != nil {
			r.log.Warn("    设置权限失败: %v", err)
		}

		// 删除
		if err := os.RemoveAll(match); err != nil {
			r.log.Warn("    ✗ 删除失败: %v", err)
			(*failed)++
		} else {
			r.log.Success("    ✓ 删除成功")
			(*removed)++
		}
	}
}

// RemoveOneDrive 移除OneDrive
func (r *AppRemover) RemoveOneDrive() error {
	mountPath := r.config.ScratchDir
	r.log.Section("移除OneDrive")

	onedrivePath := filepath.Join(mountPath, "Windows", "System32", "OneDriveSetup.exe")

	if !utils.FileExists(onedrivePath) {
		r.log.Info("OneDriveSetup.exe 不存在，跳过")
		return nil
	}

	// 获取权限
	if err := utils.Takeown(onedrivePath); err != nil {
		r.log.Warn("获取所有权失败: %v", err)
	}

	if err := utils.GrantPermission(onedrivePath); err != nil {
		r.log.Warn("设置权限失败: %v", err)
	}

	// 删除文件
	if err := os.Remove(onedrivePath); err != nil {
		r.log.Warn("✗ 删除失败: %v", err)
		return err
	}

	r.log.Success("✓ OneDrive移除成功")
	return nil
}

// RemoveScheduledTasks 移除计划任务
func (r *AppRemover) RemoveScheduledTasks() error {
	mountPath := r.config.ScratchDir
	tasksPath := filepath.Join(mountPath, "Windows", "System32", "Tasks")

	r.log.Section("移除遥测计划任务")

	// 要删除的任务文件列表
	tasks := []struct {
		path string
		desc string
	}{
		{
			filepath.Join(tasksPath, "Microsoft", "Windows", "Application Experience", "Microsoft Compatibility Appraiser"),
			"应用兼容性评估",
		},
		{
			filepath.Join(tasksPath, "Microsoft", "Windows", "Application Experience", "ProgramDataUpdater"),
			"程序数据更新",
		},
		{
			filepath.Join(tasksPath, "Microsoft", "Windows", "Customer Experience Improvement Program"),
			"客户体验改善计划(整个文件夹)",
		},
		{
			filepath.Join(tasksPath, "Microsoft", "Windows", "Chkdsk", "Proxy"),
			"磁盘检查代理",
		},
		{
			filepath.Join(tasksPath, "Microsoft", "Windows", "Windows Error Reporting", "QueueReporting"),
			"错误报告队列",
		},
	}

	removed := 0
	failed := 0

	for _, task := range tasks {
		r.log.Info("删除: %s", task.desc)

		var err error
		info, statErr := os.Stat(task.path)

		if os.IsNotExist(statErr) {
			r.log.Info("  不存在，跳过")
			continue
		}

		if statErr == nil && info.IsDir() {
			// 删除整个目录
			err = os.RemoveAll(task.path)
		} else {
			// 删除单个文件
			err = os.Remove(task.path)
		}

		if err != nil {
			r.log.Warn("  ✗ 删除失败: %v", err)
			failed++
		} else {
			r.log.Success("  ✓ 删除成功")
			removed++
		}
	}

	r.log.Info("")
	r.log.Success("计划任务移除完成: 成功 %d, 失败 %d", removed, failed)

	return nil
}