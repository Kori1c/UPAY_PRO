package web

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"upay_pro/db/sdb"
	"upay_pro/mylog"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	passkeyChallengeTTL          = 5 * time.Minute
	passkeyFlowRegistration      = "register"
	passkeyFlowAuthentication    = "authenticate"
	defaultPasskeyDisplayName    = "UPay Pro"
	passwordLoginDisabledMessage = "密码登录已禁用，请使用 Passkey 登录"
)

type passkeyUser struct {
	user        sdb.User
	credentials []webauthn.Credential
}

func (u passkeyUser) WebAuthnID() []byte {
	return []byte(strconv.FormatUint(uint64(u.user.ID), 10))
}

func (u passkeyUser) WebAuthnName() string {
	return u.user.UserName
}

func (u passkeyUser) WebAuthnDisplayName() string {
	return u.user.UserName
}

func (u passkeyUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

type passkeyConfig struct {
	RPID          string
	RPOrigin      string
	RPDisplayName string
}

type passkeyCeremonyRequest struct {
	ChallengeID string          `json:"challengeId"`
	Credential  json.RawMessage `json:"credential"`
}

func registerPublicAuthRoutes(r *gin.Engine, validate *validator.Validate) {
	r.GET("/login", func(c *gin.Context) {
		c.File("./static/admin_spa/index.html")
	})

	r.GET("/login/auth-config", func(c *gin.Context) {
		setting := sdb.GetSetting()
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"passwordLoginEnabled": setting.PasswordLoginEnabled,
				"passkeySupported":     true,
			},
		})
	})

	r.POST("/login", AuthRateLimitMiddleware("password-login", loginRateLimitMax, authRateLimitWindow), func(c *gin.Context) {
		setting := sdb.GetSetting()
		if !setting.PasswordLoginEnabled {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    1,
				"message": passwordLoginDisabledMessage,
			})
			return
		}

		var user User
		if err := c.ShouldBind(&user); err != nil {
			c.JSON(400, gin.H{"message": "参数错误"})
			return
		}
		if err := validate.Struct(user); err != nil {
			c.JSON(400, gin.H{"message": err.Error()})
			return
		}

		var userDB sdb.User
		err := sdb.DB.Where("UserName = ?", user.UserName).First(&userDB).Error
		if err != nil {
			c.JSON(400, gin.H{"message": "用户名或密码错误"})
			return
		}
		if !sdb.VerifyPassword(user.PassWord, userDB.PassWord) {
			c.JSON(400, gin.H{"message": "用户名或密码错误"})
			return
		}

		token, err := GenerateToken()
		if err != nil {
			mylog.Logger.Error("生成登录 token 失败", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "系统异常，请稍后重试"})
			return
		}
		setAuthCookie(c, token, 3600*24)
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "登录成功",
		})
	})

	r.POST("/login/passkey/options", AuthRateLimitMiddleware("passkey-options", loginRateLimitMax, authRateLimitWindow), beginPasskeyAuthenticationHandler)
	r.POST("/login/passkey/verify", AuthRateLimitMiddleware("passkey-verify", loginRateLimitMax, authRateLimitWindow), finishPasskeyAuthenticationHandler)
}

func registerPasskeyAdminRoutes(admin *gin.RouterGroup) {
	admin.GET("/api/passkeys", listPasskeysHandler)
	admin.POST("/api/passkeys/register/options", beginPasskeyRegistrationHandler)
	admin.POST("/api/passkeys/register/verify", finishPasskeyRegistrationHandler)
	admin.DELETE("/api/passkeys/:id", deletePasskeyHandler)
	admin.POST("/api/passkeys/password-login", updatePasswordLoginHandler)
}

func listPasskeysHandler(c *gin.Context) {
	user, err := currentAdminUser()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "获取当前账号失败"})
		return
	}

	var credentials []sdb.PasskeyCredential
	if err := sdb.DB.Where("user_id = ?", user.ID).Order("id ASC").Find(&credentials).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "获取 Passkey 失败"})
		return
	}

	items := make([]gin.H, 0, len(credentials))
	for _, credential := range credentials {
		items = append(items, passkeySummary(credential))
	}

	setting := sdb.GetSetting()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"passwordLoginEnabled": setting.PasswordLoginEnabled,
			"passkeys":             items,
		},
	})
}

