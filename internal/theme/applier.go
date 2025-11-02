package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tiny11-builder/internal/config"
	"tiny11-builder/internal/logger"
	"tiny11-builder/internal/utils"
)

type Applier struct {
	config    *config.Config
	log       *logger.Logger
	themeMgr  *Manager
	mountPath string
}

func NewApplier(cfg *config.Config, log *logger.Logger, themeMgr *Manager) *Applier {
	return &Applier{
		config:    cfg,
		log:       log,
		themeMgr:  themeMgr,
		mountPath: cfg.ScratchDir,
	}
}

// ApplyTheme 应用主题到挂载的镜像
func (a *Applier) ApplyTheme(theme *Theme) error {
	a.log.Section("应用主题: " + theme.Name)

	successCount := 0
	failCount := 0

	// 1. 应用品牌信息
	if theme.Branding.Enabled {
		a.log.Info("[1/7] 应用品牌信息...")
		if err := a.applyBranding(theme); err != nil {
			a.log.Warn("  ✗ 品牌信息应用失败: %v", err)
			failCount++
		} else {
			a.log.Success("  ✓ 品牌信息已应用")
			successCount++
		}
	} else {
		a.log.Info("[1/7] 跳过品牌信息 (未启用)")
	}

	// 2. 应用壁纸
	if theme.Wallpapers.Enabled {
		a.log.Info("[2/7] 应用壁纸...")
		if err := a.applyWallpapers(theme); err != nil {
			a.log.Warn("  ✗ 壁纸应用失败: %v", err)
			failCount++
		} else {
			a.log.Success("  ✓ 壁纸已应用")
			successCount++
		}
	} else {
		a.log.Info("[2/7] 跳过壁纸 (未启用)")
	}

	// 3. 应用配色方案
	if theme.Colors.Enabled {
		a.log.Info("[3/7] 应用配色方案...")
		if err := a.applyColors(theme); err != nil {
			a.log.Warn("  ✗ 配色应用失败: %v", err)
			failCount++
		} else {
			a.log.Success("  ✓ 配色方案已应用")
			successCount++
		}
	} else {
		a.log.Info("[3/7] 跳过配色 (未启用)")
	}

	// 4. 应用图标和Logo
	if theme.Images.Enabled {
		a.log.Info("[4/7] 应用图标和Logo...")
		if err := a.applyImages(theme); err != nil {
			a.log.Warn("  ✗ 图标应用失败: %v", err)
			failCount++
		} else {
			a.log.Success("  ✓ 图标和Logo已应用")
			successCount++
		}
	} else {
		a.log.Info("[4/7] 跳过图标 (未启用)")
	}

	// 5. 应用启动Logo
	if theme.Boot.Enabled {
		a.log.Info("[5/7] 应用启动配置...")
		if theme.Boot.CustomLogo {
			if err := a.applyBootLogo(theme); err != nil {
				a.log.Warn("  ✗ 启动Logo应用失败: %v", err)
				failCount++
			} else {
				a.log.Success("  ✓ 启动Logo已应用")
				successCount++
			}
		} else {
			a.log.Info("  跳过自定义启动Logo (未启用)")
		}
	} else {
		a.log.Info("[5/7] 跳过启动配置 (未启用)")
	}

	// 6. 应用声音方案
	if theme.Sounds.Enabled {
		a.log.Info("[6/7] 应用声音方案...")
		if err := a.applySounds(theme); err != nil {
			a.log.Warn("  ✗ 声音方案应用失败: %v", err)
			failCount++
		} else {
			a.log.Success("  ✓ 声音方案已应用")
			successCount++
		}
	} else {
		a.log.Info("[6/7] 跳过声音方案 (未启用)")
	}

	// 7. 应用高级设置
	if theme.Advanced.Enabled {
		a.log.Info("[7/7] 应用高级设置...")
		if err := a.applyAdvancedSettings(theme); err != nil {
			a.log.Warn("  ✗ 高级设置应用失败: %v", err)
			failCount++
		} else {
			a.log.Success("  ✓ 高级设置已应用")
			successCount++
		}
	} else {
		a.log.Info("[7/7] 跳过高级设置 (未启用)")
	}

	a.log.Info("")
	a.log.Success("主题应用完成: 成功 %d, 失败 %d", successCount, failCount)
	return nil
}

