package registry

import (
	"tiny11-builder/internal/utils"
)

// ApplyTweaks 应用注册表优化
func (m *Manager) ApplyTweaks() error {
	m.log.Section("应用注册表优化")

	tweaks := []struct {
		desc string
		fn   func() error
	}{
		{"绕过系统要求检查", m.bypassSystemRequirements},
		{"禁用赞助应用和广告", m.disableSponsoredApps},
		{"启用本地账户创建", m.enableLocalAccounts},
		{"禁用预留存储空间", m.disableReservedStorage},
		{"禁用BitLocker设备加密", m.disableBitLocker},
		{"禁用聊天图标", m.disableChatIcon},
		{"移除Edge注册表项", m.removeEdgeRegistry},
		{"禁用OneDrive文件夹备份", m.disableOneDriveBackup},
		{"禁用遥测和数据收集", m.disableTelemetry},
		{"阻止DevHome和Outlook安装", m.preventDevHomeOutlook},
		{"禁用Windows Copilot", m.disableCopilot},
		{"禁用Teams自动安装", m.disableTeams},
	}

	success := 0
	failed := 0

	for i, tweak := range tweaks {
		m.log.Info("[%d/%d] %s", i+1, len(tweaks), tweak.desc)
		if err := tweak.fn(); err != nil {
			m.log.Warn("  ✗ 失败: %v", err)
			failed++
		} else {
			m.log.Success("  ✓ 成功")
			success++
		}
	}

	m.log.Info("")
	m.log.Success("注册表优化完成: 成功 %d, 失败 %d", success, failed)

	return nil
}

// ApplyCoreTweaks 应用Core版本额外优化
func (m *Manager) ApplyCoreTweaks() error {
	m.log.Section("应用Core版本特殊优化")

	tweaks := []struct {
		desc string
		fn   func() error
	}{
		{"禁用Windows Defender", m.disableDefenderRegistry},
		{"禁用Windows Update", m.disableWindowsUpdateRegistry},
		{"隐藏设置页面", m.hideSettingsPages},
	}

	success := 0
	failed := 0

	for i, tweak := range tweaks {
		m.log.Info("[%d/%d] %s", i+1, len(tweaks), tweak.desc)
		if err := tweak.fn(); err != nil {
			m.log.Warn("  ✗ 失败: %v", err)
			failed++
		} else {
			m.log.Success("  ✓ 成功")
			success++
		}
	}

	m.log.Info("")
	m.log.Success("Core优化完成: 成功 %d, 失败 %d", success, failed)

	return nil
}

// ApplyNanoTweaks 应用 Nano 版本特殊优化
func (m *Manager) ApplyNanoTweaks() error {
	m.log.Section("应用 Nano 版本特殊优化")

	tweaks := []struct {
		desc string
		fn   func() error
	}{
		{"隐藏 Windows Update 和 Defender 设置页", m.hideNanoSettingsPages},
	}

	success := 0
	failed := 0

	for i, tweak := range tweaks {
		m.log.Info("[%d/%d] %s", i+1, len(tweaks), tweak.desc)
		if err := tweak.fn(); err != nil {
			m.log.Warn("  ✗ 失败: %v", err)
			failed++
		} else {
			m.log.Success("  ✓ 成功")
			success++
		}
	}

	m.log.Info("")
	m.log.Success("Nano 优化完成: 成功 %d, 失败 %d", success, failed)
	return nil
}

func (m *Manager) hideNanoSettingsPages() error {
	// 基于 nano11builder.ps1
	sets := []regSet{
		{
			"HKLM\\zSOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\Explorer",
			"SettingsPageVisibility",
			"REG_SZ",
			"hide:virus;windowsupdate",
		},
	}
	return applyRegSets(sets)
}

// ApplyBootTweaks 应用Boot镜像优化
func (m *Manager) ApplyBootTweaks() error {
	m.log.Section("应用Boot镜像优化")
	return m.bypassSystemRequirements()
}

