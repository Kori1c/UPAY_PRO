package web

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"upay_pro/db/rdb"
	"upay_pro/db/sdb"
	"upay_pro/dto"
	"upay_pro/mq"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func TestGenerateAndParseToken(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken returned empty token")
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}
	if claims == nil {
		t.Fatal("ParseToken returned nil claims")
	}
	if claims.UserName == "" {
		t.Fatal("claims.UserName should not be empty")
	}
}

func TestJWTAuthMiddlewareForAPIUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(JWTAuthMiddleware())
	r.GET("/admin/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/api/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["code"] != float64(-1) {
		t.Fatalf("expected code -1, got %#v", body["code"])
	}
	if body["msg"] != "未登录" {
		t.Fatalf("expected msg 未登录, got %#v", body["msg"])
	}
}

func TestJWTAuthMiddlewareForPageRedirectsToLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(JWTAuthMiddleware())
	r.GET("/admin", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, w.Code)
	}

	location := w.Header().Get("Location")
	if location != "/login" {
		t.Fatalf("expected redirect to /login, got %q", location)
	}
}

func TestRequireRedisReadyForOrderFlow(t *testing.T) {
	original := rdb.RDB
	t.Cleanup(func() {
		rdb.RDB = original
	})

	rdb.RDB = nil
	if err := requireRedisReadyForOrderFlow(); err == nil {
		t.Fatal("expected error when redis client is nil")
	}

	rdb.RDB = redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	if err := requireRedisReadyForOrderFlow(); err != nil {
		t.Fatalf("expected nil error when redis client exists, got %v", err)
	}
}

