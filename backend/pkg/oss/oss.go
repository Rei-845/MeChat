package oss

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/google/uuid"
)

type Client struct {
	bucket   *oss.Bucket
	domain   string // CDN
	endpoint string // 形如 oss-cn-shenzhen.aliyuncs.com
}

// 创建 OSS 客户端
func NewClient(endpoint, accessKeyID, accessKeySecret, bucketName, domain string) (*Client, error) {
	cli, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}
	bucket, err := cli.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	// 去掉协议前缀保留 host
	host := strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	return &Client{bucket: bucket, domain: domain, endpoint: host}, nil
}

// PutObject 上传文件并返回访问 URL
func (c *Client) PutObject(dir string, ext string, reader io.Reader, size int64) (string, error) {
	key := fmt.Sprintf("%s/%s/%s%s",
		dir,
		time.Now().Format("2006/01"),
		uuid.New().String(),
		ext,
	)

	contentType := extToContentType(ext)
	err := c.bucket.PutObject(key, reader,
		oss.ContentType(contentType),
		oss.ContentLength(size),
	)
	if err != nil {
		return "", err
	}

	return c.url(key), nil
}

// 拼访问 URL
func (c *Client) url(key string) string {
	if c.domain != "" {
		return fmt.Sprintf("https://%s/%s", c.domain, key)
	}
	return fmt.Sprintf("https://%s.%s/%s", c.bucket.BucketName, c.endpoint, key)
}

// 扩展名映射 content-type
func extToContentType(ext string) string {
	switch filepath.Ext(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
