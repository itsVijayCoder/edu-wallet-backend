# EduWallet Product Plan

Last updated: 2026-05-11

## Product Idea

EduWallet is a multi-tenant fee collection and receivables platform for Indian schools. It replaces paper receipts, manual fee reminders, cash/cheque-heavy collection, and spreadsheet reconciliation with a digital workflow where parents pay through UPI, cards, net banking, or wallets, and school admins track dues, receipts, refunds, settlements, and reports from one dashboard.

The core business model is SaaS subscription per institution, with optional payment-linked revenue where commercially and legally appropriate.

## Target Customers

| Segment | Main Pain | Why EduWallet Fits |
| --- | --- | --- |
| Private K-12 schools | Manual fee collection, delayed payments, parent follow-ups, receipt disputes | Centralized fee setup, reminders, payment links, instant receipts, dashboards |
| School groups and chains | Need tenant isolation, branch-level reporting, and standard policies across campuses | Multi-school tenancy, shared templates, group dashboards, role-based controls |
| Budget and mid-market schools | Need simple setup without full ERP complexity | Focused fee-first product with CSV import, simple onboarding, and low training burden |
| Coaching centers and small institutions | Need fast digital collection without heavy admin overhead | Lightweight fee plans, payment links, parent/student receipts, mobile-friendly flows |

## Market Rationale

UPI and digital payments are now mainstream in India. Recent public reporting based on NPCI/DFS data shows UPI processed more than 22 billion monthly transactions in March and April 2026, which supports a UPI-first payment experience for Indian parents. Razorpay also positions education fee collection as a supported use case through payment links, Smart Collect, Route, payment gateway, and 100+ payment modes.

Competitors such as Fedena and Edumarshal already offer fee management, dashboards, reminders, online payments, receipts, defaulter tracking, refunds, and exports. This means the category is validated, but EduWallet needs to win through sharper fee workflows, faster onboarding, cleaner parent payment UX, stronger reconciliation, and tenant-first architecture.

## Positioning

EduWallet should not start as a full school ERP. The strongest wedge is:

> The fastest way for Indian schools to digitize fee collection, reminders, receipts, reconciliation, and parent payments.

This keeps the product focused, makes onboarding easier, and avoids competing too early with broad ERP suites.

## MVP Goal

Launch a reliable fee collection product that allows one school to:

1. Create students, classes, fee heads, and fee schedules.
2. Generate dues for students.
3. Let parents pay online in INR through Razorpay.
4. Track paid, pending, overdue, failed, refunded, and partially paid fees.
5. Send reminders.
6. Generate receipts.
7. Export reports for finance/admin use.

## Necessary Features

### 1. Multi-Tenant School Management

| Feature | Needed For | Why It Is Necessary |
| --- | --- | --- |
| Institution/tenant records | Each school or school group | The platform monetizes per institution and must isolate each school's data |
| Branch/campus support | School chains and multi-campus schools | Enables expansion without forcing each campus into a separate account |
| Tenant-scoped users, students, fees, payments, and reports | Security and correctness | Prevents one school from seeing or modifying another school's data |
| Tenant-level configuration | Custom terms, receipt format, reminder timing, payment settings | Indian schools have different fee cycles, receipt formats, and policies |

### 2. Role-Based Access Control

| Role | Permissions | Why It Is Necessary |
| --- | --- | --- |
| Platform super admin | Manage schools, plans, billing, support access | Needed for SaaS operations |
| School owner/admin | Configure school, users, fee structures, reports | Main buyer and controller inside each institution |
| Accountant/finance staff | Collect offline payments, reconcile, issue receipts, view reports | Fee collection is usually handled by finance/admin staff |
| Class teacher or limited staff | View class-level dues and reminders only if allowed | Useful for follow-up without exposing full finance data |
| Parent/student | View dues, pay fees, download receipts | Parent self-service reduces admin workload |

### 3. Student and Guardian Records