func TestHealthzReturnsOKWhenDatabaseAndRedisAreReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testEnv := setupCreateOrderTestEnv(t)

	router := gin.New()
	registerHealthRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Code int `json:"code"`
		Data struct {
			Status string `json:"status"`
			DB     string `json:"db"`
			Redis  string `json:"redis"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	if body.Data.Status != "ok" || body.Data.DB != "ok" || body.Data.Redis != "ok" {
		t.Fatalf("unexpected health response: %#v", body.Data)
	}

	if testEnv.db == nil {
		t.Fatal("test env should keep db alive")
	}
}

func TestHealthzReturnsUnavailableWhenRedisIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupCreateOrderTestEnv(t)
	rdb.RDB = nil

	router := gin.New()
	registerHealthRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOperationsSummaryReturnsActionableRuntimeSignals(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testEnv := setupCreateOrderTestEnv(t)

	now := time.Now()
	if err := testEnv.db.Create(&[]sdb.Orders{
		{
			TradeId:        "OPS-PENDING-001",
			OrderId:        "OPS-PENDING-001",
			Amount:         10,
			ActualAmount:   10,
			Type:           "USDT-TRC20",
			Token:          "TJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeC",
			Status:         sdb.StatusWaitPay,
			StartTime:      now.UnixMilli(),
			ExpirationTime: now.Add(10 * time.Minute).UnixMilli(),
		},
		{
			TradeId:            "OPS-SUCCESS-001",
			OrderId:            "OPS-SUCCESS-001",
			Amount:             11,
			ActualAmount:       11,
			Type:               "USDT-TRC20",
			Token:              "TJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeC",
			Status:             sdb.StatusPaySuccess,
			NotifyUrl:          "https://example.com/notify",
			CallBackConfirm:    sdb.CallBackConfirmOk,
			BlockTransactionId: "tx-ok",
		},
		{
			TradeId:         "OPS-CALLBACK-001",
			OrderId:         "OPS-CALLBACK-001",
			Amount:          12,
			ActualAmount:    12,
			Type:            "USDT-TRC20",
			Token:           "TJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeC",
			Status:          sdb.StatusPaySuccess,
			NotifyUrl:       "https://example.com/notify",
			CallbackNum:     2,
			CallBackConfirm: sdb.CallBackConfirmNo,
		},
		{
			TradeId:         "OPS-NO-NOTIFY-001",
			OrderId:         "OPS-NO-NOTIFY-001",
			Amount:          12,
			ActualAmount:    12,
			Type:            "USDT-TRC20",
			Token:           "TJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeC",
			Status:          sdb.StatusPaySuccess,
			NotifyUrl:       "",
			CallBackConfirm: sdb.CallBackConfirmNo,
		},
		{
			TradeId:      "OPS-EXPIRED-001",
			OrderId:      "OPS-EXPIRED-001",
			Amount:       13,
			ActualAmount: 13,
			Type:         "USDT-TRC20",
			Token:        "TJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeC",
			Status:       sdb.StatusExpired,
		},
	}).Error; err != nil {
		t.Fatalf("failed to seed operation orders: %v", err)
	}

	if err := testEnv.db.Create(&sdb.WalletAddress{
		Currency: "USDT-TRC20",
		Token:    "TJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeD",
		Status:   sdb.TokenStatusDisable,
		Rate:     1,
	}).Error; err != nil {
		t.Fatalf("failed to seed disabled wallet: %v", err)
	}

	failedAt := time.Now().Add(-2 * time.Hour)
	succeededAt := time.Now().Add(-30 * time.Minute)
	if err := testEnv.db.Create(&[]sdb.CallbackEvent{
		{
			Model:         gorm.Model{CreatedAt: failedAt},
			OrderRowID:    3,
			TradeID:       "OPS-CALLBACK-001",
			TriggerType:   sdb.CallbackTriggerAuto,
			Result:        sdb.CallbackResultFailed,
			Message:       "签名错误",
			AttemptNumber: 2,
		},
		{
			Model:         gorm.Model{CreatedAt: succeededAt},
			OrderRowID:    2,
			TradeID:       "OPS-SUCCESS-001",
			TriggerType:   sdb.CallbackTriggerAuto,
			Result:        sdb.CallbackResultSuccess,
			AttemptNumber: 1,
		},
	}).Error; err != nil {
		t.Fatalf("failed to seed callback events: %v", err)
	}

	router := gin.New()
	admin := router.Group("/admin")
	registerOperationsRoutes(admin)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/operations/summary", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Code int `json:"code"`
		Data struct {
			Status string `json:"status"`
			Orders struct {
				Pending              int64 `json:"pending"`
				Success              int64 `json:"success"`
				Expired              int64 `json:"expired"`
				CallbackPending      int64 `json:"callbackPending"`
				CallbackFailed       int64 `json:"callbackFailed"`
				PaidMissingNotifyURL int64 `json:"paidMissingNotifyUrl"`
			} `json:"orders"`
			Callbacks struct {
				FailedLast24Hours int64  `json:"failedLast24Hours"`
				ManualQueued      int64  `json:"manualQueued"`
				LatestFailedAt    string `json:"latestFailedAt"`
				LatestSuccessAt   string `json:"latestSuccessAt"`
			} `json:"callbacks"`
			Wallets struct {
				Total    int64 `json:"total"`
				Enabled  int64 `json:"enabled"`
				Disabled int64 `json:"disabled"`
			} `json:"wallets"`
			Warnings []string `json:"warnings"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode operations response: %v", err)
	}

	if body.Data.Status != "warning" {
		t.Fatalf("expected warning status, got %q", body.Data.Status)
	}
	if body.Data.Orders.Pending != 1 || body.Data.Orders.Success != 3 || body.Data.Orders.Expired != 1 {
		t.Fatalf("unexpected order counts: %#v", body.Data.Orders)
	}
	if body.Data.Orders.CallbackPending != 1 || body.Data.Orders.CallbackFailed != 1 {
		t.Fatalf("unexpected callback counts: %#v", body.Data.Orders)
	}
	if body.Data.Orders.PaidMissingNotifyURL != 1 {
		t.Fatalf("expected 1 paid order missing notify url, got %#v", body.Data.Orders)
	}
	if body.Data.Callbacks.FailedLast24Hours != 1 || body.Data.Callbacks.ManualQueued != 0 {
		t.Fatalf("unexpected callback metrics: %#v", body.Data.Callbacks)
	}
	if body.Data.Callbacks.LatestFailedAt == "" || body.Data.Callbacks.LatestSuccessAt == "" {
		t.Fatalf("expected latest callback timestamps, got %#v", body.Data.Callbacks)
	}
	if body.Data.Wallets.Total != 2 || body.Data.Wallets.Enabled != 1 || body.Data.Wallets.Disabled != 1 {
		t.Fatalf("unexpected wallet counts: %#v", body.Data.Wallets)
	}
	if len(body.Data.Warnings) == 0 {
		t.Fatal("expected callback warning to be included")
	}
}

func TestJWTSecretUsesDedicatedSettingKey(t *testing.T) {
	originalDB := sdb.DB
	t.Cleanup(func() {
		sdb.DB = originalDB
	})

	dbPath := filepath.Join(t.TempDir(), "jwt-secret-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open temp sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&sdb.Setting{}); err != nil {
		t.Fatalf("failed to migrate setting table: %v", err)
	}
	if err := db.Create(&sdb.Setting{
		SecretKey: "merchant-secret",
		JWTSecret: "jwt-secret",
	}).Error; err != nil {
		t.Fatalf("failed to seed setting: %v", err)
	}

	sdb.DB = db

	if got := string(jwtSecret()); got != "jwt-secret" {
		t.Fatalf("jwtSecret() = %q, want %q", got, "jwt-secret")
	}
}

