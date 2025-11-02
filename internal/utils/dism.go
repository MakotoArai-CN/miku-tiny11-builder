package utils

import (
	"bytes"
	"fmt"
	"os/exec"
	"syscall"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// RunDISMCommand 运行DISM命令并正确处理中文输出
func RunDISMCommand(args ...string) (string, error) {
	cmd := exec.Command("dism", args...)
	
	// 设置隐藏窗口
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// 尝试解码输出（可能是GBK编码）
	output := decodeOutput(stdout.Bytes())
	
	if err != nil {
		errMsg := decodeOutput(stderr.Bytes())
		if errMsg != "" {
			return output, fmt.Errorf("%w: %s", err, errMsg)
		}
		return output, err
	}

	return output, nil
}

// decodeOutput 解码可能的GBK输出为UTF-8
func decodeOutput(data []byte) string {
	// 先尝试直接作为UTF-8
	str := string(data)
	
	// 如果包含乱码标志，尝试GBK解码
	if containsGarbledText(str) {
		// 尝试GBK解码
		decoder := simplifiedchinese.GBK.NewDecoder()
		decoded, _, err := transform.Bytes(decoder, data)
		if err == nil {
			return string(decoded)
		}
	}
	
	return str
}

// containsGarbledText 检测是否包含乱码
func containsGarbledText(s string) bool {
	// 检测常见的乱码模式
	for _, r := range s {
		if r == '�' {
			return true
		}
	}
	return false
}