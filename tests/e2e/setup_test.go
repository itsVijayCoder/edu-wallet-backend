package e2e

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/database"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/handler"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/repository/postgres"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/router"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/service"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/hasher"
	"github.com/itsVijayCoder/edu-wallet-backend/pkg/jwt"
)

// ---------------------------------------------------------------------------
// Shared container state (started once in TestMain)
// ---------------------------------------------------------------------------

var (
	pgPool *pgxpool.Pool
	rdb    *redis.Client
)

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		_, _ = fmt.Fprintln(os.Stdout, "skipping e2e tests in short mode")
		os.Exit(0)
	}

	ctx := context.Background()

	// --- Start Postgres container ---
	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start postgres container: %v\n", err)
		os.Exit(1)
	}

	pgConnStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get postgres connection string: %v\n", err)
		os.Exit(1)
	}

	pgPool, err = pgxpool.New(ctx, pgConnStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pg pool: %v\n", err)
		os.Exit(1)
	}

	// --- Run migrations ---
	mig, err := migrate.New("file://../../migrations", pgConnStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create migrator: %v\n", err)
		os.Exit(1)
	}
	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		fmt.Fprintf(os.Stderr, "failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	// --- Start Redis container ---
	redisContainer, err := tcredis.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(15*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start redis container: %v\n", err)
		os.Exit(1)
	}

	redisEndpoint, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get redis endpoint: %v\n", err)
		os.Exit(1)
	}

	rdb = redis.NewClient(&redis.Options{Addr: redisEndpoint})
	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping redis: %v\n", err)
		os.Exit(1)
	}

	// --- Run tests ---
	code := m.Run()

	// --- Cleanup ---
	pgPool.Close()
	_ = rdb.Close()
	_ = pgContainer.Terminate(ctx)
	_ = redisContainer.Terminate(ctx)

	os.Exit(code)
}

// ---------------------------------------------------------------------------
// TestSuite wires the full application stack using test containers.
// ---------------------------------------------------------------------------

type TestSuite struct {
	Server   *gin.Engine
	Pool     *pgxpool.Pool
	Redis    *redis.Client
	TokenMgr jwt.TokenManager
	Hasher   hasher.Hasher
}

// SetupSuite creates a fully wired test application, mirroring main.go DI
// but using the shared test containers.
func SetupSuite(t *testing.T) *TestSuite {
	t.Helper()
	gin.SetMode(gin.TestMode)

	h := hasher.NewBcryptHasher(4) // low cost for fast tests
	tokenMgr := jwt.NewTokenManager(
		"test-access-secret-that-is-at-least-32-chars!!",
		"test-refresh-secret-that-is-at-least-32-chars!",
		15*time.Minute,
		7*24*time.Hour,
		"test-app",
	)

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Repositories
	roleRepo := postgres.NewRoleRepository(pgPool)
	userRepo := postgres.NewUserRepository(pgPool)
	tenantRepo := postgres.NewTenantRepository(pgPool)
	membershipRepo := postgres.NewTenantMembershipRepository(pgPool)
	auditRepo := postgres.NewAuditRepository(pgPool)
	academicRepo := postgres.NewAcademicRepository(pgPool)
	billingRepo := postgres.NewBillingRepository(pgPool)
	paymentRepo := postgres.NewPaymentRepository(pgPool)
	operationsRepo := postgres.NewOperationsRepository(pgPool)
	transactor := database.NewTransactor(pgPool)

	// Services - use a no-op email service for e2e tests
	emailSvc := &noopEmailService{}
	authSvc := service.NewAuthService(userRepo, h, tokenMgr, rdb, 7*24*time.Hour, emailSvc, log, true, membershipRepo)
	userSvc := service.NewUserService(userRepo, roleRepo, h, rdb)
	tenantSvc := service.NewTenantService(tenantRepo, membershipRepo, roleRepo, auditRepo)
	academicSvc := service.NewAcademicService(academicRepo, postgres.NewAcademicRepository, transactor, auditRepo)
	paymentProvider := service.NewFakePaymentProvider("fake", "test_payment_secret")
	paymentSvc := service.NewPaymentService(paymentRepo, postgres.NewPaymentRepository, academicRepo, transactor, auditRepo, paymentProvider, service.NewPDFReceiptRenderer())
	billingSvc := service.NewBillingService(billingRepo, postgres.NewBillingRepository, academicRepo, transactor, auditRepo, paymentRepo)
	operationsSvc := service.NewOperationsService(operationsRepo, postgres.NewOperationsRepository, transactor, auditRepo, service.NewNotificationProvider(nil))

	// Router
	r := router.New(log, router.RouterConfig{
		AppEnv:      "test",
		AppPort:     0,
		ExternalURL: "http://localhost:0",
		CORSOrigins: []string{"*"},
	}, tokenMgr, rdb, router.Handlers{
		Health:   handler.NewHealthHandler(pgPool, rdb),
		Auth:     handler.NewAuthHandler(authSvc),
		User:     handler.NewAdminUserHandler(userSvc),
		Tenant:   handler.NewTenantHandler(tenantSvc),
		Academic: handler.NewAcademicHandler(academicSvc),
		Billing:  handler.NewBillingHandler(billingSvc),
		Payment:  handler.NewPaymentHandler(paymentSvc),
		Ops:      handler.NewOperationsHandler(operationsSvc),
	})

	return &TestSuite{
		Server:   r,
		Pool:     pgPool,
		Redis:    rdb,
		TokenMgr: tokenMgr,
		Hasher:   h,
	}
}

