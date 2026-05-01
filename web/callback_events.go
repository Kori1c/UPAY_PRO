package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"upay_pro/db/sdb"

	"github.com/gin-gonic/gin"
)

type callbackEventHistoryItem struct {
	ID               uint      `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	TriggerType      string    `json:"trigger_type"`
	TriggerTypeLabel string    `json:"trigger_type_label"`
	Result           string    `json:"result"`
	ResultLabel      string    `json:"result_label"`
	Message          string    `json:"message"`
	AttemptNumber    int       `json:"attempt_number"`
}

func buildCallbackEventHistoryItem(event sdb.CallbackEvent) callbackEventHistoryItem {
	return callbackEventHistoryItem{
		ID:               event.ID,
		CreatedAt:        event.CreatedAt,
		TriggerType:      event.TriggerType,
		TriggerTypeLabel: callbackTriggerTypeLabel(event.TriggerType),
		Result:           event.Result,
		ResultLabel:      callbackResultLabel(event.Result),
		Message:          strings.TrimSpace(event.Message),
		AttemptNumber:    event.AttemptNumber,
	}
}

func callbackTriggerTypeLabel(triggerType string) string {
	switch triggerType {
	case sdb.CallbackTriggerManual:
		return "手动补发"
	default:
		return "自动回调"
	}
}

func callbackResultLabel(result string) string {
	switch result {
	case sdb.CallbackResultSuccess:
		return "回调成功"
	case sdb.CallbackResultFailed:
		return "回调失败"
	default:
		return "已触发"
	}
}

func handleListOrderCallbackEvents(c *gin.Context) {
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

	var events []sdb.CallbackEvent
	if err := sdb.DB.Where("order_row_id = ?", order.ID).Order("id DESC").Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "获取回调历史失败",
		})
		return
	}

	items := make([]callbackEventHistoryItem, 0, len(events))
	for _, event := range events {
		items = append(items, buildCallbackEventHistoryItem(event))
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"total":  len(items),
			"events": items,
		},
	})
}
