# EduWallet — Detailed Product Plan for Indian Institutions

## 1. Product Overview

**EduWallet** is a multi-tenant SaaS platform for schools, colleges, coaching centers, tuition centers, hostels, and other educational institutions in India to digitize fee collection and finance tracking.

Today, many institutions still manage fees using paper receipts, manual registers, Excel sheets, cash/cheque collections, and WhatsApp/SMS reminders. EduWallet replaces this with a digital workflow where:

- Parents can pay fees using UPI, cards, netbanking, wallets, or other online modes.
- Admins can create fee structures, track collections, manage dues, generate receipts, and send reminders from one dashboard.
- Institutions can manage online and offline payments in a single ledger.
- EduWallet can operate as a SaaS business with subscriptions per institution and optional transaction fees.

The competitive landscape already includes products like Edumarshal, Fedena, Classplus fee modules, and bank/payment-gateway-led school fee portals. Edumarshal highlights online payments, fee notifications, customizable fee heads, defaulter alerts, receipts, dashboards, and refund support. Fedena highlights fee tracking, customized receipts, tax/discount support, pending-fee tracking, and instant parent alerts. Classplus includes fee records, receipts, and automatic reminders for coaching/tutor businesses. Razorpay education payment solutions support payment pages, payment links, UPI, cards, wallets, EMI, PayLater, and 100+ payment modes.

**Reference sources**

- Edumarshal Fee Management: <https://edumarshal.com/school-erp-modules/fees-finance-management-for-schools/>
- Fedena Fee Management: <https://fedena.com/feature-tour/school-fees-management-system>
- Classplus fee records and reminders: <https://classplusapp.com/lp/tutors-fb>
- Razorpay Education Payments: <https://razorpay.com/solutions/education/>
- Razorpay Payment Links: <https://razorpay.com/payment-links/>

---

## 2. Core Goal

EduWallet should solve one painful problem:

> Institutions should not depend on paper receipts, manual fee registers, WhatsApp reminders, cash/cheque reconciliation, and Excel tracking.

Instead, EduWallet should provide:

### For Parents

- Simple UPI/card/netbanking fee payment.
- Downloadable receipts.
- Due reminders.
- Child-wise fee history.
- Payment links through SMS/WhatsApp/email.
- Support option for failed or pending payments.

### For Admins

- One dashboard to create fee structures.
- Collect payments online and offline.
- Track paid, unpaid, partial-paid, and overdue fees.
- Generate receipts.
- Send reminders.
- Manage refunds.
- Export reports.

### For EduWallet Platform Owner

- SaaS subscription revenue per institution.
- Optional transaction fee or commission on successful payments.
- Add-on revenue from WhatsApp/SMS packages, white-label apps, custom reports, and integrations.

---

## 3. Target Customers

EduWallet can serve multiple Indian education segments:

1. Schools.
2. Colleges.
3. Coaching centers.
4. Tuition centers.
5. Training institutes.
6. Hostels.
7. Preschool chains.
8. Multi-branch institutions.
9. Skill development institutes.
10. Religious/charitable educational institutions.

The MVP should focus on **schools and coaching centers** first because their workflows are simpler and faster to onboard.

---

## 4. Main User Roles

### 4.1 Platform Super Admin

This is the EduWallet internal team.

Required features:

1. Create and manage institutions.
2. Enable or disable tenant accounts.
3. Manage SaaS subscription plans.
4. View all institutions' usage.
5. Configure platform-level payment gateway settings.
6. Set transaction fee rules.
7. Monitor payment failures.
8. View revenue from SaaS plans and transaction fees.
9. Manage support tickets.
10. View tenant-wise analytics.
11. View audit logs.
12. Suspend institutions for non-payment or misuse.
13. Configure global notification providers.
14. Configure platform-level tax or billing settings.

### 4.2 Institution Owner / Principal / Management

This is the school or college management team.

Required features:

1. Institution profile setup.
2. Branch/campus management.
3. Academic year setup.
4. Classes, sections, departments, and courses.
5. Student import and management.
6. Parent/guardian contact management.
7. Fee category and fee structure setup.
8. Dashboard for total collection, pending dues, defaulters, and trends.
9. Reports and exports.
10. Staff/admin access control.
11. Discount and concession approval.
12. Refund approval.
13. Reminder configuration.
14. Receipt format configuration.

### 4.3 Finance Admin / Accountant

This is the daily fee-management user.

Required features:

1. Create fee invoices.
2. Track paid, unpaid, partial-paid, and overdue fees.
3. Record offline payments.
4. Reconcile online payments.
5. Generate receipts.
6. Send reminders.
7. Apply discounts, concessions, scholarships, and waivers.
8. Add late fees/fines.
9. Manage refunds.
10. Export reports for accounting.
11. Track cheque/DD status.
12. Search students quickly.
13. Download day-end collection report.
14. View gateway transaction details.

