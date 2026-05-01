package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	Autoprice "upay_pro/AutoPrice"
	"upay_pro/db/sdb"
	"upay_pro/mylog"

	"upay_pro/cron"

	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"upay_pro/db/rdb"
	"upay_pro/mq"
)

type User struct {
	UserName string `json:"username" form:"username" validate:"required,min=5,max=12,alphanum"`
	PassWord string `json:"password" form:"password" validate:"required,min=5,max=18,alphanum"`
}

func Start() {
	// 创建一个新的验证器实例
	validate := validator.New()
	r := gin.Default()
	/* 	// 配置 CORS 中间件
	   	r.Use(cors.New(cors.Config{
	   		AllowOrigins:     []string{"*"},                            // 允许的源
	   		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"}, // 允许的方法
	   		AllowHeaders:     []string{"Origin", "Content-Type"},       // 允许的头
	   		ExposeHeaders:    []string{"Content-Length"},               // 可见的头
	   		AllowCredentials: true,                                     // 允许携带凭据
	   		MaxAge:           10 * time.Minute,                         // 缓存时间
	   	})) */
	// 加载模版
	r.LoadHTMLGlob("static/*.html")
	// 加载静态资源并把原始目录重定向

	r.Static("/css", "./static/css")
	r.Static("/js", "./static/js")
	r.Static("/assets", "./static/admin_spa/assets")
	registerHealthRoutes(r)

	// 首页路由：默认进入现代化管理后台
	r.GET("/", func(c *gin.Context) {
		c.File("./static/admin_spa/index.html")
	})
	registerPublicAuthRoutes(r, validate)
	// 后台路由组
	{
		admin := r.Group("/admin")
		admin.Use(JWTAuthMiddleware(), AdminOriginMiddleware())

		admin.GET("/", func(c *gin.Context) {
			c.File("./static/admin_spa/index.html")
		})

		// 当前管理员账号信息
		admin.GET("/api/account", func(c *gin.Context) {
			var user sdb.User
			result := sdb.DB.Where("deleted_at IS NULL").Order("id ASC").First(&user)
			if result.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code": -1,
					"msg":  "获取账号信息失败",
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "success",
				"data": gin.H{
					"id":       user.ID,
					"username": user.UserName,
				},
			})
		})

		registerPasskeyAdminRoutes(admin)
		registerOperationsRoutes(admin)

		// 订单管理API
		admin.GET("/api/orders", func(c *gin.Context) {
			if err := sdb.SyncExpiredOrders(); err != nil {
				mylog.Logger.Error("同步过期订单状态失败", zap.Error(err))
			}

			// 获取分页参数
			page := 1
			limit := 10
			if p := c.Query("page"); p != "" {
				if pageNum, err := strconv.Atoi(p); err == nil && pageNum > 0 {
					page = pageNum
				}
			}
			if l := c.Query("limit"); l != "" {
				if limitNum, err := strconv.Atoi(l); err == nil && limitNum > 0 && limitNum <= 100 {
					limit = limitNum
				}
			}

			// 获取搜索参数
			search := c.Query("search")
			// 获取状态过滤参数
			statusFilter := c.Query("status")

			// 计算偏移量
			offset := (page - 1) * limit

			// 构建查询条件
			query := sdb.DB.Model(&sdb.Orders{})
			if search != "" {
				// 搜索订单号(TradeId)或商城订单号(OrderId)
				query = query.Where("trade_id LIKE ? OR order_id LIKE ?", "%"+search+"%", "%"+search+"%")
			}
			// 状态过滤
			if statusFilter != "" {
				if statusInt, err := strconv.Atoi(statusFilter); err == nil {
					query = query.Where("status = ?", statusInt)
				}
			}

			// 获取总数
			var total int64
			query.Count(&total)

			// 获取订单列表（按ID倒序）
			var orders []sdb.Orders
			result := query.Order("id DESC").Offset(offset).Limit(limit).Find(&orders)
			if result.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code": -1,
					"msg":  "获取订单列表失败",
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "success",
				"data": gin.H{
					"orders": orders,
					"total":  total,
					"page":   page,
					"limit":  limit,
				},
			})
		})

		// 钱包地址管理API
		admin.GET("/api/wallets", func(c *gin.Context) {
			var wallets []sdb.WalletAddress
			result := sdb.DB.Find(&wallets)
			if result.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code": -1,
					"msg":  "获取钱包地址列表失败",
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "success",
				"data": wallets,
			})
		})

		// 统计数据API
		admin.GET("/api/stats", func(c *gin.Context) {
			if err := sdb.SyncExpiredOrders(); err != nil {
				mylog.Logger.Error("同步统计过期订单状态失败", zap.Error(err))
			}

			var userCount int64
			var successOrderCount int64
			var pendingOrderCount int64
			var walletCount int64
			var todayAmount float64
			var yesterdayAmount float64
			var totalAmount float64
			var todayOrderCount int64
			var currentMonthSuccessOrderCount int64

			sdb.DB.Model(&sdb.User{}).Count(&userCount)
			sdb.DB.Model(&sdb.Orders{}).Where("status = ?", sdb.StatusPaySuccess).Count(&successOrderCount)
			sdb.DB.Model(&sdb.Orders{}).Where("status = ?", sdb.StatusWaitPay).Count(&pendingOrderCount)
			sdb.DB.Model(&sdb.WalletAddress{}).Count(&walletCount)

			// 时间点计算
			now := time.Now()
			todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			yesterdayStart := todayStart.AddDate(0, 0, -1)
			currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

			// 今日成功金额
			sdb.DB.Model(&sdb.Orders{}).Where("status = ? AND created_at >= ?", sdb.StatusPaySuccess, todayStart).Select("COALESCE(SUM(amount), 0)").Scan(&todayAmount)

			// 昨日成功金额
			sdb.DB.Model(&sdb.Orders{}).Where("status = ? AND created_at >= ? AND created_at < ?", sdb.StatusPaySuccess, yesterdayStart, todayStart).Select("COALESCE(SUM(amount), 0)").Scan(&yesterdayAmount)

			// 累计成功金额
			sdb.DB.Model(&sdb.Orders{}).Where("status = ?", sdb.StatusPaySuccess).Select("COALESCE(SUM(amount), 0)").Scan(&totalAmount)

			// 今日订单数 (所有状态)
			sdb.DB.Model(&sdb.Orders{}).Where("created_at >= ?", todayStart).Count(&todayOrderCount)

			// 当月成交订单数（自然月，已支付）
			sdb.DB.Model(&sdb.Orders{}).Where("status = ? AND created_at >= ?", sdb.StatusPaySuccess, currentMonthStart).Count(&currentMonthSuccessOrderCount)

			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "success",
				"data": gin.H{
					"userCount":                     userCount,
					"successOrderCount":             successOrderCount,
					"pendingOrderCount":             pendingOrderCount,
					"walletCount":                   walletCount,
					"todayAmount":                   todayAmount,
					"yesterdayAmount":               yesterdayAmount,
					"totalAmount":                   totalAmount,
					"todayOrderCount":               todayOrderCount,
					"currentMonthSuccessOrderCount": currentMonthSuccessOrderCount,
				},
			})
		})

		// 修改当前管理员账号和密码
		admin.POST("/api/account", func(c *gin.Context) {
			var req struct {
				UserName string `json:"username" validate:"required,min=5,max=12,alphanum"`
				Password string `json:"password" validate:"omitempty,min=5,max=18,alphanum"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"code": 1, "message": "参数错误"})
				return
			}

			err := validate.Struct(req)
			if err != nil {
				c.JSON(400, gin.H{"code": 1, "message": err.Error()})
				return
			}

			var currentUser sdb.User
			result := sdb.DB.Where("deleted_at IS NULL").Order("id ASC").First(&currentUser)
			if result.Error != nil {
				c.JSON(500, gin.H{"code": 1, "message": "获取当前账号失败"})
				return
			}

			var duplicated int64
			sdb.DB.Model(&sdb.User{}).
				Where("deleted_at IS NULL AND UserName = ? AND id <> ?", req.UserName, currentUser.ID).
				Count(&duplicated)
			if duplicated > 0 {
				c.JSON(400, gin.H{"code": 1, "message": "账号已存在"})
				return
			}

			updates := map[string]interface{}{
				"UserName": req.UserName,
			}
			userNameChanged := currentUser.UserName != req.UserName
			if req.Password != "" {
				hash, hashErr := sdb.HashPassword(req.Password)
				if hashErr != nil {
					c.JSON(500, gin.H{"code": 1, "message": "密码加密失败"})
					return
				}
				updates["PassWord"] = hash
			}

			if err := sdb.DB.Model(&sdb.User{}).Where("id = ?", currentUser.ID).Updates(updates).Error; err != nil {
				c.JSON(500, gin.H{"code": 1, "message": "更新失败"})
				return
			}

			if userNameChanged {
				token, tokenErr := GenerateToken()
				if tokenErr != nil {
					mylog.Logger.Error("更新账号后生成 token 失败", zap.Error(tokenErr))
					c.JSON(500, gin.H{"code": 1, "message": "账号已更新，但登录状态刷新失败"})
					return
				}
				setAuthCookie(c, token, 3600*24)
			}

			c.JSON(200, gin.H{
				"code":    0,
				"message": "账号安全设置已更新",
				"relogin": false,
			})
		})

		// 添加钱包地址
		admin.POST("/api/wallets", func(c *gin.Context) {
			// 传入的币种和钱包地址和汇率和状态
			var wallet sdb.WalletAddress

			if err := c.ShouldBindJSON(&wallet); err != nil {
				c.JSON(400, gin.H{"code": 1, "message": "参数错误"})
				return
			}

			if wallet.Currency == "" || wallet.Token == "" {
				c.JSON(400, gin.H{"code": 1, "message": "币种和钱包地址不能为空"})
				return
			}

			if err := validateWalletToken(wallet.Currency, wallet.Token); err != nil {
				c.JSON(400, gin.H{"code": 1, "message": err.Error()})
				return
			}

			if wallet.Rate <= 0 {

				c.JSON(400, gin.H{"code": 1, "message": "汇率必须大于0"})
				return
			}

			// // 创建汇率维护表
			// var autoprice sdb.AutoRate

			// autoprice.Currency = wallet.Currency

			if wallet.AutoRate == true {
				mylog.Logger.Info("自动汇率已启用", zap.String("币种", wallet.Currency))
				// 自动汇率是否启用
				wallet.AutoRate = true
				// 设置钱包地址表里面的汇率字段
				// 币种
				C := ""
				// 如果order.Currency包含了"USDT"，那么C就等于"USDT"
				switch {
				case strings.Contains(wallet.Currency, "USDT"):
					C = "USDT"
				case strings.Contains(wallet.Currency, "USDC"):
					C = "USDC"
				case strings.Contains(wallet.Currency, "TRX"):
					C = "TRX"
				default:
					mylog.Logger.Error("当前币种将自动设置默认汇率：10，请检查是否错误", zap.String("币种", wallet.Currency))
				}
				price, err := Autoprice.Start(C)
				if err != nil {
					mylog.Logger.Error("获取自动汇率失败，将设置默认汇率，USDT:7,USDC:7,TRX:2.5", zap.Error(err))
					//将设置默认汇率
					// 优化后的switch语句
					switch C {
					case "USDT", "USDC":
						wallet.Rate = 7
					case "TRX":
						wallet.Rate = 2.5
					default:
						wallet.Rate = 10
					}
				} else {
					wallet.Rate = price
				}

			} else {
				wallet.AutoRate = false
			}
			// 查询数据库中的钱包记录
			var existingWallet sdb.WalletAddress
			if wallet.AutoRate == false {
				// 检查输入的币种的汇率是否存在，如果存在验证输入的汇率和数据库中的汇率是否一致，如果能找到，说明已经存在了，返回错误要求为汇率必须输入为一致；
				// 这里使用Last是为了获取最新的一条记录，因为如果有两条记录，说明之前有过修改，所以需要验证最新的一条记录的汇率是否一致
				if err := sdb.DB.Where("currency = ? ", wallet.Currency).Last(&existingWallet).Error; err == nil {

					/* fmt.Println("existingWallet.Rate:", existingWallet.Rate)
					fmt.Println("wallet.Rate:", wallet.Rate) */

					if wallet.Rate != existingWallet.Rate {
						c.JSON(400, gin.H{"code": 1, "message": fmt.Sprintf("每一个币种的汇率必须一致，你输入的钱包汇率配置错误，请把汇率设置为%v", existingWallet.Rate)})
						return

					}

				}
			}

			// 检查是否已经存在了该币种和地址都存在的记录，如果存在，返回错误，提示钱包地址在该币种下已经存在
			if err := sdb.DB.Where("currency = ? AND token = ?", wallet.Currency, wallet.Token).First(&existingWallet).Error; err == nil {
				c.JSON(400, gin.H{"code": 1, "message": "钱包地址在当前币种中已存在"})
				return
			}

			// 创建钱包地址
			if err := sdb.DB.Create(&wallet).Error; err != nil {
				c.JSON(500, gin.H{"code": 1, "message": "创建失败"})
				return
			}

			c.JSON(200, gin.H{"code": 0, "message": "添加成功", "data": wallet})

		})

		// 编辑钱包地址
		admin.PUT("/api/wallets/:id", func(c *gin.Context) {
			walletId := c.Param("id")
			var wallet sdb.WalletAddress

			if err := c.ShouldBindJSON(&wallet); err != nil {
				c.JSON(400, gin.H{"code": 1, "message": "参数错误"})
				return
			}

			if wallet.Currency == "" || wallet.Token == "" {
				c.JSON(400, gin.H{"code": 1, "message": "币种和钱包地址不能为空"})
				return
			}

			if err := validateWalletToken(wallet.Currency, wallet.Token); err != nil {
				c.JSON(400, gin.H{"code": 1, "message": err.Error()})
				return
			}

			if wallet.Rate <= 0 {
				c.JSON(400, gin.H{"code": 1, "message": "汇率必须大于0"})
				return
			}

			if wallet.AutoRate == true {
				mylog.Logger.Info("自动汇率已启用", zap.String("币种", wallet.Currency))
				// 自动汇率是否启用
				wallet.AutoRate = true
				// 设置钱包地址表里面的汇率字段
				// 币种
				C := ""
				// 如果order.Currency包含了"USDT"，那么C就等于"USDT"
				switch {
				case strings.Contains(wallet.Currency, "USDT"):
					C = "USDT"
				case strings.Contains(wallet.Currency, "USDC"):
					C = "USDC"
				case strings.Contains(wallet.Currency, "TRX"):
					C = "TRX"
				default:
					mylog.Logger.Error("当前币种将自动设置默认汇率：10，请检查是否错误", zap.String("币种", wallet.Currency))
				}
				price, err := Autoprice.Start(C)
				if err != nil {
					mylog.Logger.Error("获取自动汇率失败，将设置默认汇率，USDT:7,USDC:7,TRX:2.5", zap.Error(err))
					//将设置默认汇率
					// 优化后的switch语句
					switch C {
					case "USDT", "USDC":
						wallet.Rate = 7
					case "TRX":
						wallet.Rate = 2.5
					default:
						wallet.Rate = 10
					}
				} else {
					wallet.Rate = price
				}

			} else {
				wallet.AutoRate = false
			}

			/* 	// 检查钱包地址是否已存在（排除当前记录）
			var existingWallet sdb.WalletAddress
			if err := sdb.DB.Where("token = ? AND id != ?", wallet.Token, walletId).First(&existingWallet).Error; err == nil {
				c.JSON(400, gin.H{"code": 1, "message": "钱包地址已存在"})
				return
			} */

			// 更新钱包地址
			result := sdb.DB.Model(&sdb.WalletAddress{}).Where("id = ?", walletId).Updates(map[string]interface{}{
				"Currency": wallet.Currency,
				"Token":    wallet.Token,
				"Rate":     wallet.Rate,
				"Status":   wallet.Status,
				"AutoRate": wallet.AutoRate,
			})

			if result.Error != nil {
				c.JSON(500, gin.H{"code": 1, "message": "更新失败"})
				return
			}

			if result.RowsAffected == 0 {
				c.JSON(404, gin.H{"code": 1, "message": "钱包地址更新失败"})
				return
			}

			c.JSON(200, gin.H{"code": 0, "message": "更新成功"})

		})

		// 删除钱包地址
		admin.DELETE("/api/wallets/:id", func(c *gin.Context) {
			walletId := c.Param("id")

			// 删除钱包地址
			result := sdb.DB.Delete(&sdb.WalletAddress{}, walletId)
			if result.Error != nil {
				c.JSON(500, gin.H{"code": 1, "message": "删除失败"})
				return
			}

			if result.RowsAffected == 0 {
				c.JSON(404, gin.H{"code": 1, "message": "钱包地址不存在"})
				return
			}

			c.JSON(200, gin.H{"code": 0, "message": "删除成功"})

		})

		// 系统设置管理API
		// 获取系统设置
		admin.GET("/api/settings", func(c *gin.Context) {
			var setting sdb.Setting
			result := sdb.DB.First(&setting)
			if result.Error != nil {
				c.JSON(500, gin.H{
					"code": -1,
					"msg":  "获取系统设置失败",
				})
				return
			}
			if result.RowsAffected == 0 {
				c.JSON(500, gin.H{
					"code": -1,
					"msg":  "系统设置不存在",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "success",
				"data": settingResponseData(setting),
			})
		})

		admin.GET("/api/settings/secret-key", func(c *gin.Context) {
			var setting sdb.Setting
			result := sdb.DB.First(&setting)
			if result.Error != nil || result.RowsAffected == 0 {
				c.JSON(500, gin.H{"code": 1, "message": "通信密钥读取失败"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "success",
				"data": gin.H{
					"SecretKey": setting.SecretKey,
				},
			})
		})

		// 保存系统设置
		admin.POST("/api/settings", func(c *gin.Context) {
			var req map[string]interface{}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"code": 1, "message": "参数错误"})
				return
			}

			// 获取当前设置
			var setting sdb.Setting
			result := sdb.DB.First(&setting)
			if result.Error != nil {
				c.JSON(500, gin.H{"code": 1, "message": "获取当前设置失败"})
				return
			}

			updates, err := settingUpdatesFromRequest(req)
			if err != nil {
				c.JSON(400, gin.H{"code": 1, "message": err.Error()})
				return
			}

			previousValues := map[string]interface{}{}
			for field := range updates {
				switch field {
				case "AppName":
					previousValues[field] = setting.AppName
				case "AppUrl":
					previousValues[field] = setting.AppUrl
				case "Httpport":
					previousValues[field] = setting.Httpport
				case "ExpirationDate":
					previousValues[field] = setting.ExpirationDate
				case "CustomerServiceContact":
					previousValues[field] = setting.CustomerServiceContact
				case "SecretKey":
					previousValues[field] = setting.SecretKey
				case "Redishost":
					previousValues[field] = setting.Redishost
				case "Redisport":
					previousValues[field] = setting.Redisport
				case "Redispasswd":
					previousValues[field] = setting.Redispasswd
				case "Redisdb":
					previousValues[field] = setting.Redisdb
				case "Tgbotkey":
					previousValues[field] = setting.Tgbotkey
				case "Tgchatid":
					previousValues[field] = setting.Tgchatid
				case "Barkkey":
					previousValues[field] = setting.Barkkey
				}
			}

			restorePreviousSettings := func() {
				if len(previousValues) == 0 {
					return
				}

				if err := sdb.DB.Model(&setting).Where("id = ?", setting.ID).Updates(previousValues).Error; err != nil {
					mylog.Logger.Error("回滚系统设置失败", zap.Error(err))
					return
				}

				if reloadErr := rdb.Reload(); reloadErr != nil {
					mylog.Logger.Error("回滚后重载 Redis 失败", zap.Error(reloadErr))
				}
				if reloadErr := mq.Reload(); reloadErr != nil {
					mylog.Logger.Error("回滚后重载 MQ 失败", zap.Error(reloadErr))
				}
			}

			// 执行更新
			if len(updates) > 0 {
				result := sdb.DB.Model(&setting).Where("id = ?", setting.ID).Updates(updates)
				if result.Error != nil {
					c.JSON(500, gin.H{"code": 1, "message": "保存失败"})
					return
				}
			}

			redisFields := []string{"Redishost", "Redisport", "Redispasswd", "Redisdb"}
			redisUpdated := false
			for _, field := range redisFields {
				if _, ok := updates[field]; ok {
					redisUpdated = true
					break
				}
			}

			if redisUpdated {
				if err := rdb.Reload(); err != nil {
					mylog.Logger.Error("重载 Redis 客户端失败", zap.Error(err))
					restorePreviousSettings()
					c.JSON(500, gin.H{"code": 1, "message": "Redis 配置校验失败，已回滚到原配置"})
					return
				}
				if err := mq.Reload(); err != nil {
					mylog.Logger.Error("重载 MQ Redis 连接失败", zap.Error(err))
					restorePreviousSettings()
					c.JSON(500, gin.H{"code": 1, "message": "任务队列重连失败，已回滚到原配置"})
					return
				}
			}

			c.JSON(200, gin.H{"code": 0, "message": "保存成功"})
		})

		// 手动补单
		admin.POST("/api/manual-complete-order", func(c *gin.Context) {
			var req struct {
				OrderID string `json:"order_id" validate:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"code": 1, "message": "参数绑定错误"})
				return
			}
			// validate := validator.New()
			// 验证参数是否符合要求
			err := validate.Struct(req)
			if err != nil {
				c.JSON(400, gin.H{"code": 1, "message": "参数验证错误"})
				return
			}
			orderID := strings.TrimSpace(req.OrderID)
			order, err := sdb.GetLatestOrderByTradeOrOrderID(orderID)
			if err != nil || order.ID == 0 {
				c.JSON(400, gin.H{"code": 1, "message": "订单不存在"})
				return
			}

			if order.Status == sdb.StatusPaySuccess {
				c.JSON(200, gin.H{"code": 0, "message": "订单已手动完成"})
				return
			}

			result := sdb.DB.Model(&sdb.Orders{}).
				Where("id = ?", order.ID).
				Update("status", sdb.StatusPaySuccess)
			if result.Error != nil {
				c.JSON(500, gin.H{"code": 1, "message": "保存失败"})
				return
			}

			order.Status = sdb.StatusPaySuccess
			mylog.Logger.Info("订单已手动完成",
				zap.String("lookup_value", orderID),
				zap.String("trade_id", order.TradeId),
				zap.String("order_id", order.OrderId),
			)
			// 异步回调
			go cron.ProcessCallback(order)
			c.JSON(200, gin.H{"code": 0, "message": "订单已手动完成"})
		})

		// API密钥管理API
		// 获取波场和以太坊API密钥
		admin.GET("/api/apikeys", func(c *gin.Context) {
			var apiKey sdb.ApiKey
			result := sdb.DB.First(&apiKey)
			if result.Error != nil {
				c.JSON(500, gin.H{
					"code": -1,
					"msg":  "获取API密钥失败",
				})
				return
			}
			if result.RowsAffected == 0 {
				c.JSON(500, gin.H{
					"code": -1,
					"msg":  "API密钥不存在",
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "success",
				"data": apiKey,
			})
		})

		// 保存API密钥
		admin.POST("/api/apikeys", func(c *gin.Context) {
			var req map[string]interface{}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"code": 1, "message": "参数错误"})
				return
			}

			// 获取当前API密钥
			var apiKey sdb.ApiKey
			result := sdb.DB.First(&apiKey)
			if result.Error != nil {
				c.JSON(500, gin.H{"code": 1, "message": "获取当前API密钥失败"})
				return
			}

			// 更新字段（只更新传入的字段）
			updates := make(map[string]interface{})

			// API密钥设置
			if tronscan, ok := req["tronscan"]; ok {
				updates["Tronscan"] = tronscan
			}
			if trongrid, ok := req["trongrid"]; ok {
				updates["Trongrid"] = trongrid
			}
			if etherscan, ok := req["etherscan"]; ok {
				updates["Etherscan"] = etherscan
			}

			// 执行更新（更新获取到的apiKey记录）
			if len(updates) > 0 {
				// 更新获取到的apiKey变量对应的记录
				result := sdb.DB.Model(&apiKey).Updates(updates)
				if result.Error != nil {
					c.JSON(500, gin.H{"code": 1, "message": "保存失败"})
					return
				}
				// 检查是否有记录被更新
				if result.RowsAffected == 0 {
					c.JSON(500, gin.H{"code": 1, "message": "没有找到要更新的记录"})
					return
				}
			}

			c.JSON(200, gin.H{"code": 0, "message": "保存成功"})
		})

		// 退出登录路由
		admin.POST("/logout", func(c *gin.Context) {
			// 清除cookie
			clearAuthCookie(c)
			// 跳转到登录页
			c.JSON(200, gin.H{"code": 0, "message": "退出成功"})
		})

	}

	// 定义订单路由组
	api := r.Group("/api", AuthMiddleware())

	api.POST("/create_order", CreateTransaction)

	// 定义支付路由组
	pay := r.Group("/pay")
	// 返回支付页面【支付页面是静态页面，所以需要返回html文件】
	pay.GET("/checkout-counter/:trade_id", CheckoutCounter)

	// 检查订单状态
	pay.GET("/check-status/:trade_id", CheckOrderStatus)

	// Vue SPA History Mode Fallback
	r.NoRoute(func(c *gin.Context) {
		// 排除 API 路由，其余 GET 请求全部返回 SPA 首页
		if c.Request.Method == "GET" && !strings.HasPrefix(c.Request.URL.Path, "/api") && !strings.HasPrefix(c.Request.URL.Path, "/admin/api") {
			c.File("./static/admin_spa/index.html")
		} else {
			c.JSON(404, gin.H{"code": 404, "message": "Not Found"})
		}
	})

	// 读取系统设置
	port := sdb.GetSetting().Httpport
	// endless.ListenAndServe(":8080", r)
	endless.ListenAndServe(fmt.Sprintf(":%d", port), r)
}

func validateWalletToken(currency string, token string) error {
	token = strings.TrimSpace(token)
	switch currency {
	case "USDT-TRC20", "TRX":
		if len(token) != 34 || !strings.HasPrefix(token, "T") {
			return fmt.Errorf("TRON 钱包地址格式不正确")
		}
	case "USDT-Polygon", "USDT-BSC", "USDT-ERC20", "USDT-ArbitrumOne", "USDC-ERC20", "USDC-Polygon", "USDC-BSC", "USDC-ArbitrumOne":
		if len(token) != 42 || !strings.HasPrefix(strings.ToLower(token), "0x") {
			return fmt.Errorf("EVM 钱包地址格式不正确")
		}
	}

	return nil
}
