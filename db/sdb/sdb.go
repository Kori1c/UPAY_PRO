package sdb

import (
	"bytes"
	cryptorand "crypto/rand"
	"fmt"
	"os"
	"strconv"
	"time"
	"upay_pro/mylog"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"golang.org/x/crypto/bcrypt"
)

var DB *gorm.DB

func init() {
	// 确保目录存在
	// 创建目录
	os.MkdirAll("DBS", 0755)
	db, err := gorm.Open(sqlite.Open("DBS/upay_pro.db"), &gorm.Config{})
	if err != nil {
		mylog.Logger.Error("open db error", zap.Error(err))
		mylog.Logger.Sync()
	}
	mylog.Logger.Info("数据库链接成功")
	DB = db
	Start()
}

type User struct {
	gorm.Model
	UserName string `gorm:"column:UserName"`
	PassWord string `gorm:"column:PassWord"`
}

// 订单状态
const (
	StatusWaitPay     = 1 // 等待支付
	StatusPaySuccess  = 2 // 支付成功
	StatusExpired     = 3 // 已过期
	CallBackConfirmOk = 1 // 回调已确认
	CallBackConfirmNo = 2 // 回调未确认
)

const (
	CallbackTriggerAuto   = "auto"
	CallbackTriggerManual = "manual"
)

const (
	CallbackResultQueued  = "queued"
	CallbackResultSuccess = "success"
	CallbackResultFailed  = "failed"
)

// 订单表
type Orders struct {
	gorm.Model
	TradeId            string  // UPAY订单号
	OrderId            string  // 客户交易id
	BlockTransactionId string  // 区块id
	Amount             float64 // 订单金额，保留2位小数
	ActualAmount       float64 // 订单实际需要支付的金额，保留4位小数
	Type               string  //钱包类型
	Token              string  // 所属钱包地址
	Status             int     // 1：等待支付，2：支付成功，3：已过期

	NotifyUrl       string // 异步回调地址
	RedirectUrl     string // 同步回调地址
	CallbackNum     int    // 回调次数
	CallBackConfirm int    // 回调是否已确认 1是 2否
	CallbackMessage string // 最近一次回调结果说明
	LastCallbackAt  *time.Time
	StartTime       int64 // 订单开始时间（时间戳）
	ExpirationTime  int64 // 订单过期时间（时间戳）

}

type CallbackEvent struct {
	gorm.Model
	OrderRowID    uint   `gorm:"index"`
	TradeID       string `gorm:"index"`
	TriggerType   string
	Result        string
	Message       string
	AttemptNumber int
}

// 钱包状态
const (
	TokenStatusEnable  = 1 // 钱包启用
	TokenStatusDisable = 2 // 钱包禁用
)

// 钱包地址表
type WalletAddress struct {
	gorm.Model
	Currency string  // 币种
	Token    string  // 钱包token
	Status   int     // 1:启用 2:禁用
	Rate     float64 // 汇率
	AutoRate bool    `gorm:"column:AutoRate;default:false"` // 汇率是否自动维护

	// - 0 ：表示 false ，即 禁用 自动汇率功能
	// - 1 ：表示 true ，即 启用 自动汇率功能
}

// 汇率维护表
/* type AutoRate struct {
	gorm.Model
	Currency string `gorm:"column:currency"`                // 币种
	AutoRate bool   `gorm:"column:autoRate default:false" ` // 汇率是否自动维护
} */

type Setting struct {
	gorm.Model
	AppUrl                 string
	SecretKey              string
	JWTSecret              string
	PasswordLoginEnabled   bool `gorm:"default:true"`
	Httpport               int
	Tgbotkey               string
	Tgchatid               string
	Barkkey                string
	Redishost              string
	Redisport              int
	Redispasswd            string
	Redisdb                int
	ExpirationDate         time.Duration
	AppName                string //应用名称
	CustomerServiceContact string //客户服务联系方式

}