### 4.4 Class Teacher / Department Staff

This is an optional role.

Required features:

1. View student payment status for assigned class/department.
2. See defaulter list for assigned class.
3. Send fee reminder request.
4. View limited student information.
5. No access to financial configuration.
6. No access to receipt cancellation/refund controls.

### 4.5 Parent / Guardian

This is the main payer.

Required features:

1. Login using mobile OTP.
2. View all linked children.
3. View outstanding fees.
4. Pay via UPI, card, netbanking, wallet, EMI if enabled.
5. Download receipts.
6. View payment history.
7. Receive reminders by SMS, WhatsApp, email, and app notification.
8. Raise payment issue or support request.
9. Pay partial amount if institution allows.
10. View upcoming dues.
11. View failed payment status.

### 4.6 Student

This is optional, more useful for colleges.

Required features:

1. View fee dues.
2. Download receipts.
3. Pay fees if allowed.
4. View scholarship/concession status.
5. View no-due certificate status.
6. Raise support requests.

---

## 5. Multi-Tenant SaaS Requirements

Since EduWallet supports multiple institutions as tenants, tenant isolation is one of the most important product and architecture decisions.

### 5.1 Tenant Model

Each institution should have isolated data.

```text
EduWallet Platform
 ├── School A
 │    ├── Students
 │    ├── Parents
 │    ├── Fee structures
 │    ├── Payments
 │    ├── Receipts
 │    └── Reports
 ├── School B
 │    ├── Students
 │    ├── Parents
 │    ├── Fee structures
 │    ├── Payments
 │    ├── Receipts
 │    └── Reports
 └── College C
      ├── Departments
      ├── Students
      ├── Semester fees
      ├── Payments
      └── Reports
```

### 5.2 Tenant Features

1. Unique institution ID.
2. Custom institution logo.
3. Custom receipt format.
4. Custom fee heads.
5. Custom academic year.
6. Custom payment settings.
7. Custom reminder templates.
8. Custom user roles.
9. Branch/campus support.
10. Tenant-level reports.
11. Tenant-level SMS/WhatsApp sender settings.
12. Tenant-level tax settings if needed.
13. Tenant-level language preferences.
14. Tenant-specific payment gateway account if required.

### 5.3 SaaS Controls

1. Subscription plan limit.
2. Student count limit.
3. Admin user limit.
4. Payment volume tracking.
5. Storage limit for receipts/reports.
6. Feature access based on plan.
7. Trial account support.
8. Auto-suspend if subscription expires.
9. Billing reminders for institution.
10. Usage alerts when limits are close.
11. Upgrade/downgrade plan support.
12. Tenant-level feature flags.

---

## 6. Fee Management Features

This is the core of EduWallet.

### 6.1 Fee Structure Setup

Institutions should be able to create different fee structures.

Common fee heads:

1. Admission fee.
2. Tuition fee.
3. Term fee.
4. Exam fee.
5. Transport fee.
6. Hostel fee.
7. Lab fee.
8. Library fee.
9. Uniform/book fee.
10. Fine/penalty.
11. Security deposit.
12. Miscellaneous fee.
13. Activity fee.
14. Sports fee.
15. Development fee.
16. Caution deposit.
17. Graduation/certificate fee.
18. Bus route fee.
19. Mess fee.
20. ID card fee.

Competitors like Edumarshal and Fedena already support customizable fee structures, fee heads, discounts, online payment, receipts, alerts, and reports, so EduWallet should cover these from the first serious version.

### 6.2 Fee Frequency

EduWallet should support multiple billing types:

1. One-time.
2. Monthly.
3. Quarterly.
4. Term-wise.
5. Semester-wise.
6. Yearly.
7. Custom date range.
8. Installment-based.
9. Admission-time fee.
10. Event-based fee.

### 6.3 Fee Assignment

Admin should assign fees by:

1. Class.
2. Section.
3. Course.
4. Department.
5. Batch.
6. Individual student.
7. Transport route.
8. Hostel room.
9. Student category.
10. Custom group.
11. Academic year.
12. Branch/campus.

### 6.4 Discounts and Concessions

Required features:

1. Sibling discount.
2. Staff child discount.
3. Scholarship discount.
4. Management concession.
5. Early payment discount.
6. Category-based discount.
7. One-time manual waiver.
8. Fee-head-specific discount.
9. Percentage discount.
10. Fixed-amount discount.
11. Approval workflow for large discounts.
12. Discount audit history.
13. Discount expiry date.
14. Discount reason field.

### 6.5 Late Fee and Fine

Options needed:

1. Fixed late fee.
2. Daily late fee.
3. Weekly late fee.
4. Monthly late fee.
5. Percentage-based penalty.
6. Grace period.
7. Maximum fine cap.
8. Auto-apply fine after due date.
9. Manual fine waiver with approval.
10. Late fee by class/course.
11. Late fee by fee head.
12. Late fee exclusion for scholarship students.