| Feature | Needed For | Why It Is Necessary |
| --- | --- | --- |
| Student profile | Name, admission number, class, section, status | Fee assignment and reporting depend on accurate student records |
| Guardian profile | Parent name, phone, email, relationship | Payment links, receipts, reminders, and support need parent contact details |
| CSV import | Onboarding existing school data | Most schools already maintain spreadsheets; manual entry will slow adoption |
| Student status lifecycle | Active, transferred, graduated, inactive | Prevents incorrect future fee generation |
| Sibling linking | Family-level fee handling | Schools often offer sibling concessions and parents may pay for multiple children |

### 4. Fee Structure and Billing Engine

| Feature | Needed For | Why It Is Necessary |
| --- | --- | --- |
| Fee heads | Tuition, transport, hostel, exam, activity, admission, late fine | Schools collect many types of fees, not just tuition |
| Class/batch/category-wise fees | Different fees by grade, section, route, scholarship, quota | A single flat fee model is not realistic for schools |
| One-time, monthly, quarterly, term-wise, and annual schedules | Real school billing cycles | Indian schools commonly use term or monthly fee cycles |
| Concessions and discounts | Scholarships, staff children, sibling discounts | Prevents manual adjustments and receipt disputes |
| Late fees and fines | Overdue fee policies | Automates a common admin task and improves collections |
| Partial payments and installments | Parent affordability and school flexibility | Many parents pay in parts; blocking this would force offline workarounds |
| Waivers and manual adjustments | Principal/admin exceptions | Schools need controlled exception handling with audit trails |

### 5. Parent Payment Experience

| Feature | Needed For | Why It Is Necessary |
| --- | --- | --- |
| Mobile-first payment page | Parents paying from phones | Most UPI payments happen from mobile devices |
| UPI, cards, net banking, wallets | Parent payment choice | Razorpay and similar gateways support these modes; UPI should be the default |
| Payment links | WhatsApp/SMS/email reminders | Schools need quick collection without forcing parents to install an app |
| Due summary before payment | Trust and clarity | Parents should see child name, fee heads, amount, due date, and school name |
| Multiple children checkout | Families with siblings | Reduces repeated payments and support calls |
| Instant downloadable receipt | Parent proof of payment | Replaces paper receipts and reduces school counter visits |
| Failed payment recovery | Retry link and clear status | Payment failures are common; parent should know what to do next |

### 6. Razorpay Integration

| Feature | Needed For | Why It Is Necessary |
| --- | --- | --- |
| Razorpay order creation | Secure checkout flow | Orders are the standard payment object before collecting money |
| Webhook verification | Reliable payment status | Webhooks handle late authorization and async payment updates |
| Payment status reconciliation | Correct ledgers | The system must not rely only on frontend success callbacks |
| Refund support | Overpayments, cancellations, wrong fee collection | Refunds are a real school finance operation |
| Settlement tracking | Finance reporting | Schools need to know what was collected versus what reached the bank |
| Razorpay fee/tax capture | Net revenue and reconciliation | Gateway fees and GST affect settlement amounts |
| Idempotency and duplicate protection | Double-clicks, retries, webhook replays | Prevents duplicate receipts and incorrect ledgers |

### 7. Offline Collection Support

| Feature | Needed For | Why It Is Necessary |
| --- | --- | --- |
| Cash entry | Schools that still accept cash | Full digitization usually takes time; MVP must support transition |
| Cheque/DD entry | Common offline payment mode | Many schools still use cheques for annual or term fees |
| Cheque status | Pending, cleared, bounced | Finance needs accurate outstanding balances |
| Offline receipt generation | Counter payments | Admins need one receipt system for online and offline collections |
| Payment method tagging | Audit and reporting | Separates UPI/card/cash/cheque collection performance |

### 8. Receipts, Invoices, and Ledger

