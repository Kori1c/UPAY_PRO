package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"upay_pro/db/sdb"

	"github.com/gin-gonic/gin"
)

const (
	loginRateLimitMax    = 10
	authRateLimitWindow  = time.Minute
	sensitiveMaskPrefix  = "********"
	settingsDurationUnit = time.Nanosecond
)

type rateLimitBucket struct {
	count   int
	resetAt time.Time
}

var (
	authRateLimitMu      sync.Mutex
	authRateLimitBuckets = map[string]rateLimitBucket{}
)

func resetAuthRateLimitForTest() {
	authRateLimitMu.Lock()
	defer authRateLimitMu.Unlock()
	authRateLimitBuckets = map[string]rateLimitBucket{}
}

func AuthRateLimitMiddleware(scope string, maxAttempts int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxAttempts <= 0 || window <= 0 {
			c.Next()
			return
		}

		key := fmt.Sprintf("%s:%s:%s", scope, c.ClientIP(), c.Request.URL.Path)
		now := time.Now()

		authRateLimitMu.Lock()
		bucket := authRateLimitBuckets[key]
		if bucket.resetAt.IsZero() || now.After(bucket.resetAt) {
			bucket = rateLimitBucket{count: 1, resetAt: now.Add(window)}
			authRateLimitBuckets[key] = bucket
			authRateLimitMu.Unlock()
			c.Next()
			return
		}

		bucket.count++
		authRateLimitBuckets[key] = bucket
		limited := bucket.count > maxAttempts
		authRateLimitMu.Unlock()

		if limited {
			c.JSON(http.StatusTooManyRequests, gin.H{"code": 1, "message": "请求过于频繁，请稍后再试"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func AdminOriginMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isUnsafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		origin := originFromRequest(c.Request)
		if origin == "" {
			c.Next()
			return
		}

		if !isAllowedAdminOrigin(c, origin) {
			c.JSON(http.StatusForbidden, gin.H{"code": 1, "message": "请求来源不受信任"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func originFromRequest(r *http.Request) string {
	if origin := normalizeOrigin(r.Header.Get("Origin")); origin != "" {
		return origin
	}
	return normalizeOrigin(r.Header.Get("Referer"))
}

func isAllowedAdminOrigin(c *gin.Context, origin string) bool {
	allowed := map[string]bool{}
	if current := requestOrigin(c); current != "" {
		allowed[current] = true
	}
	if settingOrigin := normalizeOrigin(sdb.GetSetting().AppUrl); settingOrigin != "" {
		allowed[settingOrigin] = true
	}
	return allowed[origin]
}

func normalizeOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host)
}

func requestIsHTTPS(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	forwardedProto := strings.ToLower(strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0]))
	return forwardedProto == "https"
}

func setAuthCookie(c *gin.Context, value string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("token", value, maxAge, "/", "", requestIsHTTPS(c), true)
}

func clearAuthCookie(c *gin.Context) {
	setAuthCookie(c, "", -1)
}

func maskSensitiveValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return sensitiveMaskPrefix
	}
	return sensitiveMaskPrefix + value[len(value)-4:]
}

func isMaskedSensitiveValue(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), sensitiveMaskPrefix)
}

func settingResponseData(setting sdb.Setting) gin.H {
	return gin.H{
		"AppUrl":                 setting.AppUrl,
		"SecretKey":              maskSensitiveValue(setting.SecretKey),
		"Httpport":               setting.Httpport,
		"Tgbotkey":               maskSensitiveValue(setting.Tgbotkey),
		"Tgchatid":               setting.Tgchatid,
		"Barkkey":                maskSensitiveValue(setting.Barkkey),
		"Redishost":              setting.Redishost,
		"Redisport":              setting.Redisport,
		"Redispasswd":            maskSensitiveValue(setting.Redispasswd),
		"Redisdb":                setting.Redisdb,
		"AppName":                setting.AppName,
		"CustomerServiceContact": setting.CustomerServiceContact,
		"ExpirationDate":         setting.ExpirationDate,
	}
}

func settingUpdatesFromRequest(req map[string]interface{}) (map[string]interface{}, error) {
	updates := make(map[string]interface{})

	if appname, ok := req["appname"]; ok {
		if name, ok := appname.(string); ok {
			updates["AppName"] = name
		}
	}
	if customerservicecontact, ok := req["customerservicecontact"]; ok {
		updates["CustomerServiceContact"] = customerservicecontact
	}
	if appurl, ok := req["appurl"]; ok {
		if url, ok := appurl.(string); ok && url != "" {
			updates["AppUrl"] = url
		} else {
			return nil, fmt.Errorf("应用地址不能为空")
		}
	}
	if httpport, ok := req["httpport"]; ok {
		if port, ok := httpport.(float64); ok && port >= 1 && port <= 65535 {
			updates["Httpport"] = int(port)
		} else {
			return nil, fmt.Errorf("HTTP端口必须在1-65535之间")
		}
	}
	if secretkey, ok := req["secretkey"]; ok {
		if value, ok := secretkey.(string); ok && !isMaskedSensitiveValue(value) {
			updates["SecretKey"] = value
		}
	}
	if expirationdate, ok := req["expirationdate"]; ok {
		if expiration, ok := expirationdate.(float64); ok && expiration > 0 {
			updates["ExpirationDate"] = time.Duration(int64(expiration)) * settingsDurationUnit
		} else {
			return nil, fmt.Errorf("过期时间必须大于0")
		}
	}
	if redishost, ok := req["redishost"]; ok {
		if host, ok := redishost.(string); ok && host != "" {
			updates["Redishost"] = host
		} else {
			return nil, fmt.Errorf("Redis主机不能为空")
		}
	}
	if redisport, ok := req["redisport"]; ok {
		if port, ok := redisport.(float64); ok && port >= 1 && port <= 65535 {
			updates["Redisport"] = int(port)
		} else {
			return nil, fmt.Errorf("Redis端口必须在1-65535之间")
		}
	}
	if redispasswd, ok := req["redispasswd"]; ok {
		if value, ok := redispasswd.(string); ok && !isMaskedSensitiveValue(value) {
			updates["Redispasswd"] = value
		}
	}
	if redisdb, ok := req["redisdb"]; ok {
		if db, ok := redisdb.(float64); ok && db >= 0 && db <= 15 {
			updates["Redisdb"] = int(db)
		} else {
			return nil, fmt.Errorf("Redis数据库编号必须在0-15之间")
		}
	}
	if tgbotkey, ok := req["tgbotkey"]; ok {
		if value, ok := tgbotkey.(string); ok && !isMaskedSensitiveValue(value) {
			updates["Tgbotkey"] = value
		}
	}
	if tgchatid, ok := req["tgchatid"]; ok {
		updates["Tgchatid"] = tgchatid
	}
	if barkkey, ok := req["barkkey"]; ok {
		if value, ok := barkkey.(string); ok && !isMaskedSensitiveValue(value) {
			updates["Barkkey"] = value
		}
	}

	return updates, nil
}
