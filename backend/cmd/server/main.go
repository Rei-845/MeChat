package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mechat/config"
	aimod "mechat/internal/ai"
	"mechat/internal/ai/agent"
	aichain "mechat/internal/ai/chain"
	"mechat/internal/chat"
	"mechat/internal/feed"
	"mechat/internal/friend"
	"mechat/internal/level"
	"mechat/internal/user"
	"mechat/internal/vip"
	"mechat/internal/ws"
	"mechat/pkg/email"
	jwtpkg "mechat/pkg/jwt"
	"mechat/pkg/middleware"
	"mechat/pkg/mq"
	"mechat/pkg/oss"
	redispkg "mechat/pkg/redis"
	"mechat/pkg/snowflake"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfgPath := flag.String("config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Gin 运行模式
	gin.SetMode(cfg.Server.Mode)

	// 初始化日志
	zapLogger, _ := zap.NewProduction()
	if cfg.Server.Mode == "debug" {
		zapLogger, _ = zap.NewDevelopment()
	}
	defer zapLogger.Sync()

	// 雪花 ID 节点 1
	if err := snowflake.Init(1); err != nil {
		zapLogger.Fatal("init snowflake", zap.Error(err))
	}

	// 连接 MySQL
	db, err := setupMySQL(cfg)
	if err != nil {
		zapLogger.Fatal("init mysql", zap.Error(err))
	}

	// 自动建表
	if err := autoMigrate(db); err != nil {
		zapLogger.Fatal("auto migrate", zap.Error(err))
	}

	// 连接 Redis
	rdb := redispkg.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, cfg.Redis.PoolSize)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		zapLogger.Fatal("init redis", zap.Error(err))
	}

	// 连接 RabbitMQ 失败则降级 NoopMQ
	var rabbitMQ interface {
		mq.Publisher
		mq.Consumer
		Close() error
	}
	mqEnabled := true
	if rmq, err := mq.NewRabbitMQ(cfg.RabbitMQ.URL, zapLogger); err != nil {
		zapLogger.Warn("init rabbitmq failed, using no-op MQ (async AI runs synchronously)", zap.Error(err))
		rabbitMQ = &mq.NoopMQ{}
		mqEnabled = false
	} else {
		rabbitMQ = rmq
		defer rmq.Close()
	}

	// JWT 管理器
	jwtMgr := jwtpkg.NewManager(cfg.JWT.Secret, cfg.JWT.ExpireTime, rdb)

	// 邮件发送器
	mailer := email.NewSender(cfg.Email.Host, cfg.Email.Port, cfg.Email.Username, cfg.Email.Password, cfg.Email.From)

	// OSS 未配置则降级本地磁盘
	var uploader oss.Uploader
	if cfg.OSS.AccessKeyID != "" {
		ossCli, oerr := oss.NewClient(cfg.OSS.Endpoint, cfg.OSS.AccessKeyID, cfg.OSS.AccessKeySecret, cfg.OSS.Bucket, cfg.OSS.Domain)
		if oerr != nil {
			zapLogger.Warn("init oss failed, falling back to local storage", zap.Error(oerr))
		} else {
			uploader = ossCli
		}
	}
	if uploader == nil {
		local, lerr := oss.NewLocal("uploads", "/uploads")
		if lerr != nil {
			zapLogger.Fatal("init local storage", zap.Error(lerr))
		}
		uploader = local
		zapLogger.Info("using local file storage at ./uploads (served at /uploads)")
	}

	// AI Model 配了 api_key 才启用
	ctx := context.Background()
	var aiInvoker *aichain.Invoker
	if cfg.AI.APIKey != "" {
		aiTemp := float32(0.5) // 低温度提升工具调用稳定性
		llmModel, mErr := openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
			Model:       cfg.AI.Model,
			APIKey:      cfg.AI.APIKey,
			BaseURL:     cfg.AI.BaseURL,
			Temperature: &aiTemp,
		})
		if mErr != nil {
			zapLogger.Warn("init ai model failed, AI features disabled", zap.Error(mErr))
		} else {
			aiInvoker = aichain.New(llmModel)
		}
	} else {
		zapLogger.Warn("ai.api_key 未配置，AI 功能已禁用（在 config.yaml 填写后重启即可启用）")
	}

	// WebSocket Hub
	hub := ws.NewHub(rdb, zapLogger)
	go hub.Run() // Hub 事件循环

	// 自底向上依赖注入
	// Repositories
	userRepo := user.NewRepository(db)
	friendRepo := friend.NewRepository(db)
	chatRepo := chat.NewRepository(db)
	feedRepo := feed.NewRepository(db)
	vipRepo := vip.NewRepository(db)
	aiRepo := aimod.NewRepository(db)

	// Services
	userSvc := user.NewService(userRepo, jwtMgr, rdb, mailer, uploader)
	friendSvc := friend.NewService(friendRepo, userRepo, hub, rdb)
	chatSvc := chat.NewService(chatRepo, userRepo, hub, rdb, uploader, zapLogger)
	friendSvc.SetChatSvc(chatSvc) // 删好友级联清理会话
	feedSvc := feed.NewService(feedRepo, userRepo, friendSvc, rdb)
	vipSvc := vip.NewService(vipRepo, userRepo)

	levelSvc := level.NewService(userRepo, rdb)
	feedSvc.SetLevelSvc(levelSvc) // 点赞评论触发经验

	aiSvc := aimod.NewService(userRepo, chatRepo, aiRepo, aiInvoker, zapLogger)
	// 注入 Agent 工具注册表
	aiSvc.SetAgent(agent.BuildRegistry(&agent.Deps{
		Chat:    chatSvc,
		Feed:    feedSvc,
		Friend:  friendSvc,
		User:    userSvc,
		Invoker: aiInvoker,
	}))
	// 注入异步 AI 依赖 mqEnabled=false 降级同步
	aiSvc.EnableAsync(rabbitMQ, hub, mqEnabled)

	// AI 任务消费协程
	go func() {
		if err := aimod.NewConsumer(rabbitMQ, aiSvc, zapLogger).Start(); err != nil {
			zapLogger.Error("ai consumer start", zap.Error(err))
		}
	}()

	// Handlers
	authMW := middleware.Auth(jwtMgr)
	wsHandler := ws.NewHandler(hub, jwtMgr, rdb, chatSvc, zapLogger, cfg.Server.AllowedOrigins)

	userHandler := user.NewHandler(userSvc)
	userHandler.SetFriendSvc(friendSvc)
	h := &Handlers{
		user:   userHandler,
		friend: friend.NewHandler(friendSvc),
		chat:   chat.NewHandler(chatSvc),
		feed:   feed.NewHandler(feedSvc),
		ai:     aimod.NewHandler(aiSvc),
		vip:    vip.NewHandler(vipSvc),
		level:  level.NewHandler(levelSvc),
		ws:     wsHandler,
	}

	r := setupRouter(h, authMW, cfg.Server.AllowedOrigins)

	zapLogger.Info("server starting", zap.String("addr", cfg.Server.Addr))

	// HTTP 服务器
	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,  // 防 Slowloris
		IdleTimeout:       120 * time.Second, // keep-alive 空闲时长
	}

	// 优雅停机 监听信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zapLogger.Fatal("server run", zap.Error(err))
		}
	}()

	<-quit
	zapLogger.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Error("server shutdown error", zap.Error(err))
	}
	zapLogger.Info("server stopped")
}

// 连接 MySQL 并配连接池
func setupMySQL(cfg *config.Config) (*gorm.DB, error) {
	logLevel := logger.Silent
	if cfg.Server.Mode == "debug" {
		logLevel = logger.Info
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	// 连接池默认值
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	maxOpen := cfg.MySQL.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 100
	}
	maxIdle := cfg.MySQL.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 20
	}
	lifetime := cfg.MySQL.ConnMaxLifetime
	if lifetime <= 0 {
		lifetime = time.Hour
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(lifetime)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // 防 wait_timeout 失效
	return db, nil
}

// 自动建表
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&user.User{},
		&user.EmailVerifyCode{},
		&friend.Friendship{},
		&friend.FriendRequest{},
		&chat.Conversation{},
		&chat.ConversationMember{},
		&chat.Group{},
		&chat.Message{},
		&feed.Post{},
		&feed.PostLike{},
		&feed.PostComment{},
		&feed.CommentLike{},
		&vip.VIPOrder{},
		&aimod.AIMessage{},
	)
}