type PasskeyCredential struct {
	gorm.Model
	UserID            uint   `gorm:"index"`
	CredentialID      string `gorm:"uniqueIndex"`
	CredentialIDB64   string `gorm:"uniqueIndex"`
	PublicKey         []byte
	CredentialJSON    []byte
	AttestationType   string
	AttestationFormat string
	AAGUID            string
	SignCount         uint32
	Transports        string
	DeviceLabel       string
	LastUsedAt        *time.Time
}

type PasskeyChallenge struct {
	gorm.Model
	UserID      *uint  `gorm:"index"`
	FlowType    string `gorm:"index"`
	ChallengeID string `gorm:"uniqueIndex"`
	SessionData string
	ExpiresAt   time.Time `gorm:"index"`
}
type ApiKey struct {
	gorm.Model
	Tronscan  string
	Trongrid  string
	Etherscan string
}

// 创建一个单独的表用来存储订单号和队列ID

type TradeIdTaskID struct {
	gorm.Model
	// 这里的订单号是系统订单号不是商户订单号
	TradeId string `gorm:"column:TradeId"`
	TaskID  string `gorm:"column:TaskID"`
}

func Start() {
	mylog.Logger.Info("开始初始化数据库")
	mylog.Logger.Info("开始迁移数据库")
	// 迁移用户表
	DB.AutoMigrate(&User{})

	// 初始化用户表
	result := DB.First(&User{})
	if result.Error != nil {
		mylog.Logger.Info("获取用户表失败")
	}
	if result.RowsAffected == 0 {
		mylog.Logger.Info("用户表为空")

		hashedPassword, _ := HashPassword(Defaultuserpassword)
		mylog.Logger.Info("初始用户名:", zap.String("username", defaultuserusername))
		mylog.Logger.Info("初始密码:", zap.String("password", Defaultuserpassword))
		// 创建用户
		result := DB.Create(&User{
			UserName: defaultuserusername,
			PassWord: hashedPassword,
		})
		if result.Error != nil {
			mylog.Logger.Info("创建用户失败")
		} else {
			mylog.Logger.Info("创建用户成功")
		}

	}

	// 迁移订单表
	DB.AutoMigrate(&Orders{})
	DB.AutoMigrate(&CallbackEvent{})
	// 迁移钱包地址表
	DB.AutoMigrate(&WalletAddress{})
	// 迁移设置表
	hadPasswordLoginColumn := DB.Migrator().HasColumn(&Setting{}, "password_login_enabled")
	DB.AutoMigrate(&Setting{})
	if !hadPasswordLoginColumn {
		if err := DB.Model(&Setting{}).Where("1 = 1").Update("PasswordLoginEnabled", true).Error; err != nil {
			mylog.Logger.Error("初始化密码登录开关失败", zap.Error(err))
		}
	}
	// 迁移 Passkey 表
	DB.AutoMigrate(&PasskeyCredential{}, &PasskeyChallenge{})
	//迁移apikey表
	DB.AutoMigrate(&ApiKey{})
	// 检查设置表是否为空，如果为空则插入默认设置
	var settingCount int64
	DB.Model(&Setting{}).Count(&settingCount)
	// 给设置表设置默认值
	if settingCount == 0 {
		mylog.Logger.Info("设置表为空，创建默认设置")
		result := DB.Create(&Setting{
			AppUrl:                 envString("UPAY_APP_URL", "http://localhost"),
			SecretKey:              GenerateSecretKey(48),
			JWTSecret:              GenerateSecretKey(48),
			PasswordLoginEnabled:   true,
			Httpport:               envInt("UPAY_HTTP_PORT", 8090),
			Tgbotkey:               "",
			Tgchatid:               "",
			Barkkey:                "",
			Redishost:              envString("UPAY_REDIS_HOST", "127.0.0.1"),
			Redisport:              envInt("UPAY_REDIS_PORT", 6379),
			Redispasswd:            envString("UPAY_REDIS_PASSWORD", ""),
			Redisdb:                envInt("UPAY_REDIS_DB", 0),
			ExpirationDate:         ExpirationDate,
			AppName:                "",
			CustomerServiceContact: "",
		})
		if result.Error != nil {
			mylog.Logger.Error("创建默认设置失败", zap.Error(result.Error))
		} else {
			mylog.Logger.Info("默认设置创建成功")
		}
	}

	var setting Setting
	if err := DB.First(&setting).Error; err == nil && setting.JWTSecret == "" {
		jwtSecret := setting.SecretKey
		if jwtSecret == "" {
			jwtSecret = GenerateSecretKey(48)
		}

		if err := DB.Model(&setting).Update("JWTSecret", jwtSecret).Error; err != nil {
			mylog.Logger.Error("初始化 JWT 密钥失败", zap.Error(err))
		} else {
			mylog.Logger.Info("JWT 密钥初始化完成")
		}
	}

	// 给APIKEY表设置默认值
	var apikeyCount int64
	// 检查APIKEY表是否为空，如果为空则插入默认值
	DB.Model(&ApiKey{}).Count(&apikeyCount)
	if apikeyCount == 0 {
		mylog.Logger.Info("APIKEY表为空，创建默认设置")
		result := DB.Create(&ApiKey{
			Tronscan:  "",
			Trongrid:  "",
			Etherscan: "",
		})
		if result.Error != nil {
			mylog.Logger.Error("APIKEY表创建默认设置失败", zap.Error(result.Error))
		} else {
			mylog.Logger.Info("APIKEY表默认设置创建成功")
		}
	}
	// 迁移订单号和队列ID表
	DB.AutoMigrate(&TradeIdTaskID{})
	// 迁移汇率维护表
	// DB.AutoMigrate(&AutoRate{})

}

