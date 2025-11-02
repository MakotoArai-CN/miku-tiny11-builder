package utils

import (
	"sync"
)

// 全局缓冲区池 - 复用内存，减少 GC 压力
var (
	// 64KB 缓冲区池 - 用于小文件
	smallBufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 64*1024)
			return &buf
		},
	}

	// 1MB 缓冲区池 - 用于大文件复制
	largeBufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 1024*1024)
			return &buf
		},
	}
)

func GetSmallBuffer() *[]byte {
	return smallBufferPool.Get().(*[]byte)
}

func PutSmallBuffer(buf *[]byte) {
	smallBufferPool.Put(buf)
}

func GetLargeBuffer() *[]byte {
	return largeBufferPool.Get().(*[]byte)
}

func PutLargeBuffer(buf *[]byte) {
	largeBufferPool.Put(buf)
}