package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"mechat/config"
	"mechat/internal/user"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type seedUser struct {
	Email    string
	Password string
	Nickname string
	Bio      string
	VIPLevel int8
}

func main() {
	cfgPath := flag.String("config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}

	vipExpire := time.Now().AddDate(1, 0, 0)

	seeds := []seedUser{
		{Email: "alice@a.com", Password: "123456", Nickname: "Alice", Bio: "前端工程师，喜欢折腾新技术", VIPLevel: 1},
		{Email: "bob@b.com", Password: "123456", Nickname: "Bob", Bio: "后端老司机，Go 爱好者", VIPLevel: 0},
		{Email: "bob2@b.com", Password: "123456", Nickname: "bob", Bio: "全小写的 bob", VIPLevel: 0},
		{Email: "charlie@c.com", Password: "123456", Nickname: "Charlie", Bio: "产品经理，负责需求拆解", VIPLevel: 0},
	}

	for _, s := range seeds {
		hash, err := bcrypt.GenerateFromPassword([]byte(s.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("hash password for %s: %v", s.Email, err)
		}

		var u user.User
		err = db.Where("email = ?", s.Email).First(&u).Error
		switch err {
		case nil:
			// 已存在则更新
			u.Password = string(hash)
			u.Nickname = s.Nickname
			u.Bio = s.Bio
			u.VIPLevel = s.VIPLevel
			if s.VIPLevel > 0 {
				u.VIPExpiredAt = &vipExpire
			} else {
				u.VIPExpiredAt = nil
			}
			u.Status = 1
			if err := db.Save(&u).Error; err != nil {
				log.Fatalf("update %s: %v", s.Email, err)
			}
			fmt.Printf("✓ 更新 %-14s (%s / %s)  VIP=%d\n", s.Nickname, s.Email, s.Password, s.VIPLevel)
		case gorm.ErrRecordNotFound:
			nu := user.User{
				Email:    s.Email,
				Password: string(hash),
				Nickname: s.Nickname,
				Bio:      s.Bio,
				VIPLevel: s.VIPLevel,
				Status:   1,
			}
			if s.VIPLevel > 0 {
				nu.VIPExpiredAt = &vipExpire
			}
			if err := db.Create(&nu).Error; err != nil {
				log.Fatalf("create %s: %v", s.Email, err)
			}
			fmt.Printf("✓ 创建 %-14s (%s / %s)  VIP=%d\n", s.Nickname, s.Email, s.Password, s.VIPLevel)
		default:
			log.Fatalf("query %s: %v", s.Email, err)
		}
	}

	fmt.Println("\n测试账号就绪")
}
