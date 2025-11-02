package remover

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tiny11-builder/internal/config"
	"tiny11-builder/internal/logger"
	"tiny11-builder/internal/utils"
)

type NanoRemover struct {
	config *config.Config
	log    *logger.Logger
}

func NewNanoRemover(cfg *config.Config, log *logger.Logger) *NanoRemover {
	return &NanoRemover{
		config: cfg,
		log:    log,
	}
}

// RemoveAggressiveApps 移除更多应用（Nano模式）
func (r *NanoRemover) RemoveAggressiveApps() error {
	mountPath := r.config.ScratchDir
	r.log.Section("移除扩展应用列表 (Nano模式)")

	// Nano 模式额外移除的应用（基于 nano11builder.ps1）
	extraPatterns := []string{
		"*Photos*",
		"*Camera*",
		"*Paint*",
		"*Notepad*",
		"*QuickAssist*",
		"*CoreAI*",
		"*PeopleExperienceHost*",
		"*PinningConfirmationDialog*",
		"*SecureAssessmentBrowser*",
		"*AV1VideoExtension*",
		"*AVCEncoderVideoExtension*",
		"*HEIFImageExtension*",
		"*HEVCVideoExtension*",
		"*RawImageExtension*",
		"*VP9VideoExtensions*",
		"*WebpImageExtension*",
		"*SecHealthUI*",
		"*CompatibilityEnhancements*",
	}

	spinner := utils.NewSpinner("获取已安装的应用包...")
	spinner.Start()

	output, err := utils.RunCommand("dism", "/English",
		fmt.Sprintf("/Image:%s", mountPath),
		"/Get-ProvisionedAppxPackages")

	spinner.Stop(err == nil)

	if err != nil {
		return fmt.Errorf("获取应用列表失败: %w", err)
	}

	// 解析包名
	var packages []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "PackageName :") {
			pkgName := strings.TrimSpace(strings.TrimPrefix(trimmed, "PackageName :"))
			if pkgName != "" {
				packages = append(packages, pkgName)
			}
		}
	}

	r.log.Info("发现 %d 个已安装的应用包", len(packages))

	removed := 0
	failed := 0

	for i, pattern := range extraPatterns {
		r.log.Info("[%d/%d] 检查应用模式: %s", i+1, len(extraPatterns),
			utils.Colorize(pattern, utils.MikuYellow))

		// 去除通配符
		cleanPattern := strings.Trim(pattern, "*")
		found := false

		for _, pkg := range packages {
			if strings.Contains(strings.ToLower(pkg), strings.ToLower(cleanPattern)) {
				found = true
				r.log.Info("  移除: %s", pkg)

				_, err := utils.RunCommand("dism", "/English",
					fmt.Sprintf("/Image:%s", mountPath),
					"/Remove-ProvisionedAppxPackage",
					fmt.Sprintf("/PackageName:%s", pkg))

				if err != nil {
					r.log.Warn("  ✗ 失败: %v", err)
					failed++
				} else {
					r.log.Success("  ✓ 成功")
					removed++
				}
			}
		}

		if !found {
			r.log.Info("  未找到匹配的包")
		}
	}

	// 尝试移除 WindowsApps 残留文件夹
	r.log.Info("清理 WindowsApps 残留文件夹...")
	windowsAppsPath := filepath.Join(mountPath, "Program Files", "WindowsApps")
	
	if utils.DirExists(windowsAppsPath) {
		entries, err := os.ReadDir(windowsAppsPath)
		if err == nil {
			cleanedFolders := 0
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				// 检查是否匹配已移除的应用
				folderName := entry.Name()
				shouldRemove := false

				for _, pattern := range extraPatterns {
					cleanPattern := strings.Trim(pattern, "*")
					if strings.Contains(strings.ToLower(folderName), strings.ToLower(cleanPattern)) {
						shouldRemove = true
						break
					}
				}

				if shouldRemove {
					folderPath := filepath.Join(windowsAppsPath, folderName)
					r.log.Info("  删除文件夹: %s", folderName)
					
					if err := os.RemoveAll(folderPath); err != nil {
						r.log.Warn("  ✗ 删除失败: %v", err)
					} else {
						cleanedFolders++
					}
				}
			}
			
			if cleanedFolders > 0 {
				r.log.Success("清理了 %d 个残留文件夹", cleanedFolders)
			}
		}
	}

	r.log.Info("")
	r.log.Success("扩展应用移除完成: 成功 %d, 失败 %d", removed, failed)
	return nil
}