| Feature | Needed For | Why It Is Necessary |
| --- | --- | --- |
| Receipt number series per school | Compliance and audit discipline | Schools often require predictable receipt numbering |
| PDF receipt | Parent download and school records | Replaces paper receipts |
| Student ledger | Full payment history | Admins need to answer parent disputes quickly |
| Fee balance calculation | Paid, pending, overdue, waived, refunded | Core product correctness depends on this |
| Audit trail | Who changed what and when | Necessary for finance integrity and internal controls |

### 9. Reminders and Communication

| Feature | Needed For | Why It Is Necessary |
| --- | --- | --- |
| Due reminders | Before due date | Improves collection without manual calls |
| Overdue reminders | After due date | Reduces unpaid balances |
| Payment success messages | Parent confirmation | Builds trust and reduces support calls |
| Failed payment messages | Recovery | Encourages retry while the parent intent is fresh |
| SMS/email first, WhatsApp later | Practical MVP scope | SMS/email are simpler; WhatsApp adds approval, templates, and cost complexity |
| Reminder logs | Admin visibility | Staff should know which parents were contacted |

### 10. Admin Dashboard and Reports

| Feature | Needed For | Why It Is Necessary |
| --- | --- | --- |
| Collection dashboard | Daily operations | Admins need paid, pending, overdue, failed, and refunded totals |
| Defaulter list | Follow-up | This is one of the highest-value school finance workflows |
| Class/section-wise report | Staff follow-up and review | Schools commonly manage collections by class |
| Fee-head-wise report | Accounting | Shows tuition versus transport versus other collections |
| Payment method report | Payment strategy | Helps schools understand UPI/card/offline mix |
| Settlement report | Bank reconciliation | Finance teams need to match gateway settlements with internal records |
| CSV/Excel export | Existing accounting workflow | Schools often use Excel or Tally outside the product |

### 11. SaaS Billing and Monetization

| Feature | Needed For | Why It Is Necessary |
| --- | --- | --- |
| Subscription plans | SaaS revenue | Enables per-institution pricing |
| Plan limits | Student count, users, branches, reminders, reports | Makes pricing scalable and understandable |
| Trial/demo school | Sales onboarding | Schools need to see workflows before buying |
| Invoice for school subscription | B2B accounting | Institutions need proper vendor billing records |
| Usage metrics | Pricing and support | Helps identify high-volume schools and upsell opportunities |

Suggested pricing structure:

| Plan | Target | Possible Pricing Logic |
| --- | --- | --- |
| Starter | Small schools/coaching centers | Low monthly fee, limited students and users |
| Growth | Mid-size schools | Higher student limit, reports, reminders, offline collection |
| Pro | School chains | Multi-branch, advanced reports, custom receipt templates, priority support |
| Enterprise | Large institutions/groups | Custom pricing, integrations, SLA, data migration support |

Transaction fee options:

1. Pass through Razorpay/platform charges transparently.
2. Charge SaaS only and avoid payment markup during early adoption.
3. Add a small convenience/platform fee only after validating school and parent acceptance.

For early market entry, SaaS-first pricing is cleaner because schools are sensitive to payment fees and Razorpay already charges platform fees on successful transactions.

### 12. Security, Privacy, and Compliance

| Feature | Needed For | Why It Is Necessary |
| --- | --- | --- |
| Tenant isolation at database/query layer | Data protection | A multi-school SaaS cannot risk cross-tenant leakage |
| Strong authentication and refresh tokens | Account security | Finance dashboards expose sensitive student and payment data |
| Role-based permissions | Least privilege | Accountants, teachers, admins, and parents need different access |
| Audit logs | Finance and admin accountability | Required for payment edits, waivers, refunds, and receipt changes |
| Data minimization | Student/child privacy | The product should collect only what is needed for fee operations |
| Parent consent-aware data handling | Children's data | India's DPDP Act treats a child as under 18 and includes parent/lawful guardian in the data principal context |
| Webhook signature validation | Payment security | Prevents forged payment status updates |
| No card data storage | Payment compliance | Card details should remain with the payment gateway/payment network |
| Backups and recovery | Business continuity | Fee records are critical financial data |

