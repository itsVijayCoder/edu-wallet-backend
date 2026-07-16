# Parent Portal API Handoff

## Delivered

The parent API now requires a tenant-scoped JWT for a user with the `parents` role. Every parent read and payment action is additionally restricted to students linked through that user's guardian record. Requests for an unrelated child return `404 NOT_FOUND` to prevent student enumeration.

### List linked children

`GET /api/v1/parent/children`

| Parameter | Type | Default | Notes |
| --- | --- | --- | --- |
| `search` | string | — | Matches first name, last name, or admission number. |
| `page` | integer | `1` | One-based page number. |
| `page_size` | integer | `20` | Maximum accepted size is `100`. |

```json
{
  "success": true,
  "data": {
    "rows": [
      {
        "id": "uuid",
        "admission_number": "STU001",
        "first_name": "John",
        "last_name": "Doe",
        "class_name": "Class 5",
        "section_name": "A",
        "status": "active"
      }
    ],
    "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
  }
}
```

An authenticated parent with no guardian link receives an empty `rows` array and zero totals.

### Child dues filters

`GET /api/v1/parent/children/:id/dues`

| Parameter | Values / format |
| --- | --- |
| `search` | Invoice number fragment. |
| `status` | `paid`, `pending`, `partial`, `overdue`, `failed`. |
| `due_from` | `YYYY-MM-DD`, inclusive. |
| `due_to` | `YYYY-MM-DD`, inclusive. |

`pending` maps to the invoice lifecycle state `issued`; `partial` maps to `partially_paid`. The database currently has no failed-invoice lifecycle state, so `status=failed` is accepted and returns no invoices. With no `status`, the existing unpaid-dues behavior is retained. With `status=paid`, paid invoices are returned and the due totals remain zero.

### Receipt filters

`GET /api/v1/parent/receipts`

| Parameter | Values / format |
| --- | --- |
| `search` | Receipt number, admission number, or student name fragment. |
| `status` | `issued` or `cancelled`. |
| `from` | `YYYY-MM-DD`, inclusive issue date. |
| `to` | `YYYY-MM-DD`, inclusive issue date. |
| `page` | Defaults to `1`. |
| `page_size` | Defaults to `20`; maximum `100`. |

Receipt pagination uses the established API envelope: the rows are in `data` and pagination is in the top-level `meta` field.

## Frontend next steps

1. Ensure the logged-in parent has a guardian record linked by an administrator and can select the school tenant before calling these endpoints.
2. Load `/parent/children` first; use each returned `id` for dues and payment views.
3. Send filters only when a value is selected. Dates must be `YYYY-MM-DD` and invalid statuses/dates receive `400 VALIDATION_FAILED`.
4. Treat `404 NOT_FOUND` from child dues, receipts download, or payment actions as no access to that child/receipt; do not retry with another child ID.
5. Refresh generated client types from `/api/v1/docs/openapi.json` if your frontend uses the OpenAPI document.
