# EduWallet Frontend UI Guide

This guide turns the product plan into an implementation-ready frontend direction for a Next.js application. It is written for a frontend developer building the EduWallet admin, platform, and parent payment experiences.

Source product docs:

- `docs/EduWallet_Product_Plan.md`
- `docs/PRODUCT_PLAN.md`
- `docs/PROJECT_FLOW.md`

## Frontend Stack

Use this stack consistently across the app:

| Area | Choice | Usage |
| --- | --- | --- |
| Framework | Next.js App Router | Route groups by role and workflow. |
| UI system | shadcn/ui | Source-code components, composed into domain screens. |
| Styling | Tailwind CSS | Semantic tokens, responsive utilities, no one-off CSS themes. |
| Server state | TanStack Query | API reads, mutations, cache invalidation, optimistic table refresh. |
| Client state | Zustand | Auth session, selected tenant, sidebar/table preferences, transient UI state. |
| Forms | React Hook Form | All create/edit/filter forms. |
| Validation | Zod | Shared frontend schemas for forms and API payloads. |
| HTTP | Axios | Typed API client, auth interceptors, file downloads. |
| Reports | TanStack Table | Paginated, sortable, filterable admin/report tables. |
| Charts | Recharts through shadcn Chart | Dashboard trend and breakdown visualizations. |

## Product Experience

EduWallet is a finance-first SaaS dashboard, not a marketing site. The first authenticated screen should be useful immediately: dues, collections, defaulters, failed payments, and actions that move money workflows forward.

Core UX goals:

- Parents should see what they owe, trust the amount, pay quickly, and download receipts.
- School admins should understand collection health within seconds.
- Accountants should search students, record payments, reconcile receipts, and export reports with minimal page hopping.
- Platform super admins should manage tenants, tenant health, and onboarding status without entering school-specific workflows unless needed.

## UI Personality

The interface should feel modern, advanced, and professional, but still operational. Prefer compact, high-clarity layouts over decorative layouts.

Visual direction:

- Clean SaaS dashboard with strong information hierarchy.
- Mobile-first layouts that still become dense and efficient on desktop.
- Finance-grade trust: restrained color, visible audit context, clear statuses, predictable tables.
- Use a neutral base with controlled accent colors for money, overdue, success, failure, and warnings.
- Avoid oversized marketing heroes, heavy gradients, floating decorative cards, and purely illustrative dashboard filler.

Suggested token direction:

| Token | Direction |
| --- | --- |
| Background | Neutral app background with white or near-white surfaces. |
| Primary | Professional blue or teal for primary actions and active navigation. |
| Success | Used only for paid, settled, cleared, and successful states. |
| Warning | Used for due soon, pending, and partial states. |
| Destructive | Used for overdue, failed, bounced, cancelled, and delete actions. |
| Radius | Keep cards, tables, buttons, and inputs at 8px or less unless the chosen shadcn preset differs. |
| Density | Use comfortable mobile spacing, then tighter desktop tables and dashboards. |

## shadcn/ui Rules

Use shadcn/ui components before custom markup. Compose domain screens from primitives:

| Need | Component direction |
| --- | --- |
| App navigation | `Sidebar`, `Breadcrumb`, `DropdownMenu`, `Command`, `Sheet` on mobile. |
| Forms | `Form`, `FieldGroup`, `Field`, `Input`, `Select`, `Textarea`, `Checkbox`, `Switch`, `Calendar`, `Popover`. |
| Data tables | TanStack Table plus shadcn `Table`, `Checkbox`, `DropdownMenu`, `Badge`, `Pagination`. |
| Detail panels | `Sheet` for contextual edit/view, `Dialog` for blocking create flows, `Drawer` for mobile payment details. |
| Alerts | shadcn `Alert`; do not create custom alert boxes. |
| Empty states | shadcn `Empty`; include the next best action. |
| Loading | `Skeleton` and `Spinner`; avoid custom pulse blocks. |
| Status | `Badge` variants and semantic tokens, not raw color spans. |
| Charts | shadcn `Chart` wrappers around Recharts. |
| Confirmations | `AlertDialog` for destructive actions. |
| Toasts | `sonner` for mutation feedback. |

Implementation rules:

- Use `gap-*` for spacing, not `space-x-*` or `space-y-*`.
- Use semantic colors such as `bg-background`, `text-muted-foreground`, `border-border`, `bg-primary`.
- Use `cn()` for conditional class names.
- Icons in buttons should come from the configured icon library and be passed as icon components, not string keys.
- Every `Dialog`, `Sheet`, and `Drawer` needs a title for accessibility.
- Every `Avatar` needs an `AvatarFallback`.
- Use full `Card` composition: `CardHeader`, `CardTitle`, `CardDescription`, `CardContent`, `CardFooter`.

## Information Architecture

Use route groups by audience. Keep the platform app, tenant admin app, and parent app visually related but workflow-specific.

```text
app/
  (public)/
    login/
    forgot-password/
    reset-password/
  (platform)/
    platform/
      tenants/
      tenants/[id]/
      users/
      support/
  (admin)/
    admin/
      dashboard/
      academic-years/
      classes/
      sections/
      students/
      students/[id]/
      guardians/
      imports/
      fee-heads/
      fee-structures/
      fee-assignments/
      invoices/
      payments/
      offline-payments/
      receipts/
      reminders/templates/
      reminders/rules/
      reminders/logs/
      reports/
      exports/
      settings/tenant/
      settings/users/
  (parent)/
    parent/
      children/
      children/[id]/dues/
      payments/checkout/
      receipts/
```

## App Shell

Desktop admin shell:

- Persistent left sidebar with primary modules.
- Top bar with tenant selector, academic year selector, search, notifications, and user menu.
- Breadcrumbs on every workflow page.
- Page header with title, secondary context, and primary action on the right.
- Tables and forms should live directly in the page content, not nested inside decorative cards.

Mobile shell:

- Top app bar with tenant name, search icon, and menu icon.
- Sidebar becomes a `Sheet`.
- Page actions collapse into a segmented action row or overflow menu.
- Tables become responsive list rows with the most important values visible first.
- Parent payment flow should be single-column and sticky-bottom for primary pay actions.

## Navigation

Tenant admin modules:

| Module | Main jobs |
| --- | --- |
| Dashboard | Collection health, overdue amount, defaulters, method mix, recent events. |
| Academic Setup | Academic years, classes, sections. |
| Students | Student directory, guardian links, student profile, ledger. |
| Imports | CSV template, preview, errors, commit history. |
| Fees | Fee heads, fee structures, assignments, invoice generation. |
| Payments | Online payments, offline entries, payment events. |
| Receipts | Receipt register and PDF downloads. |
| Reminders | Templates, rules, manual send, logs. |
| Reports | Collections, defaulters, dues, fee heads, payment methods, offline payments. |
| Exports | Export jobs and CSV downloads. |
| Settings | Tenant profile and tenant users. |

Platform modules:

| Module | Main jobs |
| --- | --- |
| Tenants | Create schools, branches, statuses, owner assignment. |
| Users | Platform/admin user management. |
| Tenant Detail | Tenant health, branches, contact profile, billing status placeholders. |
| Support | Future payment issue lookup and tenant impersonation entry points. |

Parent modules:

| Module | Main jobs |
| --- | --- |
| Children | Child cards with due amount and receipt summary. |
| Dues | Invoice breakdown by child, due dates, partial payment constraints. |
| Checkout | Razorpay order creation, gateway handoff, verification, receipt confirmation. |
| Receipts | Receipt list and PDF downloads. |

## Screen Specs

### Login And Tenant Selection

Use a focused auth layout with school-safe language and no marketing hero.

Required states:

- Login with email and password.
- Forgot password and reset password.
- If login returns multiple tenant memberships, show a tenant selection screen.
- Store the non-tenant access token after login, then replace it with the tenant access token after `POST /api/v1/auth/select-tenant`.
- Show role and tenant context in the user menu after tenant selection.

Primary components:

- `Card` for the form container.
- `FieldGroup` and `Field` for inputs.
- `Alert` for account inactive, invalid credentials, or registration disabled.
- `Button` with `Spinner` while pending.

### Tenant Admin Dashboard

The dashboard is the daily operations landing page.

Above the fold on mobile:

- Today collection.
- Month collection.
- Total due.
- Overdue amount.
- Defaulter count.
- Quick action: Record offline payment or Send reminder.

Desktop layout:

- KPI strip with 4 to 6 compact metric cards.
- Payment method breakdown as a donut or bar chart.
- Recent payment events as a dense list.
- Defaulter preview table with a link to the full report.

Use Recharts for:

- Payment method breakdown.
- Collection trend when a trend endpoint is added later.
- Dues vs paid comparison if derived from reports.

### Academic Setup

Academic setup should be fast and table-first.