### 13. Support and Operations

| Feature | Needed For | Why It Is Necessary |
| --- | --- | --- |
| School onboarding checklist | Sales to activation | Reduces setup confusion |
| Data import validation | Clean migration | Bad imports create billing and receipt errors |
| Support user impersonation with audit | Troubleshooting | Platform support may need to diagnose school issues safely |
| Payment issue lookup | Parent/admin support | Failed/late payments need quick resolution |
| Notification delivery status | Reminder reliability | Admins should know whether reminders were sent |

## MVP Scope

Build first:

1. Multi-tenant schools.
2. Admin, accountant, and parent roles.
3. Students, guardians, classes, sections.
4. Fee heads and fee schedules.
5. Due generation.
6. Razorpay order creation and webhook handling.
7. Parent payment page.
8. Online payment ledger.
9. Offline cash/cheque entries.
10. Receipts.
11. Basic reminders by email/SMS.
12. Dashboard and CSV exports.

Defer until after MVP:

1. Full mobile app.
2. WhatsApp Business integration.
3. Tally integration.
4. BBPS/bank portal integrations.
5. Advanced analytics and forecasting.
6. Payroll, attendance, homework, exams, transport tracking.
7. AI-based collection predictions.

## Roadmap

### Phase 1: Foundation and MVP

Goal: One school can run end-to-end fee collection.

Deliverables:

1. Tenant, user, student, guardian, class, and role models.
2. Fee setup and due generation.
3. Razorpay checkout and webhooks.
4. Parent payment page.
5. Receipts and ledgers.
6. Basic dashboard and exports.
7. Offline payments.

Success metrics:

1. School can onboard within 1 day after data import.
2. Parent can complete payment in under 2 minutes.
3. Payment status accuracy is above 99% after webhook reconciliation.
4. Admin can export daily collection report.

### Phase 2: Collection Automation

Goal: Reduce manual follow-up and finance workload.

Deliverables:

1. Automated reminders.
2. Defaulter workflows.
3. Refunds.
4. Settlement reconciliation.
5. Custom receipt templates.
6. Sibling payments and concessions.
7. Better reports by fee head, class, and payment method.

Success metrics:

1. Reduction in manual reminder calls.
2. Higher on-time collection rate.
3. Fewer parent receipt/payment disputes.
4. Faster month-end reconciliation.

### Phase 3: Scale and Differentiation

Goal: Make EduWallet strong for school groups and larger institutions.

Deliverables:

1. Multi-branch group dashboard.
2. Tally/accounting exports or integration.
3. WhatsApp reminders.
4. Advanced permissions and approval workflows.
5. API integrations for existing school ERPs.
6. Subscription billing automation for EduWallet plans.

Success metrics:

1. Multi-branch schools can manage all campuses from one account.
2. Finance team can close reconciliation faster.
3. School retention and expansion revenue increase.

## Suggested Data Model

Core entities:

1. `schools`
2. `branches`
3. `users`
4. `roles`
5. `students`
6. `guardians`
7. `student_guardians`
8. `classes`
9. `sections`
10. `fee_heads`
11. `fee_plans`
12. `fee_plan_items`
13. `fee_schedules`
14. `student_fee_assignments`
15. `invoices` or `fee_demands`
16. `invoice_items`
17. `payments`
18. `payment_attempts`
19. `refunds`
20. `receipts`
21. `reminders`
22. `audit_logs`
23. `school_subscriptions`

Important implementation rule: every tenant-owned table should include `school_id`, and every query should be scoped by `school_id`.

## Backend API Modules