// bypassSystemRequirements 绕过系统要求
func (m *Manager) bypassSystemRequirements() error {
	sets := []regSet{
		{"HKLM\\zDEFAULT\\Control Panel\\UnsupportedHardwareNotificationCache", "SV1", "REG_DWORD", "0"},
		{"HKLM\\zDEFAULT\\Control Panel\\UnsupportedHardwareNotificationCache", "SV2", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Control Panel\\UnsupportedHardwareNotificationCache", "SV1", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Control Panel\\UnsupportedHardwareNotificationCache", "SV2", "REG_DWORD", "0"},
		{"HKLM\\zSYSTEM\\Setup\\LabConfig", "BypassCPUCheck", "REG_DWORD", "1"},
		{"HKLM\\zSYSTEM\\Setup\\LabConfig", "BypassRAMCheck", "REG_DWORD", "1"},
		{"HKLM\\zSYSTEM\\Setup\\LabConfig", "BypassSecureBootCheck", "REG_DWORD", "1"},
		{"HKLM\\zSYSTEM\\Setup\\LabConfig", "BypassStorageCheck", "REG_DWORD", "1"},
		{"HKLM\\zSYSTEM\\Setup\\LabConfig", "BypassTPMCheck", "REG_DWORD", "1"},
		{"HKLM\\zSYSTEM\\Setup\\MoSetup", "AllowUpgradesWithUnsupportedTPMOrCPU", "REG_DWORD", "1"},
	}
	return applyRegSets(sets)
}