func RecordCallbackEvent(order Orders, triggerType string, result string, message string, attemptNumber int) error {
	event := CallbackEvent{
		OrderRowID:    order.ID,
		TradeID:       order.TradeId,
		TriggerType:   triggerType,
		Result:        result,
		Message:       message,
		AttemptNumber: attemptNumber,
	}

	return DB.Create(&event).Error
}

const (
	ExpirationDate = time.Minute * 10
)

var (
	defaultuserusername = "admin"
	Defaultuserpassword = "admin"
)

// 设置一个生成密钥的函数

func GenerateSecretKey(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	if length <= 0 {
		return ""
	}

	randomBytes := make([]byte, length)
	if _, err := cryptorand.Read(randomBytes); err != nil {
		mylog.Logger.Error("生成安全随机密钥失败", zap.Error(err))
		return ""
	}

	var key bytes.Buffer
	key.Grow(length)
	for _, b := range randomBytes {
		key.WriteByte(chars[int(b)%len(chars)])
	}

	return key.String()
}

func GetSetting() Setting {
	var setting Setting
	DB.First(&setting)
	applySettingEnvOverrides(&setting)
	if setting.JWTSecret == "" {
		setting.JWTSecret = setting.SecretKey
	}

	/* if result.RowsAffected == 0 {
		mylog.Logger.Info("系统设置不存在，创建默认设置")
		// 创建默认设置
		defaultSetting := Setting{
			AppUrl:                 "",
			SecretKey:              GenerateSecretKey(48),
			Httpport:               8080,
			Tgbotkey:               "",
			Tgchatid:               "",
			Barkkey:                "",
			Redishost:              "127.0.0.1",
			Redisport:              6379,
			Redispasswd:            "",
			Redisdb:                0,
			ExpirationDate:         ExpirationDate,
			AppName:                "",
			CustomerServiceContact: "",
		}
		createResult := DB.Create(&defaultSetting)
		if createResult.Error != nil {
			mylog.Logger.Error("创建默认设置失败", zap.Error(createResult.Error))
			return setting // 返回空设置
		}
		return defaultSetting
	} */

	return setting
}

