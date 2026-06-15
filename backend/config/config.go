package config

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	MySQL    MySQLConfig    `yaml:"mysql"`
	Redis    RedisConfig    `yaml:"redis"`
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`
	JWT      JWTConfig      `yaml:"jwt"`
	OSS      OSSConfig      `yaml:"oss"`
	Email    EmailConfig    `yaml:"email"`
	AI       AIConfig       `yaml:"ai"`
}

type ServerConfig struct {
	Addr           string `yaml:"addr"`
	Mode           string `yaml:"mode"`
	AllowedOrigins string `yaml:"allowed_origins"` // CORS/WS 允许来源
}

type MySQLConfig struct {
	DSN             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`    // 默认 100
	MaxIdleConns    int           `yaml:"max_idle_conns"`    // 默认 20
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"` // 默认 1h
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"` // 默认 50
}

type RabbitMQConfig struct {
	URL string `yaml:"url"`
}

type JWTConfig struct {
	Secret     string        `yaml:"secret"`
	ExpireTime time.Duration `yaml:"expire_time"`
}

type OSSConfig struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	Bucket          string `yaml:"bucket"`
	Domain          string `yaml:"domain"`
}

type EmailConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

type AIConfig struct {
	Provider string        `yaml:"provider"`
	Model    string        `yaml:"model"`
	APIKey   string        `yaml:"api_key"`
	BaseURL  string        `yaml:"base_url"`
	Timeout  time.Duration `yaml:"timeout"`
}

// 加载配置
func Load(path string) (*Config, error) {
	cfg := &Config{}
	// 配置文件缺失则纯靠环境变量与缺省值 docker 用此方式启动
	if f, err := os.Open(path); err == nil {
		defer f.Close()
		if derr := yaml.NewDecoder(f).Decode(cfg); derr != nil {
			return nil, derr
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	applyEnv(cfg)
	return cfg, nil
}

// 环境变量覆盖配置
func applyEnv(cfg *Config) {
	if v := os.Getenv("MYSQL_DSN"); v != "" {
		cfg.MySQL.DSN = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("RABBITMQ_URL"); v != "" {
		cfg.RabbitMQ.URL = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("OSS_ACCESS_KEY"); v != "" {
		cfg.OSS.AccessKeyID = v
	}
	if v := os.Getenv("OSS_SECRET_KEY"); v != "" {
		cfg.OSS.AccessKeySecret = v
	}
	if v := os.Getenv("OSS_BUCKET"); v != "" {
		cfg.OSS.Bucket = v
	}
	if v := os.Getenv("OSS_ENDPOINT"); v != "" {
		cfg.OSS.Endpoint = v
	}
	if v := os.Getenv("AI_API_KEY"); v != "" {
		cfg.AI.APIKey = v
	}
	if v := os.Getenv("AI_BASE_URL"); v != "" {
		cfg.AI.BaseURL = v
	}
	if v := os.Getenv("SMTP_HOST"); v != "" {
		cfg.Email.Host = v
	}
	if v := os.Getenv("SMTP_USER"); v != "" {
		cfg.Email.Username = v
	}
	if v := os.Getenv("SMTP_PASS"); v != "" {
		cfg.Email.Password = v
	}
	if v := os.Getenv("SMTP_FROM"); v != "" {
		cfg.Email.From = v
	}
	if v := os.Getenv("SMTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Email.Port = port
		}
	}
	if v := os.Getenv("ALLOWED_ORIGIN"); v != "" {
		cfg.Server.AllowedOrigins = v
	}
	if v := os.Getenv("SERVER_MODE"); v != "" {
		cfg.Server.Mode = v
	}
	if v := os.Getenv("SERVER_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("AI_MODEL"); v != "" {
		cfg.AI.Model = v
	}

	// 缺省值
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "release"
	}
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}
	if cfg.Server.AllowedOrigins == "" {
		cfg.Server.AllowedOrigins = "*"
	}
	// From 留空回退登录账号
	if cfg.Email.From == "" {
		cfg.Email.From = cfg.Email.Username
	}
	if cfg.JWT.ExpireTime == 0 {
		cfg.JWT.ExpireTime = 720 * time.Hour
	}
	if cfg.AI.Model == "" {
		cfg.AI.Model = "deepseek-chat"
	}
	if cfg.AI.Provider == "" {
		cfg.AI.Provider = "deepseek"
	}
	if cfg.AI.Timeout == 0 {
		cfg.AI.Timeout = 30 * time.Second
	}
}
