package oss

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Uploader 对象存储抽象 (OSS 与本地磁盘实现)
type Uploader interface {
	// PutObject 保存文件并返回访问 URL
	PutObject(dir, ext string, reader io.Reader, size int64) (string, error)
}

// 接口断言
var (
	_ Uploader = (*Client)(nil)
	_ Uploader = (*Local)(nil)
)

// Local OSS 未配置时落本地磁盘
type Local struct {
	baseDir   string // 落盘根目录
	urlPrefix string // 访问前缀
}

// 创建本地存储
func NewLocal(baseDir, urlPrefix string) (*Local, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	return &Local{baseDir: baseDir, urlPrefix: strings.TrimRight(urlPrefix, "/")}, nil
}

// 落盘并返回 URL
func (l *Local) PutObject(dir, ext string, reader io.Reader, size int64) (string, error) {
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	rel := filepath.Join(dir, time.Now().Format("2006/01"), uuid.New().String()+ext)
	full := filepath.Join(l.baseDir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(full)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, reader); err != nil {
		return "", err
	}
	return l.urlPrefix + "/" + filepath.ToSlash(rel), nil
}
