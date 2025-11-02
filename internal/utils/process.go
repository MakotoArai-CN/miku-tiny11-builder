package utils

import (
	"bytes"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := TryDecodeGBK(stdout.Bytes())
	if err != nil {
		errMsg := TryDecodeGBK(stderr.Bytes())
		if errMsg != "" {
			return output, fmt.Errorf("%w: %s", err, errMsg)
		}
		return output, err
	}
	return output, nil
}
func RunCommandWithOutput(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
func Takeown(path string) error {
	_, err := RunCommand("takeown", "/F", path)
	if err != nil {
		return fmt.Errorf("takeown失败: %w", err)
	}
	return nil
}
func TakeownRecursive(path string) error {
	_, err := RunCommand("takeown", "/F", path, "/R")
	if err != nil {
		return fmt.Errorf("takeown递归失败: %w", err)
	}
	return nil
}
func GrantPermission(path string) error {
	_, err := RunCommand("icacls", path, "/grant", "Administrators:(F)")
	if err != nil {
		return fmt.Errorf("icacls失败: %w", err)
	}
	return nil
}
func GrantPermissionRecursive(path string) error {
	_, err := RunCommand("icacls", path, "/grant", "Administrators:(F)", "/T", "/C")
	if err != nil {
		return fmt.Errorf("icacls递归失败: %w", err)
	}
	return nil
}
func ExtractField(output, field string) string {
	lines := strings.Split(output, "\n")
	prefix := field + " :"
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			value := strings.TrimPrefix(trimmed, prefix)
			return strings.TrimSpace(value)
		}
	}
	return ""
}
func ExtractLanguage(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Default system UI language") {
			parts := strings.Split(trimmed, ":")
			if len(parts) >= 2 {
				lang := strings.TrimSpace(parts[1])
				if lang != "" {
					return lang
				}
			}
		}
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Default language") {
			parts := strings.Split(trimmed, ":")
			if len(parts) >= 2 {
				lang := strings.TrimSpace(parts[1])
				if lang != "" {
					return lang
				}
			}
		}
	}
	return "en-US"
}
func KillProcess(name string) error {
	cmd := exec.Command("taskkill", "/F", "/IM", name)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}
func IsProcessRunning(name string) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", name))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), name)
}
func GetSystemDrive() string {
	systemDrive := os.Getenv("SystemDrive")
	if systemDrive != "" && len(systemDrive) >= 2 && systemDrive[1] == ':' {
		return systemDrive
	}
	systemRoot := os.Getenv("SystemRoot")
	if len(systemRoot) >= 2 && systemRoot[1] == ':' {
		return systemRoot[:2]
	}
	sysDir, err := windows.GetSystemDirectory()
	if err == nil && len(sysDir) >= 2 && sysDir[1] == ':' {
		return sysDir[:2]
	}
	winDir := os.Getenv("WINDIR")
	if len(winDir) >= 2 && winDir[1] == ':' {
		return winDir[:2]
	}
	return "C:"
}
func ValidateDriveLetter(drive string) bool {
	if len(drive) != 2 {
		return false
	}
	if drive[1] != ':' {
		return false
	}
	letter := drive[0]
	return (letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z')
}

func Sleep(seconds int) {
	time.Sleep(time.Duration(seconds) * time.Second)
}
