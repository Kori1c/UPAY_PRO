package web

import (
	"net/http"
	"time"
	"upay_pro/db/sdb"
	"upay_pro/mylog"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func registerOperationsRoutes(admin *gin.RouterGroup) {
	admin.GET("/api/operations/summary", operationsSummaryHandler)
}

func operationsSummaryHandler(c *gin.Context) {
	if err := sdb.SyncExpiredOrders(); err != nil {
		mylog.Logger.Error("同步运营状态过期订单失败", zap.Error(err))
	}

	var pendingOrderCount int64
	var successOrderCount int64
	var expiredOrderCount int64
	var callbackPendingCount int64
	var callbackFailedCount int64
	var walletCount int64
	var enabledWalletCount int64
	var disabledWalletCount int64

	counts := []struct {
		label string
		query func() error
	}{
		{
			label: "pending orders",
			query: func() error {
				return sdb.DB.Model(&sdb.Orders{}).Where("status = ?", sdb.StatusWaitPay).Count(&pendingOrderCount).Error
			},
		},
		{
			label: "success orders",
			query: func() error {
				return sdb.DB.Model(&sdb.Orders{}).Where("status = ?", sdb.StatusPaySuccess).Count(&successOrderCount).Error
			},
		},
		{
			label: "expired orders",
			query: func() error {
				return sdb.DB.Model(&sdb.Orders{}).Where("status = ?", sdb.StatusExpired).Count(&expiredOrderCount).Error
			},
		},
		{
			label: "callback pending orders",
			query: func() error {
				return sdb.DB.Model(&sdb.Orders{}).
					Where("status = ? AND notify_url <> ? AND call_back_confirm <> ?", sdb.StatusPaySuccess, "", sdb.CallBackConfirmOk).
					Count(&callbackPendingCount).Error
			},
		},
		{
			label: "callback failed orders",
			query: func() error {
				return sdb.DB.Model(&sdb.Orders{}).
					Where("status = ? AND notify_url <> ? AND call_back_confirm <> ? AND callback_num > ?", sdb.StatusPaySuccess, "", sdb.CallBackConfirmOk, 0).
					Count(&callbackFailedCount).Error
			},
		},
		{
			label: "wallets",
			query: func() error {
				return sdb.DB.Model(&sdb.WalletAddress{}).Count(&walletCount).Error
			},
		},
		{
			label: "enabled wallets",
			query: func() error {
				return sdb.DB.Model(&sdb.WalletAddress{}).Where("status = ?", sdb.TokenStatusEnable).Count(&enabledWalletCount).Error
			},
		},
		{
			label: "disabled wallets",
			query: func() error {
				return sdb.DB.Model(&sdb.WalletAddress{}).Where("status = ?", sdb.TokenStatusDisable).Count(&disabledWalletCount).Error
			},
		},
	}

	for _, count := range counts {
		if err := count.query(); err != nil {
			mylog.Logger.Error("获取运营状态失败", zap.String("metric", count.label), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    -1,
				"message": "获取运营状态失败",
			})
			return
		}
	}

	warnings := make([]string, 0)
	if enabledWalletCount == 0 {
		warnings = append(warnings, "没有启用中的收款钱包")
	}
	if callbackPendingCount > 0 {
		warnings = append(warnings, "存在已支付但回调未确认的订单")
	}
	if callbackFailedCount > 0 {
		warnings = append(warnings, "存在已发生回调失败的订单")
	}

	status := "ok"
	if len(warnings) > 0 {
		status = "warning"
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"status":      status,
			"generatedAt": time.Now().Format(time.RFC3339),
			"orders": gin.H{
				"pending":         pendingOrderCount,
				"success":         successOrderCount,
				"expired":         expiredOrderCount,
				"callbackPending": callbackPendingCount,
				"callbackFailed":  callbackFailedCount,
			},
			"wallets": gin.H{
				"total":    walletCount,
				"enabled":  enabledWalletCount,
				"disabled": disabledWalletCount,
			},
			"warnings": warnings,
		},
	})
}
