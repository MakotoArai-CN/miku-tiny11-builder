package remover

import (
	"fmt"
	"strings"
	"tiny11-builder/internal/config"
	"tiny11-builder/internal/logger"
	"tiny11-builder/internal/utils"
)

// AppRemover 应用移除器
type AppRemover struct {
	config *config.Config
	log    *logger.Logger
}

// NewAppRemover 创建应用移除器
func NewAppRemover(cfg *config.Config, log *logger.Logger) *AppRemover {
	return &AppRemover{
		config: cfg,
		log:    log,
	}
}

// RemoveProvisionedApps 移除预装应用
func (r *AppRemover) RemoveProvisionedApps() error {
	mountPath := r.config.ScratchDir

	r.log.Section("移除预装应用")

	// 获取已安装的应用列表
	spinner := utils.NewSpinner("扫描已安装的应用包...")
	spinner.Start()

	output, err := utils.RunCommand("dism", "/English",
		fmt.Sprintf("/Image:%s", mountPath),
		"/Get-ProvisionedAppxPackages")

	spinner.Stop(err == nil)

	if err != nil {
		return fmt.Errorf("获取应用列表失败: %w", err)
	}

	// 解析包名列表
	packages := r.parsePackageNames(output)
	r.log.Info("发现 %d 个预装应用包", len(packages))

	// 要移除的应用前缀列表
	appPrefixes := r.getRemovalList()

	// 匹配要移除的包
	packagesToRemove := r.matchPackages(packages, appPrefixes)

	if len(packagesToRemove) == 0 {
		r.log.Info("没有需要移除的应用包")
		return nil
	}

	r.log.Info("准备移除 %d 个应用包", len(packagesToRemove))

	// 移除匹配的包
	removed := 0
	failed := 0

	for i, pkg := range packagesToRemove {
		pkgName := r.extractShortName(pkg)
		r.log.Info("[%d/%d] 移除: %s",
			i+1, len(packagesToRemove),
			utils.Colorize(pkgName, utils.MikuYellow))

		_, err := utils.RunCommand("dism", "/English",
			fmt.Sprintf("/Image:%s", mountPath),
			"/Remove-ProvisionedAppxPackage",
			fmt.Sprintf("/PackageName:%s", pkg))

		if err != nil {
			r.log.Warn("  ✗ 移除失败: %v", err)
			failed++
		} else {
			r.log.Success("  ✓ 移除成功")
			removed++
		}
	}

	// 显示统计
	r.log.Info("")
	r.log.Success("应用移除完成: 成功 %d, 失败 %d", removed, failed)

	return nil
}

// getRemovalList 获取要移除的应用列表
func (r *AppRemover) getRemovalList() []string {
	return []string{
				"AppUp.IntelManagementandSecurityStatus",
		"Clipchamp.Clipchamp",
		"DolbyLaboratories.DolbyAccess",
		"DolbyLaboratories.DolbyDigitalPlusDecoderOEM",
		"Microsoft.BingNews",
		"Microsoft.BingSearch",
		"Microsoft.BingWeather",
		"Microsoft.Copilot",
		"Microsoft.Windows.CrossDevice",
		"Microsoft.GamingApp",
		"Microsoft.GetHelp",
		"Microsoft.Getstarted",
		"Microsoft.Microsoft3DViewer",
		"Microsoft.MicrosoftOfficeHub",
		"Microsoft.MicrosoftSolitaireCollection",
		"Microsoft.MicrosoftStickyNotes",
		"Microsoft.MixedReality.Portal",
		"Microsoft.MSPaint",
		"Microsoft.Office.OneNote",
		"Microsoft.OfficePushNotificationUtility",
		"Microsoft.OutlookForWindows",
		"Microsoft.Paint",
		"Microsoft.People",
		"Microsoft.PowerAutomateDesktop",
		"Microsoft.SkypeApp",
		"Microsoft.StartExperiencesApp",
		"Microsoft.Todos",
		"Microsoft.Wallet",
		"Microsoft.Windows.DevHome",
		"Microsoft.Windows.Copilot",
		"Microsoft.Windows.Teams",
		"Microsoft.WindowsAlarms",
		"Microsoft.WindowsCamera",
		"microsoft.windowscommunicationsapps",
		"Microsoft.WindowsFeedbackHub",
		"Microsoft.WindowsMaps",
		"Microsoft.WindowsSoundRecorder",
		"Microsoft.WindowsTerminal",
		"Microsoft.Xbox.TCUI",
		"Microsoft.XboxApp",
		"Microsoft.XboxGameOverlay",
		"Microsoft.XboxGamingOverlay",
		"Microsoft.XboxIdentityProvider",
		"Microsoft.XboxSpeechToTextOverlay",
		"Microsoft.YourPhone",
		"Microsoft.ZuneMusic",
		"Microsoft.ZuneVideo",
		"MicrosoftCorporationII.MicrosoftFamily",
		"MicrosoftCorporationII.QuickAssist",
		"MSTeams",
		"MicrosoftTeams",
		"Microsoft.WindowsTerminal",
		"Microsoft.549981C3F5F10", 
	}
}

