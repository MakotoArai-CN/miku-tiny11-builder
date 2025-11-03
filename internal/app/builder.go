package app

import (
	"fmt"
	"path/filepath"

	"tiny11-builder/internal/config"
	"tiny11-builder/internal/image"
	"tiny11-builder/internal/logger"
	"tiny11-builder/internal/preinstall"
	"tiny11-builder/internal/registry"
	"tiny11-builder/internal/remover"
	"tiny11-builder/internal/theme"
	"tiny11-builder/internal/utils"
)

type Tiny11Builder struct {
	config       *config.Config
	log          *logger.Logger
	imgMgr       *image.Manager
	regMgr       *registry.Manager
	remover      *remover.AppRemover
	themeMgr     *theme.Manager
	themeApplier *theme.Applier
	preinstallMgr *preinstall.Manager
	outputISO    string
}

func NewTiny11Builder(cfg *config.Config, log *logger.Logger) *Tiny11Builder {
	themeMgr := theme.NewManager(cfg, log)
	builder := &Tiny11Builder{
		config:        cfg,
		log:           log,
		imgMgr:        image.NewManager(cfg, log),
		regMgr:        registry.NewManager(cfg, log),
		remover:       remover.NewAppRemover(cfg, log),
		themeMgr:      themeMgr,
		preinstallMgr: preinstall.NewManager(cfg, log),
	}
	builder.themeApplier = theme.NewApplier(cfg, log, themeMgr)
	return builder
}

func (b *Tiny11Builder) Build() error {
	b.log.Header("Tiny11 Builder - 标准版")

	var imageUnmounted = false

	if err := b.executeBasicSteps(); err != nil {
		return err
	}

	b.log.Step(3, "获取镜像信息")
	imageInfo, err := b.imgMgr.GetImageInfo()
	if err != nil {
		return fmt.Errorf("获取镜像信息失败: %w", err)
	}
	b.log.Info("架构: %s, 语言: %s, 索引: %d", imageInfo.Architecture, imageInfo.Language, imageInfo.Index)

	b.log.Step(4, "挂载install.wim")
	if err := b.imgMgr.MountInstallWim(imageInfo.Index); err != nil {
		return fmt.Errorf("挂载失败: %w", err)
	}

	defer func() {
		if !imageUnmounted {
			b.log.Info("执行紧急清理...")
			b.regMgr.UnloadHives()
			b.imgMgr.UnmountImage(false)
		}
	}()

	if err := b.executeRemovalSteps(); err != nil {
		return err
	}

	b.log.Step(7, "应用注册表优化")
	if err := b.regMgr.LoadHives(); err != nil {
		return fmt.Errorf("加载注册表失败: %w", err)
	}
	if err := b.regMgr.ApplyTweaks(); err != nil {
		b.log.Warn("应用优化失败: %v", err)
	}

	if b.config.ThemeName != "" {
		b.log.Step(8, "应用自定义主题: "+b.config.ThemeName)
		if err := b.applyTheme(imageInfo.Name); err != nil {
			b.log.Warn("主题应用失败: %v", err)
		}
	} else {
		b.log.Step(8, "跳过主题自定义 (未指定主题)")
	}

	b.log.Info("卸载注册表Hive...")
	if err := b.regMgr.UnloadHives(); err != nil {
		b.log.Warn("卸载注册表失败: %v", err)
	}

	//  预装软件安装 
	if len(b.config.PreinstallApps) > 0 {
		b.log.Step(9, "预装软件到系统")
		if err := b.installPreinstallApps(); err != nil {
			b.log.Warn("预装软件安装失败: %v", err)
		}
	} else {
		b.log.Info("跳过预装软件 (未选择)")
	}

	b.copyAutounattend()

	if err := b.executeFinalSteps(imageInfo); err != nil {
		return err
	}

	imageUnmounted = true
	return nil
}

//  预装软件安装函数 
func (b *Tiny11Builder) installPreinstallApps() error {
	if err := b.preinstallMgr.InstallApps(b.config.PreinstallApps); err != nil {
		return fmt.Errorf("预装软件安装失败: %w", err)
	}
	return nil
}

