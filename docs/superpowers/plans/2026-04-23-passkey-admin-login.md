# Admin Passkey Login Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add admin Passkey registration, Passkey sign-in, and password-login disablement without breaking the existing JWT-cookie admin session model.

**Architecture:** Extend the existing Gin + Gorm auth flow with WebAuthn-backed credential storage and one-time challenge records, then wire the Vue admin login page and settings page to those endpoints through a small typed WebAuthn utility layer. Keep password login as an existing code path, but gate it behind a new `PasswordLoginEnabled` setting and protect against locking out the only admin.

**Tech Stack:** Go, Gin, Gorm, SQLite, `github.com/go-webauthn/webauthn`, Vue 3, Vite, Arco Design Vue, Vitest

---

## File Map

- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `db/sdb/sdb.go`
- Create: `web/passkey.go`
- Modify: `web/web.go`
- Modify: `web/function_test.go`
- Modify: `frontend/src/api/index.ts`
- Create: `frontend/src/utils/webauthn.ts`
- Create: `frontend/src/utils/webauthn.spec.ts`
- Modify: `frontend/src/views/login/login-view.vue`
- Create: `frontend/src/views/login/login-view.spec.ts`
- Modify: `frontend/src/views/settings/settings-view.vue`
- Create: `frontend/src/views/settings/settings-view.spec.ts`

### Task 1: Add Passkey persistence primitives and migration defaults

**Files:**
- Modify: `db/sdb/sdb.go`
- Test: `web/function_test.go`

- [ ] **Step 1: Write the failing backend tests for password-login defaults and deletion safety**

```go
func TestPasskeyPasswordLoginDefaultsToEnabled(t *testing.T) {
	db := openPasskeyTestDB(t)

	var setting sdb.Setting
	if err := db.First(&setting).Error; err != nil {
		t.Fatalf("failed to load setting: %v", err)
	}

	if !setting.PasswordLoginEnabled {
		t.Fatal("expected PasswordLoginEnabled to default to true")
	}
}

func TestDeleteLastPasskeyBlockedWhenPasswordLoginDisabled(t *testing.T) {
	env := setupPasskeyAuthTestEnv(t)

	user := seedAdminUser(t, env.db)
	seedPasskeyCredential(t, env.db, user.ID, "cred-1")
	disablePasswordLogin(t, env.db)

	err := deletePasskeyCredential(env.db, user.ID, "cred-1")
	if err == nil {
		t.Fatal("expected deleting last passkey to fail when password login is disabled")
	}
}
```

- [ ] **Step 2: Run the focused Go tests and verify they fail for the right reason**

Run: `go test ./web -run 'TestPasskeyPasswordLoginDefaultsToEnabled|TestDeleteLastPasskeyBlockedWhenPasswordLoginDisabled'`

Expected: FAIL because `PasswordLoginEnabled`, Passkey tables, and deletion guard helpers do not exist yet.

- [ ] **Step 3: Add the new Gorm models and migration defaults**

```go
type PasskeyCredential struct {
	gorm.Model
	UserID         uint   `gorm:"index"`
	CredentialID   string `gorm:"uniqueIndex"`
	CredentialIDB64 string `gorm:"uniqueIndex"`
	PublicKey      []byte
	AttestationType string
	AAGUID         string
	SignCount      uint32
	Transports     string
	DeviceLabel    string
	LastUsedAt     *time.Time
}

type PasskeyChallenge struct {
	gorm.Model
	UserID      *uint  `gorm:"index"`
	FlowType    string `gorm:"index"`
	ChallengeID string `gorm:"uniqueIndex"`
	SessionData string
	ExpiresAt   time.Time `gorm:"index"`
}

type Setting struct {
	gorm.Model
	AppUrl               string
	SecretKey            string
	JWTSecret            string
	PasswordLoginEnabled bool
	// existing fields stay unchanged
}
```

- [ ] **Step 4: Migrate the new tables and backfill the settings default**

```go
DB.AutoMigrate(&Setting{})
DB.AutoMigrate(&PasskeyCredential{})
DB.AutoMigrate(&PasskeyChallenge{})

var setting Setting
if err := DB.First(&setting).Error; err == nil && !setting.PasswordLoginEnabled {
	DB.Model(&setting).Update("PasswordLoginEnabled", true)
}
```

