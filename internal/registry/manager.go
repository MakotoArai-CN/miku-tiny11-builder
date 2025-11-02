package registry

import (
	"fmt"
	"time"
	"tiny11-builder/internal/config"
	"tiny11-builder/internal/logger"
	"tiny11-builder/internal/utils"
)

// Manager 注册表管理器
type Manager struct {
	config      *config.Config
	log         *logger.Logger
	hivesLoaded bool
}

// NewManager 创建注册表管理器
func NewManager(cfg *config.Config, log *logger.Logger) *Manager {
	return &Manager{
		config: cfg,
		log:    log,
	}
}

// LoadHives 加载注册表Hive
func (m *Manager) LoadHives() error {
	mountPath := m.config.ScratchDir
	m.log.Info("加载注册表Hive...")

	hives := map[string]string{
		"HKLM\\zCOMPONENTS": "Windows\\System32\\config\\COMPONENTS",
		"HKLM\\zDEFAULT":    "Windows\\System32\\config\\default",
		"HKLM\\zNTUSER":     "Users\\Default\\ntuser.dat",
		"HKLM\\zSOFTWARE":   "Windows\\System32\\config\\SOFTWARE",
		"HKLM\\zSYSTEM":     "Windows\\System32\\config\\SYSTEM",
	}

	for hive, path := range hives {
		fullPath := fmt.Sprintf("%s\\%s", mountPath, path)
		if err := loadHive(hive, fullPath); err != nil {
			m.log.Warn("加载Hive失败 %s: %v", hive, err)
		}
	}

	m.hivesLoaded = true
	return nil
}

// UnloadHives 卸载注册表Hive
func (m *Manager) UnloadHives() error {
	if !m.hivesLoaded {
		return nil
	}

	m.log.Info("卸载注册表Hive...")

	hives := []string{
		"HKLM\\zCOMPONENTS",
		"HKLM\\zDEFAULT",
		"HKLM\\zNTUSER",
		"HKLM\\zSOFTWARE",
		"HKLM\\zSYSTEM",
	}

	// 多次尝试卸载（有时需要等待）
	maxRetries := 3
	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			m.log.Info("等待系统释放资源...")
			time.Sleep(2 * time.Second)
		}

		allSuccess := true
		for _, hive := range hives {
			if err := unloadHive(hive); err != nil {
				m.log.Warn("卸载Hive失败 %s: %v (尝试 %d/%d)", hive, err, retry+1, maxRetries)
				allSuccess = false
			}
		}

		if allSuccess {
			m.hivesLoaded = false
			m.log.Success("注册表Hive卸载成功")
			return nil
		}
	}

	// 即使失败也标记为已卸载，避免重复尝试
	m.hivesLoaded = false
	return fmt.Errorf("部分注册表Hive卸载失败")
}

// loadHive 加载单个Hive
func loadHive(hive, path string) error {
	_, err := utils.RunCommand("reg", "load", hive, path)
	return err
}

// unloadHive 卸载单个Hive
func unloadHive(hive string) error {
	// 先尝试正常卸载
	_, err := utils.RunCommand("reg", "unload", hive)
	if err == nil {
		return nil
	}

	// 如果失败，强制垃圾回收
	utils.RunCommand("reg", "unload", hive)
	time.Sleep(500 * time.Millisecond)
	
	// 再次尝试
	_, err = utils.RunCommand("reg", "unload", hive)
	return err
}