package config

import (
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	ISODrive      string
	ScratchDrive  string
	ImageIndex    int
	OutputISO     string
	Verbose       bool
	CoreMode      bool
	ThemeName     string
	PreinstallApps []string

	// 路径配置 - 全部基于程序目录
	WorkDir      string
	Tiny11Dir    string
	ScratchDir   string
	ResourcesDir string
	ThemesDir    string
	PreinstallDir string
	TempDir      string
	LogDir       string
}

func NewConfig() *Config {
	// 获取程序所在目录作为工作目录
	exePath, _ := os.Executable()
	workDir := filepath.Dir(exePath)
	
	// 如果是 go run，使用当前目录
	if filepath.Base(filepath.Dir(exePath)) == "go-build" {
		workDir, _ = os.Getwd()
	}

	cfg := &Config{
		WorkDir:   workDir,
		ThemeName: "default",
	}

	// 所有路径基于工作目录
	cfg.Tiny11Dir = filepath.Join(workDir, "build", "tiny11")
	cfg.ScratchDir = filepath.Join(workDir, "build", "scratch")
	cfg.TempDir = filepath.Join(workDir, "build", "temp")
	cfg.ResourcesDir = filepath.Join(workDir, "resources")
	cfg.ThemesDir = filepath.Join(workDir, "themes")
	cfg.PreinstallDir = filepath.Join(workDir, "preinstall")
	cfg.LogDir = filepath.Join(workDir, "logs")
	cfg.OutputISO = filepath.Join(workDir, "tiny11.iso")

	// 自动检测系统盘作为默认临时盘
	cfg.ScratchDrive = detectSystemDrive()

	return cfg
}

func detectSystemDrive() string {
	if drive := os.Getenv("SystemDrive"); drive != "" {
		return drive
	}
	return "C:"
}

func (c *Config) GetArchitecture() string {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return arch
	}
}

func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.Tiny11Dir,
		c.ScratchDir,
		c.TempDir,
		c.LogDir,
		filepath.Join(c.WorkDir, "build"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) Cleanup() error {
	buildDir := filepath.Join(c.WorkDir, "build")
	if err := os.RemoveAll(buildDir); err != nil {
		return err
	}
	return nil
}