// RemoveAggressivePackages 移除更多系统包
func (r *NanoRemover) RemoveAggressivePackages(languageCode string) error {
	mountPath := r.config.ScratchDir
	r.log.Section("移除扩展系统包 (Nano模式)")

	// 基于 nano11builder.ps1 的包列表
	packagePatterns := []string{
		// Legacy 组件
		"Microsoft-Windows-InternetExplorer-Optional-Package~",
		"Microsoft-Windows-MediaPlayer-Package~",
		"Microsoft-Windows-WordPad-FoD-Package~",
		"Microsoft-Windows-StepsRecorder-Package~",
		"Microsoft-Windows-MSPaint-FoD-Package~",
		"Microsoft-Windows-SnippingTool-FoD-Package~",
		"Microsoft-Windows-TabletPCMath-Package~",
		"Microsoft-Windows-Xps-Xps-Viewer-Opt-Package~",
		"Microsoft-Windows-PowerShell-ISE-FOD-Package~",
		"OpenSSH-Client-Package~",

		// 语言功能
		fmt.Sprintf("Microsoft-Windows-LanguageFeatures-Handwriting-%s-Package~", languageCode),
		fmt.Sprintf("Microsoft-Windows-LanguageFeatures-OCR-%s-Package~", languageCode),
		fmt.Sprintf("Microsoft-Windows-LanguageFeatures-Speech-%s-Package~", languageCode),
		fmt.Sprintf("Microsoft-Windows-LanguageFeatures-TextToSpeech-%s-Package~", languageCode),

		// IME (亚洲语言输入法)
		"*IME-ja-jp*",
		"*IME-ko-kr*",
		"*IME-zh-cn*",
		"*IME-zh-tw*",

		// 核心功能 (Nano 模式激进移除)
		"Windows-Defender-Client-Package~",
		"Microsoft-Windows-Search-Engine-Client-Package~",
		"Microsoft-Windows-Kernel-LA57-FoD-Package~",

		// 安全和身份
		"Microsoft-Windows-Hello-Face-Package~",
		"Microsoft-Windows-Hello-BioEnrollment-Package~",
		"Microsoft-Windows-BitLocker-DriveEncryption-FVE-Package~",
		"Microsoft-Windows-TPM-WMI-Provider-Package~",

		// 辅助功能
		"Microsoft-Windows-Narrator-App-Package~",
		"Microsoft-Windows-Magnifier-App-Package~",

		// 其他功能
		"Microsoft-Windows-Printing-PMCPPC-FoD-Package~",
		"Microsoft-Windows-WebcamExperience-Package~",
		"Microsoft-Media-MPEG2-Decoder-Package~",
		"Microsoft-Windows-Wallpaper-Content-Extended-FoD-Package~",
	}

	spinner := utils.NewSpinner("获取系统包列表...")
	spinner.Start()

	output, err := utils.RunCommand("dism",
		fmt.Sprintf("/image:%s", mountPath),
		"/Get-Packages",
		"/Format:Table")

	spinner.Stop(err == nil)

	if err != nil {
		return fmt.Errorf("获取系统包列表失败: %w", err)
	}

	lines := strings.Split(output, "\n")
	removed := 0
	failed := 0

	for i, pattern := range packagePatterns {
		r.log.Info("[%d/%d] 检查包模式: %s", i+1, len(packagePatterns),
			utils.Colorize(pattern, utils.MikuYellow))

		foundPackages := false

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// 检查是否匹配模式
			matchFound := false
			if strings.Contains(pattern, "*") {
				// 通配符匹配
				cleanPattern := strings.Trim(pattern, "*")
				if strings.Contains(strings.ToLower(line), strings.ToLower(cleanPattern)) {
					matchFound = true
				}
			} else {
				// 精确前缀匹配
				if strings.Contains(line, pattern) {
					matchFound = true
				}
			}

			if matchFound {
				foundPackages = true
				fields := strings.Fields(line)
				if len(fields) > 0 {
					pkgName := fields[0]
					r.log.Info("  移除: %s", pkgName)

					_, err := utils.RunCommand("dism",
						fmt.Sprintf("/image:%s", mountPath),
						"/Remove-Package",
						fmt.Sprintf("/PackageName:%s", pkgName))

					if err != nil {
						r.log.Warn("  ✗ 失败: %v", err)
						failed++
					} else {
						r.log.Success("  ✓ 成功")
						removed++
					}
				}
			}
		}

		if !foundPackages {
			r.log.Info("  未找到匹配的包")
		}
	}

	r.log.Info("")
	r.log.Success("系统包移除完成: 成功 %d, 失败 %d", removed, failed)
	return nil
}