func TestCreateTransactionOffsetsActualAmountForSameBaseAmount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testEnv := setupCreateOrderTestEnv(t)

	router := gin.New()
	router.POST("/api/create_order", AuthMiddleware(), CreateTransaction)

	first := createOrderForTest(t, router, "OFFSET-TEST-A-001", 10)
	second := createOrderForTest(t, router, "OFFSET-TEST-A-002", 10)
	third := createOrderForTest(t, router, "OFFSET-TEST-A-003", 10)

	if got, want := first.Data.ActualAmount, 10.00; got != want {
		t.Fatalf("first actual_amount = %.2f, want %.2f", got, want)
	}
	if got, want := second.Data.ActualAmount, 10.01; got != want {
		t.Fatalf("second actual_amount = %.2f, want %.2f", got, want)
	}
	if got, want := third.Data.ActualAmount, 10.02; got != want {
		t.Fatalf("third actual_amount = %.2f, want %.2f", got, want)
	}

	if first.Data.Token != second.Data.Token || second.Data.Token != third.Data.Token {
		t.Fatal("expected same wallet token to be reused while actual_amount increments")
	}

	var orders []sdb.Orders
	if err := testEnv.db.Where("order_id LIKE ?", "OFFSET-TEST-A-%").Order("id ASC").Find(&orders).Error; err != nil {
		t.Fatalf("failed to query created orders: %v", err)
	}
	if len(orders) != 3 {
		t.Fatalf("expected 3 orders, got %d", len(orders))
	}
	for _, order := range orders {
		if order.CallBackConfirm != sdb.CallBackConfirmNo {
			t.Fatalf("created order %s callback confirm = %d, want %d", order.OrderId, order.CallBackConfirm, sdb.CallBackConfirmNo)
		}
	}
}

func TestCreateTransactionReusesExistingPendingOrderForSameOrderID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testEnv := setupCreateOrderTestEnv(t)

	router := gin.New()
	router.POST("/api/create_order", AuthMiddleware(), CreateTransaction)

	first := createOrderForTest(t, router, "OFFSET-TEST-REUSE-001", 10)
	second := createOrderForTest(t, router, "OFFSET-TEST-REUSE-001", 10)

	if first.Data.TradeID != second.Data.TradeID {
		t.Fatalf("expected duplicate order to reuse trade_id %q, got %q", first.Data.TradeID, second.Data.TradeID)
	}
	if first.Data.ActualAmount != second.Data.ActualAmount {
		t.Fatalf("expected duplicate order to reuse actual_amount %.2f, got %.2f", first.Data.ActualAmount, second.Data.ActualAmount)
	}
	if second.Data.ExpirationTime < first.Data.ExpirationTime {
		t.Fatalf("expected duplicate order to refresh expiration_time, got %d then %d", first.Data.ExpirationTime, second.Data.ExpirationTime)
	}

	var count int64
	if err := testEnv.db.Model(&sdb.Orders{}).Where("order_id = ?", "OFFSET-TEST-REUSE-001").Count(&count).Error; err != nil {
		t.Fatalf("failed to count orders: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 persisted order for duplicate order_id, got %d", count)
	}
}

func TestCreateTransactionConcurrentSameAmountUsesUniqueActualAmounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testEnv := setupCreateOrderTestEnv(t)

	router := gin.New()
	router.POST("/api/create_order", AuthMiddleware(), CreateTransaction)

	const orderCount = 8
	responses := make([]dto.Response, orderCount)
	errs := make([]error, orderCount)

	var wg sync.WaitGroup
	for i := 0; i < orderCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			orderID := fmt.Sprintf("OFFSET-TEST-CONCURRENT-%03d", index)
			responses[index], errs[index] = createOrderForTestResult(router, orderID, 10)
		}(i)
	}
	wg.Wait()

	for index, err := range errs {
		if err != nil {
			t.Fatalf("concurrent order %d failed: %v", index, err)
		}
	}

	seenActualAmounts := map[float64]bool{}
	for _, response := range responses {
		if seenActualAmounts[response.Data.ActualAmount] {
			t.Fatalf("actual_amount %.2f was assigned more than once", response.Data.ActualAmount)
		}
		seenActualAmounts[response.Data.ActualAmount] = true
	}

	if len(seenActualAmounts) != orderCount {
		t.Fatalf("expected %d unique actual amounts, got %d", orderCount, len(seenActualAmounts))
	}

	var orders []sdb.Orders
	if err := testEnv.db.Where("order_id LIKE ?", "OFFSET-TEST-CONCURRENT-%").Find(&orders).Error; err != nil {
		t.Fatalf("failed to query concurrent orders: %v", err)
	}
	if len(orders) != orderCount {
		t.Fatalf("expected %d persisted orders, got %d", orderCount, len(orders))
	}
}

