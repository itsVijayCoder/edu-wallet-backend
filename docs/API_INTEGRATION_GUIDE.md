# EduWallet API Integration Guide For Frontend

This guide is for the Next.js frontend developer integrating with the EduWallet backend.

Primary API sources:

- Runtime API: `http://localhost:8080/api/v1`
- Swagger UI: `http://localhost:8080/api/v1/docs`
- OpenAPI JSON: `docs/swagger/openapi.json`
- Route source: `internal/router/router.go`
- Request/response DTOs: `internal/dto`

Regenerate checked-in API specs after backend route changes:

```bash
make swagger
```

## Base URL

Use one environment variable in the frontend:

```bash
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1
```

All paths in this document are relative to `/api/v1`.

## Response Envelope

Most JSON endpoints return the same envelope.

```ts
export type ApiResponse<T> = {
  success: boolean;
  request_id?: string;
  data?: T;
  error?: {
    code: string;
    message: string;
    details?: string[];
  };
  meta?: {
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
  };
};
```

Successful single-resource response:

```json
{
  "success": true,
  "request_id": "req_...",
  "data": {}
}
```

Successful paginated response:

```json
{
  "success": true,
  "request_id": "req_...",
  "data": [],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

Error response:

```json
{
  "success": false,
  "request_id": "req_...",
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "validation failed",
    "details": ["Email is required"]
  }
}
```

Known stable error codes include:

| Code | Meaning |
| --- | --- |
| `AUTH_INVALID_CREDENTIALS` | Invalid email or password. |
| `AUTH_ACCOUNT_INACTIVE` | User account exists but is inactive. |
| `AUTH_TOKEN_EXPIRED` | Access token expired. |
| `AUTH_INVALID_TOKEN` | Token is missing, invalid, or malformed. |
| `AUTH_REFRESH_INVALID` | Refresh token is invalid or expired. |
| `AUTH_REGISTRATION_DISABLED` | Public registration is disabled. |
| `TENANT_REQUIRED` | Endpoint requires a tenant-scoped token. |
| `TENANT_ACCESS_DENIED` | User cannot access selected tenant. |
| `FORBIDDEN` | Role or permission denied. |
| `NOT_FOUND` | Resource not found. |
| `CONFLICT` | Resource already exists. |
| `VALIDATION_FAILED` | Request validation failed. |
| `RATE_LIMITED` | Too many requests. |
| `INTERNAL_ERROR` | Unexpected backend error. |

## Auth Model

The backend uses bearer JWTs.

```http
Authorization: Bearer <access_token>
```

There are two important token phases:

1. Login token: returned by `POST /auth/login`. It identifies the user and global roles.
2. Tenant token: returned by `POST /auth/select-tenant`. It includes the selected tenant and permissions. Use this token for tenant admin and parent routes.

Recommended frontend flow:

1. Call `POST /auth/login`.
2. Store `access_token`, `refresh_token`, user, and returned `tenants`.
3. If the user needs a tenant-scoped app area, call `POST /auth/select-tenant` with `tenant_id`.
4. Replace the stored access token with the tenant access token.
5. Use the refresh token from the latest auth response.
6. On `401`, call `POST /auth/refresh` once, update tokens, then retry the failed request.
7. On repeated `401` or `AUTH_REFRESH_INVALID`, clear session and redirect to login.

Do not call tenant admin or parent APIs with only the login token. They require tenant context.

## Axios Client

Use a single Axios instance and unwrap the API envelope in one place.

```ts
import axios, { AxiosError, AxiosRequestConfig } from "axios";

export class ApiError extends Error {
  code: string;
  status?: number;
  details?: string[];
  requestId?: string;

  constructor(input: {
    code: string;
    message: string;
    status?: number;
    details?: string[];
    requestId?: string;
  }) {
    super(input.message);
    this.code = input.code;
    this.status = input.status;
    this.details = input.details;
    this.requestId = input.requestId;
  }
}

export const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_BASE_URL,
  timeout: 30000,
  headers: { "Content-Type": "application/json" },
});