func beginPasskeyRegistrationHandler(c *gin.Context) {
	user, err := currentAdminUser()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "获取当前账号失败"})
		return
	}

	passkeyUser, err := loadPasskeyUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "加载 Passkey 失败"})
		return
	}

	webAuthn, err := webAuthnForRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": err.Error()})
		return
	}

	options, session, err := webAuthn.BeginRegistration(
		passkeyUser,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementPreferred),
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementPreferred,
			UserVerification: protocol.VerificationPreferred,
		}),
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "生成 Passkey 注册参数失败"})
		return
	}

	challenge, err := createPasskeyChallenge(&user.ID, passkeyFlowRegistration, session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "保存 Passkey 挑战失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"challengeId": challenge.ChallengeID,
			"publicKey":   options.Response,
		},
	})
}

func finishPasskeyRegistrationHandler(c *gin.Context) {
	user, err := currentAdminUser()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "获取当前账号失败"})
		return
	}

	var req passkeyCeremonyRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ChallengeID == "" || len(req.Credential) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误"})
		return
	}

	session, err := consumePasskeyChallenge(req.ChallengeID, passkeyFlowRegistration)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": err.Error()})
		return
	}

	passkeyUser, err := loadPasskeyUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "加载 Passkey 失败"})
		return
	}

	webAuthn, err := webAuthnForRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": err.Error()})
		return
	}

	credentialRequest, err := requestFromCredentialJSON(c, req.Credential)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "Passkey 数据格式错误"})
		return
	}

	credential, err := webAuthn.FinishRegistration(passkeyUser, *session, credentialRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "Passkey 注册验证失败"})
		return
	}

	savedCredential, err := savePasskeyCredential(user.ID, credential)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "保存 Passkey 失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": passkeySummary(savedCredential),
	})
}

func beginPasskeyAuthenticationHandler(c *gin.Context) {
	var user sdb.User
	if err := sdb.DB.Where("deleted_at IS NULL").Order("id ASC").First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "管理员账号不存在"})
		return
	}

	passkeyUser, err := loadPasskeyUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "加载 Passkey 失败"})
		return
	}
	if len(passkeyUser.credentials) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "当前后台还没有注册 Passkey"})
		return
	}

	webAuthn, err := webAuthnForRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": err.Error()})
		return
	}

	options, session, err := webAuthn.BeginLogin(passkeyUser, webauthn.WithUserVerification(protocol.VerificationPreferred))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "生成 Passkey 登录参数失败"})
		return
	}

	challenge, err := createPasskeyChallenge(&user.ID, passkeyFlowAuthentication, session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "保存 Passkey 挑战失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"challengeId": challenge.ChallengeID,
			"publicKey":   options.Response,
		},
	})
}

func finishPasskeyAuthenticationHandler(c *gin.Context) {
	var req passkeyCeremonyRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ChallengeID == "" || len(req.Credential) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误"})
		return
	}

	session, err := consumePasskeyChallenge(req.ChallengeID, passkeyFlowAuthentication)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": err.Error()})
		return
	}

	if len(session.UserID) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "Passkey 挑战无效"})
		return
	}

	userID, err := strconv.ParseUint(string(session.UserID), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "Passkey 用户无效"})
		return
	}

	var user sdb.User
	if err := sdb.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "账号不存在"})
		return
	}

	passkeyUser, err := loadPasskeyUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "加载 Passkey 失败"})
		return
	}

	webAuthn, err := webAuthnForRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": err.Error()})
		return
	}

	credentialRequest, err := requestFromCredentialJSON(c, req.Credential)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "Passkey 数据格式错误"})
		return
	}

	credential, err := webAuthn.FinishLogin(passkeyUser, *session, credentialRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "Passkey 登录验证失败"})
		return
	}

	if err := updateUsedPasskeyCredential(user.ID, credential); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "更新 Passkey 状态失败"})
		return
	}

	token, err := GenerateToken()
	if err != nil {
		mylog.Logger.Error("Passkey 登录生成 token 失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "系统异常，请稍后重试"})
		return
	}

	setAuthCookie(c, token, 3600*24)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "登录成功"})
}

func updatePasswordLoginHandler(c *gin.Context) {
	user, err := currentAdminUser()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "获取当前账号失败"})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误"})
		return
	}

	if !req.Enabled {
		count, err := activePasskeyCount(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "检查 Passkey 失败"})
			return
		}
		if count == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "请先注册至少一个 Passkey，再禁用密码登录"})
			return
		}
	}

	if err := sdb.DB.Model(&sdb.Setting{}).Where("1 = 1").Update("PasswordLoginEnabled", req.Enabled).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "更新密码登录状态失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