// noopEmailService is a test double that silently succeeds.
type noopEmailService struct{}

func (n *noopEmailService) SendPasswordReset(_ context.Context, _, _ string) error { return nil }
func (n *noopEmailService) SendWelcome(_ context.Context, _, _ string) error       { return nil }

// ---------------------------------------------------------------------------
// truncateAndReseed clears all data (except schema_migrations) and
// re-inserts the default roles so every test starts from a clean slate.
// ---------------------------------------------------------------------------

func truncateAndReseed(t *testing.T, pool *pgxpool.Pool, rdb *redis.Client) {
	t.Helper()
	ctx := context.Background()

	// Truncate all tables except schema_migrations.
	_, err := pool.Exec(ctx, `
		DO $$
		DECLARE
			r RECORD;
		BEGIN
			FOR r IN
				SELECT tablename FROM pg_tables
				WHERE schemaname = 'public'
				  AND tablename != 'schema_migrations'
			LOOP
				EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$;
	`)
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	// Re-insert default roles.
	_, err = pool.Exec(ctx, `
		INSERT INTO roles (name, slug, description) VALUES
			('Super Admin', 'super_admin', 'Full system access'),
			('Admin', 'admin', 'Administrative access'),
			('Member', 'member', 'Standard member access')
		ON CONFLICT DO NOTHING;

		INSERT INTO permissions (code, name, category, description) VALUES
			('platform.tenants.manage', 'Manage Tenants', 'platform', 'Create and update tenant accounts'),
			('tenant.read', 'Read Tenant', 'tenant', 'Read current tenant profile'),
			('tenant.update', 'Update Tenant', 'tenant', 'Update current tenant profile'),
			('branches.create', 'Create Branches', 'tenant', 'Create tenant branches'),
			('users.manage', 'Manage Users', 'tenant', 'Create and manage tenant users'),
			('academic.manage', 'Manage Academic Setup', 'tenant', 'Create and update academic years, classes, and sections'),
			('students.manage', 'Manage Students', 'tenant', 'Create, update, and list tenant students'),
			('guardians.manage', 'Manage Guardians', 'tenant', 'Create, update, and list tenant guardians'),
			('imports.manage', 'Manage Imports', 'tenant', 'Preview and commit student imports'),
			('fees.manage', 'Manage Fees', 'tenant', 'Create fee setup, assignments, and generated invoices'),
			('payments.manage', 'Manage Payments', 'tenant', 'Record payments, process webhooks, and manage receipts'),
			('reminders.manage', 'Manage Reminders', 'tenant', 'Create reminder templates, rules, and send reminders'),
			('reports.view', 'View Reports', 'tenant', 'View dashboard and financial reports'),
			('exports.manage', 'Manage Exports', 'tenant', 'Create and download report exports')
		ON CONFLICT (code) DO UPDATE
		SET name = EXCLUDED.name,
			category = EXCLUDED.category,
			description = EXCLUDED.description;

		INSERT INTO role_permissions (role_id, permission_id)
		SELECT r.id, p.id
		FROM roles r
		CROSS JOIN permissions p
		WHERE r.slug = 'super_admin'
		ON CONFLICT DO NOTHING;

		INSERT INTO role_permissions (role_id, permission_id)
		SELECT r.id, p.id
		FROM roles r
		JOIN permissions p ON p.code IN (
			'tenant.read',
			'tenant.update',
			'branches.create',
			'users.manage',
			'academic.manage',
			'students.manage',
			'guardians.manage',
			'imports.manage',
			'fees.manage',
			'payments.manage',
			'reminders.manage',
			'reports.view',
			'exports.manage'
		)
		WHERE r.slug = 'admin'
		ON CONFLICT DO NOTHING;
	`)
	if err != nil {
		t.Fatalf("reseed roles: %v", err)
	}

	// Flush Redis.
	if err := rdb.FlushAll(ctx).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
}

// ---------------------------------------------------------------------------
// SeedUser creates a test user with the specified roles and returns the user ID.
// ---------------------------------------------------------------------------

func SeedUser(t *testing.T, pool *pgxpool.Pool, h hasher.Hasher, email, password string, roleSlugs []string) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	hash, err := h.Hash(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	var userID uuid.UUID
	err = pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, first_name, last_name, status)
		 VALUES ($1, $2, 'Test', 'User', 'active')
		 RETURNING id`,
		email, hash,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	for _, slug := range roleSlugs {
		_, err := pool.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_id)
			 SELECT $1, id FROM roles WHERE slug = $2`,
			userID, slug,
		)
		if err != nil {
			t.Fatalf("assign role %s: %v", slug, err)
		}
	}

	return userID
}
