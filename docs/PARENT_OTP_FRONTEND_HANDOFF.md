# Parent OTP Login — Frontend Handoff

## Scope

Parents/guardians can sign in with the phone number stored on their linked guardian record. The guardian must be linked to an active user with the `parents` role.

All endpoints are under the API v1 base URL:

```text
POST /api/v1/auth/send-otp
POST /api/v1/auth/verify-otp
```

The existing frontend exports remain the integration surface:

```ts
authApi.sendOtp(request: SendOtpRequest)
authApi.verifyOtp(request: VerifyOtpRequest)
```

`verifyOtp` returns the existing `LoginResponse`, so persist the returned session and continue normal post-login routing.

## Suggested UI flow

1. Show a phone input. Submit an E.164 number, for example `+919876543210`.
2. If `AUTH_TENANT_REQUIRED` is returned, show the school/tenant picker, then repeat the send request with `tenant_slug`.
3. On success, show the OTP screen and a 5-minute countdown using `data.expires_in_seconds`.
4. Submit the received 4–6 digit numeric code to `verify-otp` with the same phone number.
5. Store `access_token`, `refresh_token`, user, and tenants exactly as for email login. The returned access token is already scoped to the selected tenant, so parent routes can be called immediately.

## Send OTP

```http
POST /api/v1/auth/send-otp
Content-Type: application/json

{
  "phone": "+919876543210",
  "tenant_slug": "greenfield-public-school"
}
```

`tenant_slug` is optional. Send it only when the same phone belongs to more than one active tenant, or when the user has already selected a school.

```json
{
  "success": true,
  "data": {
    "message": "OTP sent to +919876****10",
    "expires_in_seconds": 300
  },
  "request_id": "req_abc123"
}
```

Resend is throttled per phone and tenant for 60 seconds. Disable the resend control during that interval; if the backend still responds with a rate-limit error, show the supplied message and keep the OTP input visible.

## Verify OTP

```http
POST /api/v1/auth/verify-otp
Content-Type: application/json

{
  "phone": "+919876543210",
  "otp": "123456"
}
```

```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOi...",
    "refresh_token": "eyJhbGciOi...",
    "expires_at": "2026-07-16T10:00:00Z",
    "user": {
      "id": "uuid",
      "email": "parent@example.com",
      "first_name": "Parent",
      "last_name": "Name",
      "name": "Parent Name",
      "role": "parents",
      "roles": ["parents"],
      "permissions": [],
      "status": "active"
    },
    "tenants": [
      {
        "tenant_id": "uuid",
        "tenant_name": "Greenfield Public School",
        "tenant_slug": "greenfield-public-school",
        "role": "parents",
        "permissions": [],
        "status": "active"
      }
    ]
  },
  "request_id": "req_def456"
}
```

The returned `tenants` list has the tenant selected when the OTP was sent. Do not call `/auth/select-tenant` after OTP verification unless your session store deliberately does so; the OTP access token already contains the tenant context.

## Error handling

All API errors use this envelope:

```ts
type APIErrorResponse = {
  success: false
  error: {
    code: string
    message: string
    details?: string[]
  }
  request_id?: string
}
```

| Code | Status | Frontend action |
| --- | --- | --- |
| `VALIDATION_FAILED` | 400 | Highlight malformed phone or OTP input. Phone must be E.164; OTP must be numeric and 4–6 digits. |
| `AUTH_PHONE_NOT_FOUND` | 404 | Show “No parent account is linked to this phone number.” |
| `AUTH_TENANT_REQUIRED` | 400 | Ask the user to choose their school, then retry send with `tenant_slug`. |
| `AUTH_RATE_LIMITED` | 429 | Keep the OTP view and require the user to wait before resending. |
| `AUTH_OTP_INVALID` | 401 | Keep the OTP view and allow correction. Five invalid attempts invalidate the code. |
| `AUTH_OTP_EXPIRED` | 401 | Return to the phone screen and request a new OTP. |
| `AUTH_ACCOUNT_INACTIVE` | 403 | Show the account is inactive and direct the parent to the school administrator. |
| `RATE_LIMITED` | 429 | Generic API throttle on OTP verification; ask the user to wait and retry. |

## Integration notes

- Never log the OTP, access token, refresh token, or full phone number in frontend telemetry.
- Preserve the phone and selected tenant slug only in transient login state until authentication completes.
- Use `expires_at` to schedule normal access-token refresh behavior.
- The backend requires a configured SMS delivery provider in deployed environments; a delivery failure is returned as the normal generic `INTERNAL_ERROR` envelope and should be shown as a retryable “Unable to send code right now” message.