### 6.6 Partial Payment

Many Indian institutions need this because parents may pay fees in installments.

Required features:

1. Allow full payment only.
2. Allow partial payment.
3. Minimum payable amount.
4. Installment due tracking.
5. Balance amount calculation.
6. Separate receipt for each partial payment.
7. Parent-visible pending balance.
8. Admin setting to disable partial payment.
9. Partial payment remarks.
10. Auto-adjust remaining balance.

### 6.7 Offline Payment Entry

Even if the product is digital, many schools may continue collecting cash or cheque initially.

Support:

1. Cash payment entry.
2. Cheque payment entry.
3. Bank transfer entry.
4. Demand draft entry.
5. UTR/reference number.
6. Cheque pending/cleared/bounced status.
7. Manual receipt generation.
8. Offline payment approval by accountant/principal.
9. Cash counter/day-end report.
10. Offline payment audit log.
11. Attachment upload for bank proof.
12. Duplicate reference detection.

---

## 7. Online Payment Features

India-first payment support is critical.

### 7.1 Payment Modes

EduWallet should support:

1. UPI.
2. UPI QR.
3. UPI intent.
4. Debit card.
5. Credit card.
6. Netbanking.
7. Wallets.
8. EMI for large school/college fees.
9. PayLater, optional.
10. Bank transfer/virtual account, optional.
11. Payment links.
12. Payment pages.

Razorpay's education payment solution supports payment pages, payment links, 100+ payment modes, EMI, and PayLater. Razorpay also supports UPI, cards, netbanking, and wallets. This makes payment-gateway integration a strong fit for EduWallet's India-first fee collection flow.

### 7.2 Payment Gateway Integrations

Start with one gateway and then expand.

#### Phase 1

1. Razorpay.
2. Cashfree or PayU as backup.

#### Phase 2

1. Multiple gateway routing.
2. Gateway failover.
3. Institution-specific gateway account.
4. Platform master gateway account with split settlement.
5. Smart routing based on payment method.
6. Gateway health monitoring.

### 7.3 Payment Flow

```text
Parent receives payment link / logs in
        ↓
Selects child
        ↓
Views pending fees
        ↓
Chooses fee items
        ↓
Applies partial payment if allowed
        ↓
Pays using UPI/card/netbanking
        ↓
Payment success/failure webhook received
        ↓
Receipt generated
        ↓
Parent and admin notified
        ↓
Dashboard and reports updated
```

### 7.4 Payment Statuses

Track every transaction with clear status:

1. Created.
2. Pending.
3. Processing.
4. Success.
5. Failed.
6. Expired.
7. Cancelled.
8. Refunded.
9. Partially refunded.
10. Manually verified.
11. Reconciliation mismatch.
12. Webhook pending.
13. Settlement pending.
14. Settled.

### 7.5 Payment Links

Payment links are very useful for Indian schools.

Required features:

1. Generate individual payment link.
2. Generate class-wise bulk links.
3. Send link by WhatsApp/SMS/email.
4. Link expiry.
5. Link with fixed amount.
6. Link with editable amount if allowed.
7. Link tracking: opened, paid, expired.
8. Bulk payment link generation by CSV/Excel.
9. Payment link resend option.
10. Payment link reminder automation.

Razorpay Payment Links support partial payments, bulk link creation, real-time tracking with webhooks, and multiple payment methods. This is directly relevant for EduWallet's parent payment-link workflow.

---

## 8. Receipts and Invoices

### 8.1 Receipt Features

1. Auto receipt after successful payment.
2. Manual receipt for offline payment.
3. PDF receipt download.
4. Receipt number series.
5. Institution logo.
6. Student details.
7. Fee breakdown.
8. Payment mode.
9. Transaction ID.
10. GST/tax details if applicable.
11. Digital signature/stamp option.
12. Receipt cancellation with reason.
13. Duplicate receipt download.
14. Receipt print format.
15. Receipt share through WhatsApp/email.
16. Branch-wise receipt numbering.
17. Academic-year-wise receipt numbering.
18. Cancelled receipt report.

### 8.2 Invoice / Demand Note

Before payment, the system should generate demand notes.

Required fields:

1. Student name.
2. Admission number.
3. Class/course.
4. Academic year.
5. Fee invoice number.
6. Due date.
7. Fee heads.
8. Discounts.
9. Fine.
10. Tax if applicable.
11. Pending amount.
12. Payment link.
13. Institution contact details.

Required features:

1. Generate invoice per student.
2. Generate bulk invoices.
3. Regenerate invoice after fee change.
4. Cancel invoice with audit log.
5. Parent-visible demand note.
6. Download invoice as PDF.

---

## 9. Reminder and Communication System

Manual reminders are one of the biggest pain points for schools and institutions.

### 9.1 Reminder Channels