// RemoveNativeImages 移除 .NET Native Images
func (r *NanoRemover) RemoveNativeImages() error {
	mountPath := r.config.ScratchDir
	assemblyPath := filepath.Join(mountPath, "Windows", "assembly")

	r.log.Section("移除预编译 .NET 程序集")

	if !utils.DirExists(assemblyPath) {
		r.log.Info("assembly 目录不存在，跳过")
		return nil
	}

	// 查找所有 NativeImages_* 目录
	entries, err := os.ReadDir(assemblyPath)
	if err != nil {
		return fmt.Errorf("读取 assembly 目录失败: %w", err)
	}

	removed := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), "NativeImages_") {
			niPath := filepath.Join(assemblyPath, entry.Name())
			r.log.Info("移除: %s", entry.Name())

			if err := os.RemoveAll(niPath); err != nil {
				r.log.Warn("  ✗ 失败: %v", err)
			} else {
				r.log.Success("  ✓ 成功")
				removed++
			}
		}
	}

	if removed == 0 {
		r.log.Info("未找到 NativeImages 目录")
	} else {
		r.log.Success("移除了 %d 个 Native Images 目录", removed)
	}

	return nil
}

// SlimDriverStore 精简驱动存储
func (r *NanoRemover) SlimDriverStore() error {
	mountPath := r.config.ScratchDir
	driverRepo := filepath.Join(mountPath, "Windows", "System32", "DriverStore", "FileRepository")

	r.log.Section("精简 DriverStore (移除非必需驱动)")

	if !utils.DirExists(driverRepo) {
		r.log.Warn("DriverStore 目录不存在: %s", driverRepo)
		return nil
	}

	// 要移除的驱动模式（基于 nano11builder.ps1）
	patternsToRemove := []string{
		"prn*",         // 打印机驱动
		"scan*",        // 扫描仪驱动
		"mfd*",         // 多功能设备驱动
		"wscsmd.inf*",  // 智能卡读卡器
		"tapdrv*",      // 磁带驱动器
		"rdpbus.inf*",  // 远程桌面虚拟总线
		"tdibth.inf*",  // 蓝牙 PAN
	}

	entries, err := os.ReadDir(driverRepo)
	if err != nil {
		return fmt.Errorf("读取 DriverStore 目录失败: %w", err)
	}

	removed := 0
	skipped := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		driverName := entry.Name()
		shouldRemove := false

		for _, pattern := range patternsToRemove {
			matched, _ := filepath.Match(pattern, strings.ToLower(driverName))
			if matched {
				shouldRemove = true
				break
			}
		}

		if shouldRemove {
			driverPath := filepath.Join(driverRepo, driverName)
			r.log.Info("移除驱动包: %s", driverName)

			if err := os.RemoveAll(driverPath); err != nil {
				r.log.Warn("  ✗ 失败: %v", err)
				skipped++
			} else {
				r.log.Success("  ✓ 成功")
				removed++
			}
		}
	}

	r.log.Info("")
	r.log.Success("驱动存储精简完成: 移除 %d 个, 跳过 %d 个", removed, skipped)
	return nil
}

