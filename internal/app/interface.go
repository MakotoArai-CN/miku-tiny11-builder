package app

// Builder 构建器接口
type Builder interface {
	// Build 执行构建流程
	Build() error

	// GetOutputISO 获取输出ISO路径
	GetOutputISO() string
}