1. SMS.
2. WhatsApp.
3. Email.
4. Push notification.
5. In-app notification.

### 9.2 Reminder Types

1. Before due date.
2. On due date.
3. After due date.
4. Repeated overdue reminder.
5. Payment success message.
6. Failed payment message.
7. Cheque bounced message.
8. Receipt generated message.
9. Refund processed message.
10. Payment link expiry reminder.
11. Partial payment balance reminder.
12. Institution subscription reminder.

### 9.3 Reminder Rules

Example rule:

```text
7 days before due date → polite reminder
1 day before due date → payment due tomorrow
Due date → payment due today
3 days after due date → overdue reminder
7 days after due date → stronger reminder with late fee
15 days after due date → management escalation report
```

### 9.4 Template Management

Institution should customize:

1. SMS template.
2. WhatsApp template.
3. Email subject and body.
4. Language: English, Tamil, Hindi, etc.
5. Variables like student name, amount, due date, payment link.
6. Template approval status for WhatsApp.
7. Reminder tone: polite, formal, urgent.
8. Institution branding.

Example template:

```text
Dear {parent_name}, fee payment of ₹{amount} for {student_name} is due on {due_date}. Pay here: {payment_link}
```

---

## 10. Dashboard Features

### 10.1 Institution Dashboard

Main metrics:

1. Total fees assigned.
2. Total collected.
3. Total pending.
4. Total overdue.
5. Collection percentage.
6. Today's collection.
7. This month's collection.
8. Payment mode split.
9. Class-wise pending amount.
10. Top defaulter classes.
11. Failed transactions.
12. Refund amount.
13. Pending cheque amount.
14. Online vs offline collection.
15. Collection trend by date.
16. Due amount by fee head.

### 10.2 Finance Dashboard

Useful widgets:

1. Daily collection.
2. Online vs offline collection.
3. Pending reconciliation.
4. Cheque pending/cleared/bounced.
5. Refund requests.
6. Late fee collected.
7. Concessions given.
8. Fee-head-wise collection.
9. Gateway failures.
10. Settlement pending.
11. Cash collection by counter/user.
12. Receipt cancellation count.

### 10.3 Parent Dashboard

1. Total due.
2. Due date.
3. Fee breakdown.
4. Payment button.
5. Receipt history.
6. Payment status.
7. Support option.
8. Linked children.
9. Upcoming dues.
10. Failed payment retry.

---

## 11. Reports Needed

Reports are very important because schools and colleges depend heavily on Excel exports.

### 11.1 Collection Reports

1. Daily collection report.
2. Monthly collection report.
3. Date range collection report.
4. Fee-head-wise collection.
5. Class-wise collection.
6. Section-wise collection.
7. Student-wise collection.
8. Payment-mode-wise collection.
9. Online payment report.
10. Offline payment report.
11. Branch-wise collection report.
12. User/counter-wise collection report.
13. Academic-year-wise collection report.
14. Transport fee collection report.
15. Hostel fee collection report.

### 11.2 Pending Reports

1. Defaulter report.
2. Class-wise due report.
3. Student-wise due report.
4. Overdue ageing report.
5. Due by fee head.
6. Due by academic year.
7. Parent contact with pending amount.
8. Partial payment balance report.
9. Long-pending dues report.
10. Course/department-wise pending report.

### 11.3 Accounting Reports

1. Receipt register.
2. Cancelled receipt report.
3. Refund report.
4. Concession report.
5. Late fee report.
6. Tax/GST report if applicable.
7. Bank reconciliation report.
8. Settlement report.
9. Cheque status report.
10. Ledger report.
11. Payment gateway charges report.
12. Platform transaction fee report.

### 11.4 Export Options

1. Excel.
2. CSV.
3. PDF.
4. Print view.
5. Scheduled email reports.
6. API export for enterprise customers.

---

## 12. Reconciliation and Settlement

This is a must-have for serious institutions.

### 12.1 Online Reconciliation

Required features:

1. Payment gateway webhook capture.
2. Payment gateway settlement report import.
3. Auto-match payment with invoice.
4. Detect mismatch.
5. Detect duplicate payment.
6. Detect payment success but receipt not generated.
7. Detect gateway success but internal failure.
8. Manual reconciliation screen.
9. Webhook retry log.
10. Gateway signature verification.
11. Reconciliation status per transaction.

### 12.2 Settlement Tracking

If EduWallet collects payment through a platform gateway:

1. Gateway transaction ID.
2. Gateway fee.
3. Platform fee.
4. GST on charges.
5. Net settlement amount.
6. Settlement date.
7. Institution bank account.
8. Settlement status.
9. Settlement UTR.
10. Settlement report download.
11. Institution payout history.
12. Settlement failure tracking.

### 12.3 Refunds

Required features:

1. Full refund.
2. Partial refund.
3. Refund approval workflow.
4. Refund reason.
5. Gateway refund tracking.
6. Refund receipt/credit note.
7. Parent notification.
8. Refund status: requested, approved, processing, successful, failed.
9. Refund audit log.
10. Refund report.

---

## 13. Student and Parent Management

### 13.1 Student Data

Fields needed:

1. Student ID/admission number.
2. Name.
3. Class/course.
4. Section/batch.
5. Roll number.
6. Academic year.
7. Parent/guardian details.
8. Mobile number.
9. Email.
10. Address.
11. Status: active, inactive, alumni, transferred.
12. Category: general, scholarship, staff child, etc.
13. Transport route.
14. Hostel details.
15. Date of admission.
16. Gender, optional.
17. Date of birth, optional.
18. Previous balance.
19. Opening balance.
20. Custom fields.

### 13.2 Parent Data

1. Father name.
2. Mother name.
3. Guardian name.
4. Mobile number.
5. WhatsApp number.
6. Email.
7. Login access.
8. Multiple children mapping.
9. Preferred language.
10. Communication opt-in status.
11. Address.

### 13.3 Bulk Import

Admin should import students using Excel/CSV.

Required features:

1. Download sample template.
2. Upload Excel/CSV.
3. Validate data.
4. Show errors before import.
5. Duplicate detection.
6. Parent-child auto-linking.
7. Import history.
8. Rollback failed import.
9. Preview before final import.
10. Required-field validation.

---

## 14. Access Control and Security

### 14.1 Role-Based Access Control

Example permissions:

| Role | Access |
|---|---|
| Super Admin | All tenants and platform controls |
| Institution Admin | Full institution access |
| Accountant | Fee, payment, receipt, refund, and report access |
| Teacher | View class dues only |
| Parent | Own child only |
| Student | Own fee data only |

### 14.2 Security Features

1. Password login.
2. OTP login for parents.
3. Two-factor authentication for admins.
4. Role-based permissions.
5. Audit logs.
6. IP/device logs for admins.
7. Session timeout.
8. Data export permission control.
9. Receipt cancellation permission.
10. Refund approval permission.
11. Password reset flow.
12. Rate limiting for OTP.
13. Secure webhook verification.
14. Encrypted sensitive data.
15. Backup and restore policy.
16. Tenant-level data isolation.

### 14.3 Audit Logs

Track:

1. Fee structure created or edited.
2. Discount applied.
3. Payment manually added.
4. Receipt cancelled.
5. Refund approved.
6. Student data edited.
7. Admin login.
8. Report exported.
9. Payment status manually changed.
10. Reminder template edited.
11. Gateway settings changed.
12. Tenant subscription changed.

---

## 15. Mobile Experience

EduWallet should be mobile-first because parents mostly pay from phones.

### 15.1 Parent Mobile Web / App

Must-have features:

1. OTP login.
2. Child selection.
3. Pending fee card.
4. UPI payment button.
5. Receipt download.
6. Payment history.
7. Notifications.
8. Support ticket.
9. Retry failed payment.
10. Share/download receipt.

### 15.2 Admin Mobile View

Useful for small schools:

1. Today's collection.
2. Pending dues.
3. Search student.
4. Collect offline payment.
5. Generate receipt.
6. Send reminder.
7. View failed payments.
8. View defaulter list.

---

## 16. Support and Dispute Handling

Payment issues will happen frequently.

### 16.1 Parent Support

1. “Payment deducted but not updated” ticket.
2. Upload screenshot.
3. Enter UPI reference number.
4. Track ticket status.
5. Admin response.
6. Refund request.
7. Duplicate payment request.
8. Receipt not received request.

### 16.2 Admin Support

1. View failed/pending payments.
2. Check gateway status.
3. Manually mark after verification.
4. Add internal notes.
5. Escalate to EduWallet support.
6. Download payment logs.
7. Retry receipt generation.
8. Retry notification.

### 16.3 EduWallet Support

1. Tenant-wise tickets.
2. Gateway issue tracking.
3. Payment logs.
4. Webhook retry logs.
5. Manual correction with audit trail.
6. Institution onboarding support.
7. Data migration support.
8. Subscription/billing support.

---

## 17. Competitive Differentiation

The competitors already offer fee collection, receipts, alerts, dashboards, and broader ERP modules. EduWallet should not try to become a full ERP in the beginning. It should become the **best fee collection and payment operating system for Indian institutions**.

### 17.1 Differentiators

1. India-first UPI payment experience.
2. Very simple onboarding for small schools and coaching centers.
3. WhatsApp-first reminders.
4. Multi-tenant SaaS from day one.
5. Clean fee dashboard.
6. Fast Excel import/export.
7. Strong reconciliation.
8. Parent-friendly payment links.
9. Offline + online payment in one ledger.
10. Lower cost than heavy ERP systems.
11. Works for schools, colleges, coaching centers, tuition centers, hostels, and training institutes.
12. Payment-first product, not a bloated ERP.
13. Simple setup for non-technical institution staff.
14. Mobile-first parent flow.