// SlimFonts 精简字体
func (r *NanoRemover) SlimFonts() error {
	mountPath := r.config.ScratchDir
	fontsPath := filepath.Join(mountPath, "Windows", "Fonts")

	r.log.Section("精简系统字体 (只保留必需字体)")

	if !utils.DirExists(fontsPath) {
		r.log.Warn("Fonts 目录不存在: %s", fontsPath)
		return nil
	}

	// 保留的字体模式（基于 nano11builder.ps1）
	keepPatterns := []string{
		"segoe*",
		"tahoma*",
		"marlett.ttf",
		"8541oem.fon",
		"segui*",
		"consol*",
		"lucon*",
		"calibri*",
		"arial*",
		"times*",
		"cou*",
		"8*",
	}

	// 明确需要移除的亚洲字体
	removePatterns := []string{
		"mingli*",
		"msjh*",
		"msyh*",
		"malgun*",
		"meiryo*",
		"yugoth*",
		"segoeuihistoric.ttf",
	}

	entries, err := os.ReadDir(fontsPath)
	if err != nil {
		return fmt.Errorf("读取 Fonts 目录失败: %w", err)
	}

	removed := 0
	kept := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fontName := entry.Name()
		fontNameLower := strings.ToLower(fontName)
		shouldKeep := false
		shouldRemove := false

		// 检查是否在保留列表
		for _, pattern := range keepPatterns {
			matched, _ := filepath.Match(pattern, fontNameLower)
			if matched {
				shouldKeep = true
				break
			}
		}

		// 检查是否在移除列表
		for _, pattern := range removePatterns {
			matched, _ := filepath.Match(pattern, fontNameLower)
			if matched {
				shouldRemove = true
				break
			}
		}

		// 如果不在保留列表或在移除列表，则删除
		if shouldRemove || !shouldKeep {
			fontPath := filepath.Join(fontsPath, fontName)

			if err := os.Remove(fontPath); err != nil {
				// 静默忽略错误
			} else {
				removed++
			}
		} else {
			kept++
		}
	}

	r.log.Info("")
	r.log.Success("字体精简完成: 移除 %d 个, 保留 %d 个", removed, kept)
	return nil
}

// RemoveSystemFolders 移除系统文件夹
func (r *NanoRemover) RemoveSystemFolders() error {
	mountPath := r.config.ScratchDir
	r.log.Section("移除非必需系统文件夹")

	// 基于 nano11builder.ps1
	foldersToRemove := []struct {
		path string
		desc string
	}{
		{filepath.Join(mountPath, "Windows", "Speech", "Engines", "TTS"), "TTS 语音合成引擎"},
		{filepath.Join(mountPath, "ProgramData", "Microsoft", "Windows Defender", "Definition Updates"), "Defender 定义更新"},
		{filepath.Join(mountPath, "Windows", "System32", "InputMethod", "CHS"), "简体中文输入法"},
		{filepath.Join(mountPath, "Windows", "System32", "InputMethod", "CHT"), "繁体中文输入法"},
		{filepath.Join(mountPath, "Windows", "System32", "InputMethod", "JPN"), "日文输入法"},
		{filepath.Join(mountPath, "Windows", "System32", "InputMethod", "KOR"), "韩文输入法"},
		{filepath.Join(mountPath, "Windows", "Temp"), "临时文件"},
		{filepath.Join(mountPath, "Windows", "Web"), "Web 内容"},
		{filepath.Join(mountPath, "Windows", "Help"), "帮助文件"},
		{filepath.Join(mountPath, "Windows", "Cursors"), "光标主题"},
	}

	removed := 0
	skipped := 0

	for i, folder := range foldersToRemove {
		r.log.Info("[%d/%d] %s", i+1, len(foldersToRemove), folder.desc)

		if !utils.DirExists(folder.path) {
			r.log.Info("  目录不存在，跳过")
			skipped++
			continue
		}

		if err := os.RemoveAll(folder.path); err != nil {
			r.log.Warn("  ✗ 失败: %v", err)
			skipped++
		} else {
			r.log.Success("  ✓ 成功")
			removed++
		}
	}

	r.log.Info("")
	r.log.Success("系统文件夹移除完成: 成功 %d, 跳过 %d", removed, skipped)
	return nil
}