// applyBranding 应用品牌信息
func (a *Applier) applyBranding(theme *Theme) error {
	branding, err := a.themeMgr.LoadBrandingData(theme)
	if err != nil {
		return fmt.Errorf("加载品牌数据失败: %w", err)
	}

	applied := 0

	// 修改系统产品名称
	if branding.ProductName != "" {
		a.setRegistryValue(
			"HKLM\\zSOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
			"ProductName",
			"REG_SZ",
			branding.ProductName,
		)
		applied++
	}

	// 修改版本信息
	if branding.VersionInfo.DisplayVersion != "" {
		a.setRegistryValue(
			"HKLM\\zSOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
			"DisplayVersion",
			"REG_SZ",
			branding.VersionInfo.DisplayVersion,
		)
		applied++
	}

	if branding.VersionInfo.BuildBranch != "" {
		a.setRegistryValue(
			"HKLM\\zSOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
			"BuildBranch",
			"REG_SZ",
			branding.VersionInfo.BuildBranch,
		)
		applied++
	}

	if branding.VersionInfo.BuildLab != "" {
		a.setRegistryValue(
			"HKLM\\zSOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
			"BuildLab",
			"REG_SZ",
			branding.VersionInfo.BuildLab,
		)
		applied++
	}

	// 修改系统信息
	if branding.SystemInfo.RegisteredOwner != "" {
		a.setRegistryValue(
			"HKLM\\zSOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
			"RegisteredOwner",
			"REG_SZ",
			branding.SystemInfo.RegisteredOwner,
		)
		applied++
	}

	if branding.SystemInfo.RegisteredOrganization != "" {
		a.setRegistryValue(
			"HKLM\\zSOFTWARE\\Microsoft\\Windows NT\\CurrentVersion",
			"RegisteredOrganization",
			"REG_SZ",
			branding.SystemInfo.RegisteredOrganization,
		)
		applied++
	}

	// 修改OEM信息
	oemInfoPath := "HKLM\\zSOFTWARE\\Microsoft\\Windows\\CurrentVersion\\OEMInformation"

	if branding.SystemInfo.Manufacturer != "" {
		a.setRegistryValue(oemInfoPath, "Manufacturer", "REG_SZ", branding.SystemInfo.Manufacturer)
		applied++
	}

	if branding.SystemInfo.Model != "" {
		a.setRegistryValue(oemInfoPath, "Model", "REG_SZ", branding.SystemInfo.Model)
		applied++
	}

	if branding.SystemInfo.SupportHours != "" {
		a.setRegistryValue(oemInfoPath, "SupportHours", "REG_SZ", branding.SystemInfo.SupportHours)
		applied++
	}

	if branding.SystemInfo.SupportPhone != "" {
		a.setRegistryValue(oemInfoPath, "SupportPhone", "REG_SZ", branding.SystemInfo.SupportPhone)
		applied++
	}

	if branding.SystemInfo.SupportURL != "" {
		a.setRegistryValue(oemInfoPath, "SupportURL", "REG_SZ", branding.SystemInfo.SupportURL)
		applied++
	}

	a.log.Info("  应用了 %d 项品牌设置", applied)
	return nil
}

