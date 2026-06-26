# EduWallet API Tester Flow

This guide gives QA a clean order for testing EduWallet APIs end to end. Follow the sections in order because later APIs depend on IDs created by earlier APIs.

All paths are relative to `/api/v1`.

## 0. Test Setup

Set these shell variables first:

```bash
export BASE_URL="https://eduwallet-api.asthrix.live/api/v1"
export SUPER_EMAIL="admin@eduwallet.in"
export SUPER_PASSWORD="<super-admin-password>"

export RUN_ID="$(date +%Y%m%d%H%M%S)"
export TENANT_SLUG="qa-school-$RUN_ID"
export TEST_DOMAIN="qa-$RUN_ID.eduwallet.test"
```

The API response envelope is:

```json
{
  "success": true,
  "request_id": "...",
  "data": {},
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

Use `jq` to extract IDs:

```bash
jq -r '.data.id'
jq -r '.data[0].id'
jq -r '.data.access_token'
```

Common query params for list APIs:

```text
page=1&page_size=20&sort_by=created_at&sort_dir=desc
```

## 1. Health, Docs, And Auth

### 1.1 Health checks

```bash
curl -sS "$BASE_URL/healthz" | jq
curl -sS "$BASE_URL/readyz" | jq
curl -sS "$BASE_URL/docs/openapi.json" | jq '.info,.paths | keys | length'
```

Expected:

- `/healthz`: `{"status":"ok"}`
- `/readyz`: `postgres.status=up`, `redis.status=up`
- OpenAPI JSON loads successfully.

### 1.2 Login as superadmin

```bash
LOGIN_JSON="$(curl -sS -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$SUPER_EMAIL\",
    \"password\": \"$SUPER_PASSWORD\"
  }")"

echo "$LOGIN_JSON" | jq
export PLATFORM_ACCESS="$(echo "$LOGIN_JSON" | jq -r '.data.access_token')"
export PLATFORM_REFRESH="$(echo "$LOGIN_JSON" | jq -r '.data.refresh_token')"
export SUPER_USER_ID="$(echo "$LOGIN_JSON" | jq -r '.data.user.id')"
```

Expected:

- `success=true`
- `data.user.roles` contains `super_admin`
- both tokens are non-empty.

### 1.3 Refresh token

```bash
curl -sS -X POST "$BASE_URL/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$PLATFORM_REFRESH\"}" | jq
```

Expected: new `access_token` and `refresh_token`.

### 1.4 Password reset endpoints

Use a non-destructive request first:

```bash
curl -sS -X POST "$BASE_URL/auth/forgot-password" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"not-existing-$RUN_ID@example.com\"}" | jq
```

Expected: success response, even if the email does not exist.

Only test `/auth/reset-password` if you have a real reset token from email/logs:

```bash
curl -sS -X POST "$BASE_URL/auth/reset-password" \
  -H "Content-Type: application/json" \
  -d '{"token":"<reset-token>","new_password":"NewStrongPass123!"}' | jq
```

### 1.5 Public registration

In production this should usually be disabled.

```bash
curl -sS -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"public-$RUN_ID@example.com\",
    \"password\": \"PublicPass123!\",
    \"first_name\": \"Public\",
    \"last_name\": \"Tester\"
  }" | jq
```

Expected in production: disabled/forbidden style error unless `AUTH_PUBLIC_REGISTRATION_ENABLED=true`.

### 1.6 Logout

Run this only after all authenticated tests are finished, or use a disposable login token:

```bash
curl -sS -X POST "$BASE_URL/auth/logout" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" | jq
```

## 2. Platform Setup

Use `PLATFORM_ACCESS` for this section.

### 2.1 Create a platform user

```bash
PLATFORM_USER_JSON="$(curl -sS -X POST "$BASE_URL/admin/users" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"platform-admin-$RUN_ID@example.com\",
    \"password\": \"PlatformPass123!\",
    \"first_name\": \"Platform\",
    \"last_name\": \"Admin\",
    \"roles\": [\"super_admin\"]
  }")"

echo "$PLATFORM_USER_JSON" | jq
export PLATFORM_USER_ID="$(echo "$PLATFORM_USER_JSON" | jq -r '.data.id')"
```

Then test list, get, update:

```bash
curl -sS "$BASE_URL/admin/users?page=1&page_size=20" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" | jq

curl -sS "$BASE_URL/admin/users/$PLATFORM_USER_ID" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" | jq

curl -sS -X PUT "$BASE_URL/admin/users/$PLATFORM_USER_ID" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"first_name":"Platform QA","status":"active","roles":["super_admin"]}' | jq
```

Do not delete this user until cleanup.

### 2.2 Create tenant with primary branch

```bash
TENANT_JSON="$(curl -sS -X POST "$BASE_URL/platform/tenants" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"QA Public School $RUN_ID\",
    \"slug\": \"$TENANT_SLUG\",
    \"legal_name\": \"QA Public School Trust\",
    \"domain\": \"$TEST_DOMAIN\",
    \"contact_email\": \"qa-school-$RUN_ID@example.com\",
    \"contact_phone\": \"+919999000001\",
    \"status\": \"active\",
    \"owner_user_id\": \"$SUPER_USER_ID\",
    \"address\": {
      \"line1\": \"1 QA Road\",
      \"line2\": \"Test Campus\",
      \"city\": \"Chennai\",
      \"state\": \"Tamil Nadu\",
      \"postal_code\": \"600001\",
      \"country\": \"India\"
    },
    \"metadata\": {\"test_run\":\"$RUN_ID\"},
    \"branch\": {
      \"name\": \"Main Campus\",
      \"code\": \"MAIN-$RUN_ID\",
      \"contact_email\": \"main-$RUN_ID@example.com\",
      \"contact_phone\": \"+919999000002\",
      \"status\": \"active\",
      \"address\": {
        \"line1\": \"1 QA Road\",
        \"city\": \"Chennai\",
        \"state\": \"Tamil Nadu\",
        \"postal_code\": \"600001\",
        \"country\": \"India\"
      },
      \"metadata\": {\"campus\":\"main\"}
    }
  }")"

echo "$TENANT_JSON" | jq
export TENANT_ID="$(echo "$TENANT_JSON" | jq -r '.data.id')"
export BRANCH_ID="$(echo "$TENANT_JSON" | jq -r '.data.branches[0].id')"
```

Test list, get, update:

```bash
curl -sS "$BASE_URL/platform/tenants?page=1&page_size=20" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" | jq

