# Admin Passkey Design

## Goal

Add Passkey support to the admin authentication flow so the operator can register one or more WebAuthn credentials in system settings, use them to sign in on the admin login page, and optionally disable password login after at least one Passkey has been enrolled.

## Current State

The current admin authentication flow is a single username/password login:

- The login page in `frontend/src/views/login/login-view.vue` submits `username` and `password` to `POST /login`.
- The backend in `web/web.go` validates the admin account from the `sdb.User` table and issues a JWT cookie.
- System settings in `frontend/src/views/settings/settings-view.vue` already manage username/password updates through `GET /admin/api/account` and `POST /admin/api/account`.
- There is no Passkey, WebAuthn, challenge storage, or password-login switch in the current system.

This means Passkey support must be introduced as an additive authentication path while preserving the current JWT cookie session model.

## Product Decision

This feature follows the approved policy:

- Passkey login is added for the admin console.
- Password login can be disabled from system settings.
- Password login may only be disabled after at least one Passkey is registered.
- If password login has been disabled, the system must prevent deletion of the last remaining Passkey.
- The system should continue to support re-enabling password login from an authenticated admin session.

The result is "Passkey-first with controlled password fallback", not "Passkey-only with no recovery path".

## User Experience

### Login Page

The login page keeps the current visual structure and gains a second sign-in path:

- Keep the account field.
- Keep the password field only when password login is enabled.
- Add a `Use Passkey` button below the main sign-in action.
- If password login is disabled, hide the password field and disable the password submit path.
- Show clear inline or toast feedback when:
  - the browser does not support WebAuthn
  - the page is not running in a secure context
  - there are no registered Passkeys for the given account
  - the challenge has expired or verification fails

The existing Enter-to-submit keyboard behavior stays attached to password login only. Passkey login stays an explicit button action.

### System Settings

The settings page gains a dedicated Passkey management section inside the existing account/security area:

- Show whether password login is currently enabled or disabled.
- Show a toggle to enable or disable password login.
- Disable that toggle until at least one Passkey exists.
- Show a compact list of registered Passkeys with:
  - display name
  - created time
  - last used time
  - optional transport/device hint when available
- Provide `Register Passkey` action.
- Provide `Delete` action per Passkey.
- When password login is disabled and only one Passkey remains, disable deletion and explain why.

This keeps all account security operations in one place and matches the current information architecture.

## Architecture

The implementation keeps the current session model and adds WebAuthn as a parallel authentication method:

1. Frontend requests WebAuthn registration or authentication options from the backend.
2. Backend creates a short-lived challenge and returns WebAuthn options payload.
3. Frontend calls `navigator.credentials.create()` or `navigator.credentials.get()`.
4. Frontend posts the resulting credential response back to the backend.
5. Backend verifies the WebAuthn response, updates credential state, and on login success issues the same JWT cookie already used by the app.

This design avoids introducing a second session mechanism and keeps the rest of the admin SPA unchanged.

## Data Model

### New Table: `PasskeyCredential`

Add a persistent credential table associated with the admin user.

Suggested fields:

- `gorm.Model`
- `UserID uint`
- `CredentialID string`
- `CredentialIDB64 string`
- `PublicKey []byte`
- `AttestationType string`
- `AAGUID string`
- `SignCount uint32`
- `Transports string`
- `DeviceLabel string`
- `LastUsedAt *time.Time`

Behavior:

- `CredentialID` should be unique.
- `CredentialIDB64` is stored as a normalized frontend-safe identifier for API payloads and UI operations.
- `DeviceLabel` is user-facing and should default to a generated label such as `Passkey 1` if the browser does not provide a meaningful name.
- `LastUsedAt` updates after successful Passkey authentication.

### New Table: `PasskeyChallenge`

Add a short-lived table for WebAuthn ceremonies.

Suggested fields:

- `gorm.Model`
- `UserID *uint`
- `FlowType string`
- `ChallengeID string`
- `SessionData string`
- `ExpiresAt time.Time`

Behavior:

- `FlowType` supports `register` and `authenticate`.
- `SessionData` stores serialized WebAuthn session state needed for verification.
- Each challenge is one-time-use.
- Expired challenges are rejected and cleaned up opportunistically.

### Existing Table Update: `Setting`

Extend the existing `Setting` table with:

- `PasswordLoginEnabled bool`

Behavior:

- Default value should be `true`.
- Existing installs should continue to work without manual migration steps.

## Backend API

### Public Login APIs

Add:

- `GET /login/auth-config`
- `POST /login/passkey/options`
- `POST /login/passkey/verify`

`GET /login/auth-config`

- Returns public login-mode metadata needed by the login page.
- Response should include at least:
  - `passwordLoginEnabled`
  - `passkeySupported` as a server-declared capability flag
- This endpoint must not disclose any user-specific credential information.

`POST /login/passkey/options`

- Accepts the admin username so the system can resolve the correct user and allowed credentials.
- Returns WebAuthn authentication options and a server-side challenge token reference.
- If password login is disabled, this becomes the primary login method.

`POST /login/passkey/verify`