Use the existing “count zero then create defaults” block to seed `PasswordLoginEnabled: true` in the first-row settings record.

- [ ] **Step 5: Add minimal helper functions for passkey persistence**

```go
func ActivePasskeyCount(db *gorm.DB, userID uint) (int64, error) {
	var count int64
	err := db.Model(&PasskeyCredential{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&count).Error
	return count, err
}

func CanDisablePasswordLogin(db *gorm.DB, userID uint) (bool, error) {
	count, err := ActivePasskeyCount(db, userID)
	return count > 0, err
}
```

- [ ] **Step 6: Run the focused Go tests and verify they pass**

Run: `go test ./web -run 'TestPasskeyPasswordLoginDefaultsToEnabled|TestDeleteLastPasskeyBlockedWhenPasswordLoginDisabled'`

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add db/sdb/sdb.go web/function_test.go
git commit -m "feat: add passkey persistence models"
```

### Task 2: Add backend WebAuthn bootstrap helpers and public auth endpoints

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Create: `web/passkey.go`
- Modify: `web/web.go`
- Test: `web/function_test.go`

- [ ] **Step 1: Write the failing backend tests for auth-config and disabled password login**

```go
func TestLoginAuthConfigReturnsPasswordLoginState(t *testing.T) {
	router := setupAdminRouterForTest(t)

	req := httptest.NewRequest(http.MethodGet, "/login/auth-config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body struct {
		Code int `json:"code"`
		Data struct {
			PasswordLoginEnabled bool `json:"passwordLoginEnabled"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &body)

	if !body.Data.PasswordLoginEnabled {
		t.Fatal("expected password login to be enabled by default")
	}
}

func TestPasswordLoginRejectedWhenDisabled(t *testing.T) {
	router := setupAdminRouterForTest(t)
	disablePasswordLogin(t, sdb.DB)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":"admin","password":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run the focused Go tests and verify they fail**

Run: `go test ./web -run 'TestLoginAuthConfigReturnsPasswordLoginState|TestPasswordLoginRejectedWhenDisabled'`

Expected: FAIL because `/login/auth-config` and password-login gating do not exist yet.

- [ ] **Step 3: Add the WebAuthn dependency**

```bash
go get github.com/go-webauthn/webauthn@latest
go mod tidy
```

- [ ] **Step 4: Create the passkey service helpers**

```go
type passkeyConfig struct {
	RPID          string
	RPOrigin      string
	RPDisplayName string
}

func resolvePasskeyConfig(c *gin.Context) (passkeyConfig, error) {
	// derive RP ID and origin from settings AppUrl, request host, and localhost rules
}

func newWebAuthn(config passkeyConfig) (*webauthn.WebAuthn, error) {
	return webauthn.New(&webauthn.Config{
		RPDisplayName: config.RPDisplayName,
		RPID:          config.RPID,
		RPOrigins:     []string{config.RPOrigin},
	})
}
```

- [ ] **Step 5: Add public auth-config and password-login guard behavior**

```go
r.GET("/login/auth-config", func(c *gin.Context) {
	setting := sdb.GetOrInitSetting()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"passwordLoginEnabled": setting.PasswordLoginEnabled,
			"passkeySupported":     true,
		},
	})
})

r.POST("/login", func(c *gin.Context) {
	setting := sdb.GetOrInitSetting()
	if !setting.PasswordLoginEnabled {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "密码登录已禁用，请使用 Passkey 登录",
		})
		return
	}

	// keep existing username/password verification
})
```

- [ ] **Step 6: Add public Passkey auth begin/verify endpoints**

```go
r.POST("/login/passkey/options", beginPasskeyAuthenticationHandler)
r.POST("/login/passkey/verify", finishPasskeyAuthenticationHandler)
```

Implement the handlers in `web/passkey.go` so they:

```go
func beginPasskeyAuthenticationHandler(c *gin.Context) {
	// load user by username
	// build allowed credentials from sdb.PasskeyCredential
	// create challenge record with expires_at
	// return challenge ID + publicKey options
}

func finishPasskeyAuthenticationHandler(c *gin.Context) {
	// load and validate challenge
	// verify assertion with go-webauthn
	// update sign_count + last_used_at
	// delete challenge
	// issue existing JWT cookie
}
```

- [ ] **Step 7: Run the focused Go tests and verify they pass**

Run: `go test ./web -run 'TestLoginAuthConfigReturnsPasswordLoginState|TestPasswordLoginRejectedWhenDisabled'`

Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add go.mod go.sum web/passkey.go web/web.go web/function_test.go
git commit -m "feat: add passkey auth endpoints"
```