func TestLoginAuthConfigReturnsPasswordLoginState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupPasskeyRouteTestEnv(t)

	router := gin.New()
	registerPublicAuthRoutes(router, validator.New())

	req := httptest.NewRequest(http.MethodGet, "/login/auth-config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Code int `json:"code"`
		Data struct {
			PasswordLoginEnabled bool `json:"passwordLoginEnabled"`
			PasskeySupported     bool `json:"passkeySupported"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode auth config response: %v", err)
	}

	if body.Code != 0 {
		t.Fatalf("expected code 0, got %d", body.Code)
	}
	if !body.Data.PasswordLoginEnabled {
		t.Fatal("expected password login to be enabled by default")
	}
	if !body.Data.PasskeySupported {
		t.Fatal("expected passkey support flag to be true")
	}
}

func TestPasswordLoginRejectedWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupPasskeyRouteTestEnv(t)

	if err := sdb.DB.Model(&sdb.Setting{}).Where("1 = 1").Update("password_login_enabled", false).Error; err != nil {
		t.Fatalf("failed to disable password login: %v", err)
	}

	router := gin.New()
	registerPublicAuthRoutes(router, validator.New())

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"admin","password":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["message"] != "密码登录已禁用，请使用 Passkey 登录" {
		t.Fatalf("unexpected message: %#v", body["message"])
	}
}

func TestLoginCookieUsesSecureAttributesBehindHTTPSProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupPasskeyRouteTestEnv(t)
	resetAuthRateLimitForTest()

	router := gin.New()
	registerPublicAuthRoutes(router, validator.New())

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"admin","password":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	cookie := w.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "HttpOnly") {
		t.Fatalf("expected HttpOnly cookie, got %q", cookie)
	}
	if !strings.Contains(cookie, "SameSite=Lax") {
		t.Fatalf("expected SameSite=Lax cookie, got %q", cookie)
	}
	if !strings.Contains(cookie, "Secure") {
		t.Fatalf("expected Secure cookie behind HTTPS proxy, got %q", cookie)
	}
}

func TestLoginRateLimitBlocksRepeatedAttempts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupPasskeyRouteTestEnv(t)
	resetAuthRateLimitForTest()

	router := gin.New()
	registerPublicAuthRoutes(router, validator.New())

	for i := 0; i < loginRateLimitMax; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"admin","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.10:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("attempt %d was limited too early", i+1)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"admin","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.10:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminOriginMiddlewareRejectsCrossSiteWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupPasskeyRouteTestEnv(t)

	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	router := gin.New()
	admin := router.Group("/admin")
	admin.Use(JWTAuthMiddleware(), AdminOriginMiddleware())
	admin.POST("/api/settings", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/api/settings", bytes.NewBufferString(`{}`))
	req.Header.Set("Origin", "https://evil.example")
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSettingResponseMasksSensitiveFieldsAndSaveIgnoresMaskedValues(t *testing.T) {
	setting := sdb.Setting{
		SecretKey:   "merchant-secret-value",
		Redispasswd: "redis-secret-value",
		Tgbotkey:    "telegram-secret-value",
		Barkkey:     "bark-secret-value",
	}

	data := settingResponseData(setting)
	for _, key := range []string{"SecretKey", "Redispasswd", "Tgbotkey", "Barkkey"} {
		value, ok := data[key].(string)
		if !ok {
			t.Fatalf("expected %s to be a string, got %#v", key, data[key])
		}
		if value == "" || value == "merchant-secret-value" || value == "redis-secret-value" || value == "telegram-secret-value" || value == "bark-secret-value" {
			t.Fatalf("expected %s to be masked, got %q", key, value)
		}
	}

	updates, err := settingUpdatesFromRequest(map[string]interface{}{
		"secretkey":   data["SecretKey"],
		"redispasswd": data["Redispasswd"],
		"tgbotkey":    data["Tgbotkey"],
		"barkkey":     data["Barkkey"],
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, key := range []string{"SecretKey", "Redispasswd", "Tgbotkey", "Barkkey"} {
		if _, ok := updates[key]; ok {
			t.Fatalf("expected masked %s to be ignored by save updates", key)
		}
	}
}

func TestDisablePasswordLoginRequiresAtLeastOnePasskey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupPasskeyRouteTestEnv(t)

	router := gin.New()
	admin := router.Group("/admin")
	registerPasskeyAdminRoutes(admin)

	req := httptest.NewRequest(http.MethodPost, "/admin/api/passkeys/password-login", bytes.NewBufferString(`{"enabled":false}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteLastPasskeyBlockedWhenPasswordLoginDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupPasskeyRouteTestEnv(t)

	var user sdb.User
	if err := sdb.DB.First(&user).Error; err != nil {
		t.Fatalf("failed to load user: %v", err)
	}
	credential := sdb.PasskeyCredential{
		UserID:          user.ID,
		CredentialID:    "raw-credential-id",
		CredentialIDB64: "cmF3LWNyZWRlbnRpYWwtaWQ",
		PublicKey:       []byte("public-key"),
		DeviceLabel:     "测试 Passkey",
	}
	if err := sdb.DB.Create(&credential).Error; err != nil {
		t.Fatalf("failed to seed passkey credential: %v", err)
	}
	if err := sdb.DB.Model(&sdb.Setting{}).Where("1 = 1").Update("password_login_enabled", false).Error; err != nil {
		t.Fatalf("failed to disable password login: %v", err)
	}

	router := gin.New()
	admin := router.Group("/admin")
	registerPasskeyAdminRoutes(admin)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/admin/api/passkeys/%d", credential.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBuildOrderListItemCallbackState(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name             string
		order            sdb.Orders
		expectedState    string
		expectedLabel    string
		expectedMessage  string
		expectCallbackAt bool
		expectedCanRetry bool
	}{
		{
			name: "unpaid order does not trigger callback",
			order: sdb.Orders{
				Model:     gorm.Model{ID: 1, CreatedAt: now},
				TradeId:   "ORDER-UNPAID",
				OrderId:   "ORDER-UNPAID",
				Status:    sdb.StatusWaitPay,
				NotifyUrl: "https://example.com/notify",
			},
			expectedState:    "not_applicable",
			expectedLabel:    "未触发",
			expectedMessage:  "",
			expectCallbackAt: false,
			expectedCanRetry: false,
		},
		{
			name: "paid order without notify url is not applicable",
			order: sdb.Orders{
				Model:     gorm.Model{ID: 2, CreatedAt: now},
				TradeId:   "ORDER-NO-NOTIFY",
				OrderId:   "ORDER-NO-NOTIFY",
				Status:    sdb.StatusPaySuccess,
				NotifyUrl: "",
			},
			expectedState:    "not_applicable",
			expectedLabel:    "无需回调",
			expectedMessage:  "",
			expectCallbackAt: false,
			expectedCanRetry: false,
		},
		{
			name: "confirmed callback order is confirmed",
			order: sdb.Orders{
				Model:           gorm.Model{ID: 3, CreatedAt: now},
				TradeId:         "ORDER-CONFIRMED",
				OrderId:         "ORDER-CONFIRMED",
				Status:          sdb.StatusPaySuccess,
				NotifyUrl:       "https://example.com/notify",
				CallBackConfirm: sdb.CallBackConfirmOk,
			},
			expectedState:    "confirmed",
			expectedLabel:    "已确认",
			expectedMessage:  "",
			expectCallbackAt: false,
			expectedCanRetry: false,
		},
		{
			name: "failed callback order keeps latest failure reason",
			order: sdb.Orders{
				Model:           gorm.Model{ID: 4, CreatedAt: now},
				TradeId:         "ORDER-FAILED",
				OrderId:         "ORDER-FAILED",
				Status:          sdb.StatusPaySuccess,
				NotifyUrl:       "https://example.com/notify",
				CallbackNum:     2,
				CallbackMessage: "签名验证失败",
				LastCallbackAt:  &now,
				CallBackConfirm: sdb.CallBackConfirmNo,
			},
			expectedState:    "failed",
			expectedLabel:    "回调失败",
			expectedMessage:  "签名验证失败",
			expectCallbackAt: true,
			expectedCanRetry: true,
		},
		{
			name: "paid order without attempts is pending callback",
			order: sdb.Orders{
				Model:           gorm.Model{ID: 5, CreatedAt: now},
				TradeId:         "ORDER-PENDING",
				OrderId:         "ORDER-PENDING",
				Status:          sdb.StatusPaySuccess,
				NotifyUrl:       "https://example.com/notify",
				CallbackNum:     0,
				CallBackConfirm: sdb.CallBackConfirmNo,
			},
			expectedState:    "pending",
			expectedLabel:    "待回调",
			expectedMessage:  "",
			expectCallbackAt: false,
			expectedCanRetry: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			item := buildOrderListItem(tc.order)

			if item.CallbackState != tc.expectedState {
				t.Fatalf("expected callback state %q, got %q", tc.expectedState, item.CallbackState)
			}
			if item.CallbackStateLabel != tc.expectedLabel {
				t.Fatalf("expected callback label %q, got %q", tc.expectedLabel, item.CallbackStateLabel)
			}
			if item.CallbackMessage != tc.expectedMessage {
				t.Fatalf("expected callback message %q, got %q", tc.expectedMessage, item.CallbackMessage)
			}
			if tc.expectCallbackAt && item.LastCallbackAt == nil {
				t.Fatal("expected last_callback_at to be present")
			}
			if !tc.expectCallbackAt && item.LastCallbackAt != nil {
				t.Fatalf("expected last_callback_at to be nil, got %v", item.LastCallbackAt)
			}
			if item.CanRetryCallback != tc.expectedCanRetry {
				t.Fatalf("expected can_retry_callback %v, got %v", tc.expectedCanRetry, item.CanRetryCallback)
			}
		})
	}
}

func TestRetryOrderCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testEnv := setupCreateOrderTestEnv(t)

	eligible := sdb.Orders{
		TradeId:         "RETRY-OK-001",
		OrderId:         "RETRY-OK-001",
		Amount:          10,
		ActualAmount:    10,
		Type:            "USDT-TRC20",
		Token:           "TJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeC",
		Status:          sdb.StatusPaySuccess,
		NotifyUrl:       "https://example.com/notify",
		CallBackConfirm: sdb.CallBackConfirmNo,
	}
	unpaid := sdb.Orders{
		TradeId:         "RETRY-WAIT-001",
		OrderId:         "RETRY-WAIT-001",
		Amount:          11,
		ActualAmount:    11,
		Type:            "USDT-TRC20",
		Token:           "TJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeC",
		Status:          sdb.StatusWaitPay,
		NotifyUrl:       "https://example.com/notify",
		CallBackConfirm: sdb.CallBackConfirmNo,
	}
	confirmed := sdb.Orders{
		TradeId:         "RETRY-DONE-001",
		OrderId:         "RETRY-DONE-001",
		Amount:          12,
		ActualAmount:    12,
		Type:            "USDT-TRC20",
		Token:           "TJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeC",
		Status:          sdb.StatusPaySuccess,
		NotifyUrl:       "https://example.com/notify",
		CallBackConfirm: sdb.CallBackConfirmOk,
	}
	noNotify := sdb.Orders{
		TradeId:         "RETRY-NONE-001",
		OrderId:         "RETRY-NONE-001",
		Amount:          13,
		ActualAmount:    13,
		Type:            "USDT-TRC20",
		Token:           "TJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeC",
		Status:          sdb.StatusPaySuccess,
		NotifyUrl:       "",
		CallBackConfirm: sdb.CallBackConfirmNo,
	}

	if err := testEnv.db.Create(&[]sdb.Orders{eligible, unpaid, confirmed, noNotify}).Error; err != nil {
		t.Fatalf("failed to seed orders: %v", err)
	}

	var savedOrders []sdb.Orders
	if err := testEnv.db.Order("id ASC").Find(&savedOrders).Error; err != nil {
		t.Fatalf("failed to load seeded orders: %v", err)
	}
	if len(savedOrders) != 4 {
		t.Fatalf("expected 4 seeded orders, got %d", len(savedOrders))
	}

	triggered := make([]string, 0, 1)
	originalTrigger := triggerOrderCallbackAsync
	triggerOrderCallbackAsync = func(order sdb.Orders) {
		triggered = append(triggered, order.TradeId)
	}
	t.Cleanup(func() {
		triggerOrderCallbackAsync = originalTrigger
	})

	router := gin.New()
	admin := router.Group("/admin")
	admin.POST("/api/orders/:id/retry-callback", handleRetryOrderCallback)

	testCases := []struct {
		name           string
		orderID        uint
		expectedStatus int
		expectedCode   int
		expectedCalls  int
	}{
		{
			name:           "eligible order triggers retry",
			orderID:        savedOrders[0].ID,
			expectedStatus: http.StatusOK,
			expectedCode:   0,
			expectedCalls:  1,
		},
		{
			name:           "unpaid order cannot retry",
			orderID:        savedOrders[1].ID,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   1,
			expectedCalls:  1,
		},
		{
			name:           "confirmed order cannot retry",
			orderID:        savedOrders[2].ID,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   1,
			expectedCalls:  1,
		},
		{
			name:           "order without notify url cannot retry",
			orderID:        savedOrders[3].ID,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   1,
			expectedCalls:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/api/orders/%d/retry-callback", tc.orderID), nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			var body struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if body.Code != tc.expectedCode {
				t.Fatalf("expected code %d, got %d", tc.expectedCode, body.Code)
			}
			if len(triggered) != tc.expectedCalls {
				t.Fatalf("expected retry trigger count %d, got %d", tc.expectedCalls, len(triggered))
			}
		})
	}

	var queuedEvents []sdb.CallbackEvent
	if err := testEnv.db.Where("order_row_id = ? AND trigger_type = ? AND result = ?", savedOrders[0].ID, sdb.CallbackTriggerManual, sdb.CallbackResultQueued).Find(&queuedEvents).Error; err != nil {
		t.Fatalf("failed to query queued retry events: %v", err)
	}
	if len(queuedEvents) != 1 {
		t.Fatalf("expected 1 queued callback event for eligible order, got %d", len(queuedEvents))
	}
}

func TestListOrderCallbackEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testEnv := setupCreateOrderTestEnv(t)

	order := sdb.Orders{
		TradeId:         "CALLBACK-HISTORY-001",
		OrderId:         "CALLBACK-HISTORY-001",
		Amount:          10,
		ActualAmount:    10,
		Type:            "USDT-TRC20",
		Token:           "TJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeC",
		Status:          sdb.StatusPaySuccess,
		NotifyUrl:       "https://example.com/notify",
		CallBackConfirm: sdb.CallBackConfirmNo,
	}
	if err := testEnv.db.Create(&order).Error; err != nil {
		t.Fatalf("failed to seed order: %v", err)
	}

	oldest := time.Now().Add(-5 * time.Minute)
	middle := time.Now().Add(-3 * time.Minute)
	latest := time.Now().Add(-1 * time.Minute)
	events := []sdb.CallbackEvent{
		{
			Model:         gorm.Model{CreatedAt: oldest},
			OrderRowID:    order.ID,
			TradeID:       order.TradeId,
			TriggerType:   sdb.CallbackTriggerAuto,
			Result:        sdb.CallbackResultFailed,
			Message:       "请求超时",
			AttemptNumber: 1,
		},
		{
			Model:         gorm.Model{CreatedAt: middle},
			OrderRowID:    order.ID,
			TradeID:       order.TradeId,
			TriggerType:   sdb.CallbackTriggerManual,
			Result:        sdb.CallbackResultQueued,
			Message:       "管理员手动触发补发",
			AttemptNumber: 0,
		},
		{
			Model:         gorm.Model{CreatedAt: latest},
			OrderRowID:    order.ID,
			TradeID:       order.TradeId,
			TriggerType:   sdb.CallbackTriggerManual,
			Result:        sdb.CallbackResultSuccess,
			Message:       "",
			AttemptNumber: 2,
		},
	}
	if err := testEnv.db.Create(&events).Error; err != nil {
		t.Fatalf("failed to seed callback events: %v", err)
	}

	router := gin.New()
	admin := router.Group("/admin")
	admin.GET("/api/orders/:id/callback-events", handleListOrderCallbackEvents)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/api/orders/%d/callback-events", order.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Code int `json:"code"`
		Data struct {
			Total  int                        `json:"total"`
			Events []callbackEventHistoryItem `json:"events"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Code != 0 {
		t.Fatalf("expected code 0, got %d", body.Code)
	}
	if body.Data.Total != 3 {
		t.Fatalf("expected total 3, got %d", body.Data.Total)
	}
	if len(body.Data.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(body.Data.Events))
	}

	if body.Data.Events[0].Result != sdb.CallbackResultSuccess || body.Data.Events[0].TriggerType != sdb.CallbackTriggerManual {
		t.Fatalf("expected newest event to be manual success, got %#v", body.Data.Events[0])
	}
	if body.Data.Events[0].ResultLabel != "回调成功" {
		t.Fatalf("expected success label, got %q", body.Data.Events[0].ResultLabel)
	}
	if body.Data.Events[1].Result != sdb.CallbackResultQueued || body.Data.Events[1].TriggerTypeLabel != "手动补发" {
		t.Fatalf("expected middle event to be manual queued, got %#v", body.Data.Events[1])
	}
	if body.Data.Events[2].Message != "请求超时" || body.Data.Events[2].ResultLabel != "回调失败" {
		t.Fatalf("expected oldest event to preserve failure reason, got %#v", body.Data.Events[2])
	}
}

func TestListOrderCallbackEventsReturnsNotFoundForUnknownOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupCreateOrderTestEnv(t)

	router := gin.New()
	admin := router.Group("/admin")
	admin.GET("/api/orders/:id/callback-events", handleListOrderCallbackEvents)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/orders/999999/callback-events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

type createOrderTestEnv struct {
	db *gorm.DB
}

func setupCreateOrderTestEnv(t *testing.T) createOrderTestEnv {
	t.Helper()

	originalDB := sdb.DB
	originalRedis := rdb.RDB
	originalMQClient := mq.Client
	originalMQInspector := mq.Inspector
	originalMQServer := mq.Server

	dbPath := filepath.Join(t.TempDir(), "upay-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open temp sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&sdb.Orders{}, &sdb.CallbackEvent{}, &sdb.WalletAddress{}, &sdb.Setting{}, &sdb.TradeIdTaskID{}); err != nil {
		t.Fatalf("failed to migrate temp sqlite db: %v", err)
	}

	if err := db.Create(&sdb.Setting{
		AppUrl:         "http://localhost:8090",
		SecretKey:      "test-secret",
		JWTSecret:      "test-jwt-secret",
		ExpirationDate: 10 * time.Minute,
	}).Error; err != nil {
		t.Fatalf("failed to seed settings: %v", err)
	}

	if err := db.Create(&sdb.WalletAddress{
		Currency: "USDT-TRC20",
		Token:    "TJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeC",
		Status:   sdb.TokenStatusEnable,
		Rate:     1,
	}).Error; err != nil {
		t.Fatalf("failed to seed wallet address: %v", err)
	}

	miniRedis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("failed to ping miniredis: %v", err)
	}

	sdb.DB = db
	rdb.RDB = redisClient
	mq.Client = nil
	mq.Inspector = nil
	mq.Server = nil

	t.Cleanup(func() {
		sdb.DB = originalDB
		rdb.RDB = originalRedis
		mq.Client = originalMQClient
		mq.Inspector = originalMQInspector
		mq.Server = originalMQServer
		_ = redisClient.Close()
		miniRedis.Close()
	})

	return createOrderTestEnv{db: db}
}

func setupPasskeyRouteTestEnv(t *testing.T) {
	t.Helper()

	originalDB := sdb.DB
	dbPath := filepath.Join(t.TempDir(), "passkey-route-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open temp sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&sdb.User{}, &sdb.Setting{}, &sdb.PasskeyCredential{}, &sdb.PasskeyChallenge{}); err != nil {
		t.Fatalf("failed to migrate temp sqlite db: %v", err)
	}

	hashedPassword, err := sdb.HashPassword("admin")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	if err := db.Create(&sdb.User{
		UserName: "admin",
		PassWord: hashedPassword,
	}).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	if err := db.Create(&sdb.Setting{
		AppUrl:               "http://localhost:8090",
		SecretKey:            "test-secret",
		JWTSecret:            "test-jwt-secret",
		PasswordLoginEnabled: true,
		ExpirationDate:       10 * time.Minute,
	}).Error; err != nil {
		t.Fatalf("failed to seed setting: %v", err)
	}

	sdb.DB = db
	t.Cleanup(func() {
		sdb.DB = originalDB
	})
}

func createOrderForTest(t *testing.T, router http.Handler, orderID string, amount float64) dto.Response {
	t.Helper()

	response, err := createOrderForTestResult(router, orderID, amount)
	if err != nil {
		t.Fatal(err)
	}

	return response
}

func createOrderForTestResult(router http.Handler, orderID string, amount float64) (dto.Response, error) {
	var response dto.Response

	requestBody := map[string]any{
		"type":         "USDT-TRC20",
		"order_id":     orderID,
		"amount":       amount,
		"notify_url":   "https://example.com/notify",
		"redirect_url": "https://example.com/return",
	}
	requestBody["signature"] = createOrderSignatureForTest(orderID, amount)

	body, err := json.Marshal(requestBody)
	if err != nil {
		return response, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/create_order", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		return response, fmt.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		return response, fmt.Errorf("failed to decode create_order response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return response, fmt.Errorf("expected response status_code 200, got %d", response.StatusCode)
	}

	return response, nil
}

func createOrderSignatureForTest(orderID string, amount float64) string {
	params := []string{
		"type=USDT-TRC20",
		fmt.Sprintf("amount=%g", amount),
		"notify_url=https://example.com/notify",
		fmt.Sprintf("order_id=%s", orderID),
		"redirect_url=https://example.com/return",
	}

	// Keep the same ordering logic as AuthMiddleware.
	for i := 0; i < len(params)-1; i++ {
		for j := i + 1; j < len(params); j++ {
			if params[j] < params[i] {
				params[i], params[j] = params[j], params[i]
			}
		}
	}

	signatureString := stringsJoinForTest(params, "&") + "test-secret"
	return fmt.Sprintf("%x", md5.Sum([]byte(signatureString)))
}

func stringsJoinForTest(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, part := range parts[1:] {
		result += sep + part
	}
	return result
}

func almostEqualFloat(a float64, b float64) bool {
	return math.Abs(a-b) < 0.000001
}