// disableSponsoredApps 禁用赞助应用
func (m *Manager) disableSponsoredApps() error {
	sets := []regSet{
		{"HKLM\\zNTUSER\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "OemPreInstalledAppsEnabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "PreInstalledAppsEnabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "SilentInstalledAppsEnabled", "REG_DWORD", "0"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\CloudContent", "DisableWindowsConsumerFeatures", "REG_DWORD", "1"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "ContentDeliveryAllowed", "REG_DWORD", "0"},
		{"HKLM\\zSOFTWARE\\Microsoft\\PolicyManager\\current\\device\\Start", "ConfigureStartPins", "REG_SZ", "{\"pinnedList\": [{}]}"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "FeatureManagementEnabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "PreInstalledAppsEverEnabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "SoftLandingEnabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "SubscribedContentEnabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "SubscribedContent-310093Enabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "SubscribedContent-338388Enabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "SubscribedContent-338389Enabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "SubscribedContent-338393Enabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "SubscribedContent-353694Enabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "SubscribedContent-353696Enabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager", "SystemPaneSuggestionsEnabled", "REG_DWORD", "0"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\PushToInstall", "DisablePushToInstall", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\MRT", "DontOfferThroughWUAU", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\CloudContent", "DisableConsumerAccountStateContent", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\CloudContent", "DisableCloudOptimizedContent", "REG_DWORD", "1"},
	}

	if err := applyRegSets(sets); err != nil {
		return err
	}

	// 删除特定键
	utils.RunCommand("reg", "delete", "HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager\\Subscriptions", "/f")
	utils.RunCommand("reg", "delete", "HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\ContentDeliveryManager\\SuggestedApps", "/f")

	return nil
}

// enableLocalAccounts 启用本地账户
func (m *Manager) enableLocalAccounts() error {
	sets := []regSet{
		{"HKLM\\zSOFTWARE\\Microsoft\\Windows\\CurrentVersion\\OOBE", "BypassNRO", "REG_DWORD", "1"},
	}
	return applyRegSets(sets)
}

// disableReservedStorage 禁用预留存储
func (m *Manager) disableReservedStorage() error {
	sets := []regSet{
		{"HKLM\\zSOFTWARE\\Microsoft\\Windows\\CurrentVersion\\ReserveManager", "ShippedWithReserves", "REG_DWORD", "0"},
	}
	return applyRegSets(sets)
}

// disableBitLocker 禁用BitLocker
func (m *Manager) disableBitLocker() error {
	sets := []regSet{
		{"HKLM\\zSYSTEM\\ControlSet001\\Control\\BitLocker", "PreventDeviceEncryption", "REG_DWORD", "1"},
	}
	return applyRegSets(sets)
}

// disableChatIcon 禁用聊天图标
func (m *Manager) disableChatIcon() error {
	sets := []regSet{
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\Windows Chat", "ChatIcon", "REG_DWORD", "3"},
		{"HKLM\\zNTUSER\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced", "TaskbarMn", "REG_DWORD", "0"},
	}
	return applyRegSets(sets)
}

// removeEdgeRegistry 移除Edge注册表
func (m *Manager) removeEdgeRegistry() error {
	utils.RunCommand("reg", "delete", "HKLM\\zSOFTWARE\\WOW6432Node\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\Microsoft Edge", "/f")
	utils.RunCommand("reg", "delete", "HKLM\\zSOFTWARE\\WOW6432Node\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\Microsoft Edge Update", "/f")
	return nil
}

// disableOneDriveBackup 禁用OneDrive备份
func (m *Manager) disableOneDriveBackup() error {
	sets := []regSet{
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\OneDrive", "DisableFileSyncNGSC", "REG_DWORD", "1"},
	}
	return applyRegSets(sets)
}

// disableTelemetry 禁用遥测
func (m *Manager) disableTelemetry() error {
	sets := []regSet{
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\AdvertisingInfo", "Enabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Windows\\CurrentVersion\\Privacy", "TailoredExperiencesWithDiagnosticDataEnabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Speech_OneCore\\Settings\\OnlineSpeechPrivacy", "HasAccepted", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Input\\TIPC", "Enabled", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\InputPersonalization", "RestrictImplicitInkCollection", "REG_DWORD", "1"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\InputPersonalization", "RestrictImplicitTextCollection", "REG_DWORD", "1"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\InputPersonalization\\TrainedDataStore", "HarvestContacts", "REG_DWORD", "0"},
		{"HKLM\\zNTUSER\\Software\\Microsoft\\Personalization\\Settings", "AcceptedPrivacyPolicy", "REG_DWORD", "0"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\DataCollection", "AllowTelemetry", "REG_DWORD", "0"},
		{"HKLM\\zSYSTEM\\ControlSet001\\Services\\dmwappushservice", "Start", "REG_DWORD", "4"},
	}
	return applyRegSets(sets)
}

// preventDevHomeOutlook 阻止DevHome和Outlook安装
func (m *Manager) preventDevHomeOutlook() error {
	sets := []regSet{
		{"HKLM\\zSOFTWARE\\Microsoft\\Windows\\CurrentVersion\\WindowsUpdate\\Orchestrator\\UScheduler\\OutlookUpdate", "workCompleted", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Microsoft\\Windows\\CurrentVersion\\WindowsUpdate\\Orchestrator\\UScheduler\\DevHomeUpdate", "workCompleted", "REG_DWORD", "1"},
	}

	if err := applyRegSets(sets); err != nil {
		return err
	}

	utils.RunCommand("reg", "delete", "HKLM\\zSOFTWARE\\Microsoft\\WindowsUpdate\\Orchestrator\\UScheduler_Oobe\\OutlookUpdate", "/f")
	utils.RunCommand("reg", "delete", "HKLM\\zSOFTWARE\\Microsoft\\WindowsUpdate\\Orchestrator\\UScheduler_Oobe\\DevHomeUpdate", "/f")

	return nil
}

// disableCopilot 禁用Copilot
func (m *Manager) disableCopilot() error {
	sets := []regSet{
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\WindowsCopilot", "TurnOffWindowsCopilot", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Edge", "HubsSidebarEnabled", "REG_DWORD", "0"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\Explorer", "DisableSearchBoxSuggestions", "REG_DWORD", "1"},
	}
	return applyRegSets(sets)
}

// disableTeams 禁用Teams
func (m *Manager) disableTeams() error {
	sets := []regSet{
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Teams", "DisableInstallation", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\Windows Mail", "PreventRun", "REG_DWORD", "1"},
	}
	return applyRegSets(sets)
}

// disableDefenderRegistry 禁用Windows Defender (Core版本)
func (m *Manager) disableDefenderRegistry() error {
	// 禁用Defender服务
	services := []string{
		"WinDefend",
		"WdNisSvc",
		"WdNisDrv",
		"WdFilter",
		"Sense",
	}

	for _, service := range services {
		path := "HKLM\\zSYSTEM\\ControlSet001\\Services\\" + service
		utils.RunCommand("reg", "add", path, "/v", "Start", "/t", "REG_DWORD", "/d", "4", "/f")
	}

	// 禁用Defender策略
	sets := []regSet{
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows Defender", "DisableAntiSpyware", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows Defender\\Real-Time Protection", "DisableRealtimeMonitoring", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows Defender\\Real-Time Protection", "DisableBehaviorMonitoring", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows Defender\\Real-Time Protection", "DisableOnAccessProtection", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows Defender\\Real-Time Protection", "DisableScanOnRealtimeEnable", "REG_DWORD", "1"},
	}

	return applyRegSets(sets)
}

// disableWindowsUpdateRegistry 禁用Windows Update (Core版本)
func (m *Manager) disableWindowsUpdateRegistry() error {
	sets := []regSet{
		// 禁用Windows Update服务
		{"HKLM\\zSYSTEM\\ControlSet001\\Services\\wuauserv", "Start", "REG_DWORD", "4"},
		// 禁用Update Orchestrator
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\WindowsUpdate", "DoNotConnectToWindowsUpdateInternetLocations", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\WindowsUpdate", "DisableWindowsUpdateAccess", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\WindowsUpdate", "WUServer", "REG_SZ", "localhost"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\WindowsUpdate", "WUStatusServer", "REG_SZ", "localhost"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\WindowsUpdate", "UpdateServiceUrlAlternate", "REG_SZ", "localhost"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\WindowsUpdate\\AU", "UseWUServer", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Policies\\Microsoft\\Windows\\WindowsUpdate\\AU", "NoAutoUpdate", "REG_DWORD", "1"},
		{"HKLM\\zSOFTWARE\\Microsoft\\Windows\\CurrentVersion\\OOBE", "DisableOnline", "REG_DWORD", "1"},
	}

	if err := applyRegSets(sets); err != nil {
		return err
	}

	// 删除Update服务
	utils.RunCommand("reg", "delete", "HKLM\\zSYSTEM\\ControlSet001\\Services\\WaaSMedicSVC", "/f")
	utils.RunCommand("reg", "delete", "HKLM\\zSYSTEM\\ControlSet001\\Services\\UsoSvc", "/f")

	// 添加RunOnce命令以在首次启动后禁用
	utils.RunCommand("reg", "add", "HKLM\\zSOFTWARE\\Microsoft\\Windows\\CurrentVersion\\RunOnce",
		"/v", "StopWUPostOOBE1", "/t", "REG_SZ", "/d", "net stop wuauserv", "/f")
	utils.RunCommand("reg", "add", "HKLM\\zSOFTWARE\\Microsoft\\Windows\\CurrentVersion\\RunOnce",
		"/v", "StopWUPostOOBE2", "/t", "REG_SZ", "/d", "sc stop wuauserv", "/f")
	utils.RunCommand("reg", "add", "HKLM\\zSOFTWARE\\Microsoft\\Windows\\CurrentVersion\\RunOnce",
		"/v", "StopWUPostOOBE3", "/t", "REG_SZ", "/d", "sc config wuauserv start= disabled", "/f")

	return nil
}

// hideSettingsPages 隐藏设置页面 (Core版本)
func (m *Manager) hideSettingsPages() error {
	sets := []regSet{
		{"HKLM\\zSOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\Explorer", "SettingsPageVisibility", "REG_SZ", "hide:virus;windowsupdate"},
	}
	return applyRegSets(sets)
}

// regSet 注册表设置结构
type regSet struct {
	path  string
	name  string
	typ   string
	value string
}

// applyRegSets 批量应用注册表设置
func applyRegSets(sets []regSet) error {
	for _, set := range sets {
		_, err := utils.RunCommand("reg", "add", set.path, "/v", set.name, "/t", set.typ, "/d", set.value, "/f")
		if err != nil {
			// 记录错误但继续
			continue
		}
	}
	return nil
}

func (m *Manager) hideWindowsUpdatePage() error {
	sets := []regSet{
		{"HKLM\\zSOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\Explorer",
			"SettingsPageVisibility", "REG_SZ", "hide:virus;windowsupdate"},
	}
	return applyRegSets(sets)
}

func (m *Manager) hideVirusProtectionPage() error {
	// 已在 hideWindowsUpdatePage 中一起设置
	return nil
}