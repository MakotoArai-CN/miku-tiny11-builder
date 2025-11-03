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
	Enabled bool         `json:"enabled"`
	Apps    []AppPackage `json:"apps"`
}

type AppPackage struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Source      string `json:"source"`
	InstallCmd  string `json:"installCmd"`
	Silent      bool   `json:"silent"`
	PostScript  string `json:"postScript"`
}

func NewManager(cfg *config.Config, log *logger.Logger) *Manager {
	return &Manager{
		config: cfg,
		log:    log,
	}
}

//  优化配置加载，处理文件不存在的情况
func (m *Manager) LoadConfig() (*PreinstallConfig, error) {
	configPath := filepath.Join(m.config.PreinstallDir, "preinstall.json")

	// 检查文件是否存在
	if !utils.FileExists(configPath) {
		m.log.Info("预装配置文件不存在，跳过: %s", configPath)
		return &PreinstallConfig{Enabled: false}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		m.log.Warn("无法读取预装配置文件: %v", err)
		return &PreinstallConfig{Enabled: false}, nil
	}

	var cfg PreinstallConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		m.log.Warn("预装配置文件格式错误: %v", err)
		return &PreinstallConfig{Enabled: false}, nil
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

	if len(appsToInstall) == 0 {
		m.log.Info("没有选择要预装的软件")
		return nil
	}

	for i, app := range appsToInstall {
		m.log.Info("[%d/%d] 预装: %s v%s", i+1, len(appsToInstall), app.Name, app.Version)

		//  检查安装包是否存在
		srcPath := filepath.Join(m.config.PreinstallDir, app.Source)
		if !utils.FileExists(srcPath) {
			m.log.Warn("  ✗ 安装包不存在: %s", srcPath)
			continue
		}

		if err := m.installApp(app, mountPath); err != nil {
			m.log.Warn("  ✗ 安装失败: %v", err)
			continue
		}

		m.log.Success("  ✓ 配置成功")
	}

	return nil
}

func (m *Manager) filterApps(apps []AppPackage, selected []string) []AppPackage {
	if len(selected) == 0 {
		return []AppPackage{}
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
	srcPath := filepath.Join(m.config.PreinstallDir, app.Source)

	// 创建预装目录
	destDir := filepath.Join(mountPath, "Windows", "Setup", "PreInstall")
	os.MkdirAll(destDir, 0755)

	// 复制安装包
	destPath := filepath.Join(destDir, filepath.Base(srcPath))
	if err := utils.CopyFile(srcPath, destPath); err != nil {
		return err
	}

	// 创建安装脚本
	scriptPath := filepath.Join(mountPath, "Windows", "Setup", "Scripts", "SetupComplete.cmd")
	os.MkdirAll(filepath.Dir(scriptPath), 0755)

	installCmd := app.InstallCmd
	if app.Silent {
		installCmd += " /S /Silent"
	}

	script := fmt.Sprintf("@echo off\necho Installing %s...\ncd %%SystemRoot%%\\Setup\\PreInstall\n%s\n",
		app.Name, installCmd)

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