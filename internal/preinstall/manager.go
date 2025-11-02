package preinstall

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"tiny11-builder/internal/config"
	"tiny11-builder/internal/logger"
	"tiny11-builder/internal/utils"
)

type Manager struct {
	config *config.Config
	log    *logger.Logger
}

type PreinstallConfig struct {
	Enabled bool          `json:"enabled"`
	Apps    []AppPackage  `json:"apps"`
}

type AppPackage struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Source      string `json:"source"`      // 安装包路径
	InstallCmd  string `json:"installCmd"`  // 安装命令
	Silent      bool   `json:"silent"`      // 静默安装
	PostScript  string `json:"postScript"`  // 安装后脚本
}

func NewManager(cfg *config.Config, log *logger.Logger) *Manager {
	return &Manager{
		config: cfg,
		log:    log,
	}
}

func (m *Manager) LoadConfig() (*PreinstallConfig, error) {
	configPath := filepath.Join(m.config.PreinstallDir, "preinstall.json")
	
	if !utils.FileExists(configPath) {
		m.log.Info("预装配置不存在，跳过")
		return &PreinstallConfig{Enabled: false}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg PreinstallConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (m *Manager) InstallApps(selectedApps []string) error {
	cfg, err := m.LoadConfig()
	if err != nil {
		return err
	}

	if !cfg.Enabled || len(cfg.Apps) == 0 {
		m.log.Info("无预装应用")
		return nil
	}

	m.log.Section("预装软件到系统")

	mountPath := m.config.ScratchDir
	appsToInstall := m.filterApps(cfg.Apps, selectedApps)

	for i, app := range appsToInstall {
		m.log.Info("[%d/%d] 预装: %s v%s", i+1, len(appsToInstall), app.Name, app.Version)
		
		if err := m.installApp(app, mountPath); err != nil {
			m.log.Warn("  ✗ 安装失败: %v", err)
			continue
		}
		
		m.log.Success("  ✓ 安装成功")
	}

	return nil
}

func (m *Manager) filterApps(apps []AppPackage, selected []string) []AppPackage {
	if len(selected) == 0 {
		return apps
	}

	var filtered []AppPackage
	selectedMap := make(map[string]bool)
	for _, id := range selected {
		selectedMap[id] = true
	}

	for _, app := range apps {
		if selectedMap[app.ID] {
			filtered = append(filtered, app)
		}
	}

	return filtered
}

func (m *Manager) installApp(app AppPackage, mountPath string) error {
	// 复制安装包到镜像
	srcPath := filepath.Join(m.config.PreinstallDir, app.Source)
	if !utils.FileExists(srcPath) {
		return fmt.Errorf("安装包不存在: %s", srcPath)
	}

	destDir := filepath.Join(mountPath, "Windows", "Setup", "PreInstall")
	os.MkdirAll(destDir, 0755)

	destPath := filepath.Join(destDir, filepath.Base(srcPath))
	if err := utils.CopyFile(srcPath, destPath); err != nil {
		return err
	}

	// 创建首次启动脚本
	scriptPath := filepath.Join(mountPath, "Windows", "Setup", "Scripts", "SetupComplete.cmd")
	os.MkdirAll(filepath.Dir(scriptPath), 0755)

	// 追加安装命令
	installCmd := app.InstallCmd
	if app.Silent {
		installCmd += " /S /Silent"
	}

	script := fmt.Sprintf("@echo off\necho Installing %s...\ncd %%SystemRoot%%\\Setup\\PreInstall\n%s\n", 
		app.Name, installCmd)

	// 追加模式写入
	f, err := os.OpenFile(scriptPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(script)
	return err
}

func (m *Manager) ListAvailableApps() ([]AppPackage, error) {
	cfg, err := m.LoadConfig()
	if err != nil {
		return nil, err
	}

	return cfg.Apps, nil
}