// parsePackageNames 解析包名列表
func (r *AppRemover) parsePackageNames(output string) []string {
	var packages []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "PackageName :") {
			pkgName := strings.TrimSpace(strings.TrimPrefix(trimmed, "PackageName :"))
			if pkgName != "" && !strings.Contains(pkgName, "...") {
				packages = append(packages, pkgName)
			}
		}
	}

	return packages
}

// matchPackages 匹配要移除的包
func (r *AppRemover) matchPackages(packages []string, prefixes []string) []string {
	var matched []string

	for _, pkg := range packages {
		for _, prefix := range prefixes {
			if strings.Contains(pkg, prefix) {
				matched = append(matched, pkg)
				break
			}
		}
	}

	return matched
}

// extractShortName 提取包的短名称
func (r *AppRemover) extractShortName(fullName string) string {
	// 从完整包名中提取简短名称
	// 例: Microsoft.BingNews_4.2.27001.0_neutral_~_8wekyb3d8bbwe -> Microsoft.BingNews
	parts := strings.Split(fullName, "_")
	if len(parts) > 0 {
		return parts[0]
	}
	return fullName
}

// RemoveSystemPackages 移除系统包
func (r *AppRemover) RemoveSystemPackages(languageCode string) error {
	mountPath := r.config.ScratchDir

	r.log.Section("移除系统组件包")

	// 获取所有包
	output, err := utils.RunCommand("dism",
		fmt.Sprintf("/Image:%s", mountPath),
		"/Get-Packages",
		"/Format:Table")

	if err != nil {
		return fmt.Errorf("获取系统包列表失败: %w", err)
	}

	// 要移除的包模式
	packagePatterns := r.getSystemPackagePatterns(languageCode)

	removed := 0
	failed := 0

	for i, pattern := range packagePatterns {
		r.log.Info("[%d/%d] 检查包: %s",
			i+1, len(packagePatterns),
			utils.Colorize(pattern, utils.MikuYellow))

		// 查找匹配的包
		packages := r.findMatchingPackages(output, pattern)

		if len(packages) == 0 {
			r.log.Info("  未找到匹配的包")
			continue
		}

		// 移除找到的包
		for _, pkg := range packages {
			r.log.Info("  移除: %s", pkg)

			_, err := utils.RunCommand("dism",
				fmt.Sprintf("/Image:%s", mountPath),
				"/Remove-Package",
				fmt.Sprintf("/PackageName:%s", pkg))

			if err != nil {
				r.log.Warn("  ✗ 移除失败: %v", err)
				failed++
			} else {
				r.log.Success("  ✓ 移除成功")
				removed++
			}
		}
	}

	r.log.Info("")
	r.log.Success("系统包移除完成: 成功 %d, 失败 %d", removed, failed)

	return nil
}

// getSystemPackagePatterns 获取系统包模式列表
func (r *AppRemover) getSystemPackagePatterns(languageCode string) []string {
	return []string{
		"Microsoft-Windows-InternetExplorer-Optional-Package~31bf3856ad364e35",
		"Microsoft-Windows-Kernel-LA57-FoD-Package~31bf3856ad364e35~amd64",
		fmt.Sprintf("Microsoft-Windows-LanguageFeatures-Handwriting-%s-Package~31bf3856ad364e35", languageCode),
		fmt.Sprintf("Microsoft-Windows-LanguageFeatures-OCR-%s-Package~31bf3856ad364e35", languageCode),
		fmt.Sprintf("Microsoft-Windows-LanguageFeatures-Speech-%s-Package~31bf3856ad364e35", languageCode),
		fmt.Sprintf("Microsoft-Windows-LanguageFeatures-TextToSpeech-%s-Package~31bf3856ad364e35", languageCode),
		"Microsoft-Windows-MediaPlayer-Package~31bf3856ad364e35",
		"Microsoft-Windows-Wallpaper-Content-Extended-FoD-Package~31bf3856ad364e35",
		"Windows-Defender-Client-Package~31bf3856ad364e35~",
		"Microsoft-Windows-WordPad-FoD-Package~",
		"Microsoft-Windows-TabletPCMath-Package~",
		"Microsoft-Windows-StepsRecorder-Package~",
	}
}

// findMatchingPackages 查找匹配的包
func (r *AppRemover) findMatchingPackages(output, pattern string) []string {
	var matches []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, pattern) {
			// 提取包标识（第一列）
			fields := strings.Fields(trimmed)
			if len(fields) > 0 {
				matches = append(matches, fields[0])
			}
		}
	}

	return matches
}