### Task 3: Add authenticated Passkey management APIs and challenge lifecycle tests

**Files:**
- Create: `web/passkey.go`
- Modify: `web/web.go`
- Modify: `web/function_test.go`

- [ ] **Step 1: Write the failing backend tests for password-toggle guard and challenge expiry**

```go
func TestDisablePasswordLoginRequiresAtLeastOnePasskey(t *testing.T) {
	router, token := setupAuthenticatedAdminRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/admin/api/passkeys/password-login", bytes.NewBufferString(`{"enabled":false}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestExpiredPasskeyChallengeRejected(t *testing.T) {
	env := setupPasskeyAuthTestEnv(t)
	challengeID := seedExpiredChallenge(t, env.db)

	err := verifyPasskeyChallenge(env.db, challengeID)
	if err == nil {
		t.Fatal("expected expired challenge verification to fail")
	}
}
```

- [ ] **Step 2: Run the focused Go tests and verify they fail**

Run: `go test ./web -run 'TestDisablePasswordLoginRequiresAtLeastOnePasskey|TestExpiredPasskeyChallengeRejected'`

Expected: FAIL because admin passkey routes and challenge guards do not exist yet.

- [ ] **Step 3: Add authenticated passkey routes**

```go
admin.GET("/api/passkeys", listPasskeysHandler)
admin.POST("/api/passkeys/register/options", beginPasskeyRegistrationHandler)
admin.POST("/api/passkeys/register/verify", finishPasskeyRegistrationHandler)
admin.DELETE("/api/passkeys/:id", deletePasskeyHandler)
admin.POST("/api/passkeys/password-login", updatePasswordLoginHandler)
```

- [ ] **Step 4: Implement the management handlers with explicit business guards**

```go
func updatePasswordLoginHandler(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	// bind request
	// if disabling and no passkeys exist -> 400
	// update settings row
}

func deletePasskeyHandler(c *gin.Context) {
	// load credential by ID and current admin
	// if password login disabled and count == 1 -> 400
	// delete credential
}
```

- [ ] **Step 5: Implement challenge storage lifecycle helpers**

```go
func createPasskeyChallenge(db *gorm.DB, flowType string, userID *uint, sessionData string, ttl time.Duration) (*sdb.PasskeyChallenge, error) {
	return challenge, db.Create(challenge).Error
}

func consumePasskeyChallenge(db *gorm.DB, challengeID string, flowType string) (*sdb.PasskeyChallenge, error) {
	// find challenge by id + flow type
	// reject if expires_at is in the past
	// delete after successful load
}
```

- [ ] **Step 6: Expand the tests to cover credential list shape and last-passkey protection**

```go
func TestPasskeyListReturnsCredentialSummaries(t *testing.T) {
	// seed a credential
	// GET /admin/api/passkeys
	// expect one item with id, deviceLabel, createdAt, lastUsedAt, transports
}
```

- [ ] **Step 7: Run the focused Go tests and verify they pass**

Run: `go test ./web -run 'TestDisablePasswordLoginRequiresAtLeastOnePasskey|TestExpiredPasskeyChallengeRejected|TestPasskeyListReturnsCredentialSummaries'`

Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add web/passkey.go web/web.go web/function_test.go
git commit -m "feat: add passkey management api"
```

### Task 4: Add frontend Passkey API typing and WebAuthn utility coverage

**Files:**
- Modify: `frontend/src/api/index.ts`
- Create: `frontend/src/utils/webauthn.ts`
- Create: `frontend/src/utils/webauthn.spec.ts`

- [ ] **Step 1: Write the failing frontend tests for WebAuthn option conversion**

```ts
import { describe, expect, it } from 'vitest'
import {
  decodeBase64Url,
  normalizeRequestOptions,
  serializeAssertionCredential,
} from './webauthn'

describe('webauthn helpers', () => {
  it('converts challenge and allowCredentials ids into Uint8Array buffers', () => {
    const options = normalizeRequestOptions({
      challenge: 'AQID',
      allowCredentials: [{ id: 'BAUG', type: 'public-key' }],
    } as any)

    expect(options.challenge).toBeInstanceOf(Uint8Array)
    expect(options.allowCredentials?.[0]?.id).toBeInstanceOf(Uint8Array)
  })
})
```