api.interceptors.request.use((config) => {
  const token = authStore.getState().accessToken;
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

export async function request<T>(config: AxiosRequestConfig): Promise<T> {
  try {
    const response = await api.request<ApiResponse<T>>(config);
    const body = response.data;
    if (!body.success) {
      throw new ApiError({
        code: body.error?.code ?? "API_ERROR",
        message: body.error?.message ?? "Request failed",
        details: body.error?.details,
        requestId: body.request_id,
        status: response.status,
      });
    }
    return body.data as T;
  } catch (error) {
    const err = error as AxiosError<ApiResponse<unknown>>;
    if (err.response?.data?.error) {
      throw new ApiError({
        code: err.response.data.error.code,
        message: err.response.data.error.message,
        details: err.response.data.error.details,
        requestId: err.response.data.request_id,
        status: err.response.status,
      });
    }
    throw error;
  }
}
```

For paginated endpoints, return both `data` and `meta`:

```ts
export async function requestPage<T>(config: AxiosRequestConfig) {
  const response = await api.request<ApiResponse<T[]>>(config);
  if (!response.data.success) throw new Error(response.data.error?.message);
  return {
    rows: response.data.data ?? [],
    meta: response.data.meta!,
  };
}
```

## File Downloads

Some endpoints return raw files, not the JSON envelope.

Use `responseType: "blob"` for:

- `GET /admin/imports/students/template`
- `GET /admin/receipts/{id}/download`
- `GET /parent/receipts/{id}/download`
- `GET /admin/exports/{id}/download`

```ts
export async function downloadFile(path: string, filenameFallback: string) {
  const response = await api.get<Blob>(path, { responseType: "blob" });
  const disposition = response.headers["content-disposition"];
  const filename =
    disposition?.match(/filename="?([^"]+)"?/)?.[1] ?? filenameFallback;

  const url = URL.createObjectURL(response.data);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}
```

## Shared Conventions

| Convention | Frontend rule |
| --- | --- |
| UUID | Treat all `*_id` fields as UUID strings. Validate with Zod `z.string().uuid()`. |
| Dates | Send API dates as `YYYY-MM-DD`. |
| Timestamps | Returned as ISO timestamps. Display in the user's locale. |
| Money | Backend uses paise integers. Display INR, submit paise. |
| Pagination | Use `page`, `page_size`, `sort_by`, `sort_dir`. Defaults are page `1`, page size `20`, sort `created_at desc`. |
| Sort direction | `asc` or `desc`. |
| Soft delete | Delete endpoints usually soft delete and return a message envelope. |
| PATCH | OpenAPI marks PATCH bodies as generic, but handlers expect typed partial update DTOs. Send only changed fields. |
| Rate limits | Auth, password reset, tenant selection, parent payment order, verify, and webhooks are rate limited. |

Money helpers:

```ts
export function paiseToInr(paise: number) {
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 2,
  }).format(paise / 100);
}