Pages:

- Academic years.
- Classes.
- Sections.

UX details:

- Use data tables with status filter, search, sort, pagination.
- Create/edit in `Sheet` on desktop and `Drawer` on mobile.
- Show active academic year as a `Badge`.
- Block destructive deletes with `AlertDialog`.
- Sections need linked academic year and class selectors.

### Students And Guardians

Student directory should be optimized for search and finance follow-up.

Student list:

- Search by name, admission number, parent phone, or email where backend supports search.
- Filters: academic year, class, section, status.
- Columns: admission number, student, class/section, guardian, category, status, balance summary if available.
- Row actions: view profile, edit, ledger, link guardian.

Student profile:

- Header: student identity, class/section, status, guardian contact.
- Tabs: Overview, Guardians, Ledger, Invoices, Receipts.
- Ledger uses a chronological table with debit, credit, and running balance.

Student import:

- Download template action.
- Upload CSV through drag/drop or file picker.
- Preview screen with valid/invalid rows.
- Error table grouped by row number and field.
- Commit button enabled only when invalid row count is zero.

### Fee Setup And Billing

Fee workflows should make amount composition clear.

Fee heads:

- Table by code, name, category, taxable status, tax rate, status.
- Use category filter and search.

Fee structures:

- Show billing cycle, academic year, partial-payment policy, due day, total amount.
- Structure editor uses an itemized fee-head table.
- Amount inputs should display INR while submitting paise.
- Validate that at least one item exists.

Assignments and invoice generation:

- Assignment form should use a segmented control for assignment type: class, section, student.
- Show only relevant selectors based on assignment type.
- Invoice generation should clearly show issue date, due date, billing period, selected students, generated count, skipped count.
- Make idempotent behavior visible: repeated generation may skip existing invoices.

### Parent Dues And Checkout

This is the most important mobile flow.

Dues screen:

- Show school/tenant name, child name, class/section, total due, overdue amount.
- Invoice cards should show invoice number, due date, fee heads, total, paid, balance, status.
- Payment selection must respect `allow_partial`, `allow_partial_payment`, and `minimum_payable_paise`.
- Multi-invoice payment is allowed when paying full balances. Partial payment should be one invoice at a time.

Checkout flow:

1. Parent selects invoice(s) and amount.
2. Frontend calls `POST /api/v1/parent/payments/orders`.
3. Open Razorpay Checkout or provider URL from the response.
4. On gateway success, call `POST /api/v1/parent/payments/verify`.
5. Show receipt success state and offer PDF download.
6. If verification fails, show retry and "payment pending" support text.

Mobile UX:

- Sticky bottom pay bar with selected amount.
- Fee breakdown in accordion sections.
- Receipts and support actions after payment.

### Payments And Receipts

Payments table:

- Filters: date range, status, method, provider, student, search.
- Columns: paid at, student, method, provider, status, amount, applied amount, receipt, reconciliation.
- Detail `Sheet` shows allocations and gateway IDs.

Offline payment:

- Create as a dedicated form or modal from student profile and payments page.
- Allocation table by invoice.
- Method-specific fields for cheque/DD/bank transfer.
- Show clearance status clearly.

Receipts:

- Receipt register table with date, student, receipt number, amount, method, status.
- Download PDF as a blob through Axios.
- Parent receipts use the same table pattern but scoped to the parent user.

### Reminders

Reminder templates:

- Template editor with channel, subject, body, tone, status.
- Preview panel with sample student, amount, due date, and school name.

Reminder rules:

- Rule table with trigger type, offset days, target statuses, channel, status.
- Rule form validates offset and max attempts.

Manual send:

- Filter targets by invoice IDs, student, class, section, academic year, and due date.
- Show queued, sent, failed, and skipped counts after sending.

Logs:

- Table by scheduled time, attempted time, channel, recipient, status, error message.

### Reports And Exports

Reports should be built for finance users who scan, filter, and export.

Use TanStack Table for:

- Collections report.
- Defaulters report.
- Dues report.
- Fee-head report.
- Payment-method report.
- Offline-payment report.

Shared report controls:

- Date range: `from`, `to`.
- As-of date: `as_of` for dues and defaulter style views.
- Academic year, class, section, student.
- Payment method and provider where relevant.
- Export action creates an export job, then downloads the CSV when ready.

Table behavior:

- Sticky header on desktop.
- Column visibility menu.
- Density toggle: comfortable, compact.
- CSV export action in page header.
- Mobile list fallback for critical reports.

