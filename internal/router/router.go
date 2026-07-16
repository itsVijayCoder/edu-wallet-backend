package router

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/handler"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/middleware"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/jwt"
)

const (
	authBodyLimitBytes       = 16 * 1024
	paymentBodyLimitBytes    = 32 * 1024
	importBodyLimitBytes     = 6 * 1024 * 1024
	webhookBodyLimitBytes    = 1 * 1024 * 1024
	offlinePaymentLimitBytes = 64 * 1024
)

// Handlers aggregates all handler structs for dependency injection into the router.
type Handlers struct {
	Docs     *handler.DocsHandler
	Health   *handler.HealthHandler
	Auth     *handler.AuthHandler
	User     *handler.AdminUserHandler
	Tenant   *handler.TenantHandler
	Academic *handler.AcademicHandler
	Billing  *handler.BillingHandler
	Payment  *handler.PaymentHandler
	Ops      *handler.OperationsHandler
	Parent   *handler.ParentHandler
}

// RouterConfig holds router-level configuration.
type RouterConfig struct {
	AppEnv         string
	AppPort        int
	ExternalURL    string
	CORSOrigins    []string
	TrustedProxies []string
}

// New creates a fully configured *gin.Engine with all middleware and route groups.
func New(log *slog.Logger, cfg RouterConfig, tokenMgr jwt.TokenManager, rdb *redis.Client, h Handlers) *gin.Engine {
	// Set gin mode based on environment.
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	// Only honor X-Forwarded-For from explicitly configured infrastructure.
	// Otherwise a client could forge its source IP and bypass IP-based limits.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Error("invalid trusted proxy configuration; forwarded headers disabled", "error", err)
		_ = r.SetTrustedProxies(nil)
	}
	r.MaxMultipartMemory = importBodyLimitBytes

	// --- Global middleware chain ---
	r.Use(
		middleware.Recovery(log),
		middleware.RequestID(),
		middleware.SecurityHeaders(),
		middleware.CORS(cfg.AppEnv, cfg.CORSOrigins),
		middleware.Logger(log),
	)

	// --- API v1 ---
	v1 := r.Group("/api/v1")
	{
		v1.GET("/docs", h.Docs.SwaggerUI)
		v1.GET("/docs/api-test", h.Docs.APITestGuide)
		v1.GET("/docs/openapi.json", h.Docs.OpenAPIJSON)
		v1.GET("/docs/swagger.json", h.Docs.OpenAPIJSON)

		// Health probes (no auth required).
		v1.GET("/healthz", h.Health.Healthz)
		v1.GET("/readyz", h.Health.Readyz)

		// Auth routes.
		auth := v1.Group("/auth")
		{
			auth.POST("/login", middleware.BodySizeLimit(authBodyLimitBytes), middleware.RateLimit(rdb, 5, time.Minute), h.Auth.Login)
			auth.POST("/send-otp", middleware.BodySizeLimit(authBodyLimitBytes), h.Auth.SendOTP)
			auth.POST("/verify-otp", middleware.BodySizeLimit(authBodyLimitBytes), middleware.RateLimit(rdb, 10, 5*time.Minute), h.Auth.VerifyOTP)
			auth.POST("/register", middleware.BodySizeLimit(authBodyLimitBytes), middleware.RateLimit(rdb, 5, time.Minute), h.Auth.Register)
			auth.POST("/refresh", h.Auth.Refresh)
			auth.POST("/select-tenant", middleware.BodySizeLimit(authBodyLimitBytes), middleware.RateLimit(rdb, 20, time.Minute), middleware.Auth(tokenMgr), h.Auth.SelectTenant)
			auth.POST("/logout", middleware.Auth(tokenMgr), h.Auth.Logout)
			auth.POST("/forgot-password", middleware.BodySizeLimit(authBodyLimitBytes), middleware.RateLimit(rdb, 3, time.Hour), h.Auth.ForgotPassword)
			auth.POST("/reset-password", middleware.BodySizeLimit(authBodyLimitBytes), middleware.RateLimit(rdb, 5, 15*time.Minute), h.Auth.ResetPassword)
		}

		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/razorpay", middleware.BodySizeLimit(webhookBodyLimitBytes), middleware.RateLimit(rdb, 120, time.Minute), h.Payment.RazorpayWebhook)
		}

		// Platform routes are reserved for platform super admins.
		platform := v1.Group("/platform", middleware.Auth(tokenMgr), middleware.RoleGuard("super_admin"))
		{
			tenants := platform.Group("/tenants")
			{
				tenants.POST("", h.Tenant.PlatformCreate)
				tenants.GET("", h.Tenant.PlatformList)
				tenants.GET("/:id", h.Tenant.PlatformGet)
				tenants.PATCH("/:id", h.Tenant.PlatformUpdate)
				tenants.POST("/:id/branches", h.Tenant.PlatformCreateBranch)
			}
		}

		// Admin routes (authenticated + role-guarded).
		admin := v1.Group("/admin", middleware.Auth(tokenMgr), middleware.RoleGuard("super_admin", "admin"))
		{
			users := admin.Group("/users")
			{
				users.POST("", h.User.Create)
				users.GET("", h.User.List)
				users.GET("/:id", h.User.GetByID)
				users.PUT("/:id", h.User.Update)
				users.DELETE("/:id", h.User.Delete)
			}
		}

		adminTenant := v1.Group("/admin", middleware.Auth(tokenMgr), middleware.RequireTenant())
		{
			adminTenant.GET("/tenant", middleware.PermissionGuard("tenant.read"), h.Tenant.AdminGet)
			adminTenant.PATCH("/tenant", middleware.PermissionGuard("tenant.update"), h.Tenant.AdminUpdate)
			adminTenant.POST("/tenant/users", middleware.PermissionGuard("users.manage"), h.User.CreateTenantUser)

			academicYears := adminTenant.Group("/academic-years", middleware.PermissionGuard("academic.manage"))
			{
				academicYears.POST("", h.Academic.CreateAcademicYear)
				academicYears.GET("", h.Academic.ListAcademicYears)
				academicYears.GET("/:id", h.Academic.GetAcademicYear)
				academicYears.PATCH("/:id", h.Academic.UpdateAcademicYear)
				academicYears.DELETE("/:id", h.Academic.DeleteAcademicYear)
			}

			classes := adminTenant.Group("/classes", middleware.PermissionGuard("academic.manage"))
			{
				classes.POST("", h.Academic.CreateClass)
				classes.GET("", h.Academic.ListClasses)
				classes.GET("/:id", h.Academic.GetClass)
				classes.PATCH("/:id", h.Academic.UpdateClass)
				classes.DELETE("/:id", h.Academic.DeleteClass)
			}

			sections := adminTenant.Group("/sections", middleware.PermissionGuard("academic.manage"))
			{
				sections.POST("", h.Academic.CreateSection)
				sections.GET("", h.Academic.ListSections)
				sections.GET("/:id", h.Academic.GetSection)
				sections.PATCH("/:id", h.Academic.UpdateSection)
				sections.DELETE("/:id", h.Academic.DeleteSection)
			}

			students := adminTenant.Group("/students", middleware.PermissionGuard("students.manage"))
			{
				students.POST("", h.Academic.CreateStudent)
				students.GET("", h.Academic.ListStudents)
				students.GET("/:id", h.Academic.GetStudent)
				students.PATCH("/:id", h.Academic.UpdateStudent)
				students.DELETE("/:id", h.Academic.DeleteStudent)
				students.POST("/:id/guardians", h.Academic.LinkStudentGuardian)
				students.DELETE("/:id/guardians/:guardian_id", h.Academic.UnlinkStudentGuardian)
			}

			guardians := adminTenant.Group("/guardians", middleware.PermissionGuard("guardians.manage"))
			{
				guardians.POST("", h.Academic.CreateGuardian)
				guardians.GET("", h.Academic.ListGuardians)
				guardians.GET("/:id", h.Academic.GetGuardian)
				guardians.PATCH("/:id", h.Academic.UpdateGuardian)
				guardians.DELETE("/:id", h.Academic.DeleteGuardian)
				guardians.GET("/:id/students", h.Parent.ListGuardianStudents)
				guardians.POST("/:id/user", h.Parent.LinkGuardianUser)
				guardians.DELETE("/:id/user", h.Parent.UnlinkGuardianUser)
			}

			adminTenant.GET("/parents", middleware.PermissionGuard("guardians.manage"), h.Parent.ListParents)

			imports := adminTenant.Group("/imports", middleware.PermissionGuard("imports.manage"))
			{
				imports.GET("", h.Academic.ListImports)
				imports.GET("/students/template", h.Academic.StudentImportTemplate)
				imports.POST("/students/preview", middleware.BodySizeLimit(importBodyLimitBytes), h.Academic.PreviewStudentImport)
				imports.POST("/students/commit", middleware.BodySizeLimit(authBodyLimitBytes), h.Academic.CommitStudentImport)
			}

			adminTenant.GET("/students/:id/ledger", middleware.PermissionGuard("fees.manage"), h.Billing.GetStudentLedger)

			feeHeads := adminTenant.Group("/fee-heads", middleware.PermissionGuard("fees.manage"))
			{
				feeHeads.POST("", h.Billing.CreateFeeHead)
				feeHeads.GET("", h.Billing.ListFeeHeads)
				feeHeads.GET("/:id", h.Billing.GetFeeHead)
				feeHeads.PATCH("/:id", h.Billing.UpdateFeeHead)
				feeHeads.DELETE("/:id", h.Billing.DeleteFeeHead)
			}

			feeStructures := adminTenant.Group("/fee-structures", middleware.PermissionGuard("fees.manage"))
			{
				feeStructures.POST("", h.Billing.CreateFeeStructure)
				feeStructures.GET("", h.Billing.ListFeeStructures)
				feeStructures.GET("/:id", h.Billing.GetFeeStructure)
				feeStructures.PATCH("/:id", h.Billing.UpdateFeeStructure)
				feeStructures.DELETE("/:id", h.Billing.DeleteFeeStructure)
			}

			feeAssignments := adminTenant.Group("/fee-assignments", middleware.PermissionGuard("fees.manage"))
			{
				feeAssignments.POST("", h.Billing.CreateFeeAssignment)
				feeAssignments.GET("", h.Billing.ListFeeAssignments)
				feeAssignments.GET("/:id", h.Billing.GetFeeAssignment)
				feeAssignments.PATCH("/:id", h.Billing.UpdateFeeAssignment)
				feeAssignments.DELETE("/:id", h.Billing.DeleteFeeAssignment)
			}

			invoices := adminTenant.Group("/invoices", middleware.PermissionGuard("fees.manage"))
			{
				invoices.POST("/generate", h.Billing.GenerateInvoices)
				invoices.GET("", h.Billing.ListInvoices)
				invoices.GET("/:id", h.Billing.GetInvoice)
			}

			adminTenant.POST("/offline-payments", middleware.PermissionGuard("payments.manage"), middleware.BodySizeLimit(offlinePaymentLimitBytes), h.Payment.CreateOfflinePayment)

			payments := adminTenant.Group("/payments", middleware.PermissionGuard("payments.manage"))
			{
				payments.GET("", h.Payment.ListPayments)
				payments.GET("/:id", h.Payment.GetPayment)
			}

			receipts := adminTenant.Group("/receipts", middleware.PermissionGuard("payments.manage"))
			{
				receipts.GET("", h.Payment.ListReceipts)
				receipts.GET("/:id", h.Payment.GetReceipt)
				receipts.GET("/:id/download", h.Payment.DownloadReceipt)
			}

			adminTenant.GET("/payment-events", middleware.PermissionGuard("payments.manage"), h.Payment.ListPaymentEvents)

			reminderTemplates := adminTenant.Group("/reminder-templates", middleware.PermissionGuard("reminders.manage"))
			{
				reminderTemplates.POST("", h.Ops.CreateReminderTemplate)
				reminderTemplates.GET("", h.Ops.ListReminderTemplates)
				reminderTemplates.GET("/:id", h.Ops.GetReminderTemplate)
				reminderTemplates.PATCH("/:id", h.Ops.UpdateReminderTemplate)
				reminderTemplates.DELETE("/:id", h.Ops.DeleteReminderTemplate)
			}

			reminderRules := adminTenant.Group("/reminder-rules", middleware.PermissionGuard("reminders.manage"))
			{
				reminderRules.POST("", h.Ops.CreateReminderRule)
				reminderRules.GET("", h.Ops.ListReminderRules)
				reminderRules.GET("/:id", h.Ops.GetReminderRule)
				reminderRules.PATCH("/:id", h.Ops.UpdateReminderRule)
				reminderRules.DELETE("/:id", h.Ops.DeleteReminderRule)
			}

			adminTenant.POST("/reminders/send", middleware.PermissionGuard("reminders.manage"), h.Ops.SendReminders)
			adminTenant.GET("/reminder-logs", middleware.PermissionGuard("reminders.manage"), h.Ops.ListReminderLogs)

			adminTenant.GET("/dashboard", middleware.PermissionGuard("reports.view"), h.Ops.GetDashboard)
			reports := adminTenant.Group("/reports", middleware.PermissionGuard("reports.view"))
			{
				reports.GET("/collections", h.Ops.CollectionsReport)
				reports.GET("/defaulters", h.Ops.DefaultersReport)
				reports.GET("/dues", h.Ops.DuesReport)
				reports.GET("/fee-heads", h.Ops.FeeHeadsReport)
				reports.GET("/payment-methods", h.Ops.PaymentMethodsReport)
				reports.GET("/offline-payments", h.Ops.OfflinePaymentsReport)
			}

			exports := adminTenant.Group("/exports", middleware.PermissionGuard("exports.manage"))
			{
				exports.POST("", h.Ops.CreateExport)
				exports.GET("", h.Ops.ListExports)
				exports.GET("/:id", h.Ops.GetExport)
				exports.GET("/:id/download", h.Ops.DownloadExport)
			}
		}

		parent := v1.Group("/parent", middleware.Auth(tokenMgr), middleware.RequireTenant(), middleware.RoleGuard("parents"))
		{
			parent.GET("/children", h.Parent.ListLinkedChildren)
			parent.GET("/children/:id/dues", h.Billing.GetParentChildDues)
			parent.POST("/payments/orders", middleware.BodySizeLimit(paymentBodyLimitBytes), middleware.RateLimit(rdb, 30, time.Minute), h.Payment.CreatePaymentOrder)
			parent.POST("/payments/verify", middleware.BodySizeLimit(paymentBodyLimitBytes), middleware.RateLimit(rdb, 60, time.Minute), h.Payment.VerifyPayment)
			parent.GET("/receipts", h.Payment.ListParentReceipts)
			parent.GET("/receipts/:id/download", h.Payment.DownloadParentReceipt)
		}

		// --- ADD YOUR ROUTES HERE ---
	}

	return r
}
