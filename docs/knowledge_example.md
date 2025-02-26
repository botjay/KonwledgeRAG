# 枚举类型

## MailStatus 邮件发送状态枚举

**标签：** `completed` · `failed` · `mail` · `pending` · `sending` · `status` · `status 邮件发送状态枚举` · `发送中` · `发送失败` · `已完成` · `待发送` · `邮件发送状态枚举`

| 变量 | 原值 | 描述 |
|---|---|---|
| MailStatusPending | 1 | 待发送 |
| MailStatusSending | 2 | 发送中 |
| MailStatusCompleted | 3 | 已完成 |
| MailStatusFailed | 4 | 发送失败 |

# 数据库表

## order_details

| 字段 | 类型 | 描述 |
|---|---|---|
| id | bigint | 自增ID |
| trade_date | character | 交易日期（2006-01-02） |
| user_id | bigint | 用户id |
| order_id | bigint | 订单ID |
| currency | character | - |
| trade_amount | numeric | 成交金额 |
| trade_quantity | numeric | 成交数量 |
| order_status | character | 订单状态：init-初始化，pending-待处理，processing-处理中，completed-已完成，cancelled-已取消 |
| fee | numeric | - |