### 17.2 Positioning

Use this positioning:

> EduWallet is not a full school ERP. It is a focused fee collection, payment, receipt, reminder, and reconciliation platform for Indian institutions.

This makes it easier to sell because institutions can adopt EduWallet without replacing their entire ERP immediately.

---

## 18. Monetization Plan

### 18.1 SaaS Subscription

Plans can be based on student count.

| Plan | Target | Features |
|---|---|---|
| Starter | Small schools/coaching centers | Basic fee setup, payment links, receipts |
| Growth | Mid-size schools | Reminders, reports, bulk import, partial payments |
| Pro | Colleges/multi-branch institutions | Advanced reports, reconciliation, role control |
| Enterprise | Large groups | API, custom gateway, white-label, dedicated support |

### 18.2 Transaction Fee

Possible options:

1. Fixed platform fee per successful transaction.
2. Percentage fee on transaction.
3. Institution pays gateway charges.
4. Parent pays convenience fee.
5. Mixed model.

Example pricing model:

```text
SaaS subscription: ₹2,999/month per institution
Platform transaction fee: ₹3–₹10 per successful payment
Gateway charges: passed to institution or parent
```

### 18.3 Add-On Revenue

1. WhatsApp message package.
2. SMS package.
3. Custom domain.
4. White-label parent app.
5. Data migration service.
6. ERP integration.
7. Custom reports.
8. Additional branches.
9. Advanced reconciliation.
10. Priority support.
11. Accounting software export.
12. API access.
13. Dedicated onboarding.

---

## 19. MVP Feature Scope

### 19.1 MVP Version 1 — Must Build First

This should be enough to sell to small schools and coaching centers.

#### Admin

1. Institution login.
2. Student import.
3. Class/section setup.
4. Fee structure creation.
5. Assign fees to students/classes.
6. Generate fee dues.
7. Online payment link.
8. UPI/card payment gateway integration.
9. Payment status update by webhook.
10. Receipt generation.
11. Manual offline payment entry.
12. Basic dashboard.
13. Due list.
14. Defaulter list.
15. Send reminder by SMS/WhatsApp/email.
16. Export Excel/CSV.

#### Parent

1. OTP login.
2. View child fees.
3. Pay online.
4. Download receipt.
5. View payment history.
6. Retry failed payment.

#### Platform

1. Create tenant.
2. Manage subscription plan manually.
3. View institution usage.
4. Basic support logs.
5. Basic platform revenue view.

---

## 20. Version 2 Features

After MVP validation:

1. Partial payments.
2. Discounts/concessions.
3. Late fee automation.
4. Refunds.
5. Advanced reports.
6. Payment reconciliation.
7. WhatsApp template automation.
8. Multi-branch institutions.
9. Role permissions.
10. Receipt customization.
11. Parent support tickets.
12. Gateway settlement reports.
13. Scheduled reports.
14. Mobile admin view.
15. Cheque/DD lifecycle.
16. Approval workflows.

---

## 21. Version 3 Features

For scale and enterprise:

1. White-label parent app.
2. Institution custom domain.
3. Multiple payment gateway routing.
4. ERP integrations.
5. Accounting software export.
6. Advanced approval workflows.
7. AI-based defaulter prediction.
8. Auto-generated collection insights.
9. No-due certificate.
10. Scholarship workflow.
11. Transport/hostel fee automation.
12. API access for institutions.
13. Franchise/group institution dashboard.
14. Bank-led payment portal integrations.
15. Multi-currency support for international schools, if needed later.
16. Advanced data warehouse/BI layer.

---

## 22. Suggested Product Modules

Build the product as these modules:

```text
1. Tenant Management
2. User & Role Management
3. Student Management
4. Parent Management
5. Academic Year Management
6. Class / Course / Section Management
7. Fee Structure Management
8. Invoice / Demand Note Management
9. Payment Management
10. Receipt Management
11. Reminder Management
12. Refund Management
13. Reconciliation Management
14. Reports & Exports
15. Subscription & Billing
16. Support Ticket Management
17. Audit Log Management
18. Notification Provider Management
19. Gateway Webhook Management
20. Settlement Management
```

---

## 23. Suggested Database Entities

Main tables/entities:

```text
tenants
tenant_branches
users
roles
permissions
role_permissions
students
parents
student_parent_links
academic_years
classes
sections
courses
batches
fee_heads
fee_structures
fee_structure_items
student_fee_assignments
invoices
invoice_items
payments
payment_attempts
receipts
receipt_series
refunds
discounts
concessions
late_fee_rules
reminder_templates
reminder_logs
notification_logs
gateway_webhooks
settlements
support_tickets
audit_logs
subscription_plans
tenant_subscriptions
payment_gateway_accounts
imports
import_errors
export_jobs
```

