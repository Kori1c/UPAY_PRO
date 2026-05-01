package notification

// 这里是bark的通知服务

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"upay_pro/db/sdb"
	"upay_pro/mylog"

	"go.uber.org/zap"
)

func sendBarkNotification(barkURL, title, body string) error {
	// 创建通知内容
	notification := map[string]string{
		"title": title,
		"body":  body,
	}

	// 将通知内容编码为 JSON
	jsonData, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	// 发送 POST 请求
	resp, err := http.Post(barkURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send notification, status code: %d", resp.StatusCode)
	}

	return nil
}

func Bark_Start(order sdb.Orders) error {

	if sdb.GetSetting().Barkkey == "" {
		mylog.Logger.Info("Barkkey为空，不能发送通知")
		return nil
	}

	// 替换为你的 Bark URL
	barkURL := "https://api.day.app/" + sdb.GetSetting().Barkkey // 你的 Bark 服务器 URL
	title := "UPAY_PRO 订单通知"
	// 将数据库中的数字翻译会自然语言
	var Status string
	switch order.Status {
	case 1:
		Status = "待支付"
	case 2:
		Status = "支付成功"
	case 3:
		Status = "已过期"
	default:
		Status = "未知状态"
	}

	var CallBackConfirm string
	if order.CallBackConfirm == sdb.CallBackConfirmOk {
		CallBackConfirm = "已回调"
	} else {
		CallBackConfirm = "未回调"
	}

	body := fmt.Sprintf("订单号:%s\n币种:%s\n支付金额%.2f\n支付状态:%s\n区块ID:%s\n回调状态：%s\n", order.TradeId, order.Type, order.ActualAmount, Status, order.BlockTransactionId, CallBackConfirm)
	// body := "您的订单已成功创建！\n感谢您的购买！\n请查看您的订单详情。"

	// 发送通知
	err := sendBarkNotification(barkURL, title, body)
	if err != nil {
		mylog.Logger.Error("发送 Bark 通知失败", zap.Error(err))
		return err
	}

	mylog.Logger.Info("Bark 通知发送成功")
	return nil
}
