package test

import "fmt"

// @ai 邮件发送状态枚举
const (
	MailStatusPending   = 1 // 待发送
	MailStatusSending   = 2 // 发送中
	MailStatusCompleted = 3 // 已完成
	MailStatusFailed    = 4 // 发送失败
)

func Test() {
	fmt.Println("test")
}