func (b *Tiny11Builder) applyTheme(originalEditionName string) error {
	themePath := filepath.Join(b.config.ThemesDir, b.config.ThemeName)
	if !utils.DirExists(themePath) {
		return fmt.Errorf("主题目录不存在: %s", themePath)
	}

	themeFile := filepath.Join(themePath, "theme.json")
	if !utils.FileExists(themeFile) {
		return fmt.Errorf("主题配置文件不存在: %s", themeFile)
	}

	activeTheme, err := b.themeMgr.LoadTheme(b.config.ThemeName)
	if err != nil {
		return fmt.Errorf("加载主题失败: %w", err)
	}

	warnings := b.themeMgr.ValidateTheme(activeTheme)
	if len(warnings) > 0 {
		b.log.Warn("主题验证警告:")
		for _, warn := range warnings {
			b.log.Warn("  • %s", warn)
		}
	}

	if err := b.themeApplier.ApplyTheme(activeTheme); err != nil {
		return err
	}

	return nil
}

func (b *Tiny11Builder) executeBasicSteps() error {
	b.log.Step(1, "验证ISO镜像")
	if err := b.imgMgr.ValidateISO(); err != nil {
		return fmt.Errorf("ISO验证失败: %w", err)
	}

	b.log.Step(2, "复制Windows镜像文件")
	if err := b.imgMgr.CopyImageFiles(); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}

	return nil
}

func (b *Tiny11Builder) executeRemovalSteps() error {
	b.log.Step(5, "移除预装应用")
	if err := b.remover.RemoveProvisionedApps(); err != nil {
		return fmt.Errorf("移除应用失败: %w", err)
	}

	b.log.Step(6, "移除Edge和OneDrive")
	if err := b.remover.RemoveEdge(); err != nil {
		b.log.Warn("移除Edge失败: %v", err)
	}

	if err := b.remover.RemoveOneDrive(); err != nil {
		b.log.Warn("移除OneDrive失败: %v", err)
	}

	if err := b.remover.RemoveScheduledTasks(); err != nil {
		b.log.Warn("移除计划任务失败: %v", err)
	}

	return nil
}

func (b *Tiny11Builder) executeFinalSteps(imageInfo *image.ImageInfo) error {
	b.log.Step(10, "清理和优化镜像")
	if err := b.imgMgr.CleanupImage(); err != nil {
		b.log.Warn("清理镜像失败（跳过）: %v", err)
	}

	b.log.Step(11, "导出优化后的镜像")
	if err := b.imgMgr.UnmountImage(true); err != nil {
		return fmt.Errorf("卸载失败: %w", err)
	}

	if err := b.imgMgr.ExportImage(imageInfo.Index); err != nil {
		return fmt.Errorf("导出失败: %w", err)
	}

	b.log.Step(12, "处理boot.wim")
	if err := b.processBootWim(); err != nil {
		return fmt.Errorf("处理boot.wim失败: %w", err)
	}

	b.log.Step(13, "创建ISO镜像")
	isoPath, err := b.imgMgr.CreateISO()
	if err != nil {
		return fmt.Errorf("创建ISO失败: %w", err)
	}
	b.outputISO = isoPath

	b.log.Step(14, "清理临时文件")
	if err := b.imgMgr.Cleanup(); err != nil {
		b.log.Warn("清理临时文件失败: %v", err)
	}

	return nil
}

