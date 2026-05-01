package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"upay_pro/cron"
	"upay_pro/db/sdb"

	"github.com/gin-gonic/gin"
)

const (
	callbackStateNotApplicable = "not_applicable"
	callbackStatePending       = "pending"
	callbackStateFailed        = "failed"
	callbackStateConfirmed     = "confirmed"
)

type orderListItem struct {
	ID                 uint       `json:"id"`
	CreatedAt          time.Time  `json:"CreatedAt"`
	TradeID            string     `json:"trade_id"`
	OrderID            string     `json:"order_id"`
	Amount             float64    `json:"amount"`
	ActualAmount       float64    `json:"actual_amount"`
	Type               string     `json:"type"`
	Token              string     `json:"token"`
	Status             int        `json:"status"`
	CallbackNum        int        `json:"callback_num"`
	CallBackConfirm    int        `json:"call_back_confirm"`
	CallbackState      string     `json:"callback_state"`
	CallbackStateLabel string     `json:"callback_state_label"`
	CallbackMessage    string     `json:"callback_message"`
	LastCallbackAt     *time.Time `json:"last_callback_at"`
	CanRetryCallback   bool       `json:"can_retry_callback"`
}

var triggerOrderCallbackAsync = func(order sdb.Orders) {
	go cron.ProcessCallbackWithSource(order, sdb.CallbackTriggerManual)
}

func buildOrderListItem(order sdb.Orders) orderListItem {
	callbackState, callbackLabel, callbackMessage := callbackPresentation(order)

	return orderListItem{
		ID:                 order.ID,
		CreatedAt:          order.CreatedAt,
		TradeID:            order.TradeId,
		OrderID:            order.OrderId,
		Amount:             order.Amount,
		ActualAmount:       order.ActualAmount,
		Type:               order.Type,
		Token:              order.Token,
		Status:             order.Status,
		CallbackNum:        order.CallbackNum,
		CallBackConfirm:    order.CallBackConfirm,
		CallbackState:      callbackState,
		CallbackStateLabel: callbackLabel,
		CallbackMessage:    callbackMessage,
		LastCallbackAt:     order.LastCallbackAt,
		CanRetryCallback:   canRetryOrderCallback(order),
	}
}

func callbackPresentation(order sdb.Orders) (state string, label string, message string) {
	switch {
	case order.Status != sdb.StatusPaySuccess:
		return callbackStateNotApplicable, "未触发", ""
	case strings.TrimSpace(order.NotifyUrl) == "":
		return callbackStateNotApplicable, "无需回调", ""
	case order.CallBackConfirm == sdb.CallBackConfirmOk:
		return callbackStateConfirmed, "已确认", ""
	case order.CallbackNum > 0:
		return callbackStateFailed, "回调失败", strings.TrimSpace(order.CallbackMessage)
	default:
		return callbackStatePending, "待回调", ""
	}
}

func canRetryOrderCallback(order sdb.Orders) bool {
	return order.Status == sdb.StatusPaySuccess &&
		strings.TrimSpace(order.NotifyUrl) != "" &&
		order.CallBackConfirm != sdb.CallBackConfirmOk
}

func retryOrderCallbackErrorMessage(order sdb.Orders) string {
	switch {
	case order.Status != sdb.StatusPaySuccess:
		return "订单未支付，暂不支持补发回调"
	case strings.TrimSpace(order.NotifyUrl) == "":
		return "当前订单未配置回调地址"
	case order.CallBackConfirm == sdb.CallBackConfirmOk:
		return "当前订单回调已确认，无需补发"
	default:
		return "当前订单暂不支持补发回调"
	}
}

func handleRetryOrderCallback(c *gin.Context) {
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "订单 ID 无效",
		})
		return
	}

	var order sdb.Orders
	if err := sdb.DB.First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    1,
			"message": "订单不存在",
		})
		return
	}

	if !canRetryOrderCallback(order) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": retryOrderCallbackErrorMessage(order),
		})
		return
	}

	if err := sdb.RecordCallbackEvent(order, sdb.CallbackTriggerManual, sdb.CallbackResultQueued, "管理员手动触发补发", 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "记录回调补发任务失败",
		})
		return
	}

	triggerOrderCallbackAsync(order)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "回调补发任务已触发",
	})
}
