package utils

import (
	"bytes"
	"io"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// DecodeGBK 将GBK编码转换为UTF-8
func DecodeGBK(data []byte) (string, error) {
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return string(data), err
	}
	return string(decoded), nil
}

// TryDecodeGBK 尝试解码GBK，如果失败则返回原始UTF-8字符串
func TryDecodeGBK(data []byte) string {
	// 检测是否包含乱码字符
	str := string(data)
	if !containsGarbledChars(str) {
		return str
	}

	// 尝试GBK解码
	decoded, err := DecodeGBK(data)
	if err != nil {
		return str
	}
	return decoded
}

// containsGarbledChars 检测是否包含乱码字符
func containsGarbledChars(s string) bool {
	for _, r := range s {
		// � (U+FFFD) 是替换字符，通常表示解码失败
		if r == '\uFFFD' {
			return true
		}
		// 检测常见的乱码模式（高位ASCII范围的字节组合）
		if r > 127 && r < 256 {
			return true
		}
	}
	return false
}