---

## 24. Important Workflows

### 24.1 Institution Onboarding

```text
Create tenant
↓
Add institution profile
↓
Create academic year
↓
Create classes/sections/courses
↓
Import students and parents
↓
Create fee heads
↓
Create fee structure
↓
Assign fees
↓
Generate dues
↓
Start collection
```

### 24.2 Parent Payment

```text
Parent receives reminder/payment link
↓
Opens fee page
↓
Verifies mobile OTP
↓
Selects child
↓
Views pending fee
↓
Pays using UPI/card/netbanking
↓
Webhook confirms payment
↓
Receipt generated
↓
Parent receives receipt
↓
Admin dashboard updates
```

### 24.3 Offline Payment

```text
Parent pays cash/cheque at school
↓
Accountant searches student
↓
Selects invoice/fee head
↓
Adds payment mode and reference
↓
System generates receipt
↓
Dashboard updates
↓
Audit log records accountant action
```

### 24.4 Reminder Automation

```text
System checks due dates daily
↓
Finds unpaid invoices
↓
Applies reminder rule
↓
Sends SMS/WhatsApp/email
↓
Stores reminder log
↓
Admin can see reminder history
```

### 24.5 Reconciliation

```text
Payment gateway sends webhook
↓
System updates payment status
↓
Settlement report imported/fetched
↓
System matches transaction ID
↓
Matched payments marked settled
↓
Mismatches shown to finance admin
```

### 24.6 Refund Workflow

```text
Parent/admin requests refund
↓
Finance admin reviews payment
↓
Institution admin approves refund
↓
Gateway refund initiated
↓
Refund status updated by webhook
↓
Parent notified
↓
Refund report updated
```

---

## 25. Recommended Tech Architecture

### 25.1 Frontend

Recommended stack:

1. Next.js.
2. shadcn/ui.
3. Tailwind CSS.
4. TanStack Query.
5. Zustand.
6. React Hook Form.
7. Zod.
8. TanStack Table for reports.
9. Recharts for dashboards.
10. Mobile-first responsive UI.

### 25.2 Backend

Recommended stack:

1. Go or Node.js.
2. PostgreSQL.
3. Redis for queues/cache.
4. Background worker for reminders, webhooks, reports, and exports.
5. Object storage for receipts/reports.
6. Payment gateway SDK.
7. WhatsApp/SMS/email providers.
8. REST API or GraphQL API.
9. Webhook verification service.
10. Scheduled job runner.

### 25.3 Infrastructure

1. Cloudflare for frontend/CDN.
2. Backend on VPS/AWS/GCP/Fly.io/Render/Railway depending budget.
3. Managed PostgreSQL.
4. Managed Redis or self-hosted Redis.
5. S3/R2 for file storage.
6. Queue system for reminder jobs.
7. Monitoring and logging.
8. Error tracking.
9. Automated backups.
10. CI/CD pipeline.

### 25.4 Multi-Tenant Data Strategy

Recommended for MVP:

- Single PostgreSQL database.
- Shared tables.
- Every tenant-owned table must include `tenant_id`.
- Use strict backend middleware to enforce tenant isolation.
- Add indexes on `tenant_id`, `student_id`, `invoice_id`, `payment_id`, and date columns.

Possible enterprise upgrade:

- Separate database per large institution.
- Separate schema per institution for high isolation.

---

## 26. Key Product Screens

### 26.1 Platform Super Admin Screens

1. All institutions.
2. Create institution.
3. Institution detail page.
4. Subscription plans.
5. Payment volume.
6. Revenue dashboard.
7. Support tickets.
8. Gateway health.
9. Audit logs.
10. Feature flags.
11. Platform settings.

### 26.2 Institution Admin Screens

1. Dashboard.
2. Students.
3. Parents.
4. Classes/sections/courses.
5. Fee structures.
6. Generate dues.
7. Payments.
8. Receipts.
9. Defaulters.
10. Reminders.
11. Reports.
12. Settings.
13. Discounts/concessions.
14. Refunds.
15. Reconciliation.
16. Staff/users.

### 26.3 Parent Portal Screens

1. Login with mobile OTP.
2. My children.
3. Pending fees.
4. Payment page.
5. Receipt history.
6. Support.
7. Payment status.
8. Profile.

---

## 27. What to Build First

For fastest market validation, build this order:

1. Tenant setup + institution profile.
2. Student and parent import.
3. Class/section/course setup.
4. Fee heads and fee structure.
5. Generate student dues.
6. Parent payment page.
7. Razorpay/UPI payment integration.
8. Webhook-based payment confirmation.
9. Receipt PDF generation.
10. Admin dashboard and reports.
11. Reminder system.
12. Offline payment entry.
13. Defaulter report.
14. CSV/Excel export.

This gives a usable MVP.

---

## 28. MVP Success Metrics

Track these from day one:

1. Number of institutions onboarded.
2. Number of active students.
3. Total fee amount generated.
4. Total fee amount collected.
5. Online payment success rate.
6. Payment failure rate.
7. Average collection time.
8. Number of reminders sent.
9. Reminder-to-payment conversion rate.
10. Monthly recurring revenue.
11. Transaction fee revenue.
12. Support tickets per 100 payments.
13. Receipt generation success rate.
14. Offline vs online collection ratio.
15. Number of defaulters reduced after reminders.
16. Institution churn rate.
17. Parent payment repeat rate.
18. Gateway settlement mismatch rate.

---

## 29. Final Feature Checklist

### 29.1 Must Have

1. Multi-tenant institution management.
2. Student and parent management.
3. Fee structure setup.
4. Fee assignment.
5. Online payments.
6. Offline payment entry.
7. Receipts.
8. Reminders.
9. Dashboard.
10. Reports.
11. Defaulter tracking.
12. Role-based access.
13. Payment webhook handling.
14. Export to Excel/CSV.
15. Parent payment page.
16. Payment status tracking.

### 29.2 Should Have

1. Partial payments.
2. Discounts/concessions.
3. Late fee automation.
4. Refunds.
5. Reconciliation.
6. WhatsApp templates.
7. Multi-branch support.
8. Custom receipt format.
9. Support tickets.
10. Settlement reports.
11. Cheque/DD tracking.
12. Approval workflows.

### 29.3 Later

1. White-label app.
2. Advanced analytics.
3. ERP integrations.
4. AI collection assistant.
5. Accounting integrations.
6. Multi-gateway routing.
7. No-due certificate.
8. Scholarship workflow.
9. Custom institution domain.
10. Advanced BI dashboards.

---

## 30. Best Starting Position

Start EduWallet as:

> A simple, India-first fee collection SaaS for schools, colleges, coaching centers, and institutions with UPI payments, automatic receipts, reminders, dashboards, and reports.

Do **not** start as a full ERP. Competing directly with large ERP platforms like Fedena/Edumarshal on all modules will make the product too big. Start with fee collection and become very strong there. Then expand into transport, hostel, attendance, accounting integrations, and ERP integrations later.

---

## 31. Recommended MVP Roadmap

### Month 1 — Foundation

1. Multi-tenant setup.
2. Institution onboarding.
3. User roles.
4. Student/parent import.
5. Class/section setup.
6. Basic UI dashboard.

### Month 2 — Fee Core

1. Fee heads.
2. Fee structures.
3. Fee assignment.
4. Invoice/demand generation.
5. Due tracking.
6. Offline payment entry.
7. Receipt generation.

### Month 3 — Online Payments

1. Razorpay integration.
2. UPI/card/netbanking checkout.
3. Payment links.
4. Webhook handling.
5. Parent payment page.
6. Payment status tracking.
7. Auto receipt generation.

### Month 4 — Reports and Reminders

1. Defaulter report.
2. Collection reports.
3. CSV/Excel export.
4. Reminder templates.
5. SMS/WhatsApp/email reminders.
6. Basic support tickets.

### Month 5 — Market Pilot

1. Onboard 3–5 pilot institutions.
2. Migrate student data.
3. Collect feedback from admins and parents.
4. Fix payment and receipt issues.
5. Improve reports.
6. Prepare pricing.

### Month 6 — Paid Launch

1. Launch SaaS plans.
2. Add subscription billing.
3. Add reconciliation improvements.
4. Add partial payment.
5. Add discount/concession support.
6. Start sales outreach.

---

## 32. Sales Pitch

### Simple Pitch

EduWallet helps schools, colleges, and coaching centers collect fees online through UPI/card, generate receipts automatically, send reminders, and track dues from one dashboard.

### Management Pitch

EduWallet reduces manual work, improves fee collection speed, reduces missed payments, improves financial visibility, and gives management real-time control over collections and dues.

### Parent Pitch

Parents can pay fees from mobile, download receipts instantly, and avoid standing in queues or depending on paper receipts.

---

## 33. One-Line Tagline Ideas

1. Fee collection made simple for Indian institutions.
2. UPI-first fee management for schools and colleges.
3. Collect, track, remind, and reconcile fees in one dashboard.
4. Replace paper receipts with digital fee collection.
5. Smart fee management for schools, colleges, and coaching centers.

---

## 34. Conclusion

EduWallet has strong potential if it focuses on one clear wedge: **fee collection and payment operations for Indian institutions**.

The first version should avoid becoming a full ERP. It should focus on:

1. Student and parent data.
2. Fee structures.
3. Online payments.
4. Offline payment entry.
5. Receipts.
6. Reminders.
7. Defaulter reports.
8. Collection dashboards.
9. Exports.
10. Reconciliation.

Once EduWallet becomes trusted for collections, it can expand into transport, hostel, accounting, ERP integrations, and white-label apps.