- Accepts the WebAuthn assertion result from the browser.
- Verifies challenge, RP ID, origin, credential ownership, and signature counter.
- On success, issues the same `token` cookie as the existing password login route.

Existing `POST /login`

- Must check `PasswordLoginEnabled`.
- If password login is disabled, reject password authentication with a clear business error instead of silently continuing.

### Authenticated Admin APIs

Add:

- `GET /admin/api/passkeys`
- `POST /admin/api/passkeys/register/options`
- `POST /admin/api/passkeys/register/verify`
- `DELETE /admin/api/passkeys/:id`
- `POST /admin/api/passkeys/password-login`

`GET /admin/api/passkeys`

- Returns the current password-login status.
- Returns all registered Passkeys for the current admin user.

`POST /admin/api/passkeys/register/options`

- Starts a registration ceremony for the authenticated admin.
- Returns creation options for `navigator.credentials.create()`.

`POST /admin/api/passkeys/register/verify`

- Verifies the attestation response.
- Persists the credential.
- Returns the saved credential summary for immediate UI refresh.

`DELETE /admin/api/passkeys/:id`

- Deletes the selected Passkey.
- Rejects deletion if this is the last Passkey while password login is disabled.

`POST /admin/api/passkeys/password-login`

- Accepts `{ enabled: boolean }`.
- Rejects disabling when the current admin has zero registered Passkeys.

## RP ID and Origin Rules

WebAuthn depends on stable domain matching, so the system needs explicit rules:

- Derive the relying party origin from `Setting.AppUrl` when it is configured.
- Use the request host as a fallback in local development.
- Accept `localhost` and `127.0.0.1` during local development.
- In production, reject registration/authentication when the derived RP ID does not match the current origin host.
- Return a clear configuration error instead of attempting verification with mismatched host data.

This makes failures diagnosable and avoids silently creating Passkeys tied to the wrong domain.

## Frontend Structure

### API Layer

Extend `frontend/src/api/index.ts` with typed methods for:

- loading Passkey settings
- starting registration
- verifying registration
- toggling password login
- starting Passkey login
- verifying Passkey login

### WebAuthn Utilities

Add a focused utility module for:

- base64url encoding/decoding
- converting backend payloads into browser WebAuthn option objects
- converting browser credential responses back into JSON-safe payloads
- secure-context and browser capability detection

This logic should live outside the page components so the views stay readable and testable.

### Login View

Update `frontend/src/views/login/login-view.vue` to:

- fetch login-mode metadata from `GET /login/auth-config` on mount
- hide the password field when password login is disabled
- keep the existing visual structure intact
- add `Use Passkey` action and loading states
- surface precise user feedback for unsupported browsers, missing credentials, and verification failures

### Settings View

Update `frontend/src/views/settings/settings-view.vue` to:

- render a Passkey section under the account/security area
- display the password-login toggle
- render the Passkey list
- support register/delete flows with optimistic refresh or immediate reload after each action

## Security Rules

- Registration requires an authenticated admin session.
- Challenges are single-use and time-limited.
- Successful authentication must update the credential sign counter.
- Password login cannot be disabled until at least one Passkey exists.
- The last remaining Passkey cannot be deleted while password login is disabled.
- Password login state changes must not invalidate the current JWT session unless the username or password itself changes according to existing behavior.
- Errors should avoid leaking whether a username exists beyond what the current login flow already reveals.

## Error Handling

Expected backend error cases:

- invalid username
- no credential registered
- expired or missing challenge
- RP ID / origin mismatch
- duplicate credential during registration
- signature counter verification failure
- attempt to disable password login without a Passkey
- attempt to delete the last Passkey while password login is disabled

Expected frontend behavior:

- keep the current page stable on all failures
- surface readable toast messages
- restore loading states correctly after cancellation or browser-level WebAuthn errors
- allow the user to retry without refreshing the page

## Testing

### Backend

Add tests for:

- password-login toggle cannot be disabled with zero Passkeys
- deleting the last Passkey is blocked when password login is disabled
- Passkey login verification issues JWT cookie on success
- expired or reused challenges are rejected
- settings migration defaults `PasswordLoginEnabled` to true for existing data

### Frontend

Add tests for:

- login page hides password field when password login is disabled
- Passkey button triggers the WebAuthn helper flow
- settings page blocks disabling password login when there are no Passkeys
- settings page renders Passkey list items and deletion guard states

### Manual Verification

Verify on:

- desktop Chrome on localhost
- mobile device or responsive emulation for the settings layout
- both password-enabled and password-disabled states

## Out of Scope

The following are not part of this change:

- merchant-side Passkey support
- multi-user admin management
- backup codes or email recovery
- platform-native biometric UX customization beyond the browser default
- replacing JWT cookie auth with a new session system

## Implementation Notes

- Prefer the existing Go backend and Gin routing style instead of introducing a separate auth module.
- Keep the new WebAuthn helpers as small focused files rather than overloading `web/web.go` further if the implementation starts to sprawl.
- Preserve current admin login behavior for existing installs until the operator explicitly disables password login.