| Module | Example Responsibilities |
| --- | --- |
| Auth | Login, refresh, logout, password reset |
| Schools | Tenant setup, branch setup, school settings |
| Users/Roles | Staff and parent access management |
| Students | Student records, guardian links, imports |
| Fees | Fee heads, fee plans, schedules, concessions |
| Billing | Generate dues, calculate balances, mark overdue |
| Payments | Razorpay orders, callbacks, webhooks, offline entries |
| Receipts | Receipt generation, PDF download, receipt search |
| Reminders | Due/overdue/success/failed notification jobs |
| Reports | Dashboard summaries, exports, defaulter lists |
| Subscriptions | EduWallet SaaS plan billing and limits |
| Audit | Change history for sensitive actions |

## Key Risks

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Schools resist transaction fees | Slower adoption | Start SaaS-first or pass-through pricing; avoid confusing hidden fees |
| Bad imported data | Wrong dues and parent disputes | Add import preview, validation, and rollback |
| Payment webhook mismatch | Incorrect paid/pending status | Use signed webhooks, payment verification, idempotency, and reconciliation jobs |
| Overbuilding full ERP | Slow launch | Stay fee-first until collection workflows are strong |
| Parent trust issues | Lower online payment conversion | Clear school branding, due breakdown, secure checkout, instant receipt |
| Data privacy mistakes | Legal and reputation risk | Minimize child data, use role-based access, audit logs, retention policies |
| Offline payments remain high | Lower payment-linked revenue | Still digitize offline records first; convert parents gradually through reminders and payment links |

## Product Differentiators

1. Fee-first product instead of heavy ERP.
2. UPI-first parent payment flow.
3. Strong reconciliation and settlement tracking.
4. Multi-tenant architecture from day one.
5. Fast CSV-based onboarding.
6. Clean defaulter and reminder workflow.
7. School-branded receipts and payment pages.
8. Simple SaaS pricing for schools.

## Launch Checklist

Before pilot launch:

1. Create one demo school with realistic Indian fee data.
2. Test online payments in Razorpay test mode.
3. Test webhook replay and duplicate payment protection.
4. Import at least 500 sample students through CSV.
5. Generate term/monthly dues.
6. Send test reminders.
7. Download receipts.
8. Export collection and defaulter reports.
9. Validate tenant isolation.
10. Prepare onboarding guide for school admins.

## Pilot Strategy

Start with 2-3 schools or coaching centers that still use spreadsheets and manual receipts. Avoid very large school groups in the first pilot because they will require complex custom reports and integrations.

Pilot goals:

1. Prove parents complete UPI/card payments without staff assistance.
2. Prove admins trust dashboard balances.
3. Prove receipt disputes go down.
4. Measure reminder-to-payment conversion.
5. Measure time saved in daily and monthly reconciliation.

## Sources Checked

These sources were checked on 2026-05-11 to keep payment and competitor assumptions current:

1. Razorpay pricing and payment method information: https://razorpay.com/pricing/
2. Razorpay education payment solutions: https://razorpay.com/solutions/education/
3. Razorpay webhook documentation: https://razorpay.com/docs/webhooks/
4. Razorpay subscriptions documentation: https://razorpay.com/docs/payments/subscriptions/
5. Fedena school fees management feature page: https://fedena.com/feature-tour/school-fees-management-system
6. Edumarshal fees and finance management page: https://edumarshal.com/school-erp-modules/fees-finance-management-for-schools/
7. Classplus App Store listing for fee management context: https://apps.apple.com/in/app/classplus/id1324522260
8. RBI payment aggregator and gateway guidelines: https://www.rbi.org.in/scripts/FS_Notification.aspx?Id=11822
9. Digital Personal Data Protection Act, 2023 PDF from MeitY: https://www.meity.gov.in/static/uploads/2024/02/Digital-Personal-Data-Protection-Act-2023.pdf
10. Public reporting on UPI April 2026 transaction volume from ET BFSI/PTI: https://bfsi.economictimes.indiatimes.com/amp/articles/upi-achieves-22-35-bn-transactions-in-april-dfs/130814926