- [ ] **Step 2: Run the focused Vitest suite and verify it fails**

Run: `cd frontend && npm test -- src/utils/webauthn.spec.ts`

Expected: FAIL because `frontend/src/utils/webauthn.ts` does not exist yet.

- [ ] **Step 3: Add typed API contracts for Passkey flows**

```ts
export interface LoginAuthConfig {
  passwordLoginEnabled: boolean
  passkeySupported: boolean
}

export interface PasskeyItem {
  id: number
  credentialId: string
  deviceLabel: string
  transports: string[]
  createdAt: string
  lastUsedAt?: string
}

export const adminApi = {
  getLoginAuthConfig: () => api.get<{ code: number; data: LoginAuthConfig }>('/login/auth-config'),
  beginPasskeyLogin: (payload: { username: string }) =>
    api.post<{ code: number; data: PasskeyLoginBeginResponse }>('/login/passkey/options', payload),
  finishPasskeyLogin: (payload: PasskeyVerifyPayload) =>
    api.post<{ code: number; message: string }>('/login/passkey/verify', payload),
  getPasskeys: () => api.get<{ code: number; data: PasskeySettingsResponse }>('/admin/api/passkeys'),
  beginPasskeyRegistration: () =>
    api.post<{ code: number; data: PasskeyRegistrationBeginResponse }>('/admin/api/passkeys/register/options', {}),
  finishPasskeyRegistration: (payload: PasskeyVerifyPayload) =>
    api.post<{ code: number; data: PasskeyItem }>('/admin/api/passkeys/register/verify', payload),
  setPasswordLoginEnabled: (enabled: boolean) =>
    api.post<{ code: number; message: string }>('/admin/api/passkeys/password-login', { enabled }),
  deletePasskey: (id: number) =>
    api.delete<{ code: number; message: string }>(`/admin/api/passkeys/${id}`),
}
```

- [ ] **Step 4: Implement the WebAuthn utility module**

```ts
export function decodeBase64Url(value: string): Uint8Array {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
  const raw = atob(padded)
  return Uint8Array.from(raw, (char) => char.charCodeAt(0))
}

export function encodeBase64Url(bytes: ArrayBuffer): string {
  const raw = String.fromCharCode(...new Uint8Array(bytes))
  return btoa(raw).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

export function supportsPasskey(): boolean {
  return typeof window !== 'undefined'
    && window.isSecureContext
    && typeof window.PublicKeyCredential !== 'undefined'
}
```

- [ ] **Step 5: Run the focused Vitest suite and verify it passes**