// RemoveSystemServices 移除系统服务
func (r *NanoRemover) RemoveSystemServices() error {
	mountPath := r.config.ScratchDir
	r.log.Section("移除非必需系统服务")

	// 加载注册表
	systemHive := filepath.Join(mountPath, "Windows", "System32", "config", "SYSTEM")
	
	r.log.Info("加载 SYSTEM 注册表...")
	_, err := utils.RunCommand("reg", "load", "HKLM\\zSYSTEM", systemHive)
	if err != nil {
		return fmt.Errorf("加载 SYSTEM hive 失败: %w", err)
	}

	// 确保卸载
	defer func() {
		r.log.Info("卸载 SYSTEM 注册表...")
		utils.RunCommand("reg", "unload", "HKLM\\zSYSTEM")
	}()

	// 基于 nano11builder.ps1 的服务列表
	servicesToRemove := []string{
		"Spooler",              // 打印后台处理程序
		"PrintNotify",          // 打印机通知
		"Fax",                  // 传真
		"RemoteRegistry",       // 远程注册表
		"diagsvc",              // 诊断服务
		"WerSvc",               // Windows 错误报告
		"PcaSvc",               // 程序兼容性助手
		"MapsBroker",           // 地图管理器
		"WalletService",        // 电子钱包
		"BthAvctpSvc",          // 蓝牙 AVCTP
		"BluetoothUserService", // 蓝牙用户服务
		"wuauserv",             // Windows Update
		"UsoSvc",               // Update Orchestrator
		"WaaSMedicSvc",         // Windows Update Medic
	}

	removed := 0
	failed := 0

	for i, service := range servicesToRemove {
		r.log.Info("[%d/%d] 移除服务: %s", i+1, len(servicesToRemove),
			utils.Colorize(service, utils.MikuYellow))

		servicePath := fmt.Sprintf("HKLM\\zSYSTEM\\ControlSet001\\Services\\%s", service)

		_, err := utils.RunCommand("reg", "delete", servicePath, "/f")

		if err != nil {
			// 服务可能不存在，这是正常的
			r.log.Info("  服务不存在或已移除")
			failed++
		} else {
			r.log.Success("  ✓ 成功")
			removed++
		}
	}

	r.log.Info("")
	r.log.Success("服务移除完成: 成功 %d, 不存在 %d", removed, failed)
	return nil
}

// CleanupWindowsAppsLeftovers 清理 WindowsApps 残留
func (r *NanoRemover) CleanupWindowsAppsLeftovers(packagesToRemove []string) error {
	mountPath := r.config.ScratchDir
	windowsAppsPath := filepath.Join(mountPath, "Program Files", "WindowsApps")

	r.log.Info("清理 WindowsApps 残留文件夹...")

	if !utils.DirExists(windowsAppsPath) {
		r.log.Info("WindowsApps 目录不存在，跳过")
		return nil
	}

	entries, err := os.ReadDir(windowsAppsPath)
	if err != nil {
		return fmt.Errorf("读取 WindowsApps 目录失败: %w", err)
	}

	cleaned := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		folderName := entry.Name()
		shouldRemove := false

		// 检查是否匹配已移除的应用
		for _, pkg := range packagesToRemove {
			// 提取包的基本名称
			pkgBaseName := strings.Split(pkg, "_")[0]
			if strings.Contains(folderName, pkgBaseName) {
				shouldRemove = true
				break
			}
		}

		if shouldRemove {
			folderPath := filepath.Join(windowsAppsPath, folderName)
			r.log.Info("  删除残留: %s", folderName)

			if err := os.RemoveAll(folderPath); err != nil {
				r.log.Warn("  ✗ 失败: %v", err)
			} else {
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		r.log.Success("清理了 %d 个残留文件夹", cleaned)
	} else {
		r.log.Info("没有发现残留文件夹")
	}

	return nil
}