func (b *Tiny11Builder) copyAutounattend() error {
	b.log.Info("复制自动应答文件...")

	autoUnattendSrc := filepath.Join(b.config.ResourcesDir, "autounattend.xml")
	autoUnattendDst := filepath.Join(b.config.ScratchDir, "Windows", "System32", "Sysprep", "autounattend.xml")

	if !utils.FileExists(autoUnattendSrc) {
		b.log.Warn("autounattend.xml 不存在: %s", autoUnattendSrc)
		b.log.Info("尝试从主题或默认位置获取...")

		if b.config.ThemeName != "default" && b.config.ThemeName != "" {
			themePath := filepath.Join(b.config.WorkDir, "themes", b.config.ThemeName, "autounattend.xml")
			if utils.FileExists(themePath) {
				autoUnattendSrc = themePath
				b.log.Success("使用主题中的autounattend.xml")
			}
		}

		if !utils.FileExists(autoUnattendSrc) {
			b.log.Info("创建默认的autounattend.xml...")
			if err := b.createDefaultAutounattend(autoUnattendSrc); err != nil {
				return fmt.Errorf("创建默认autounattend.xml失败: %w", err)
			}
		}
	}

	dstDir := filepath.Dir(autoUnattendDst)
	if err := utils.EnsureDir(dstDir); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	if err := utils.CopyFile(autoUnattendSrc, autoUnattendDst); err != nil {
		return fmt.Errorf("复制autounattend.xml失败: %w", err)
	}

	b.log.Success("autounattend.xml 复制成功")

	isoAutounattend := filepath.Join(b.config.Tiny11Dir, "autounattend.xml")
	if err := utils.CopyFile(autoUnattendSrc, isoAutounattend); err != nil {
		b.log.Warn("复制autounattend.xml到ISO根目录失败: %v", err)
	} else {
		b.log.Info("已复制到ISO根目录")
	}

	return nil
}

func (b *Tiny11Builder) createDefaultAutounattend(path string) error {
	defaultContent := `<?xml version="1.0" encoding="utf-8"?>
<unattend xmlns="urn:schemas-microsoft-com:unattend">
    <settings pass="oobeSystem">
        <component xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State" 
                   xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" 
                   name="Microsoft-Windows-Shell-Setup" 
                   processorArchitecture="amd64" 
                   publicKeyToken="31bf3856ad364e35" 
                   language="neutral" 
                   versionScope="nonSxS">
            <OOBE>
                <HideEULAPage>true</HideEULAPage>
                <HideOEMRegistrationScreen>true</HideOEMRegistrationScreen>
                <HideOnlineAccountScreens>true</HideOnlineAccountScreens>
                <HideWirelessSetupInOOBE>false</HideWirelessSetupInOOBE>
                <ProtectYourPC>3</ProtectYourPC>
                <SkipUserOOBE>false</SkipUserOOBE>
                <SkipMachineOOBE>false</SkipMachineOOBE>
            </OOBE>
        </component>
    </settings>
    <settings pass="windowsPE">
        <component xmlns:wcm="http://schemas.microsoft.com/WMIConfig/2002/State" 
                   xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" 
                   name="Microsoft-Windows-Setup" 
                   processorArchitecture="amd64" 
                   publicKeyToken="31bf3856ad364e35" 
                   language="neutral" 
                   versionScope="nonSxS">
            <DynamicUpdate>
                <Enable>false</Enable>
                <WillShowUI>OnError</WillShowUI>
            </DynamicUpdate>
            <ImageInstall>
                <OSImage>
                    <Compact>true</Compact>
                    <WillShowUI>OnError</WillShowUI>
                    <InstallFrom>
                        <MetaData wcm:action="add">
                            <Key>/IMAGE/INDEX</Key>
                            <Value>1</Value>
                        </MetaData>
                    </InstallFrom>
                </OSImage>
            </ImageInstall>
            <UserData>
                <ProductKey>
                    <Key></Key>
                </ProductKey>
                <AcceptEula>true</AcceptEula>
            </UserData>
        </component>
    </settings>
</unattend>`

	if err := utils.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}

	return utils.WriteFile(path, []byte(defaultContent))
}

func (b *Tiny11Builder) processBootWim() error {
	var bootUnmounted = false

	if err := b.imgMgr.MountBootWim(); err != nil {
		return err
	}

	defer func() {
		if !bootUnmounted {
			b.log.Info("执行boot.wim紧急清理...")
			b.regMgr.UnloadHives()
			b.imgMgr.UnmountImage(false)
		}
	}()

	if err := b.regMgr.LoadHives(); err != nil {
		return err
	}

	if err := b.regMgr.ApplyBootTweaks(); err != nil {
		b.log.Warn("应用Boot优化失败: %v", err)
	}

	b.regMgr.UnloadHives()

	if err := b.imgMgr.UnmountImage(true); err != nil {
		return err
	}

	bootUnmounted = true
	return nil
}

func (b *Tiny11Builder) GetOutputISO() string {
	return b.outputISO
}