Run: `cd frontend && npm test -- src/utils/webauthn.spec.ts`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/api/index.ts frontend/src/utils/webauthn.ts frontend/src/utils/webauthn.spec.ts
git commit -m "feat: add frontend passkey utilities"
```

### Task 5: Wire Passkey sign-in into the login page

**Files:**
- Modify: `frontend/src/views/login/login-view.vue`
- Create: `frontend/src/views/login/login-view.spec.ts`

- [ ] **Step 1: Write the failing login view tests**

```ts
import { mount, flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import LoginView from './login-view.vue'
import { adminApi } from '../../api'

vi.mock('../../api', () => ({
  adminApi: {
    getLoginAuthConfig: vi.fn(),
    login: vi.fn(),
    beginPasskeyLogin: vi.fn(),
    finishPasskeyLogin: vi.fn(),
  },
}))

describe('login view', () => {
  it('hides the password field when password login is disabled', async () => {
    vi.mocked(adminApi.getLoginAuthConfig).mockResolvedValue({
      code: 0,
      data: { passwordLoginEnabled: false, passkeySupported: true },
    } as any)

    const wrapper = mount(LoginView)
    await flushPromises()

    expect(wrapper.text()).not.toContain('密码')
    expect(wrapper.text()).toContain('使用 Passkey 登录')
  })
})
```

- [ ] **Step 2: Run the focused Vitest suite and verify it fails**

Run: `cd frontend && npm test -- src/views/login/login-view.spec.ts`

Expected: FAIL because the login view does not fetch auth-config or render a Passkey action yet.

- [ ] **Step 3: Add login-page state for auth mode and Passkey loading**

```ts
const authConfig = ref({ passwordLoginEnabled: true, passkeySupported: true })
const passkeyLoading = ref(false)

onMounted(async () => {
  const res = await adminApi.getLoginAuthConfig()
  if (res.code === 0) {
    authConfig.value = res.data
  }
})
```

- [ ] **Step 4: Implement Passkey login flow**

```ts
async function handlePasskeyLogin() {
  if (!form.username) {
    Message.warning('请先输入账号')
    return
  }

  if (!supportsPasskey()) {
    Message.error('当前浏览器或环境不支持 Passkey')
    return
  }

  passkeyLoading.value = true
  try {
    const begin = await adminApi.beginPasskeyLogin({ username: form.username })
    const credential = await getAssertionCredential(begin.data.publicKey)
    const verify = await adminApi.finishPasskeyLogin({
      challengeId: begin.data.challengeId,
      credential: serializeAssertionCredential(credential),
    })
    if (verify.code === 0) {
      router.push('/dashboard')
    }
  } finally {
    passkeyLoading.value = false
  }
}
```

- [ ] **Step 5: Update the template to conditionally render password login**

```vue
<a-form v-if="authConfig.passwordLoginEnabled" @submit="handleSubmit">
  <!-- existing username + password fields -->
</a-form>

<a-button
  type="outline"
  long
  size="large"
  :loading="passkeyLoading"
  class="login-passkey"
  @click="handlePasskeyLogin"
>
  使用 Passkey 登录
</a-button>
```

- [ ] **Step 6: Run the focused Vitest suite and verify it passes**

Run: `cd frontend && npm test -- src/views/login/login-view.spec.ts`

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add frontend/src/views/login/login-view.vue frontend/src/views/login/login-view.spec.ts
git commit -m "feat: add passkey login ui"
```

### Task 6: Add Passkey management UI to system settings

**Files:**
- Modify: `frontend/src/views/settings/settings-view.vue`
- Create: `frontend/src/views/settings/settings-view.spec.ts`

- [ ] **Step 1: Write the failing settings tests**

```ts
import { mount, flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import SettingsView from './settings-view.vue'
import { adminApi } from '../../api'

vi.mock('../../api', () => ({
  adminApi: {
    getSettings: vi.fn(),
    getAccount: vi.fn(),
    getPasskeys: vi.fn(),
    setPasswordLoginEnabled: vi.fn(),
    beginPasskeyRegistration: vi.fn(),
    finishPasskeyRegistration: vi.fn(),
    deletePasskey: vi.fn(),
  },
}))

describe('settings passkey section', () => {
  it('disables the password-login switch when no passkeys exist', async () => {
    vi.mocked(adminApi.getPasskeys).mockResolvedValue({
      code: 0,
      data: { passwordLoginEnabled: true, passkeys: [] },
    } as any)

    const wrapper = mount(SettingsView)
    await flushPromises()

    expect(wrapper.text()).toContain('Passkey')
    expect(wrapper.find('[data-testid=\"password-login-switch\"]').attributes('disabled')).toBeDefined()
  })
})
```

- [ ] **Step 2: Run the focused Vitest suite and verify it fails**

Run: `cd frontend && npm test -- src/views/settings/settings-view.spec.ts`

Expected: FAIL because settings does not render any Passkey management UI yet.

- [ ] **Step 3: Add Passkey settings state and loader**

```ts
const passkeyLoading = ref(false)
const passkeyBusy = ref(false)
const passkeyState = reactive({
  passwordLoginEnabled: true,
  passkeys: [] as PasskeyItem[],
})

async function fetchPasskeys() {
  passkeyLoading.value = true
  try {
    const res = await adminApi.getPasskeys()
    if (res.code === 0) {
      passkeyState.passwordLoginEnabled = res.data.passwordLoginEnabled
      passkeyState.passkeys = res.data.passkeys
    }
  } finally {
    passkeyLoading.value = false
  }
}
```

- [ ] **Step 4: Implement registration, toggle, and deletion handlers**

```ts
async function handleRegisterPasskey() {
  const begin = await adminApi.beginPasskeyRegistration()
  const credential = await createRegistrationCredential(begin.data.publicKey)
  await adminApi.finishPasskeyRegistration({
    challengeId: begin.data.challengeId,
    credential: serializeRegistrationCredential(credential),
  })
  await fetchPasskeys()
}

async function handleTogglePasswordLogin(enabled: boolean) {
  await adminApi.setPasswordLoginEnabled(enabled)
  await fetchPasskeys()
}

async function handleDeletePasskey(id: number) {
  await adminApi.deletePasskey(id)
  await fetchPasskeys()
}
```

- [ ] **Step 5: Add the Passkey settings section to the template**

```vue
<form-section-card
  title="登录安全"
  description="管理后台 Passkey 与密码登录策略。"
>
  <a-switch
    data-testid="password-login-switch"
    :model-value="passkeyState.passwordLoginEnabled"
    :disabled="!passkeyState.passkeys.length || passkeyBusy"
    @change="handleTogglePasswordLogin"
  />

  <a-button type="primary" @click="handleRegisterPasskey">
    注册 Passkey
  </a-button>

  <div class="passkey-list">
    <article v-for="passkey in passkeyState.passkeys" :key="passkey.id">
      <strong>{{ passkey.deviceLabel }}</strong>
      <span>{{ passkey.createdAt }}</span>
      <span>{{ passkey.lastUsedAt || '未使用' }}</span>
      <a-button
        status="danger"
        :disabled="!passkeyState.passwordLoginEnabled && passkeyState.passkeys.length === 1"
        @click="handleDeletePasskey(passkey.id)"
      >
        删除
      </a-button>
    </article>
  </div>
</form-section-card>
```

- [ ] **Step 6: Run the focused Vitest suite and verify it passes**

Run: `cd frontend && npm test -- src/views/settings/settings-view.spec.ts`

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add frontend/src/views/settings/settings-view.vue frontend/src/views/settings/settings-view.spec.ts
git commit -m "feat: add passkey settings management"
```

### Task 7: Full verification and regression check

**Files:**
- Modify: `web/function_test.go`
- Modify: `frontend/src/utils/webauthn.spec.ts`
- Modify: `frontend/src/views/login/login-view.spec.ts`
- Modify: `frontend/src/views/settings/settings-view.spec.ts`

- [ ] **Step 1: Run the complete backend test suite**

Run: `go test ./web`

Expected: PASS

- [ ] **Step 2: Run the complete frontend test suite**

Run: `cd frontend && npm test`

Expected: PASS

- [ ] **Step 3: Run the production frontend build**

Run: `cd frontend && npm run build`

Expected: PASS with Vite build output and no type errors.

- [ ] **Step 4: Run targeted manual auth verification**

Run:

```bash
go test ./web -run 'TestLoginAuthConfigReturnsPasswordLoginState|TestPasswordLoginRejectedWhenDisabled|TestDisablePasswordLoginRequiresAtLeastOnePasskey|TestExpiredPasskeyChallengeRejected'
cd frontend && npm test -- src/views/login/login-view.spec.ts src/views/settings/settings-view.spec.ts
```

Expected: PASS

Manual checklist:

- Sign in with `admin/admin` while password login is enabled.
- Register one Passkey from system settings.
- Disable password login and confirm the password field disappears from the login page after refresh.
- Sign in using Passkey.
- Confirm deleting the only remaining Passkey is blocked while password login is disabled.
- Re-enable password login from a logged-in session and verify the login form returns.

- [ ] **Step 5: Commit**

```bash
git add web/function_test.go frontend/src/utils/webauthn.spec.ts frontend/src/views/login/login-view.spec.ts frontend/src/views/settings/settings-view.spec.ts
git commit -m "test: cover admin passkey flows"
```

## Self-Review

- Spec coverage:
  - Login page Passkey entry: covered in Task 5
  - Settings Passkey management: covered in Task 6
  - Password-login disablement rules: covered in Tasks 1, 2, and 3
  - Challenge lifecycle and JWT reuse: covered in Tasks 2 and 3
  - Frontend/browser capability handling: covered in Tasks 4 and 5
- Placeholder scan:
  - No `TODO`, `TBD`, or “implement later” markers remain
- Type consistency:
  - `PasswordLoginEnabled`, `PasskeyCredential`, `PasskeyChallenge`, and `PasskeyItem` naming is consistent across data, API, and UI tasks

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-23-passkey-admin-login.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