func deletePasskeyHandler(c *gin.Context) {
	user, err := currentAdminUser()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "获取当前账号失败"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "Passkey 参数错误"})
		return
	}

	var credential sdb.PasskeyCredential
	if err := sdb.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&credential).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "Passkey 不存在"})
		return
	}

	setting := sdb.GetSetting()
	if !setting.PasswordLoginEnabled {
		count, err := activePasskeyCount(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "检查 Passkey 失败"})
			return
		}
		if count <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "密码登录已禁用，不能删除最后一个 Passkey"})
			return
		}
	}

	if err := sdb.DB.Delete(&credential).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "删除 Passkey 失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func currentAdminUser() (sdb.User, error) {
	var user sdb.User
	err := sdb.DB.Where("deleted_at IS NULL").Order("id ASC").First(&user).Error
	return user, err
}

func activePasskeyCount(userID uint) (int64, error) {
	var count int64
	err := sdb.DB.Model(&sdb.PasskeyCredential{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&count).Error
	return count, err
}

func loadPasskeyUser(user sdb.User) (passkeyUser, error) {
	var records []sdb.PasskeyCredential
	if err := sdb.DB.Where("user_id = ?", user.ID).Find(&records).Error; err != nil {
		return passkeyUser{}, err
	}

	credentials := make([]webauthn.Credential, 0, len(records))
	for _, record := range records {
		credential, err := credentialFromRecord(record)
		if err != nil {
			return passkeyUser{}, err
		}
		credentials = append(credentials, credential)
	}

	return passkeyUser{
		user:        user,
		credentials: credentials,
	}, nil
}

func credentialFromRecord(record sdb.PasskeyCredential) (webauthn.Credential, error) {
	if len(record.CredentialJSON) > 0 {
		var credential webauthn.Credential
		if err := json.Unmarshal(record.CredentialJSON, &credential); err == nil {
			return credential, nil
		}
	}

	credentialID, err := base64.RawURLEncoding.DecodeString(record.CredentialIDB64)
	if err != nil || len(credentialID) == 0 {
		credentialID = []byte(record.CredentialID)
	}

	return webauthn.Credential{
		ID:                credentialID,
		PublicKey:         record.PublicKey,
		AttestationType:   record.AttestationType,
		AttestationFormat: record.AttestationFormat,
		Transport:         transportsFromString(record.Transports),
		Authenticator: webauthn.Authenticator{
			SignCount: record.SignCount,
		},
	}, nil
}

func savePasskeyCredential(userID uint, credential *webauthn.Credential) (sdb.PasskeyCredential, error) {
	credentialJSON, err := json.Marshal(credential)
	if err != nil {
		return sdb.PasskeyCredential{}, err
	}

	credentialIDB64 := base64.RawURLEncoding.EncodeToString(credential.ID)
	label, err := nextPasskeyLabel(userID)
	if err != nil {
		return sdb.PasskeyCredential{}, err
	}

	record := sdb.PasskeyCredential{
		UserID:            userID,
		CredentialID:      credentialIDB64,
		CredentialIDB64:   credentialIDB64,
		PublicKey:         credential.PublicKey,
		CredentialJSON:    credentialJSON,
		AttestationType:   credential.AttestationType,
		AttestationFormat: credential.AttestationFormat,
		AAGUID:            hex.EncodeToString(credential.Authenticator.AAGUID),
		SignCount:         credential.Authenticator.SignCount,
		Transports:        transportsToString(credential.Transport),
		DeviceLabel:       label,
	}

	return record, sdb.DB.Create(&record).Error
}

func updateUsedPasskeyCredential(userID uint, credential *webauthn.Credential) error {
	credentialJSON, err := json.Marshal(credential)
	if err != nil {
		return err
	}

	now := time.Now()
	credentialIDB64 := base64.RawURLEncoding.EncodeToString(credential.ID)
	return sdb.DB.Model(&sdb.PasskeyCredential{}).
		Where("user_id = ? AND credential_id_b64 = ?", userID, credentialIDB64).
		Updates(map[string]interface{}{
			"CredentialJSON": credentialJSON,
			"SignCount":      credential.Authenticator.SignCount,
			"LastUsedAt":     &now,
		}).Error
}

func nextPasskeyLabel(userID uint) (string, error) {
	count, err := activePasskeyCount(userID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Passkey %d", count+1), nil
}

func passkeySummary(credential sdb.PasskeyCredential) gin.H {
	var lastUsedAt any
	if credential.LastUsedAt != nil {
		lastUsedAt = credential.LastUsedAt.Format(time.RFC3339)
	}

	return gin.H{
		"id":           credential.ID,
		"credentialId": credential.CredentialIDB64,
		"deviceLabel":  credential.DeviceLabel,
		"transports":   splitTransports(credential.Transports),
		"createdAt":    credential.CreatedAt.Format(time.RFC3339),
		"lastUsedAt":   lastUsedAt,
	}
}

func createPasskeyChallenge(userID *uint, flowType string, session *webauthn.SessionData) (sdb.PasskeyChallenge, error) {
	sessionData, err := json.Marshal(session)
	if err != nil {
		return sdb.PasskeyChallenge{}, err
	}

	challenge := sdb.PasskeyChallenge{
		UserID:      userID,
		FlowType:    flowType,
		ChallengeID: uuid.NewString(),
		SessionData: string(sessionData),
		ExpiresAt:   time.Now().Add(passkeyChallengeTTL),
	}

	return challenge, sdb.DB.Create(&challenge).Error
}

func consumePasskeyChallenge(challengeID string, flowType string) (*webauthn.SessionData, error) {
	var challenge sdb.PasskeyChallenge
	err := sdb.DB.Where("challenge_id = ? AND flow_type = ?", challengeID, flowType).First(&challenge).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("Passkey 挑战不存在或已失效")
	}
	if err != nil {
		return nil, err
	}

	if time.Now().After(challenge.ExpiresAt) {
		_ = sdb.DB.Delete(&challenge).Error
		return nil, errors.New("Passkey 挑战已过期，请重试")
	}

	var session webauthn.SessionData
	if err := json.Unmarshal([]byte(challenge.SessionData), &session); err != nil {
		_ = sdb.DB.Delete(&challenge).Error
		return nil, errors.New("Passkey 挑战数据无效")
	}

	if err := sdb.DB.Delete(&challenge).Error; err != nil {
		return nil, err
	}

	return &session, nil
}

func requestFromCredentialJSON(c *gin.Context, credential json.RawMessage) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPost, c.Request.URL.String(), bytes.NewReader(credential))
	if err != nil {
		return nil, err
	}
	req.Header = c.Request.Header.Clone()
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func webAuthnForRequest(c *gin.Context) (*webauthn.WebAuthn, error) {
	config, err := resolvePasskeyConfig(c)
	if err != nil {
		return nil, err
	}

	return webauthn.New(&webauthn.Config{
		RPID:          config.RPID,
		RPDisplayName: config.RPDisplayName,
		RPOrigins:     []string{config.RPOrigin},
	})
}