curl -sS "$BASE_URL/platform/tenants/$TENANT_ID" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" | jq

curl -sS -X PATCH "$BASE_URL/platform/tenants/$TENANT_ID" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"contact_phone":"+919999000099","metadata":{"test_run":"updated"}}' | jq
```

### 2.3 Create another branch

```bash
BRANCH_JSON="$(curl -sS -X POST "$BASE_URL/platform/tenants/$TENANT_ID/branches" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"North Branch\",
    \"code\": \"NORTH-$RUN_ID\",
    \"contact_email\": \"north-$RUN_ID@example.com\",
    \"contact_phone\": \"+919999000003\",
    \"status\": \"active\",
    \"address\": {
      \"line1\": \"2 QA Road\",
      \"city\": \"Chennai\",
      \"state\": \"Tamil Nadu\",
      \"postal_code\": \"600002\",
      \"country\": \"India\"
    },
    \"metadata\": {\"campus\":\"north\"}
  }")"

echo "$BRANCH_JSON" | jq
export BRANCH_2_ID="$(echo "$BRANCH_JSON" | jq -r '.data.id')"
```

There is no branch update/delete API in the current route set.

## 3. Tenant Context And Tenant Users

Tenant-scoped APIs require a token from `/auth/select-tenant`.

```bash
TENANT_TOKEN_JSON="$(curl -sS -X POST "$BASE_URL/auth/select-tenant" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{\"tenant_id\":\"$TENANT_ID\"}")"

