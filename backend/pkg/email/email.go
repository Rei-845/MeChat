package email

import (
	"fmt"
	"time"

	"gopkg.in/gomail.v2"
)

const smtpTimeout = 15 * time.Second

type Sender struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// 创建邮件发送器
func NewSender(host string, port int, username, password, from string) *Sender {
	return &Sender{host: host, port: port, username: username, password: password, from: from}
}

// 发送验证码邮件
func (s *Sender) SendVerifyCode(to, code, purpose string) error {
	subject := "MeChat 验证码"
	body := fmt.Sprintf(`
<div style="font-family:sans-serif;max-width:480px;margin:0 auto">
  <h2>MeChat 验证码</h2>
  <p>您正在进行 <b>%s</b> 操作，验证码为：</p>
  <div style="font-size:32px;font-weight:bold;letter-spacing:8px;color:#4F46E5;padding:16px 0">%s</div>
  <p style="color:#888">验证码 5 分钟内有效，请勿泄露给他人。</p>
</div>`, purposeLabel(purpose), code)

	return s.send(to, subject, body)
}

// 发送邮件
func (s *Sender) send(to, subject, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(s.host, s.port, s.username, s.password)
	d.SSL = s.port == 465 // 465 走 SSL 587 走 STARTTLS

	// gomail 无 context 用超时 goroutine 兜底
	type result struct{ err error }
	ch := make(chan result, 1)
	go func() { ch <- result{d.DialAndSend(m)} }()
	select {
	case r := <-ch:
		return r.err
	case <-time.After(smtpTimeout):
		return fmt.Errorf("smtp: send timeout after %s", smtpTimeout)
	}
}

// 用途文案
func purposeLabel(p string) string {
	switch p {
	case "register":
		return "注册"
	case "login":
		return "登录"
	case "reset":
		return "重置密码"
	default:
		return "验证"
	}
}