func envString(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func applySettingEnvOverrides(setting *Setting) {
	if setting == nil {
		return
	}

	if value := os.Getenv("UPAY_APP_URL"); value != "" {
		setting.AppUrl = value
	}
	if value := os.Getenv("UPAY_HTTP_PORT"); value != "" {
		setting.Httpport = envInt("UPAY_HTTP_PORT", setting.Httpport)
	}
	if value := os.Getenv("UPAY_REDIS_HOST"); value != "" {
		setting.Redishost = value
	}
	if value := os.Getenv("UPAY_REDIS_PORT"); value != "" {
		setting.Redisport = envInt("UPAY_REDIS_PORT", setting.Redisport)
	}
	if value := os.Getenv("UPAY_REDIS_PASSWORD"); value != "" {
		setting.Redispasswd = value
	}
	if value := os.Getenv("UPAY_REDIS_DB"); value != "" {
		setting.Redisdb = envInt("UPAY_REDIS_DB", setting.Redisdb)
	}
}

func HashPassword(password string) (string, error) {
	cost := 12 // 计算成本，值越大越安全但越耗时
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// VerifyPassword 验证输入密码是否匹配存储的哈希
func VerifyPassword(inputPassword, storedHash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(inputPassword))
	return err == nil // true 表示验证通过
}

// 因为同样的钱包类型，可能有多个钱包地址，所以这里返回一个数组
func GetWalletAddress(type_ string) []WalletAddress {

	var walletAddress []WalletAddress

	DB.Where("currency = ? and status = ?", type_, TokenStatusEnable).Find(&walletAddress)
	return walletAddress
}

func DisableWalletAddress(currency string, token string) bool {
	if currency == "" || token == "" {
		return false
	}

	result := DB.Model(&WalletAddress{}).
		Where("currency = ? AND token = ? AND status = ?", currency, token, TokenStatusEnable).
		Update("status", TokenStatusDisable)

	if result.Error != nil {
		mylog.Logger.Error("禁用钱包地址失败", zap.String("currency", currency), zap.String("token", token), zap.Error(result.Error))
		return false
	}

	return result.RowsAffected > 0
}

func (n WalletAddress) String() string {
	return fmt.Sprintf("%s:%v", n.Token, n.Rate)
}

func GetOrderByOrderId(orderId string) Orders {
	var order Orders
	DB.Where("order_id = ?", orderId).Last(&order)
	return order
}

func GetLatestOrderByTradeOrOrderID(identifier string) (Orders, error) {
	var order Orders
	result := DB.Where("trade_id = ?", identifier).Order("id DESC").Limit(1).Find(&order)
	if result.Error != nil {
		return order, result.Error
	}
	if result.RowsAffected > 0 {
		return order, nil
	}

	result = DB.Where("order_id = ?", identifier).Order("id DESC").Limit(1).Find(&order)
	if result.Error != nil {
		return order, result.Error
	}
	if result.RowsAffected == 0 {
		return order, gorm.ErrRecordNotFound
	}

	return order, nil
}

func SyncExpiredOrders() error {
	nowMillis := time.Now().UnixMilli()
	return DB.Model(&Orders{}).
		Where("status = ? AND expiration_time > 0 AND expiration_time <= ?", StatusWaitPay, nowMillis).
		Update("status", StatusExpired).Error
}

func SyncOrderStatusByTradeID(tradeID string) (Orders, error) {
	var order Orders
	if err := DB.Where("trade_id = ?", tradeID).Last(&order).Error; err != nil {
		return order, err
	}

	if order.Status == StatusWaitPay && order.ExpirationTime > 0 && order.ExpirationTime <= time.Now().UnixMilli() {
		order.Status = StatusExpired
		if err := DB.Model(&order).Update("status", StatusExpired).Error; err != nil {
			return order, err
		}
	}

	return order, nil
}

func GetApiKey() ApiKey {
	var apikey ApiKey
	DB.First(&apikey)
	return apikey
}

func GetUserByUsername() string {
	var user User
	re := DB.First(&user)
	if re.Error != nil {
		mylog.Logger.Error("查询用户失败", zap.Error(re.Error))
		return ""
	}
	return user.UserName
}