export function rupeesToPaise(value: string) {
  return Math.round(Number(value || 0) * 100);
}
```

## TanStack Query Keys

Use stable, nested query keys:

```ts
export const qk = {
  auth: ["auth"] as const,
  tenant: ["tenant"] as const,
  tenants: (params?: unknown) => ["platform", "tenants", params] as const,
  dashboard: (params?: unknown) => ["admin", "dashboard", params] as const,
  academicYears: (params?: unknown) => ["academic-years", params] as const,
  classes: (params?: unknown) => ["classes", params] as const,
  sections: (params?: unknown) => ["sections", params] as const,
  students: (params?: unknown) => ["students", params] as const,
  student: (id: string) => ["students", id] as const,
  studentLedger: (id: string) => ["students", id, "ledger"] as const,
  feeHeads: (params?: unknown) => ["fee-heads", params] as const,
  feeStructures: (params?: unknown) => ["fee-structures", params] as const,
  invoices: (params?: unknown) => ["invoices", params] as const,
  payments: (params?: unknown) => ["payments", params] as const,
  receipts: (params?: unknown) => ["receipts", params] as const,
  reports: (type: string, params?: unknown) => ["reports", type, params] as const,
  parentDues: (childId: string) => ["parent", "children", childId, "dues"] as const,
};
```

After mutations, invalidate the relevant list and detail keys. For example, after creating an offline payment, invalidate `payments`, `receipts`, `invoices`, `studentLedger(student_id)`, `dashboard`, and affected reports.

## Important Request Examples

Login:

```json
{
  "email": "admin@eduwallet.in",
  "password": "password"
}
```

Select tenant:

```json
{
  "tenant_id": "00000000-0000-0000-0000-000000000000"
}
```

Create fee structure:

```json
{
  "academic_year_id": "00000000-0000-0000-0000-000000000000",
  "name": "Class 6 Term 1",
  "code": "C6-T1",
  "billing_cycle": "term",
  "status": "active",
  "allow_partial_payment": true,
  "minimum_partial_amount_paise": 50000,
  "due_day": 10,
  "items": [
    {
      "fee_head_id": "00000000-0000-0000-0000-000000000000",
      "name": "Tuition Fee",
      "amount_paise": 250000,
      "sort_order": 1
    }
  ]
}
```

Create parent payment order:

```json
{
  "student_id": "00000000-0000-0000-0000-000000000000",
  "invoice_ids": ["00000000-0000-0000-0000-000000000000"],
  "amount_paise": 250000,
  "idempotency_key": "checkout-1700000000-student-id"
}
```

Verify Razorpay payment:

```json
{
  "provider_order_id": "order_...",
  "provider_payment_id": "pay_...",
  "signature": "razorpay_signature",
  "payment_method": "upi"
}
```

Record offline payment:

```json
{
  "student_id": "00000000-0000-0000-0000-000000000000",
  "payment_method": "cash",
  "received_on": "2026-06-08",
  "clearance_status": "cleared",
  "allocations": [
    {
      "invoice_id": "00000000-0000-0000-0000-000000000000",
      "amount_paise": 250000
    }
  ],
  "remarks": "Collected at counter"
}
```

## Response Data Types

Use these DTO names as TypeScript model names in the frontend. The backend wraps them inside `ApiResponse<T>` unless the endpoint downloads a file.

| Backend DTO | Used by |
| --- | --- |
| `LoginResponse` | `POST /auth/login` |
| `TokenPair` | `POST /auth/refresh`, `POST /auth/select-tenant` |
| `UserResponse` | Register, admin users |
| `TenantResponse` | Tenant create/list/detail/update/current |
| `BranchResponse` | Create branch |
| `TenantUserResponse` | Create tenant user |
| `AcademicYearResponse` | Academic year endpoints |
| `ClassResponse` | Class endpoints |
| `SectionResponse` | Section endpoints |
| `StudentResponse` | Student endpoints |
| `GuardianResponse` | Guardian endpoints |
| `StudentImportPreviewResponse` | Student import preview |
| `StudentImportCommitResponse` | Student import commit |
| `ImportResponse` | Student import history |
| `FeeHeadResponse` | Fee head endpoints |
| `FeeStructureResponse` | Fee structure endpoints |
| `FeeAssignmentResponse` | Create fee assignment |
| `GenerateInvoicesResponse` | Generate invoices |
| `InvoiceResponse` | Invoice list/detail and parent dues |
| `StudentLedgerResponse` | Student ledger |
| `ParentDuesResponse` | Parent child dues |
| `PaymentOrderResponse` | Parent payment order |
| `PaymentVerificationResponse` | Parent payment verify |
| `WebhookProcessResponse` | Razorpay webhook |
| `PaymentResponse` | Online/offline payment endpoints |
| `ReceiptResponse` | Receipt endpoints |
| `PaymentEventResponse` | Payment events and dashboard recent events |
| `ReminderTemplateResponse` | Reminder template endpoints |
| `ReminderRuleResponse` | Reminder rule endpoints |
| `SendReminderResponse` | Manual reminder send |
| `ReminderLogResponse` | Reminder logs |
| `DashboardResponse` | Admin dashboard |
| `CollectionReportRowResponse` | Collections and offline-payment reports |
| `DefaulterReportRowResponse` | Defaulters report |
| `DueReportRowResponse` | Dues report |
| `FeeHeadCollectionRowResponse` | Fee-head report |
| `PaymentMethodReportRowResponse` | Payment-method report |
| `ExportJobResponse` | Export jobs |

## Complete API Catalog

The catalog below lists every available backend route exposed by `internal/router/router.go` and `docs/swagger/openapi.json`.

### Public, Docs, And Health

| Method | Path | Auth | Request | Response / frontend use |
| --- | --- | --- | --- | --- |
| `GET` | `/healthz` | Public | None | Liveness probe. |
| `GET` | `/readyz` | Public | None | Readiness probe. |
| `GET` | `/docs` | Public | None | Swagger UI. |
| `GET` | `/docs/openapi.json` | Public | None | OpenAPI JSON. |
| `GET` | `/docs/swagger.json` | Public | None | Swagger-compatible JSON. |

### Auth

| Method | Path | Auth | Request | Response / frontend use |
| --- | --- | --- | --- | --- |
| `POST` | `/auth/login` | Public | `LoginRequest` | `LoginResponse`; store tokens, user, tenant memberships. |
| `POST` | `/auth/register` | Public, may be disabled | `RegisterRequest` | `UserResponse`; public registration depends on backend config. |
| `POST` | `/auth/refresh` | Public with refresh token body | `RefreshRequest` | `TokenPair`; rotate stored tokens. |
| `POST` | `/auth/select-tenant` | Bearer login token | `SelectTenantRequest` | `TokenPair`; replace access token with tenant token. |
| `POST` | `/auth/logout` | Bearer token | None | Message; clear local session. |
| `POST` | `/auth/forgot-password` | Public | `ForgotPasswordRequest` | Message; backend always returns success-like response to prevent email enumeration. |
| `POST` | `/auth/reset-password` | Public | `ResetPasswordRequest` | Message. |

### Platform Tenants

Requires bearer token with `super_admin` role.

| Method | Path | Request | Query | Response / frontend use |
| --- | --- | --- | --- | --- |
| `POST` | `/platform/tenants` | `CreateTenantRequest` | None | `TenantResponse`; create school/institution. |
| `GET` | `/platform/tenants` | None | `page`, `page_size`, `sort_by`, `sort_dir` | `TenantResponse[]`; platform tenant table. |
| `GET` | `/platform/tenants/{id}` | None | Path `id` | `TenantResponse`; tenant detail. |
| `PATCH` | `/platform/tenants/{id}` | `UpdateTenantRequest` partial | Path `id` | `TenantResponse`; update platform tenant profile/status. |
| `POST` | `/platform/tenants/{id}/branches` | `CreateBranchRequest` | Path `id` | `BranchResponse`; add branch/campus. |

### Admin Users

`/admin/users` requires bearer token with `super_admin` or `admin` role. `/admin/tenant/users` requires tenant token and `users.manage`.

| Method | Path | Request | Query | Response / frontend use |
| --- | --- | --- | --- | --- |
| `POST` | `/admin/users` | `CreateUserRequest` | None | `UserResponse`; platform/admin user creation. |
| `GET` | `/admin/users` | None | `page`, `page_size`, `sort_by`, `sort_dir` | `UserResponse[]`; user table. |
| `GET` | `/admin/users/{id}` | None | Path `id` | `UserResponse`; user detail. |
| `PUT` | `/admin/users/{id}` | `UpdateUserRequest` | Path `id` | `UserResponse`; update user. |
| `DELETE` | `/admin/users/{id}` | None | Path `id` | Message; soft delete/deactivate user. |
| `POST` | `/admin/tenant/users` | `CreateTenantUserRequest` | None | `TenantUserResponse`; create user inside selected tenant. |

### Tenant Settings

Requires tenant token.

| Method | Path | Permission | Request | Query | Response / frontend use |
| --- | --- | --- | --- | --- | --- |
| `GET` | `/admin/tenant` | `tenant.read` | None | None | `TenantResponse`; current tenant settings. |
| `PATCH` | `/admin/tenant` | `tenant.update` | `UpdateTenantRequest` partial | None | `TenantResponse`; update selected tenant profile. |

### Academic Setup

Requires tenant token and `academic.manage`.

| Method | Path | Request | Query | Response / frontend use |
| --- | --- | --- | --- | --- |
| `POST` | `/admin/academic-years` | `CreateAcademicYearRequest` | None | `AcademicYearResponse`; create year. |
| `GET` | `/admin/academic-years` | None | Pagination, `status`, `search` | `AcademicYearResponse[]`; academic year table. |
| `GET` | `/admin/academic-years/{id}` | None | Path `id` | `AcademicYearResponse`; detail/edit sheet. |
| `PATCH` | `/admin/academic-years/{id}` | `UpdateAcademicYearRequest` partial | Path `id` | `AcademicYearResponse`; update year. |
| `DELETE` | `/admin/academic-years/{id}` | None | Path `id` | Message; soft delete. |
| `POST` | `/admin/classes` | `CreateClassRequest` | None | `ClassResponse`; create class. |
| `GET` | `/admin/classes` | None | Pagination, `status`, `search` | `ClassResponse[]`; class table. |
| `GET` | `/admin/classes/{id}` | None | Path `id` | `ClassResponse`; class detail. |
| `PATCH` | `/admin/classes/{id}` | `UpdateClassRequest` partial | Path `id` | `ClassResponse`; update class. |
| `DELETE` | `/admin/classes/{id}` | None | Path `id` | Message; soft delete. |
| `POST` | `/admin/sections` | `CreateSectionRequest` | None | `SectionResponse`; create section. |
| `GET` | `/admin/sections` | None | Pagination, `academic_year_id`, `class_id`, `status`, `search` | `SectionResponse[]`; section table. |
| `GET` | `/admin/sections/{id}` | None | Path `id` | `SectionResponse`; section detail. |
| `PATCH` | `/admin/sections/{id}` | `UpdateSectionRequest` partial | Path `id` | `SectionResponse`; update section. |
| `DELETE` | `/admin/sections/{id}` | None | Path `id` | Message; soft delete. |

### Students, Guardians, And Imports

Student routes require tenant token and `students.manage`. Guardian CRUD requires `guardians.manage`. Imports require `imports.manage`.

| Method | Path | Permission | Request | Query | Response / frontend use |
| --- | --- | --- | --- | --- | --- |
| `POST` | `/admin/students` | `students.manage` | `CreateStudentRequest` | None | `StudentResponse`; create student and optional guardian links. |
| `GET` | `/admin/students` | `students.manage` | None | Pagination, `academic_year_id`, `class_id`, `section_id`, `status`, `search` | `StudentResponse[]`; searchable student directory. |
| `GET` | `/admin/students/{id}` | `students.manage` | None | Path `id` | `StudentResponse`; student profile. |
| `PATCH` | `/admin/students/{id}` | `students.manage` | `UpdateStudentRequest` partial | Path `id` | `StudentResponse`; update student. |
| `DELETE` | `/admin/students/{id}` | `students.manage` | None | Path `id` | Message; soft delete. |
| `POST` | `/admin/students/{id}/guardians` | `students.manage` | `StudentGuardianRequest` | Path `id` | `StudentResponse`; link guardian. |
| `DELETE` | `/admin/students/{id}/guardians/{guardian_id}` | `students.manage` | None | Path `id`, `guardian_id` | `StudentResponse`; unlink guardian. |
| `POST` | `/admin/guardians` | `guardians.manage` | `CreateGuardianRequest` | None | `GuardianResponse`; create guardian. |
| `GET` | `/admin/guardians` | `guardians.manage` | None | Pagination, `search` | `GuardianResponse[]`; guardian directory. |
| `GET` | `/admin/guardians/{id}` | `guardians.manage` | None | Path `id` | `GuardianResponse`; guardian detail. |
| `PATCH` | `/admin/guardians/{id}` | `guardians.manage` | `UpdateGuardianRequest` partial | Path `id` | `GuardianResponse`; update guardian. |
| `DELETE` | `/admin/guardians/{id}` | `guardians.manage` | None | Path `id` | Message; soft delete. |
| `GET` | `/admin/imports` | `imports.manage` | None | Pagination, `status`, `import_type` | `ImportResponse[]`; import history. |
| `GET` | `/admin/imports/students/template` | `imports.manage` | None | None | CSV file; download template. |
| `POST` | `/admin/imports/students/preview` | `imports.manage` | `StudentImportUploadRequest`, multipart `file`, or raw `text/csv` | Optional `filename` for raw CSV | `StudentImportPreviewResponse`; validate CSV before commit. |
| `POST` | `/admin/imports/students/commit` | `imports.manage` | `StudentImportCommitRequest` | None | `StudentImportCommitResponse`; commit clean preview. |

### Billing, Fees, Invoices, And Ledgers

Requires tenant token and `fees.manage`, except parent dues which requires a tenant token for the parent app.

| Method | Path | Permission | Request | Query | Response / frontend use |
| --- | --- | --- | --- | --- | --- |
| `POST` | `/admin/fee-heads` | `fees.manage` | `CreateFeeHeadRequest` | None | `FeeHeadResponse`; create fee head. |
| `GET` | `/admin/fee-heads` | `fees.manage` | None | Pagination, `status`, `category`, `search` | `FeeHeadResponse[]`; fee-head table. |
| `GET` | `/admin/fee-heads/{id}` | `fees.manage` | None | Path `id` | `FeeHeadResponse`; fee-head detail. |
| `PATCH` | `/admin/fee-heads/{id}` | `fees.manage` | `UpdateFeeHeadRequest` partial | Path `id` | `FeeHeadResponse`; update fee head. |
| `DELETE` | `/admin/fee-heads/{id}` | `fees.manage` | None | Path `id` | Message; soft delete. |
| `POST` | `/admin/fee-structures` | `fees.manage` | `CreateFeeStructureRequest` | None | `FeeStructureResponse`; create structure and items. |
| `GET` | `/admin/fee-structures` | `fees.manage` | None | Pagination, `academic_year_id`, `status`, `billing_cycle`, `search` | `FeeStructureResponse[]`; structure table. |
| `GET` | `/admin/fee-structures/{id}` | `fees.manage` | None | Path `id` | `FeeStructureResponse`; includes items. |
| `PATCH` | `/admin/fee-structures/{id}` | `fees.manage` | `UpdateFeeStructureRequest` partial | Path `id` | `FeeStructureResponse`; update structure and optional items. |
| `DELETE` | `/admin/fee-structures/{id}` | `fees.manage` | None | Path `id` | Message; soft delete. |
| `POST` | `/admin/fee-assignments` | `fees.manage` | `CreateFeeAssignmentRequest` | None | `FeeAssignmentResponse`; assign fee structure to class, section, or student. |
| `POST` | `/admin/invoices/generate` | `fees.manage` | `GenerateInvoicesRequest` | None | `GenerateInvoicesResponse`; create invoices idempotently. |
| `GET` | `/admin/invoices` | `fees.manage` | None | Pagination, `student_id`, `academic_year_id`, `class_id`, `section_id`, `status`, `due_from`, `due_to`, `search` | `InvoiceResponse[]`; invoice table. |
| `GET` | `/admin/invoices/{id}` | `fees.manage` | None | Path `id` | `InvoiceResponse`; invoice detail with items. |
| `GET` | `/admin/students/{id}/ledger` | `fees.manage` | None | Path `id` | `StudentLedgerResponse`; student ledger. |
| `GET` | `/parent/children/{id}/dues` | Tenant token | None | Path `id` | `ParentDuesResponse`; parent due summary and invoices. |

### Parent Payments, Admin Payments, Receipts, And Webhooks

Parent routes require tenant token. Admin finance routes require tenant token and `payments.manage`. The Razorpay webhook is not a browser endpoint; Razorpay calls it with signature headers.

| Method | Path | Permission / auth | Request | Query | Response / frontend use |
| --- | --- | --- | --- | --- | --- |
| `POST` | `/parent/payments/orders` | Tenant token | `CreatePaymentOrderRequest` | None | `PaymentOrderResponse`; create Razorpay/provider order. |
| `POST` | `/parent/payments/verify` | Tenant token | `VerifyPaymentRequest` | None | `PaymentVerificationResponse`; verify checkout and receive receipt. |
| `GET` | `/parent/receipts` | Tenant token | None | Pagination, `student_id`, `status`, `from`, `to`, `search` | `ReceiptResponse[]`; parent receipt list. |
| `GET` | `/parent/receipts/{id}/download` | Tenant token | None | Path `id` | PDF file; parent receipt download. |
| `POST` | `/webhooks/razorpay` | Public with Razorpay signature headers | `RazorpayWebhookRequest` raw JSON | Headers `X-Razorpay-Signature`, `X-Razorpay-Event-Id` | `WebhookProcessResponse`; backend reconciliation. |
| `POST` | `/admin/offline-payments` | `payments.manage` | `CreateOfflinePaymentRequest` | None | `PaymentResponse`; record cleared or pending offline payment. |
| `GET` | `/admin/payments` | `payments.manage` | None | Pagination, `student_id`, `status`, `payment_method`, `provider`, `from`, `to`, `search` | `PaymentResponse[]`; payment table. |
| `GET` | `/admin/payments/{id}` | `payments.manage` | None | Path `id` | `PaymentResponse`; payment detail and allocations. |
| `GET` | `/admin/receipts` | `payments.manage` | None | Pagination, `student_id`, `status`, `from`, `to`, `search` | `ReceiptResponse[]`; receipt register. |
| `GET` | `/admin/receipts/{id}` | `payments.manage` | None | Path `id` | `ReceiptResponse`; receipt detail. |
| `GET` | `/admin/receipts/{id}/download` | `payments.manage` | None | Path `id` | PDF file; admin receipt download. |
| `GET` | `/admin/payment-events` | `payments.manage` | None | Pagination, `student_id`, `event_type`, `status`, `from`, `to` | `PaymentEventResponse[]`; payment activity feed. |

### Reminders

Requires tenant token and `reminders.manage`.

| Method | Path | Request | Query | Response / frontend use |
| --- | --- | --- | --- | --- |
| `POST` | `/admin/reminder-templates` | `CreateReminderTemplateRequest` | None | `ReminderTemplateResponse`; create template. |
| `GET` | `/admin/reminder-templates` | None | Pagination, `channel`, `status`, `search` | `ReminderTemplateResponse[]`; template table. |
| `GET` | `/admin/reminder-templates/{id}` | None | Path `id` | `ReminderTemplateResponse`; template detail. |
| `PATCH` | `/admin/reminder-templates/{id}` | `UpdateReminderTemplateRequest` partial | Path `id` | `ReminderTemplateResponse`; update template. |
| `DELETE` | `/admin/reminder-templates/{id}` | None | Path `id` | Message; soft delete. |
| `POST` | `/admin/reminder-rules` | `CreateReminderRuleRequest` | None | `ReminderRuleResponse`; create rule. |
| `GET` | `/admin/reminder-rules` | None | Pagination, `channel`, `trigger_type`, `status`, `search` | `ReminderRuleResponse[]`; rule table. |
| `GET` | `/admin/reminder-rules/{id}` | None | Path `id` | `ReminderRuleResponse`; rule detail. |
| `PATCH` | `/admin/reminder-rules/{id}` | `UpdateReminderRuleRequest` partial | Path `id` | `ReminderRuleResponse`; update rule. |
| `DELETE` | `/admin/reminder-rules/{id}` | None | Path `id` | Message; soft delete. |
| `POST` | `/admin/reminders/send` | `SendReminderRequest` | None | `SendReminderResponse`; queue or send reminders. |
| `GET` | `/admin/reminder-logs` | None | Pagination, `student_id`, `invoice_id`, `channel`, `status`, `from`, `to` | `ReminderLogResponse[]`; reminder log table. |

### Dashboard, Reports, And Exports

Dashboard and reports require tenant token and `reports.view`. Exports require tenant token and `exports.manage`.

| Method | Path | Permission | Request | Query | Response / frontend use |
| --- | --- | --- | --- | --- | --- |
| `GET` | `/admin/dashboard` | `reports.view` | None | Optional `as_of` | `DashboardResponse`; KPI cards and charts. |
| `GET` | `/admin/reports/collections` | `reports.view` | None | Pagination, `from`, `to`, `as_of`, `academic_year_id`, `student_id`, `class_id`, `section_id`, `payment_method`, `provider` | `CollectionReportRowResponse[]`; collections table. |
| `GET` | `/admin/reports/defaulters` | `reports.view` | None | Same shared report filters | `DefaulterReportRowResponse[]`; defaulter table. |
| `GET` | `/admin/reports/dues` | `reports.view` | None | Same shared report filters | `DueReportRowResponse[]`; dues table. |
| `GET` | `/admin/reports/fee-heads` | `reports.view` | None | Same shared report filters | `FeeHeadCollectionRowResponse[]`; fee-head collection table/chart. |
| `GET` | `/admin/reports/payment-methods` | `reports.view` | None | Same shared report filters | `PaymentMethodReportRowResponse[]`; method mix table/chart. |
| `GET` | `/admin/reports/offline-payments` | `reports.view` | None | Same shared report filters; backend defaults provider to `offline` when omitted | `CollectionReportRowResponse[]`; offline collection table. |
| `POST` | `/admin/exports` | `exports.manage` | `CreateExportRequest` | None | `ExportJobResponse`; create CSV export job. |
| `GET` | `/admin/exports` | `exports.manage` | None | Pagination, `export_type`, `status`, `from`, `to` | `ExportJobResponse[]`; export job table. |
| `GET` | `/admin/exports/{id}` | `exports.manage` | None | Path `id` | `ExportJobResponse`; export job detail. |
| `GET` | `/admin/exports/{id}/download` | `exports.manage` | None | Path `id` | CSV file; download export. |

## Request Schema Reference

These are the request body schemas currently exposed by OpenAPI. Keep frontend Zod schemas aligned with these names and fields.

| Schema | Required fields | Notes |
| --- | --- | --- |
| `LoginRequest` | `email`, `password` | Password minimum is 8. |
| `RegisterRequest` | `email`, `password`, `first_name`, `last_name` | Public registration may be disabled. |
| `RefreshRequest` | `refresh_token` | Rotates access and refresh tokens. |
| `SelectTenantRequest` | `tenant_id` | Requires login access token. |
| `ForgotPasswordRequest` | `email` | Always returns success message. |
| `ResetPasswordRequest` | `token`, `new_password` | Password minimum is 8. |
| `CreateUserRequest` | `email`, `password`, `first_name`, `last_name`, `roles` | Platform/admin user. |
| `UpdateUserRequest` | None | Supports `email`, `first_name`, `last_name`, `status`, `roles`. |
| `CreateTenantUserRequest` | `email`, `password`, `first_name`, `last_name`, `role` | Role enum: `admin`, `staff`, `parents`, `student`. |
| `AddressRequest` | None | Embedded in tenant, branch, guardian, student requests. |
| `CreateTenantRequest` | `name`, `slug` | Optional `branch`, `owner_user_id`, contact fields, status. |
| `CreateBranchRequest` | `name`, `code` | Optional address and contact fields. |
| `CreateAcademicYearRequest` | `name`, `code`, `start_date`, `end_date` | Status enum: `active`, `inactive`, `closed`. |
| `CreateClassRequest` | `name`, `code` | Status enum: `active`, `inactive`. |
| `CreateSectionRequest` | `academic_year_id`, `class_id`, `name`, `code` | Optional branch and capacity. |
| `CreateGuardianRequest` | `name` | Optional phone, WhatsApp phone, email, address, opt-in. |
| `StudentGuardianRequest` | `guardian_id` | Used for linking guardians to students. |
| `CreateStudentRequest` | `academic_year_id`, `class_id`, `section_id`, `admission_number`, `first_name` | Optional guardian links, opening balance, category. |
| `StudentImportUploadRequest` | `csv` for JSON mode | Multipart uses `file`; raw CSV uses body plus optional `filename` query. |
| `StudentImportCommitRequest` | `import_id` | Commit only clean previews. |
| `CreateFeeHeadRequest` | `name`, `code` | Category supports school fee categories such as `tuition`, `transport`, `hostel`, `fine`, `custom`. |
| `CreateFeeStructureItemRequest` | `fee_head_id`, `amount_paise` | Used inside fee structure items. |
| `CreateFeeStructureRequest` | `academic_year_id`, `name`, `code`, `items` | Billing cycle enum: `one_time`, `monthly`, `quarterly`, `term`, `yearly`, `custom`. |
| `CreateFeeAssignmentRequest` | `fee_structure_id`, `assignment_type` | Assignment type enum: `class`, `section`, `student`. |
| `GenerateInvoicesRequest` | `assignment_id` | Optional student subset and billing period. |
| `CreatePaymentOrderRequest` | `student_id`, `invoice_ids` | Optional `amount_paise` and `idempotency_key`. |
| `VerifyPaymentRequest` | `provider_order_id`, `provider_payment_id`, `signature` | Payment method enum: `online`, `upi`, `card`, `netbanking`, `wallet`, `other`. |
| `CreateOfflinePaymentRequest` | `student_id`, `payment_method`, `allocations` | Method enum: `cash`, `cheque`, `dd`, `bank_transfer`, `upi`, `other`. |
| `OfflinePaymentAllocationRequest` | `invoice_id`, `amount_paise` | Used inside offline payment allocations. |
| `CreateReminderTemplateRequest` | `name`, `code`, `body` | Channel enum: `email`, `sms`, `whatsapp`, `in_app`. |
| `CreateReminderRuleRequest` | `name`, `code` | Trigger type enum: `before_due`, `on_due`, `after_due`, `manual`. |
| `SendReminderRequest` | None | Target using invoice IDs, student, class, section, academic year, or due date filters. |
| `CreateExportRequest` | `export_type` | Export type enum: `collections`, `defaulters`, `dues`, `payment_methods`, `fee_heads`, `offline_payments`, `receipt_register`. |

## Razorpay Frontend Integration Notes

The backend owns payment order creation, verification, webhook processing, allocation, receipts, and ledger updates. The frontend should not mark invoices paid from Razorpay callback data alone.

Frontend sequence:

1. Call `GET /parent/children/{id}/dues`.
2. Let parent choose invoice IDs and amount.
3. Call `POST /parent/payments/orders`.
4. Use `order_id`, `amount_paise`, `currency`, `provider`, and `checkout_url` from `PaymentOrderResponse`.
5. Open Razorpay Checkout with the backend order ID.
6. On success callback, call `POST /parent/payments/verify`.
7. Use `PaymentVerificationResponse.receipt` for the success screen.
8. Re-fetch dues and receipts after verification.

Failure handling:

- If order creation fails, keep parent on the dues screen.
- If Razorpay closes without success, show a non-destructive retry state.
- If verify fails, show payment pending or failed messaging and re-fetch dues/receipts before allowing another attempt.
- The webhook may reconcile payment later, so avoid irreversible frontend-only failure assumptions.

## Suggested API Module Layout

```text
src/lib/api/
  client.ts
  types.ts
  auth.ts
  tenants.ts
  users.ts
  academic.ts
  students.ts
  billing.ts
  payments.ts
  reminders.ts
  reports.ts
  exports.ts
```

Keep each module small:

- `types.ts` contains `ApiResponse`, `PaginationMeta`, shared params, and DTO types.
- Domain modules export plain API functions.
- Feature folders wrap API functions with TanStack Query hooks.
- Zod schemas live with the feature forms that use them.

## Integration Checklist

- Configure `NEXT_PUBLIC_API_BASE_URL`.
- Build Axios client with bearer token injection and envelope unwrapping.
- Add refresh-token retry on `401`.
- Implement login then tenant selection before tenant routes.
- Model paise fields as integers and format as INR only for display.
- Implement blob download helper for CSV/PDF endpoints.
- Use TanStack Query for all reads and mutations.
- Keep table filters aligned with backend query params.
- Use Zod for every request body form.
- Re-check `docs/swagger/openapi.json` after backend changes.