## Component Architecture

Recommended directories:

```text
src/
  app/
  components/
    app-shell/
    data-table/
    forms/
    charts/
    money/
    status/
    ui/
  features/
    auth/
    tenants/
    academic/
    students/
    billing/
    payments/
    reminders/
    reports/
    parent/
  lib/
    api/
    auth/
    format/
    query/
    schemas/
  stores/
```

Component boundaries:

- `components/ui`: shadcn generated components only.
- `components/data-table`: shared TanStack Table wrappers.
- `components/money`: INR and paise display helpers.
- `features/*`: domain components, schemas, query hooks, and screen sections.
- `lib/api`: Axios client and generated or handwritten API functions.

## State And Data Rules

Use TanStack Query for server state:

- Lists, details, dashboards, reports, receipts, exports.
- Mutations for create, update, delete, send, generate, verify.
- Invalidate by domain query keys after mutations.
- Use `keepPreviousData` style pagination behavior so tables do not flicker.

Use Zustand only for client state:

- Access token and refresh token if not using httpOnly cookies.
- Selected tenant ID and tenant display metadata.
- Sidebar collapsed state.
- Table density and column preferences.
- Checkout draft selection before order creation.

Do not duplicate API data in Zustand. Cache API data in TanStack Query.

## Forms And Validation

All create/edit forms should use:

- React Hook Form.
- Zod schema.
- shadcn form and field components.
- Server validation errors mapped into field-level or form-level UI.

Validation conventions:

- Dates use `YYYY-MM-DD`.
- IDs are UUID strings.
- Money inputs display rupees but submit paise.
- Enums should be represented with `Select`, `RadioGroup`, or `ToggleGroup`.
- Binary values use `Switch` or `Checkbox`.

## Money And Date Formatting

Backend money fields use paise, for example `amount_paise`, `total_due_paise`, `balance_amount_paise`.

Frontend rules:

- Never store money as floating point rupees for API payloads.
- Convert rupees to paise at form boundary.
- Convert paise to INR for display.
- Use `Intl.NumberFormat("en-IN", { style: "currency", currency: "INR" })`.
- Dates sent to APIs should be `YYYY-MM-DD`.
- Timestamps returned by APIs should display with local timezone and an explicit date/time format.

## Permissions And Visibility

Backend permissions are enforced by API middleware, but the UI should hide unavailable navigation and actions when claims or profile data expose permissions.

Permission groups:

| Permission | UI actions |
| --- | --- |
| `tenant.read` | View tenant settings. |
| `tenant.update` | Edit tenant profile. |
| `users.manage` | Create tenant users. |
| `academic.manage` | Manage academic years, classes, sections. |
| `students.manage` | Manage students, guardians, links, imports. |
| `guardians.manage` | Manage guardians. |
| `imports.manage` | Upload and commit student imports. |
| `fees.manage` | Manage fee heads, structures, assignments, invoices, ledgers. |
| `payments.manage` | Record offline payments, view payments and receipts. |
| `reminders.manage` | Manage templates, rules, sends, logs. |
| `reports.view` | Dashboard and report views. |
| `exports.manage` | Export creation and downloads. |

## Loading, Empty, And Error States

Every table and dashboard section needs:

- Skeleton loading state.
- Empty state with one next action.
- Error state with retry.
- Permission denied state.
- Offline/network failure state for payment flows.

Mutation feedback:

- Use toast for success.
- Use inline errors for validation.
- Use alert dialogs for destructive confirmation.
- Keep users on the same page after create/update unless the next workflow is obvious.

## Accessibility

Minimum requirements:

- Keyboard accessible navigation, dialogs, sheets, tables, and form controls.
- Visible focus states from shadcn theme tokens.
- `aria-invalid` on invalid controls.
- Dialog and Sheet titles always present.
- Charts must include text summaries or adjacent numeric data.
- Tables must keep readable text sizes on mobile.

## Implementation Checklist

- Set up Next.js App Router with route groups for public, platform, admin, and parent.
- Initialize shadcn/ui and install required UI components before use.
- Add Axios client, TanStack Query provider, Zustand auth/session store.
- Build auth, tenant selection, and guarded route layouts first.
- Build dashboard and shared data table components before individual report pages.
- Build shared money, date, status badge, pagination, and file-download helpers.
- Implement parent dues and checkout flow as a mobile-first path.
- Keep generated OpenAPI types or handwritten API types in sync with `docs/swagger/openapi.json`.
