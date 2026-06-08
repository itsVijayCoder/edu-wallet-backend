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

// Handlers aggregates all handler structs for dependency injection into the router.
type Handlers struct {
	Health   *handler.HealthHandler
	Auth     *handler.AuthHandler
	User     *handler.AdminUserHandler
	Tenant   *handler.TenantHandler
	Academic *handler.AcademicHandler
	Billing  *handler.BillingHandler
}

// RouterConfig holds router-level configuration.
type RouterConfig struct {
	AppEnv      string
	AppPort     int
	ExternalURL string
	CORSOrigins []string
}

// New creates a fully configured *gin.Engine with all middleware and route groups.
func New(log *slog.Logger, cfg RouterConfig, tokenMgr jwt.TokenManager, rdb *redis.Client, h Handlers) *gin.Engine {
	// Set gin mode based on environment.
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

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
		// Health probes (no auth required).
		v1.GET("/healthz", h.Health.Healthz)
		v1.GET("/readyz", h.Health.Readyz)

		// Auth routes.
		auth := v1.Group("/auth")
		{
			auth.POST("/login", middleware.RateLimit(rdb, 5, time.Minute), h.Auth.Login)
			auth.POST("/register", middleware.RateLimit(rdb, 5, time.Minute), h.Auth.Register)
			auth.POST("/refresh", h.Auth.Refresh)
			auth.POST("/select-tenant", middleware.Auth(tokenMgr), h.Auth.SelectTenant)
			auth.POST("/logout", middleware.Auth(tokenMgr), h.Auth.Logout)
			auth.POST("/forgot-password", middleware.RateLimit(rdb, 3, time.Hour), h.Auth.ForgotPassword)
			auth.POST("/reset-password", h.Auth.ResetPassword)
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
			}

			imports := adminTenant.Group("/imports", middleware.PermissionGuard("imports.manage"))
			{
				imports.GET("", h.Academic.ListImports)
				imports.GET("/students/template", h.Academic.StudentImportTemplate)
				imports.POST("/students/preview", h.Academic.PreviewStudentImport)
				imports.POST("/students/commit", h.Academic.CommitStudentImport)
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
			}

			invoices := adminTenant.Group("/invoices", middleware.PermissionGuard("fees.manage"))
			{
				invoices.POST("/generate", h.Billing.GenerateInvoices)
				invoices.GET("", h.Billing.ListInvoices)
				invoices.GET("/:id", h.Billing.GetInvoice)
			}
		}

		parent := v1.Group("/parent", middleware.Auth(tokenMgr), middleware.RequireTenant())
		{
			parent.GET("/children/:id/dues", h.Billing.GetParentChildDues)
		}

		// --- ADD YOUR ROUTES HERE ---
	}

	return r
}
