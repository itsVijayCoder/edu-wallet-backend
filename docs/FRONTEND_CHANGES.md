# Frontend Integration Guide — Recent Backend Changes

This document covers two changes that affect the frontend:

1. **Parents / Guardian-User Link feature** (new endpoints + response fields)
2. **Invoice status priority fix** (`partially_paid` now wins over `overdue`)

---

## 1. Parents / Guardian-User Link Feature

### Overview

Guardians (contacts) can now be linked to parent-role **user accounts**. This lets the frontend show whether a guardian has login access, filter by linked/unlinked, and display a unified "Parents" view.

### New guardian response fields

Every guardian response (list, get, link, unlink) now includes two new fields:

```jsonc
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "...",
  "name": "Riya Sharma",
  "relationship": "mother",
  "phone": "9000000000",
  "email": "riya.sharma@example.test",
  "preferred_language": "en",
  "communication_opt_in": true,
  "address": { ... },
  "user_id": "660e8400-e29b-41d4-a716-446655440000",   // NEW — null if not linked
  "user_status": "active",                               // NEW — null if not linked
  "metadata": {},
  "created_at": "2026-07-10T12:00:00Z",
  "updated_at": "2026-07-10T12:00:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `user_id` | `string (uuid) \| null` | The linked parent user account. `null` means the guardian has no login account. |
| `user_status` | `string \| null` | Status of the linked user (`active`, `inactive`, `pending`). `null` if not linked. |

> Existing guardian endpoints are unchanged apart from these two added fields.

---

### New endpoints

#### 1.1 Link a guardian to a parent user

```
POST /api/v1/admin/guardians/:id/user
```

**Permission:** `guardians.manage`

**Request body:**

```json
{
  "user_id": "660e8400-e29b-41d4-a716-446655440000"
}
```

**Response (200 OK):** Updated `GuardianResponse` with `user_id` and `user_status` populated.

```json
{
  "success": true,
  "data": {
    "id": "550e8400-...",
    "name": "Riya Sharma",
    "user_id": "660e8400-...",
    "user_status": "active",
    ...
  }
}
```

**Rules:**
- The `user_id` must belong to a user with the **`parents`** role. Otherwise you get a `400 PARENT_ROLE_MISSING`.
- Each user can only be linked to **one** guardian per tenant. A second link attempt returns `409 GUARDIAN_USER_ALREADY_LINKED`.
- **Idempotent:** linking the same user that's already linked returns `200` with the current guardian (no error).

**Error codes:**

| HTTP | Code | When |
|------|------|------|
| 400 | `VALIDATION_FAILED` | Missing or invalid `user_id` |
| 400 | `PARENT_ROLE_MISSING` | User does not have the `parents` role |
| 404 | `NOT_FOUND` | Guardian or user not found |
| 409 | `GUARDIAN_USER_ALREADY_LINKED` | User is already linked to another guardian |

---

#### 1.2 Unlink a guardian's user

```
DELETE /api/v1/admin/guardians/:id/user
```

**Permission:** `guardians.manage`

**No request body.**

**Response (200 OK):** Updated `GuardianResponse` with `user_id` and `user_status` set to `null`.

- **Idempotent:** unlinking a guardian with no user link returns `200` with the current guardian (no error).

---

#### 1.3 List a guardian's students (reverse lookup)

```
GET /api/v1/admin/guardians/:id/students
```

**Permission:** `guardians.manage`

Returns all students linked to this guardian.

**Response (200 OK):**

```json
{
  "success": true,
  "data": [
    {
      "student_id": "770e8400-...",
      "admission_number": "ADM-100",
      "first_name": "Aarav",
      "last_name": "Sharma",
      "relationship": "mother",
      "is_primary": true,
      "class_name": "Grade 5",
      "section_name": "Section A",
      "status": "active"
    }
  ]
}
```

---

#### 1.4 Unified Parents endpoint

```
GET /api/v1/admin/parents?linked=true|false&page=1&page_size=20&search=riya
```

**Permission:** `guardians.manage`

Returns a paginated list of guardian summaries, each with their linked students and user info. This is the primary endpoint for the admin "Parents" page.

**Query parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `linked` | `true \| false` | Filter by link status. Omit for all parents. `true` = only linked to a user account. `false` = only unlinked. |
| `search` | `string` | Search by guardian name, email, or phone. |
| `page` | `int` | Page number (default 1). |
| `page_size` | `int` | Items per page (default 20). |
| `sort_by` | `string` | Sort column (default `created_at`). |
| `sort_dir` | `asc \| desc` | Sort direction (default `desc`). |

**Response (200 OK):**

```json
{
  "success": true,
  "data": [
    {
      "guardian_id": "550e8400-...",
      "name": "Riya Sharma",
      "relationship": "mother",
      "phone": "9000000000",
      "email": "riya.sharma@example.test",
      "user_id": "660e8400-...",
      "user_status": "active",
      "linked_students": [
        {
          "student_id": "770e8400-...",
          "admission_number": "ADM-100",
          "first_name": "Aarav",
          "last_name": "Sharma",
          "relationship": "mother",
          "is_primary": true,
          "class_name": "Grade 5",
          "section_name": "Section A",
          "status": "active"
        }
      ]
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 45,
    "total_pages": 3
  }
}
```

**Filtered examples:**
- `GET /api/v1/admin/parents?linked=true` — only guardians with a linked user account
- `GET /api/v1/admin/parents?linked=false` — only guardians without a linked user account
- `GET /api/v1/admin/parents` — all guardians (regardless of link status)

---

### Users list — filter by role

The existing users list endpoint now supports a `role` query parameter:

```
GET /api/v1/admin/users?role=parents&page=1&page_size=20
```

**Response:** Standard paginated `UserResponse[]`. Use this to populate the "select a parent user" dropdown when linking a guardian.

```json
{
  "success": true,
  "data": [
    {
      "id": "660e8400-...",
      "email": "parent@example.com",
      "first_name": "Riya",
      "last_name": "Sharma",
      "status": "active",
      "roles": ["parents"]
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 12,
    "total_pages": 1
  }
}
```

> Only the `parents` role slug is supported for this filter today.

---

### Suggested Frontend UX

1. **Parents list page** — Call `GET /api/v1/admin/parents?linked=...` with a toggle filter (All / Linked / Unlinked). Render each card with guardian info, user status badge, and linked students.

2. **Link action** — On a guardian card with `user_id === null`, show a "Link User" button that opens a picker. Populate the picker from `GET /api/v1/admin/users?role=parents`. On confirm, call `POST /api/v1/admin/guardians/:id/user` with the selected `user_id`.

3. **Unlink action** — On a guardian card with `user_id !== null`, show an "Unlink" button. On confirm, call `DELETE /api/v1/admin/guardians/:id/user`.

4. **Status badges:**
   - `user_id` present + `user_status === "active"` → green badge "Has account"
   - `user_id` present + `user_status === "inactive"` → red badge "Inactive account"
   - `user_id === null` → gray badge "No account"

5. **Error handling for link:**
   - `409 GUARDIAN_USER_ALREADY_LINKED` → "This parent user is already linked to another guardian."
   - `400 PARENT_ROLE_MISSING` → "Selected user does not have the parent role."
   - Linking is idempotent, so re-linking the same user is safe (returns 200, no error).

---

## 2. Invoice Status Priority Fix

### What changed

The priority order for computing invoice status has changed. This affects any frontend code that displays or filters by invoice status.

**Before (old priority):**

```
paid  >  overdue  >  partially_paid  >  issued
```

**After (new priority):**

```
paid  >  partially_paid  >  overdue  >  issued
```

### Practical impact

An invoice with a **partial payment** that is **past its due date** now shows `partially_paid` instead of `overdue`.

| Invoice state | Old status | New status |
|---------------|-----------|------------|
| Fully paid | `paid` | `paid` (unchanged) |
| Partially paid, before due date | `partially_paid` | `partially_paid` (unchanged) |
| Partially paid, **after due date** | `overdue` | `partially_paid` **(changed)** |
| Unpaid, after due date | `overdue` | `overdue` (unchanged) |
| Unpaid, before due date | `issued` | `issued` (unchanged) |

### What the frontend should check

1. **Status badges / colors:** If you have a mapping like `overdue → red`, a partially-paid invoice past its due date will now show as `partially_paid` (likely amber/orange) instead of `overdue` (red). This is the correct behavior — the parent has made a payment.
2. **Filters:** If you filter invoices by `status=overdue`, partially-paid past-due invoices will no longer appear in that filter. They'll appear under `partially_paid` instead.
3. **Dashboards / reports:** The `overdue_paise` amounts in dashboards and defaulter reports are **unaffected** — they are computed by `due_date`, not by status string.
4. **No new endpoints, no new fields.** This is purely a status value change.

---

## Standard Response Envelope

All endpoints use the same envelope (unchanged):

```jsonc
// Success
{ "success": true, "request_id": "...", "data": { ... } }

// Success + pagination
{ "success": true, "data": [ ... ], "meta": { "page": 1, "page_size": 20, "total": 45, "total_pages": 3 } }

// Error
{ "success": false, "error": { "code": "NOT_FOUND", "message": "resource not found" } }

// Validation error
{ "success": false, "error": { "code": "VALIDATION_FAILED", "message": "validation failed", "details": ["user_id is required"] } }
```

---

## Route Summary

| Method | Path | Permission | Description |
|--------|------|-----------|-------------|
| `GET` | `/api/v1/admin/parents` | `guardians.manage` | Unified parents list (with `linked` filter) |
| `GET` | `/api/v1/admin/guardians/:id/students` | `guardians.manage` | Guardian's linked students |
| `POST` | `/api/v1/admin/guardians/:id/user` | `guardians.manage` | Link guardian to parent user |
| `DELETE` | `/api/v1/admin/guardians/:id/user` | `guardians.manage` | Unlink guardian user |
| `GET` | `/api/v1/admin/users?role=parents` | (existing) | List users filtered by parents role |
| `GET` | `/api/v1/admin/guardians` | `guardians.manage` | Guardian list (now includes `user_id`, `user_status`) |
| `GET` | `/api/v1/admin/guardians/:id` | `guardians.manage` | Guardian detail (now includes `user_id`, `user_status`) |

---

## Migration Note

Backend migration `000011_add_guardian_user_link` adds a nullable `user_id` column on the `guardians` table with a unique constraint per tenant. This runs automatically on deploy — no frontend action needed.