echo "$TENANT_TOKEN_JSON" | jq
export TENANT_ACCESS="$(echo "$TENANT_TOKEN_JSON" | jq -r '.data.access_token')"
export TENANT_REFRESH="$(echo "$TENANT_TOKEN_JSON" | jq -r '.data.refresh_token')"
```

Expected: a new tenant-scoped access token. Use `TENANT_ACCESS` for all `/admin/*` tenant APIs.

### 3.1 Get and update current tenant

```bash
curl -sS "$BASE_URL/admin/tenant" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X PATCH "$BASE_URL/admin/tenant" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"legal_name":"QA Public School Updated Trust"}' | jq
```

### 3.2 Create tenant users

Create tenant admin:

```bash
TENANT_ADMIN_JSON="$(curl -sS -X POST "$BASE_URL/admin/tenant/users" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"tenant-admin-$RUN_ID@example.com\",
    \"password\": \"TenantAdmin123!\",
    \"first_name\": \"Tenant\",
    \"last_name\": \"Admin\",
    \"role\": \"admin\"
  }")"

echo "$TENANT_ADMIN_JSON" | jq
export TENANT_ADMIN_USER_ID="$(echo "$TENANT_ADMIN_JSON" | jq -r '.data.user.id')"
```

Create staff:

```bash
curl -sS -X POST "$BASE_URL/admin/tenant/users" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"staff-$RUN_ID@example.com\",
    \"password\": \"StaffPass123!\",
    \"first_name\": \"Staff\",
    \"last_name\": \"Tester\",
    \"role\": \"staff\"
  }" | jq
```

Create parent login user:

```bash
PARENT_USER_JSON="$(curl -sS -X POST "$BASE_URL/admin/tenant/users" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"parent-$RUN_ID@example.com\",
    \"password\": \"ParentPass123!\",
    \"first_name\": \"Parent\",
    \"last_name\": \"Tester\",
    \"role\": \"parents\"
  }")"

echo "$PARENT_USER_JSON" | jq
export PARENT_EMAIL="parent-$RUN_ID@example.com"
export PARENT_PASSWORD="ParentPass123!"
```

## 4. Academic Setup

### 4.1 Academic year

```bash
ACADEMIC_YEAR_JSON="$(curl -sS -X POST "$BASE_URL/admin/academic-years" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Academic Year 2026-27\",
    \"code\": \"AY2026-$RUN_ID\",
    \"start_date\": \"2026-06-01\",
    \"end_date\": \"2027-05-31\",
    \"status\": \"active\",
    \"is_active\": true,
    \"metadata\": {\"test_run\":\"$RUN_ID\"}
  }")"

echo "$ACADEMIC_YEAR_JSON" | jq
export ACADEMIC_YEAR_ID="$(echo "$ACADEMIC_YEAR_JSON" | jq -r '.data.id')"
export ACADEMIC_YEAR_CODE="$(echo "$ACADEMIC_YEAR_JSON" | jq -r '.data.code')"
```

Test list, get, update:

```bash
curl -sS "$BASE_URL/admin/academic-years?page=1&page_size=20" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/academic-years/$ACADEMIC_YEAR_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X PATCH "$BASE_URL/admin/academic-years/$ACADEMIC_YEAR_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"name":"Academic Year 2026-2027","status":"active","is_active":true}' | jq
```

### 4.2 Class

```bash
CLASS_JSON="$(curl -sS -X POST "$BASE_URL/admin/classes" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Grade 1\",
    \"code\": \"G1-$RUN_ID\",
    \"sort_order\": 1,
    \"status\": \"active\",
    \"metadata\": {\"level\":\"primary\"}
  }")"

echo "$CLASS_JSON" | jq
export CLASS_ID="$(echo "$CLASS_JSON" | jq -r '.data.id')"
export CLASS_CODE="$(echo "$CLASS_JSON" | jq -r '.data.code')"
```

Test list, get, update:

```bash
curl -sS "$BASE_URL/admin/classes?page=1&page_size=20" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/classes/$CLASS_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X PATCH "$BASE_URL/admin/classes/$CLASS_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"sort_order":2,"status":"active"}' | jq
```

### 4.3 Section

```bash
SECTION_JSON="$(curl -sS -X POST "$BASE_URL/admin/sections" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"academic_year_id\": \"$ACADEMIC_YEAR_ID\",
    \"class_id\": \"$CLASS_ID\",
    \"branch_id\": \"$BRANCH_ID\",
    \"name\": \"A\",
    \"code\": \"A-$RUN_ID\",
    \"capacity\": 40,
    \"status\": \"active\",
    \"metadata\": {\"room\":\"101\"}
  }")"

echo "$SECTION_JSON" | jq
export SECTION_ID="$(echo "$SECTION_JSON" | jq -r '.data.id')"
export SECTION_CODE="$(echo "$SECTION_JSON" | jq -r '.data.code')"
```

Test list, get, update:

```bash
curl -sS "$BASE_URL/admin/sections?academic_year_id=$ACADEMIC_YEAR_ID&class_id=$CLASS_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/sections/$SECTION_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X PATCH "$BASE_URL/admin/sections/$SECTION_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"capacity":45,"status":"active"}' | jq
```

## 5. Students, Guardians, Links, And Imports

### 5.1 Create guardian

```bash
GUARDIAN_JSON="$(curl -sS -X POST "$BASE_URL/admin/guardians" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Ravi Kumar\",
    \"relationship\": \"father\",
    \"phone\": \"+919999100001\",
    \"whatsapp_phone\": \"+919999100001\",
    \"email\": \"guardian-$RUN_ID@example.com\",
    \"preferred_language\": \"en\",
    \"communication_opt_in\": true,
    \"address\": {
      \"line1\": \"10 Guardian Street\",
      \"city\": \"Chennai\",
      \"state\": \"Tamil Nadu\",
      \"postal_code\": \"600010\",
      \"country\": \"India\"
    },
    \"metadata\": {\"test_run\":\"$RUN_ID\"}
  }")"

echo "$GUARDIAN_JSON" | jq
export GUARDIAN_ID="$(echo "$GUARDIAN_JSON" | jq -r '.data.id')"
```

Test list, get, update:

```bash
curl -sS "$BASE_URL/admin/guardians?search=Ravi" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/guardians/$GUARDIAN_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X PATCH "$BASE_URL/admin/guardians/$GUARDIAN_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"preferred_language":"ta","communication_opt_in":true}' | jq
```

### 5.2 Create student with guardian link

```bash
STUDENT_JSON="$(curl -sS -X POST "$BASE_URL/admin/students" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"academic_year_id\": \"$ACADEMIC_YEAR_ID\",
    \"class_id\": \"$CLASS_ID\",
    \"section_id\": \"$SECTION_ID\",
    \"branch_id\": \"$BRANCH_ID\",
    \"admission_number\": \"ADM-$RUN_ID-001\",
    \"first_name\": \"Ananya\",
    \"last_name\": \"Kumar\",
    \"roll_number\": \"1\",
    \"status\": \"active\",
    \"category\": \"general\",
    \"phone\": \"+919999200001\",
    \"email\": \"student-$RUN_ID@example.com\",
    \"opening_balance_paise\": 0,
    \"address\": {
      \"line1\": \"10 Guardian Street\",
      \"city\": \"Chennai\",
      \"state\": \"Tamil Nadu\",
      \"postal_code\": \"600010\",
      \"country\": \"India\"
    },
    \"metadata\": {\"test_run\":\"$RUN_ID\"},
    \"guardians\": [{
      \"guardian_id\": \"$GUARDIAN_ID\",
      \"relationship\": \"father\",
      \"is_primary\": true
    }]
  }")"

echo "$STUDENT_JSON" | jq
export STUDENT_ID="$(echo "$STUDENT_JSON" | jq -r '.data.id')"
```

Test list, get, update:

```bash
curl -sS "$BASE_URL/admin/students?academic_year_id=$ACADEMIC_YEAR_ID&class_id=$CLASS_ID&section_id=$SECTION_ID&search=Ananya" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/students/$STUDENT_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X PATCH "$BASE_URL/admin/students/$STUDENT_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"roll_number":"01","category":"general","status":"active"}' | jq
```

### 5.3 Link and unlink another guardian

```bash
GUARDIAN_2_JSON="$(curl -sS -X POST "$BASE_URL/admin/guardians" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Meena Kumar\",
    \"relationship\": \"mother\",
    \"phone\": \"+919999100002\",
    \"email\": \"guardian2-$RUN_ID@example.com\",
    \"preferred_language\": \"en\",
    \"communication_opt_in\": true,
    \"address\": {\"city\":\"Chennai\",\"state\":\"Tamil Nadu\",\"country\":\"India\"}
  }")"

export GUARDIAN_2_ID="$(echo "$GUARDIAN_2_JSON" | jq -r '.data.id')"

curl -sS -X POST "$BASE_URL/admin/students/$STUDENT_ID/guardians" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"guardian_id\": \"$GUARDIAN_2_ID\",
    \"relationship\": \"mother\",
    \"is_primary\": false
  }" | jq

curl -sS -X DELETE "$BASE_URL/admin/students/$STUDENT_ID/guardians/$GUARDIAN_2_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq
```

### 5.4 Student import template, preview, commit, list

Download template:

```bash
curl -sS "$BASE_URL/admin/imports/students/template" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -o student_import_template.csv
```

Preview import. The `csv` field contains raw CSV text:

```bash
IMPORT_CSV="admission_number,first_name,last_name,roll_number,academic_year_code,class_code,section_code,category,phone,email,opening_balance_paise,guardian_name,guardian_relationship,guardian_phone,guardian_email,guardian_communication_opt_in
ADM-$RUN_ID-002,Kavin,Rao,2,$ACADEMIC_YEAR_CODE,$CLASS_CODE,$SECTION_CODE,general,+919999200002,student2-$RUN_ID@example.com,0,Latha Rao,mother,+919999100003,guardian3-$RUN_ID@example.com,true"

IMPORT_PREVIEW_JSON="$(jq -n --arg filename "students-$RUN_ID.csv" --arg csv "$IMPORT_CSV" \
  '{filename:$filename,csv:$csv}' | \
  curl -sS -X POST "$BASE_URL/admin/imports/students/preview" \
    -H "Authorization: Bearer $TENANT_ACCESS" \
    -H "Content-Type: application/json" \
    -d @-)"

echo "$IMPORT_PREVIEW_JSON" | jq
export IMPORT_ID="$(echo "$IMPORT_PREVIEW_JSON" | jq -r '.data.import_id')"
```

Commit only if `invalid_rows=0`:

```bash
curl -sS -X POST "$BASE_URL/admin/imports/students/commit" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{\"import_id\":\"$IMPORT_ID\"}" | jq

curl -sS "$BASE_URL/admin/imports?page=1&page_size=20" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq
```

## 6. Billing, Fees, Invoices, Dues, And Ledger

### 6.1 Create fee heads

```bash
TUITION_HEAD_JSON="$(curl -sS -X POST "$BASE_URL/admin/fee-heads" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Tuition Fee\",
    \"code\": \"TUITION-$RUN_ID\",
    \"description\": \"Monthly tuition fee\",
    \"category\": \"tuition\",
    \"status\": \"active\",
    \"taxable\": false,
    \"tax_rate_bps\": 0,
    \"metadata\": {\"test_run\":\"$RUN_ID\"}
  }")"

echo "$TUITION_HEAD_JSON" | jq
export TUITION_HEAD_ID="$(echo "$TUITION_HEAD_JSON" | jq -r '.data.id')"

TRANSPORT_HEAD_JSON="$(curl -sS -X POST "$BASE_URL/admin/fee-heads" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Transport Fee\",
    \"code\": \"TRANSPORT-$RUN_ID\",
    \"category\": \"transport\",
    \"status\": \"active\",
    \"taxable\": false
  }")"

export TRANSPORT_HEAD_ID="$(echo "$TRANSPORT_HEAD_JSON" | jq -r '.data.id')"
```

Test list, get, update:

```bash
curl -sS "$BASE_URL/admin/fee-heads?status=active&category=tuition" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/fee-heads/$TUITION_HEAD_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X PATCH "$BASE_URL/admin/fee-heads/$TUITION_HEAD_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"description":"Updated tuition fee","status":"active"}' | jq
```

### 6.2 Create fee structure

```bash
FEE_STRUCTURE_JSON="$(curl -sS -X POST "$BASE_URL/admin/fee-structures" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"academic_year_id\": \"$ACADEMIC_YEAR_ID\",
    \"name\": \"Grade 1 Term Fee\",
    \"code\": \"G1-TERM-$RUN_ID\",
    \"description\": \"Grade 1 term billing\",
    \"billing_cycle\": \"term\",
    \"status\": \"active\",
    \"allow_partial_payment\": true,
    \"minimum_partial_amount_paise\": 50000,
    \"due_day\": 10,
    \"metadata\": {\"test_run\":\"$RUN_ID\"},
    \"items\": [
      {
        \"fee_head_id\": \"$TUITION_HEAD_ID\",
        \"name\": \"Term Tuition\",
        \"description\": \"Term tuition\",
        \"amount_paise\": 150000,
        \"sort_order\": 1,
        \"optional\": false
      },
      {
        \"fee_head_id\": \"$TRANSPORT_HEAD_ID\",
        \"name\": \"Transport\",
        \"description\": \"Optional transport\",
        \"amount_paise\": 50000,
        \"sort_order\": 2,
        \"optional\": true
      }
    ]
  }")"

echo "$FEE_STRUCTURE_JSON" | jq
export FEE_STRUCTURE_ID="$(echo "$FEE_STRUCTURE_JSON" | jq -r '.data.id')"
```

Test list, get, update:

```bash
curl -sS "$BASE_URL/admin/fee-structures?academic_year_id=$ACADEMIC_YEAR_ID&status=active" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/fee-structures/$FEE_STRUCTURE_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X PATCH "$BASE_URL/admin/fee-structures/$FEE_STRUCTURE_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"description":"Updated Grade 1 term billing","allow_partial_payment":true,"minimum_partial_amount_paise":50000}' | jq
```

### 6.3 Assign fee structure

Use `assignment_type=section` to target this test section.

```bash
ASSIGNMENT_JSON="$(curl -sS -X POST "$BASE_URL/admin/fee-assignments" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"fee_structure_id\": \"$FEE_STRUCTURE_ID\",
    \"assignment_type\": \"section\",
    \"academic_year_id\": \"$ACADEMIC_YEAR_ID\",
    \"section_id\": \"$SECTION_ID\",
    \"effective_from\": \"2026-06-01\",
    \"status\": \"active\",
    \"metadata\": {\"test_run\":\"$RUN_ID\"}
  }")"

echo "$ASSIGNMENT_JSON" | jq
export ASSIGNMENT_ID="$(echo "$ASSIGNMENT_JSON" | jq -r '.data.id')"
```

Other valid assignment shapes:

```json
{"assignment_type":"class","class_id":"<class_id>"}
{"assignment_type":"student","student_id":"<student_id>"}
```

### 6.4 Generate invoices

```bash
INVOICE_GEN_JSON="$(curl -sS -X POST "$BASE_URL/admin/invoices/generate" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"assignment_id\": \"$ASSIGNMENT_ID\",
    \"issue_date\": \"2026-06-05\",
    \"due_date\": \"2026-06-10\",
    \"billing_period_start\": \"2026-06-01\",
    \"billing_period_end\": \"2026-08-31\",
    \"student_ids\": [\"$STUDENT_ID\"],
    \"metadata\": {\"test_run\":\"$RUN_ID\"}
  }")"

echo "$INVOICE_GEN_JSON" | jq
export INVOICE_ID="$(echo "$INVOICE_GEN_JSON" | jq -r '.data.invoices[0].id')"
export INVOICE_AMOUNT="$(echo "$INVOICE_GEN_JSON" | jq -r '.data.invoices[0].balance_amount_paise')"
```

Expected: `generated_count=1`.

Test list and get:

```bash
curl -sS "$BASE_URL/admin/invoices?student_id=$STUDENT_ID&academic_year_id=$ACADEMIC_YEAR_ID&class_id=$CLASS_ID&section_id=$SECTION_ID&status=unpaid" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/invoices/$INVOICE_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq
```

### 6.5 Student ledger and parent dues

```bash
curl -sS "$BASE_URL/admin/students/$STUDENT_ID/ledger" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq
```

Parent dues require a parent tenant token. Login as the parent user created earlier:

```bash
PARENT_LOGIN_JSON="$(curl -sS -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$PARENT_EMAIL\",
    \"password\": \"$PARENT_PASSWORD\"
  }")"

export PARENT_PLATFORM_ACCESS="$(echo "$PARENT_LOGIN_JSON" | jq -r '.data.access_token')"

PARENT_TENANT_JSON="$(curl -sS -X POST "$BASE_URL/auth/select-tenant" \
  -H "Authorization: Bearer $PARENT_PLATFORM_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{\"tenant_id\":\"$TENANT_ID\"}")"

export PARENT_TENANT_ACCESS="$(echo "$PARENT_TENANT_JSON" | jq -r '.data.access_token')"

curl -sS "$BASE_URL/parent/children/$STUDENT_ID/dues" \
  -H "Authorization: Bearer $PARENT_TENANT_ACCESS" | jq
```

QA authorization check: create a second parent user and confirm whether they can access this same student. If access is allowed, log it as an authorization gap unless that is intentionally accepted for the current MVP.

## 7. Payments, Receipts, Webhooks, And Events

### 7.1 Offline payment

```bash
OFFLINE_PAYMENT_JSON="$(curl -sS -X POST "$BASE_URL/admin/offline-payments" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"student_id\": \"$STUDENT_ID\",
    \"payment_method\": \"cash\",
    \"allocations\": [{
      \"invoice_id\": \"$INVOICE_ID\",
      \"amount_paise\": 50000
    }],
    \"received_on\": \"2026-06-06\",
    \"reference_number\": \"CASH-$RUN_ID-001\",
    \"clearance_status\": \"cleared\",
    \"remarks\": \"QA partial offline payment\",
    \"metadata\": {\"test_run\":\"$RUN_ID\"}
  }")"

echo "$OFFLINE_PAYMENT_JSON" | jq
export PAYMENT_ID="$(echo "$OFFLINE_PAYMENT_JSON" | jq -r '.data.payment.id')"
export RECEIPT_ID="$(echo "$OFFLINE_PAYMENT_JSON" | jq -r '.data.receipt.id')"
```

Expected:

- Payment created.
- Receipt created.
- Invoice balance decreases.

### 7.2 List and get payments

```bash
curl -sS "$BASE_URL/admin/payments?student_id=$STUDENT_ID&provider=offline" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/payments/$PAYMENT_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq
```

### 7.3 Receipts

```bash
curl -sS "$BASE_URL/admin/receipts?student_id=$STUDENT_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/receipts/$RECEIPT_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/receipts/$RECEIPT_ID/download" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -o "receipt-$RECEIPT_ID.pdf"

curl -sS "$BASE_URL/parent/receipts?student_id=$STUDENT_ID" \
  -H "Authorization: Bearer $PARENT_TENANT_ACCESS" | jq

curl -sS "$BASE_URL/parent/receipts/$RECEIPT_ID/download" \
  -H "Authorization: Bearer $PARENT_TENANT_ACCESS" \
  -o "parent-receipt-$RECEIPT_ID.pdf"
```

### 7.4 Payment events

```bash
curl -sS "$BASE_URL/admin/payment-events?student_id=$STUDENT_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq
```

### 7.5 Parent online payment order

Use the current invoice balance from `/admin/invoices/{id}`. Do not exceed the invoice balance.

```bash
ORDER_JSON="$(curl -sS -X POST "$BASE_URL/parent/payments/orders" \
  -H "Authorization: Bearer $PARENT_TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"student_id\": \"$STUDENT_ID\",
    \"invoice_ids\": [\"$INVOICE_ID\"],
    \"amount_paise\": 50000,
    \"idempotency_key\": \"checkout-$RUN_ID-$STUDENT_ID\",
    \"metadata\": {\"test_run\":\"$RUN_ID\"}
  }")"

echo "$ORDER_JSON" | jq
export PROVIDER_ORDER_ID="$(echo "$ORDER_JSON" | jq -r '.data.order_id')"
```

If `PAYMENT_PROVIDER=fake`, generate a fake payment signature:

```bash
export FAKE_PAYMENT_ID="pay_$RUN_ID"
export PAYMENT_FAKE_SIGNING_SECRET="test_payment_secret"
export PAYMENT_SIGNATURE="$(printf "%s|%s" "$PROVIDER_ORDER_ID" "$FAKE_PAYMENT_ID" | openssl dgst -sha256 -hmac "$PAYMENT_FAKE_SIGNING_SECRET" | awk '{print $2}')"

curl -sS -X POST "$BASE_URL/parent/payments/verify" \
  -H "Authorization: Bearer $PARENT_TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"provider_order_id\": \"$PROVIDER_ORDER_ID\",
    \"provider_payment_id\": \"$FAKE_PAYMENT_ID\",
    \"signature\": \"$PAYMENT_SIGNATURE\",
    \"payment_method\": \"upi\",
    \"metadata\": {\"test_run\":\"$RUN_ID\"}
  }" | jq
```

If `PAYMENT_PROVIDER=razorpay`, complete the real Razorpay checkout and use the returned `razorpay_order_id`, `razorpay_payment_id`, and `razorpay_signature`.

### 7.6 Razorpay webhook

Use this only when you know the active webhook secret. The service requires:

- `X-Razorpay-Signature`
- `X-Razorpay-Event-Id`
- raw JSON body

Do not reuse an order that was already completed through `/parent/payments/verify`. Create a fresh payment order for webhook testing:

```bash
WEBHOOK_ORDER_JSON="$(curl -sS -X POST "$BASE_URL/parent/payments/orders" \
  -H "Authorization: Bearer $PARENT_TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"student_id\": \"$STUDENT_ID\",
    \"invoice_ids\": [\"$INVOICE_ID\"],
    \"amount_paise\": 50000,
    \"idempotency_key\": \"webhook-$RUN_ID-$STUDENT_ID\",
    \"metadata\": {\"test_run\":\"$RUN_ID\",\"path\":\"webhook\"}
  }")"

export WEBHOOK_PROVIDER_ORDER_ID="$(echo "$WEBHOOK_ORDER_JSON" | jq -r '.data.order_id')"
```

Fake-provider/local signature example:

```bash
WEBHOOK_BODY="$(jq -n \
  --arg order "$WEBHOOK_PROVIDER_ORDER_ID" \
  --arg payment "pay_webhook_$RUN_ID" \
  '{event:"payment.captured",payload:{payment:{entity:{order_id:$order,id:$payment,amount:50000,currency:"INR",status:"captured",method:"upi",captured:true}}}}')"

WEBHOOK_SIGNATURE="$(printf "%s" "$WEBHOOK_BODY" | openssl dgst -sha256 -hmac "$PAYMENT_FAKE_SIGNING_SECRET" | awk '{print $2}')"

curl -sS -X POST "$BASE_URL/webhooks/razorpay" \
  -H "Content-Type: application/json" \
  -H "X-Razorpay-Event-Id: evt_$RUN_ID" \
  -H "X-Razorpay-Signature: $WEBHOOK_SIGNATURE" \
  -d "$WEBHOOK_BODY" | jq
```

Expected:

- `processed` for a successful captured event.
- duplicate `X-Razorpay-Event-Id` should not double-apply payment.

## 8. Operations, Reminders, Dashboard, Reports, And Exports

### 8.1 Reminder template

```bash
TEMPLATE_JSON="$(curl -sS -X POST "$BASE_URL/admin/reminder-templates" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Due Reminder\",
    \"code\": \"DUE-$RUN_ID\",
    \"channel\": \"email\",
    \"subject\": \"Fee reminder\",
    \"body\": \"Dear guardian, {{student_name}} has due amount {{amount_due}} for invoice {{invoice_number}}.\",
    \"tone\": \"polite\",
    \"status\": \"active\",
    \"metadata\": {\"test_run\":\"$RUN_ID\"}
  }")"

echo "$TEMPLATE_JSON" | jq
export TEMPLATE_ID="$(echo "$TEMPLATE_JSON" | jq -r '.data.id')"
```

Test list, get, update:

```bash
curl -sS "$BASE_URL/admin/reminder-templates?channel=email&status=active" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/reminder-templates/$TEMPLATE_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X PATCH "$BASE_URL/admin/reminder-templates/$TEMPLATE_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"subject":"Updated fee reminder","tone":"formal"}' | jq
```

### 8.2 Reminder rule

```bash
RULE_JSON="$(curl -sS -X POST "$BASE_URL/admin/reminder-rules" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"template_id\": \"$TEMPLATE_ID\",
    \"name\": \"Before Due Reminder\",
    \"code\": \"BEFORE-DUE-$RUN_ID\",
    \"channel\": \"email\",
    \"trigger_type\": \"before_due\",
    \"offset_days\": 2,
    \"target_statuses\": [\"unpaid\", \"partial\", \"overdue\"],
    \"status\": \"active\",
    \"max_attempts\": 3,
    \"metadata\": {\"test_run\":\"$RUN_ID\"}
  }")"

echo "$RULE_JSON" | jq
export RULE_ID="$(echo "$RULE_JSON" | jq -r '.data.id')"
```

Test list, get, update:

```bash
curl -sS "$BASE_URL/admin/reminder-rules?channel=email&trigger_type=before_due&status=active" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/reminder-rules/$RULE_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X PATCH "$BASE_URL/admin/reminder-rules/$RULE_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"offset_days":1,"max_attempts":2}' | jq
```

### 8.3 Send reminders and logs

```bash
SEND_REMINDER_JSON="$(curl -sS -X POST "$BASE_URL/admin/reminders/send" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"rule_id\": \"$RULE_ID\",
    \"template_id\": \"$TEMPLATE_ID\",
    \"channel\": \"email\",
    \"invoice_ids\": [\"$INVOICE_ID\"],
    \"process_now\": true,
    \"metadata\": {\"test_run\":\"$RUN_ID\"}
  }")"

echo "$SEND_REMINDER_JSON" | jq

curl -sS "$BASE_URL/admin/reminder-logs?student_id=$STUDENT_ID&invoice_id=$INVOICE_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq
```

### 8.4 Dashboard

```bash
curl -sS "$BASE_URL/admin/dashboard?as_of=2026-06-26" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq
```

### 8.5 Reports

```bash
curl -sS "$BASE_URL/admin/reports/collections?from=2026-06-01&to=2026-06-30" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/reports/defaulters?as_of=2026-06-26&academic_year_id=$ACADEMIC_YEAR_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/reports/dues?as_of=2026-06-26&academic_year_id=$ACADEMIC_YEAR_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/reports/fee-heads?from=2026-06-01&to=2026-06-30" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/reports/payment-methods?from=2026-06-01&to=2026-06-30" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/reports/offline-payments?from=2026-06-01&to=2026-06-30" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq
```

### 8.6 Exports

```bash
EXPORT_JSON="$(curl -sS -X POST "$BASE_URL/admin/exports" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"export_type\": \"collections\",
    \"format\": \"csv\",
    \"from\": \"2026-06-01\",
    \"to\": \"2026-06-30\",
    \"metadata\": {\"test_run\":\"$RUN_ID\"}
  }")"

echo "$EXPORT_JSON" | jq
export EXPORT_ID="$(echo "$EXPORT_JSON" | jq -r '.data.id')"

curl -sS "$BASE_URL/admin/exports?export_type=collections" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/exports/$EXPORT_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS "$BASE_URL/admin/exports/$EXPORT_ID/download" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -o "export-$EXPORT_ID.csv"
```

Also test these `export_type` values:

```text
collections
defaulters
dues
payment_methods
fee_heads
offline_payments
receipt_register
```

## 9. Delete And Cleanup Tests

Do delete tests on disposable records that have no downstream dependencies.

### 9.1 Delete disposable user

```bash
DELETE_USER_JSON="$(curl -sS -X POST "$BASE_URL/admin/users" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"delete-user-$RUN_ID@example.com\",
    \"password\": \"DeletePass123!\",
    \"first_name\": \"Delete\",
    \"last_name\": \"User\",
    \"roles\": [\"admin\"]
  }")"

DELETE_USER_ID="$(echo "$DELETE_USER_JSON" | jq -r '.data.id')"

curl -sS -X DELETE "$BASE_URL/admin/users/$DELETE_USER_ID" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" | jq
```

### 9.2 Delete disposable academic records

Create and delete in reverse dependency order:

```bash
TMP_YEAR_JSON="$(curl -sS -X POST "$BASE_URL/admin/academic-years" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\":\"Delete Year\",
    \"code\":\"DEL-YEAR-$RUN_ID\",
    \"start_date\":\"2026-06-01\",
    \"end_date\":\"2027-05-31\",
    \"status\":\"inactive\",
    \"is_active\":false
  }")"
TMP_YEAR_ID="$(echo "$TMP_YEAR_JSON" | jq -r '.data.id')"

TMP_CLASS_JSON="$(curl -sS -X POST "$BASE_URL/admin/classes" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Delete Class\",\"code\":\"DEL-CLASS-$RUN_ID\",\"status\":\"active\"}")"
TMP_CLASS_ID="$(echo "$TMP_CLASS_JSON" | jq -r '.data.id')"

TMP_SECTION_JSON="$(curl -sS -X POST "$BASE_URL/admin/sections" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"academic_year_id\":\"$TMP_YEAR_ID\",
    \"class_id\":\"$TMP_CLASS_ID\",
    \"name\":\"Delete Section\",
    \"code\":\"DEL-SEC-$RUN_ID\",
    \"status\":\"active\"
  }")"
TMP_SECTION_ID="$(echo "$TMP_SECTION_JSON" | jq -r '.data.id')"

curl -sS -X DELETE "$BASE_URL/admin/sections/$TMP_SECTION_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X DELETE "$BASE_URL/admin/classes/$TMP_CLASS_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X DELETE "$BASE_URL/admin/academic-years/$TMP_YEAR_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq
```

### 9.3 Delete disposable guardian and student

```bash
TMP_GUARDIAN_JSON="$(curl -sS -X POST "$BASE_URL/admin/guardians" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Delete Guardian\",\"email\":\"delete-guardian-$RUN_ID@example.com\"}")"
TMP_GUARDIAN_ID="$(echo "$TMP_GUARDIAN_JSON" | jq -r '.data.id')"

TMP_STUDENT_JSON="$(curl -sS -X POST "$BASE_URL/admin/students" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"academic_year_id\":\"$ACADEMIC_YEAR_ID\",
    \"class_id\":\"$CLASS_ID\",
    \"section_id\":\"$SECTION_ID\",
    \"admission_number\":\"DEL-STU-$RUN_ID\",
    \"first_name\":\"Delete\",
    \"last_name\":\"Student\",
    \"status\":\"active\",
    \"category\":\"general\"
  }")"
TMP_STUDENT_ID="$(echo "$TMP_STUDENT_JSON" | jq -r '.data.id')"

curl -sS -X DELETE "$BASE_URL/admin/students/$TMP_STUDENT_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X DELETE "$BASE_URL/admin/guardians/$TMP_GUARDIAN_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq
```

### 9.4 Delete disposable fee records

Delete fee structure before fee head:

```bash
TMP_HEAD_JSON="$(curl -sS -X POST "$BASE_URL/admin/fee-heads" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Delete Fee Head\",\"code\":\"DEL-FEE-$RUN_ID\",\"category\":\"custom\",\"status\":\"active\"}")"
TMP_HEAD_ID="$(echo "$TMP_HEAD_JSON" | jq -r '.data.id')"

TMP_STRUCTURE_JSON="$(curl -sS -X POST "$BASE_URL/admin/fee-structures" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"academic_year_id\":\"$ACADEMIC_YEAR_ID\",
    \"name\":\"Delete Fee Structure\",
    \"code\":\"DEL-FS-$RUN_ID\",
    \"billing_cycle\":\"one_time\",
    \"status\":\"active\",
    \"allow_partial_payment\":false,
    \"items\":[{
      \"fee_head_id\":\"$TMP_HEAD_ID\",
      \"amount_paise\":1000,
      \"optional\":false
    }]
  }")"
TMP_STRUCTURE_ID="$(echo "$TMP_STRUCTURE_JSON" | jq -r '.data.id')"

curl -sS -X DELETE "$BASE_URL/admin/fee-structures/$TMP_STRUCTURE_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X DELETE "$BASE_URL/admin/fee-heads/$TMP_HEAD_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq
```

### 9.5 Delete disposable reminder records

Delete reminder rule before template:

```bash
TMP_TEMPLATE_JSON="$(curl -sS -X POST "$BASE_URL/admin/reminder-templates" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Delete Template\",\"code\":\"DEL-TPL-$RUN_ID\",\"channel\":\"email\",\"body\":\"Delete body\",\"status\":\"active\"}")"
TMP_TEMPLATE_ID="$(echo "$TMP_TEMPLATE_JSON" | jq -r '.data.id')"

TMP_RULE_JSON="$(curl -sS -X POST "$BASE_URL/admin/reminder-rules" \
  -H "Authorization: Bearer $TENANT_ACCESS" \
  -H "Content-Type: application/json" \
  -d "{
    \"template_id\":\"$TMP_TEMPLATE_ID\",
    \"name\":\"Delete Rule\",
    \"code\":\"DEL-RULE-$RUN_ID\",
    \"channel\":\"email\",
    \"trigger_type\":\"manual\",
    \"status\":\"active\"
  }")"
TMP_RULE_ID="$(echo "$TMP_RULE_JSON" | jq -r '.data.id')"

curl -sS -X DELETE "$BASE_URL/admin/reminder-rules/$TMP_RULE_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq

curl -sS -X DELETE "$BASE_URL/admin/reminder-templates/$TMP_TEMPLATE_ID" \
  -H "Authorization: Bearer $TENANT_ACCESS" | jq
```

## 10. Negative And Permission Tests

Run these once the happy path works.

### 10.1 Missing token

```bash
curl -sS "$BASE_URL/admin/tenant" | jq
```

Expected: unauthorized/forbidden error.

### 10.2 Login token without tenant context

```bash
curl -sS "$BASE_URL/admin/tenant" \
  -H "Authorization: Bearer $PLATFORM_ACCESS" | jq
```

Expected: `TENANT_REQUIRED`.

### 10.3 Invalid tenant access

```bash
curl -sS -X POST "$BASE_URL/auth/select-tenant" \
  -H "Authorization: Bearer $PARENT_PLATFORM_ACCESS" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"00000000-0000-0000-0000-000000000000"}' | jq
```

Expected: tenant access denied.

### 10.4 Duplicate and validation checks

Try each of these:

- Create tenant with duplicate `slug`.
- Create branch with duplicate `code` under same tenant.
- Create academic year with duplicate `code`.
- Create class with duplicate `code`.
- Create student with duplicate `admission_number`.
- Create fee head with duplicate `code`.
- Create invoice twice with same assignment, student, billing period, and due date. Expected second call should skip or avoid duplicate generation.
- Pay more than invoice balance. Expected: validation/conflict error.
- Partial pay below `minimum_partial_amount_paise`. Expected: validation error.

## 11. Endpoint Coverage Matrix

### Public and auth

| Method | Path | Flow section |
| --- | --- | --- |
| GET | `/healthz` | 1.1 |
| GET | `/readyz` | 1.1 |
| GET | `/docs` | Manual browser check |
| GET | `/docs/api-test` | Browser tester guide |
| GET | `/docs/openapi.json` | 1.1 |
| GET | `/docs/swagger.json` | 1.1 |
| POST | `/auth/login` | 1.2 |
| POST | `/auth/register` | 1.5 |
| POST | `/auth/refresh` | 1.3 |
| POST | `/auth/select-tenant` | 3 |
| POST | `/auth/logout` | 1.6 |
| POST | `/auth/forgot-password` | 1.4 |
| POST | `/auth/reset-password` | 1.4 |
| POST | `/webhooks/razorpay` | 7.6 |

### Platform and users

| Method | Path | Flow section |
| --- | --- | --- |
| POST | `/admin/users` | 2.1, 9.1 |
| GET | `/admin/users` | 2.1 |
| GET | `/admin/users/{id}` | 2.1 |
| PUT | `/admin/users/{id}` | 2.1 |
| DELETE | `/admin/users/{id}` | 9.1 |
| POST | `/platform/tenants` | 2.2 |
| GET | `/platform/tenants` | 2.2 |
| GET | `/platform/tenants/{id}` | 2.2 |
| PATCH | `/platform/tenants/{id}` | 2.2 |
| POST | `/platform/tenants/{id}/branches` | 2.3 |

### Tenant admin and academics

| Method | Path | Flow section |
| --- | --- | --- |
| GET | `/admin/tenant` | 3.1 |
| PATCH | `/admin/tenant` | 3.1 |
| POST | `/admin/tenant/users` | 3.2 |
| POST | `/admin/academic-years` | 4.1, 9.2 |
| GET | `/admin/academic-years` | 4.1 |
| GET | `/admin/academic-years/{id}` | 4.1 |
| PATCH | `/admin/academic-years/{id}` | 4.1 |
| DELETE | `/admin/academic-years/{id}` | 9.2 |
| POST | `/admin/classes` | 4.2, 9.2 |
| GET | `/admin/classes` | 4.2 |
| GET | `/admin/classes/{id}` | 4.2 |
| PATCH | `/admin/classes/{id}` | 4.2 |
| DELETE | `/admin/classes/{id}` | 9.2 |
| POST | `/admin/sections` | 4.3, 9.2 |
| GET | `/admin/sections` | 4.3 |
| GET | `/admin/sections/{id}` | 4.3 |
| PATCH | `/admin/sections/{id}` | 4.3 |
| DELETE | `/admin/sections/{id}` | 9.2 |
| POST | `/admin/guardians` | 5.1, 9.3 |
| GET | `/admin/guardians` | 5.1 |
| GET | `/admin/guardians/{id}` | 5.1 |
| PATCH | `/admin/guardians/{id}` | 5.1 |
| DELETE | `/admin/guardians/{id}` | 9.3 |
| POST | `/admin/students` | 5.2, 9.3 |
| GET | `/admin/students` | 5.2 |
| GET | `/admin/students/{id}` | 5.2 |
| PATCH | `/admin/students/{id}` | 5.2 |
| DELETE | `/admin/students/{id}` | 9.3 |
| POST | `/admin/students/{id}/guardians` | 5.3 |
| DELETE | `/admin/students/{id}/guardians/{guardian_id}` | 5.3 |
| GET | `/admin/imports` | 5.4 |
| GET | `/admin/imports/students/template` | 5.4 |
| POST | `/admin/imports/students/preview` | 5.4 |
| POST | `/admin/imports/students/commit` | 5.4 |

### Billing and payments

| Method | Path | Flow section |
| --- | --- | --- |
| POST | `/admin/fee-heads` | 6.1, 9.4 |
| GET | `/admin/fee-heads` | 6.1 |
| GET | `/admin/fee-heads/{id}` | 6.1 |
| PATCH | `/admin/fee-heads/{id}` | 6.1 |
| DELETE | `/admin/fee-heads/{id}` | 9.4 |
| POST | `/admin/fee-structures` | 6.2, 9.4 |
| GET | `/admin/fee-structures` | 6.2 |
| GET | `/admin/fee-structures/{id}` | 6.2 |
| PATCH | `/admin/fee-structures/{id}` | 6.2 |
| DELETE | `/admin/fee-structures/{id}` | 9.4 |
| POST | `/admin/fee-assignments` | 6.3 |
| POST | `/admin/invoices/generate` | 6.4 |
| GET | `/admin/invoices` | 6.4 |
| GET | `/admin/invoices/{id}` | 6.4 |
| GET | `/admin/students/{id}/ledger` | 6.5 |
| GET | `/parent/children/{id}/dues` | 6.5 |
| POST | `/admin/offline-payments` | 7.1 |
| GET | `/admin/payments` | 7.2 |
| GET | `/admin/payments/{id}` | 7.2 |
| GET | `/admin/receipts` | 7.3 |
| GET | `/admin/receipts/{id}` | 7.3 |
| GET | `/admin/receipts/{id}/download` | 7.3 |
| GET | `/admin/payment-events` | 7.4 |
| POST | `/parent/payments/orders` | 7.5 |
| POST | `/parent/payments/verify` | 7.5 |
| GET | `/parent/receipts` | 7.3 |
| GET | `/parent/receipts/{id}/download` | 7.3 |

### Operations

| Method | Path | Flow section |
| --- | --- | --- |
| POST | `/admin/reminder-templates` | 8.1, 9.5 |
| GET | `/admin/reminder-templates` | 8.1 |
| GET | `/admin/reminder-templates/{id}` | 8.1 |
| PATCH | `/admin/reminder-templates/{id}` | 8.1 |
| DELETE | `/admin/reminder-templates/{id}` | 9.5 |
| POST | `/admin/reminder-rules` | 8.2, 9.5 |
| GET | `/admin/reminder-rules` | 8.2 |
| GET | `/admin/reminder-rules/{id}` | 8.2 |
| PATCH | `/admin/reminder-rules/{id}` | 8.2 |
| DELETE | `/admin/reminder-rules/{id}` | 9.5 |
| POST | `/admin/reminders/send` | 8.3 |
| GET | `/admin/reminder-logs` | 8.3 |
| GET | `/admin/dashboard` | 8.4 |
| GET | `/admin/reports/collections` | 8.5 |
| GET | `/admin/reports/defaulters` | 8.5 |
| GET | `/admin/reports/dues` | 8.5 |
| GET | `/admin/reports/fee-heads` | 8.5 |
| GET | `/admin/reports/payment-methods` | 8.5 |
| GET | `/admin/reports/offline-payments` | 8.5 |
| POST | `/admin/exports` | 8.6 |
| GET | `/admin/exports` | 8.6 |
| GET | `/admin/exports/{id}` | 8.6 |
| GET | `/admin/exports/{id}/download` | 8.6 |

## 12. Recommended Pass Criteria

A full API pass should confirm:

- All health and docs endpoints respond.
- Superadmin can log in, refresh, create platform tenants, and select tenant.
- Tenant token is required for tenant-scoped APIs.
- Tenant admin user can execute tenant admin flows.
- Staff/parent users cannot access APIs outside their permissions.
- Academic setup supports create, list, get, update, and delete on disposable records.
- Student guardian link supports create, list through student detail, link, unlink, and import.
- Billing creates fee heads, structures, assignments, invoices, dues, and ledger entries.
- Offline payment updates invoice balance, creates payment, receipt, ledger entries, and payment events.
- Parent payment order and verification work for the configured provider.
- Razorpay webhook signature and idempotency behavior work.
- Reports/dashboard reflect invoice and payment data.
- Export CSV files download and contain expected rows.
- Duplicate, invalid UUID, invalid date, missing token, and wrong permission cases return structured errors.