// applyWallpapers 应用壁纸
func (a *Applier) applyWallpapers(theme *Theme) error {
	applied := 0

	// 创建壁纸目录
	wallpaperDir := filepath.Join(a.mountPath, "Windows", "Web", "Wallpaper", "Miku")
	if err := os.MkdirAll(wallpaperDir, 0755); err != nil {
		return fmt.Errorf("创建壁纸目录失败: %w", err)
	}

	// 复制桌面壁纸（仅当设置了路径）
	if theme.Wallpapers.Desktop != "" {
		srcDesktop := filepath.Join(theme.ThemePath, theme.Wallpapers.Desktop)

		if !utils.FileExists(srcDesktop) {
			a.log.Warn("  桌面壁纸文件不存在: %s", theme.Wallpapers.Desktop)
		} else {
			dstDesktop := filepath.Join(wallpaperDir, "desktop.jpg")

			if err := utils.CopyFile(srcDesktop, dstDesktop); err != nil {
				a.log.Warn("  复制桌面壁纸失败: %v", err)
			} else {
				applied++
				a.log.Info("  ✓ 桌面壁纸已复制")

				// 仅当明确设置为默认时才修改注册表
				if theme.Wallpapers.SetAsDefault {
					a.setRegistryValue(
						"HKLM\\zNTUSER\\Control Panel\\Desktop",
						"Wallpaper",
						"REG_SZ",
						"%SystemRoot%\\Web\\Wallpaper\\Miku\\desktop.jpg",
					)
					a.log.Info("  ✓ 已设置为默认壁纸")
				}
			}
		}
	}

	// 复制锁屏壁纸（仅当设置了路径）
	if theme.Wallpapers.Lockscreen != "" {
		srcLock := filepath.Join(theme.ThemePath, theme.Wallpapers.Lockscreen)

		if !utils.FileExists(srcLock) {
			a.log.Warn("  锁屏壁纸文件不存在: %s", theme.Wallpapers.Lockscreen)
		} else {
			dstLock := filepath.Join(wallpaperDir, "lockscreen.jpg")

			if err := utils.CopyFile(srcLock, dstLock); err != nil {
				a.log.Warn("  复制锁屏壁纸失败: %v", err)
			} else {
				applied++
				a.log.Info("  ✓ 锁屏壁纸已复制")

				// 设置锁屏壁纸
				a.setRegistryValue(
					"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\Personalization",
					"LockScreenImage",
					"REG_SZ",
					"%SystemRoot%\\Web\\Wallpaper\\Miku\\lockscreen.jpg",
				)
			}
		}
	}

	if applied == 0 {
		a.log.Info("  未设置任何壁纸路径")
	}

	return nil
}

// applyColors 应用配色方案
func (a *Applier) applyColors(theme *Theme) error {
	colors, err := a.themeMgr.LoadColorScheme(theme)
	if err != nil {
		return err
	}

	if !colors.Registry.ApplySystemWide {
		a.log.Info("  配色方案设置为不应用到系统")
		return nil
	}

	applied := 0

	// 应用强调色（仅当设置了值）
	if colors.Registry.AccentColor != "" {
		a.setRegistryValue(
			"HKLM\\zSOFTWARE\\Microsoft\\Windows\\DWM",
			"AccentColor",
			"REG_DWORD",
			colors.Registry.AccentColor,
		)

		a.setRegistryValue(
			"HKLM\\zNTUSER\\SOFTWARE\\Microsoft\\Windows\\DWM",
			"AccentColor",
			"REG_DWORD",
			colors.Registry.AccentColor,
		)
		applied++
		a.log.Info("  ✓ 强调色: %s", colors.Registry.AccentColor)
	}

	// 应用开始菜单颜色（仅当设置了值）
	if colors.Registry.StartColor != "" {
		a.setRegistryValue(
			"HKLM\\zNTUSER\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Themes\\Personalize",
			"ColorPrevalence",
			"REG_DWORD",
			"1",
		)
		applied++
		a.log.Info("  ✓ 开始菜单配色")
	}

	// 仅当明确启用时才设置透明效果
	if theme.Colors.ApplyTransparency {
		a.setRegistryValue(
			"HKLM\\zNTUSER\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Themes\\Personalize",
			"EnableTransparency",
			"REG_DWORD",
			"1",
		)
		applied++
		a.log.Info("  ✓ 透明效果已启用")
	}

	a.log.Info("  应用了 %d 项配色设置", applied)
	return nil
}