func resolvePasskeyConfig(c *gin.Context) (passkeyConfig, error) {
	setting := sdb.GetSetting()
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	if origin == "" {
		origin = requestOrigin(c)
	}

	originURL, err := url.Parse(origin)
	if err != nil || originURL.Hostname() == "" {
		return passkeyConfig{}, errors.New("Passkey 当前访问地址无效")
	}

	if isLoopbackIPHost(originURL.Hostname()) {
		return passkeyConfig{}, errors.New("本地测试 Passkey 请使用 localhost:8090 访问，不要使用 127.0.0.1")
	}

	rpID := originURL.Hostname()
	if appURL := strings.TrimSpace(setting.AppUrl); appURL != "" {
		if parsed, parseErr := url.Parse(appURL); parseErr == nil && parsed.Hostname() != "" && !isLocalHost(rpID) {
			rpID = parsed.Hostname()
		}
	}

	if !isLocalHost(originURL.Hostname()) && originURL.Hostname() != rpID {
		return passkeyConfig{}, errors.New("Passkey 页面地址与当前访问域名不一致，请检查系统设置里的页面地址")
	}

	displayName := strings.TrimSpace(setting.AppName)
	if displayName == "" {
		displayName = defaultPasskeyDisplayName
	}

	return passkeyConfig{
		RPID:          rpID,
		RPOrigin:      origin,
		RPDisplayName: displayName,
	}, nil
}

func requestOrigin(c *gin.Context) string {
	if c.Request.Host == "" {
		return ""
	}
	scheme := "http"
	if requestIsHTTPS(c) {
		scheme = "https"
	}
	return strings.ToLower(scheme + "://" + c.Request.Host)
}

func isLocalHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func isLoopbackIPHost(host string) bool {
	return host == "127.0.0.1" || host == "::1"
}

func transportsToString(transports []protocol.AuthenticatorTransport) string {
	parts := make([]string, 0, len(transports))
	for _, transport := range transports {
		if transport != "" {
			parts = append(parts, string(transport))
		}
	}
	return strings.Join(parts, ",")
}

func transportsFromString(value string) []protocol.AuthenticatorTransport {
	parts := splitTransports(value)
	transports := make([]protocol.AuthenticatorTransport, 0, len(parts))
	for _, part := range parts {
		transports = append(transports, protocol.AuthenticatorTransport(part))
	}
	return transports
}

func splitTransports(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}

	rawParts := strings.Split(value, ",")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}
