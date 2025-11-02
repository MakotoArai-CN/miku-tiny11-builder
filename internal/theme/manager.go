package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"tiny11-builder/internal/config"
	"tiny11-builder/internal/logger"
)

type Manager struct {
	config      *config.Config
	log         *logger.Logger
	themesDir   string
	activeTheme *Theme
}

type Theme struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`

	Branding   BrandingConfig   `json:"branding"`
	Wallpapers WallpapersConfig `json:"wallpapers"`
	Colors     ColorsConfig     `json:"colors"`
	Images     ImagesConfig     `json:"images"`
	Boot       BootConfig       `json:"boot"`
	Sounds     SoundsConfig     `json:"sounds"`
	Fonts      FontsConfig      `json:"fonts"`
	Advanced   AdvancedConfig   `json:"advanced"`

	ThemePath string `json:"-"`
}

type BrandingConfig struct {
	Enabled    bool   `json:"enabled"`
	ConfigFile string `json:"configFile"`
}

type WallpapersConfig struct {
	Enabled      bool   `json:"enabled"`
	Desktop      string `json:"desktop"`
	Lockscreen   string `json:"lockscreen"`
	SetAsDefault bool   `json:"setAsDefault"`
}

type ColorsConfig struct {
	Enabled           bool   `json:"enabled"`
	ConfigFile        string `json:"configFile"`
	ApplyTransparency bool   `json:"applyTransparency"`
}

type ImagesConfig struct {
	Enabled    bool   `json:"enabled"`
	SystemLogo string `json:"systemLogo"`
	OEMLogo    string `json:"oemLogo"`
	UserTile   string `json:"userTile"`
	BrandIcon  string `json:"brandIcon"`
}

type BootConfig struct {
	Enabled         bool   `json:"enabled"`
	CustomLogo      bool   `json:"customLogo"`
	LogoFile        string `json:"logoFile"`
	BackgroundColor string `json:"backgroundColor"`
}

type SoundsConfig struct {
	Enabled  bool   `json:"enabled"`
	Startup  string `json:"startup"`
	Shutdown string `json:"shutdown"`
	Logon    string `json:"logon"`
}

type FontsConfig struct {
	Enabled    bool   `json:"enabled"`
	SystemFont string `json:"systemFont"`
}

type AdvancedConfig struct {
	Enabled        bool                   `json:"enabled"`
	ModifyExplorer bool                   `json:"modifyExplorer"`
	Settings       AdvancedSettingsDetail `json:"settings"`
}

type AdvancedSettingsDetail struct {
	AccentColor         string `json:"accentColor"`
	TaskbarTransparency bool   `json:"taskbarTransparency"`
	RoundedCorners      bool   `json:"roundedCorners"`
	ShowFileExtensions  bool   `json:"showFileExtensions"`
	ShowHiddenFiles     bool   `json:"showHiddenFiles"`
}

type BrandingData struct {
	ProductName string `json:"productName"`
	Edition     struct {
		MapOriginal bool              `json:"mapOriginal"`
		Mappings    map[string]string `json:"mappings"`
	} `json:"edition"`
	SystemInfo struct {
		RegisteredOwner        string `json:"registeredOwner"`
		RegisteredOrganization string `json:"registeredOrganization"`
		Manufacturer           string `json:"manufacturer"`
		Model                  string `json:"model"`
		SupportHours           string `json:"supportHours"`
		SupportPhone           string `json:"supportPhone"`
		SupportURL             string `json:"supportURL"`
	} `json:"systemInfo"`
	VersionInfo struct {
		DisplayVersion string `json:"displayVersion"`
		BuildBranch    string `json:"buildBranch"`
		BuildLab       string `json:"buildLab"`
		CSDVersion     string `json:"csdVersion"`
	} `json:"versionInfo"`
	OOBE struct {
		OEMName         string `json:"oemName"`
		WelcomeTitle    string `json:"welcomeTitle"`
		WelcomeSubtitle string `json:"welcomeSubtitle"`
	} `json:"oobe"`
}

type ColorScheme struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Colors      struct {
		Accent              string `json:"accent"`
		AccentLight         string `json:"accentLight"`
		AccentDark          string `json:"accentDark"`
		Secondary           string `json:"secondary"`
		SecondaryLight      string `json:"secondaryLight"`
		SecondaryDark       string `json:"secondaryDark"`
		Background          string `json:"background"`
		BackgroundDark      string `json:"backgroundDark"`
		Surface             string `json:"surface"`
		SurfaceDark         string `json:"surfaceDark"`
		Text                string `json:"text"`
		TextSecondary       string `json:"textSecondary"`
		TextDark            string `json:"textDark"`
		TextDarkSecondary   string `json:"textDarkSecondary"`
		Success             string `json:"success"`
		Warning             string `json:"warning"`
		Error               string `json:"error"`
		Info                string `json:"info"`
	} `json:"colors"`
	Registry struct {
		ApplySystemWide  bool   `json:"applySystemWide"`
		AccentColor      string `json:"accentColor"`
		AccentColorMenu  string `json:"accentColorMenu"`
		StartColor       string `json:"startColor"`
		TaskbarColor     string `json:"taskbarColor"`
		TitleBarColor    string `json:"titleBarColor"`
	} `json:"registry"`
}

func NewManager(cfg *config.Config, log *logger.Logger) *Manager {
	themesDir := filepath.Join(cfg.WorkDir, "themes")
	return &Manager{
		config:    cfg,
		log:       log,
		themesDir: themesDir,
	}
}

func (m *Manager) LoadTheme(themeName string) (*Theme, error) {
	themePath := filepath.Join(m.themesDir, themeName)
	themeFile := filepath.Join(themePath, "theme.json")

	if _, err := os.Stat(themeFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("主题不存在: %s", themeName)
	}

	data, err := os.ReadFile(themeFile)
	if err != nil {
		return nil, fmt.Errorf("读取主题配置失败: %w", err)
	}

	var theme Theme
	if err := json.Unmarshal(data, &theme); err != nil {
		return nil, fmt.Errorf("解析主题配置失败: %w", err)
	}

	theme.ThemePath = themePath
	m.activeTheme = &theme

	m.log.Success("加载主题: %s v%s", theme.Name, theme.Version)
	m.log.Info("  作者: %s", theme.Author)
	m.log.Info("  描述: %s", theme.Description)

	return &theme, nil
}

func (m *Manager) ListThemes() ([]string, error) {
	if _, err := os.Stat(m.themesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("主题目录不存在: %s", m.themesDir)
	}

	entries, err := os.ReadDir(m.themesDir)
	if err != nil {
		return nil, err
	}

	var themes []string
	for _, entry := range entries {
		if entry.IsDir() {
			themeFile := filepath.Join(m.themesDir, entry.Name(), "theme.json")
			if _, err := os.Stat(themeFile); err == nil {
				themes = append(themes, entry.Name())
			}
		}
	}

	return themes, nil
}

func (m *Manager) GetActiveTheme() *Theme {
	return m.activeTheme
}

func (m *Manager) ValidateTheme(theme *Theme) []string {
	var warnings []string

	if theme.Wallpapers.Enabled {
		if theme.Wallpapers.Desktop != "" {
			desktop := filepath.Join(theme.ThemePath, theme.Wallpapers.Desktop)
			if _, err := os.Stat(desktop); os.IsNotExist(err) {
				warnings = append(warnings, fmt.Sprintf("桌面壁纸不存在: %s", theme.Wallpapers.Desktop))
			}
		}

		if theme.Wallpapers.Lockscreen != "" {
			lockscreen := filepath.Join(theme.ThemePath, theme.Wallpapers.Lockscreen)
			if _, err := os.Stat(lockscreen); os.IsNotExist(err) {
				warnings = append(warnings, fmt.Sprintf("锁屏壁纸不存在: %s", theme.Wallpapers.Lockscreen))
			}
		}
	}

	if theme.Images.Enabled {
		images := map[string]string{
			"系统Logo":  theme.Images.SystemLogo,
			"OEM Logo": theme.Images.OEMLogo,
			"用户头像":    theme.Images.UserTile,
			"品牌图标":    theme.Images.BrandIcon,
		}

		for name, path := range images {
			if path != "" {
				fullPath := filepath.Join(theme.ThemePath, path)
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					warnings = append(warnings, fmt.Sprintf("%s不存在: %s", name, path))
				}
			}
		}
	}

	if theme.Boot.Enabled && theme.Boot.CustomLogo {
		if theme.Boot.LogoFile != "" {
			logoPath := filepath.Join(theme.ThemePath, theme.Boot.LogoFile)
			if _, err := os.Stat(logoPath); os.IsNotExist(err) {
				warnings = append(warnings, fmt.Sprintf("启动Logo不存在: %s", theme.Boot.LogoFile))
			}
		}
	}

	return warnings
}

func (m *Manager) LoadBrandingData(theme *Theme) (*BrandingData, error) {
	if !theme.Branding.Enabled {
		return nil, nil
	}

	brandingFile := filepath.Join(theme.ThemePath, theme.Branding.ConfigFile)
	data, err := os.ReadFile(brandingFile)
	if err != nil {
		return nil, fmt.Errorf("读取品牌配置失败: %w", err)
	}

	var branding BrandingData
	if err := json.Unmarshal(data, &branding); err != nil {
		return nil, fmt.Errorf("解析品牌配置失败: %w", err)
	}

	return &branding, nil
}

func (m *Manager) LoadColorScheme(theme *Theme) (*ColorScheme, error) {
	if !theme.Colors.Enabled {
		return nil, nil
	}

	colorFile := filepath.Join(theme.ThemePath, theme.Colors.ConfigFile)
	data, err := os.ReadFile(colorFile)
	if err != nil {
		return nil, fmt.Errorf("读取配色配置失败: %w", err)
	}

	var colors ColorScheme
	if err := json.Unmarshal(data, &colors); err != nil {
		return nil, fmt.Errorf("解析配色配置失败: %w", err)
	}

	return &colors, nil
}