// applyImages 应用图标和Logo
func (a *Applier) applyImages(theme *Theme) error {
	// 创建OEM目录
	oemDir := filepath.Join(a.mountPath, "Windows", "System32", "oem")
	if err := os.MkdirAll(oemDir, 0755); err != nil {
		return fmt.Errorf("创建OEM目录失败: %w", err)
	}

	applied := 0

	// 复制OEM Logo（仅当设置了路径）
	if theme.Images.OEMLogo != "" {
		srcLogo := filepath.Join(theme.ThemePath, theme.Images.OEMLogo)
		dstLogo := filepath.Join(oemDir, "logo.bmp")

		if utils.FileExists(srcLogo) {
			if err := utils.CopyFile(srcLogo, dstLogo); err == nil {
				// 设置OEM Logo路径
				a.setRegistryValue(
					"HKLM\\zSOFTWARE\\Microsoft\\Windows\\CurrentVersion\\OEMInformation",
					"Logo",
					"REG_SZ",
					"%SystemRoot%\\System32\\oem\\logo.bmp",
				)
				applied++
				a.log.Info("  ✓ OEM Logo")
			}
		} else {
			a.log.Warn("  OEM Logo文件不存在: %s", theme.Images.OEMLogo)
		}
	}

	// 复制系统Logo（仅当设置了路径）
	if theme.Images.SystemLogo != "" {
		srcSysLogo := filepath.Join(theme.ThemePath, theme.Images.SystemLogo)
		dstSysLogo := filepath.Join(oemDir, "systemlogo.png")

		if utils.FileExists(srcSysLogo) {
			if utils.CopyFile(srcSysLogo, dstSysLogo) == nil {
				applied++
				a.log.Info("  ✓ 系统Logo")
			}
		} else {
			a.log.Warn("  系统Logo文件不存在: %s", theme.Images.SystemLogo)
		}
	}

	// 复制用户头像（仅当设置了路径）
	if theme.Images.UserTile != "" {
		srcTile := filepath.Join(theme.ThemePath, theme.Images.UserTile)
		userTileDir := filepath.Join(a.mountPath, "ProgramData", "Microsoft", "User Account Pictures")
		os.MkdirAll(userTileDir, 0755)

		dstTile := filepath.Join(userTileDir, "user.png")

		if utils.FileExists(srcTile) {
			if utils.CopyFile(srcTile, dstTile) == nil {
				applied++
				a.log.Info("  ✓ 用户头像")
			}
		} else {
			a.log.Warn("  用户头像文件不存在: %s", theme.Images.UserTile)
		}
	}

	// 复制品牌图标（仅当设置了路径）
	if theme.Images.BrandIcon != "" {
		srcIcon := filepath.Join(theme.ThemePath, theme.Images.BrandIcon)
		dstIcon := filepath.Join(oemDir, "brand.ico")

		if utils.FileExists(srcIcon) {
			if utils.CopyFile(srcIcon, dstIcon) == nil {
				applied++
				a.log.Info("  ✓ 品牌图标")
			}
		} else {
			a.log.Warn("  品牌图标文件不存在: %s", theme.Images.BrandIcon)
		}
	}

	a.log.Info("  应用了 %d 项图标设置", applied)
	return nil
}

// applyBootLogo 应用启动Logo
func (a *Applier) applyBootLogo(theme *Theme) error {
	if theme.Boot.LogoFile == "" {
		a.log.Info("  未设置启动Logo文件")
		return nil
	}

	srcLogo := filepath.Join(theme.ThemePath, theme.Boot.LogoFile)
	if !utils.FileExists(srcLogo) {
		return fmt.Errorf("启动Logo文件不存在: %s", srcLogo)
	}

	// 复制到系统目录
	bootLogoDir := filepath.Join(a.mountPath, "Windows", "System32")
	dstLogo := filepath.Join(bootLogoDir, "bootlogo.bmp")

	if err := utils.CopyFile(srcLogo, dstLogo); err != nil {
		return err
	}

	a.log.Info("  ✓ 启动Logo已复制")

	// 设置启动背景色（仅当指定了颜色）
	if theme.Boot.BackgroundColor != "" {
		color := a.parseColorToRGB(theme.Boot.BackgroundColor)
		a.setRegistryValue(
			"HKLM\\zSYSTEM\\ControlSet001\\Control\\BootControl",
			"BootProgressColor",
			"REG_DWORD",
			color,
		)
		a.log.Info("  ✓ 启动背景色: %s", theme.Boot.BackgroundColor)
	}

	return nil
}

// applySounds 应用声音方案
func (a *Applier) applySounds(theme *Theme) error {
	soundsDir := filepath.Join(a.mountPath, "Windows", "Media", "Miku")
	if err := os.MkdirAll(soundsDir, 0755); err != nil {
		return fmt.Errorf("创建声音目录失败: %w", err)
	}

	// 声音文件映射
	soundMappings := map[string]struct {
		src      string
		filename string
		regKey   string
		name     string
	}{
		"startup": {
			src:      theme.Sounds.Startup,
			filename: "startup.wav",
			regKey:   "SystemStart",
			name:     "启动音",
		},
		"shutdown": {
			src:      theme.Sounds.Shutdown,
			filename: "shutdown.wav",
			regKey:   "SystemExit",
			name:     "关机音",
		},
		"logon": {
			src:      theme.Sounds.Logon,
			filename: "logon.wav",
			regKey:   "WindowsLogon",
			name:     "登录音",
		},
	}

	applied := 0

	for _, mapping := range soundMappings {
		// 仅当设置了路径时才处理
		if mapping.src == "" {
			continue
		}

		srcSound := filepath.Join(theme.ThemePath, mapping.src)
		if !utils.FileExists(srcSound) {
			a.log.Warn("  %s文件不存在: %s", mapping.name, mapping.src)
			continue
		}

		dstSound := filepath.Join(soundsDir, mapping.filename)
		if err := utils.CopyFile(srcSound, dstSound); err != nil {
			a.log.Warn("  复制%s失败: %v", mapping.name, err)
			continue
		}

		// 设置注册表
		soundPath := fmt.Sprintf("%%SystemRoot%%\\Media\\Miku\\%s", mapping.filename)
		a.setRegistryValue(
			fmt.Sprintf("HKLM\\zNTUSER\\AppEvents\\Schemes\\Apps\\.Default\\%s\\.Current", mapping.regKey),
			"",
			"REG_SZ",
			soundPath,
		)

		applied++
		a.log.Info("  ✓ %s", mapping.name)
	}

	a.log.Info("  应用了 %d 项声音设置", applied)
	return nil
}

// applyAdvancedSettings 应用高级设置
func (a *Applier) applyAdvancedSettings(theme *Theme) error {
	applied := 0
	settings := theme.Advanced.Settings

	// 任务栏透明度
	if settings.TaskbarTransparency {
		if err := a.setRegistryValue(
			"HKLM\\zNTUSER\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Themes\\Personalize",
			"EnableTransparency",
			"REG_DWORD",
			"1",
		); err == nil {
			applied++
			a.log.Info("  ✓ 任务栏透明度")
		}
	}

	// 圆角窗口
	if settings.RoundedCorners {
		if err := a.setRegistryValue(
			"HKLM\\zNTUSER\\SOFTWARE\\Microsoft\\Windows\\DWM",
			"UseRoundedCorners",
			"REG_DWORD",
			"1",
		); err == nil {
			applied++
			a.log.Info("  ✓ 圆角窗口")
		}
	}

	// 显示文件扩展名
	if settings.ShowFileExtensions {
		if err := a.setRegistryValue(
			"HKLM\\zNTUSER\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced",
			"HideFileExt",
			"REG_DWORD",
			"0",
		); err == nil {
			applied++
			a.log.Info("  ✓ 显示文件扩展名")
		}
	}

	// 显示隐藏文件
	if settings.ShowHiddenFiles {
		if err := a.setRegistryValue(
			"HKLM\\zNTUSER\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced",
			"Hidden",
			"REG_DWORD",
			"1",
		); err == nil {
			applied++
			a.log.Info("  ✓ 显示隐藏文件")
		}
	}

	// 应用强调色（仅当设置了非空值）
	if settings.AccentColor != "" {
		colorValue := a.parseColorToRGB(settings.AccentColor)
		if err := a.setRegistryValue(
			"HKLM\\zNTUSER\\SOFTWARE\\Microsoft\\Windows\\DWM",
			"ColorizationColor",
			"REG_DWORD",
			colorValue,
		); err == nil {
			applied++
			a.log.Info("  ✓ 强调色: %s", settings.AccentColor)
		}
	}

	a.log.Info("  应用了 %d 项高级设置", applied)
	return nil
}

// setRegistryValue 设置注册表值（辅助函数）
func (a *Applier) setRegistryValue(path, name, valueType, value string) error {
	_, err := utils.RunCommand("reg", "add", path, "/v", name, "/t", valueType, "/d", value, "/f")
	return err
}

// parseColorToRGB 将十六进制颜色转换为DWORD格式
func (a *Applier) parseColorToRGB(hexColor string) string {
	// 移除 # 符号
	hexColor = strings.TrimPrefix(hexColor, "#")

	// 如果已经是0x格式，直接返回
	if strings.HasPrefix(hexColor, "0x") {
		return hexColor
	}

	// 转换 #RRGGBB 为 0xBBGGRR (Windows格式)
	if len(hexColor) == 6 {
		r := hexColor[0:2]
		g := hexColor[2:4]
		b := hexColor[4:6]
		return "0x" + b + g + r
	}

	return "0x000000"
}