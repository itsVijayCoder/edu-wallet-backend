package apidoc

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const defaultServerURL = "http://localhost:8080"

var routeParamPattern = regexp.MustCompile(`:([A-Za-z_][A-Za-z0-9_]*)`)

// Parameter describes one OpenAPI operation parameter.
type Parameter struct {
	Name        string
	In          string
	Description string
	Type        string
	Format      string
	Required    bool
	Enum        []string
}

// RequestBody describes one OpenAPI request body.
type RequestBody struct {
	SchemaRef    string
	ContentTypes []string
	Required     bool
	Description  string
}

// Endpoint describes one documented API route.
type Endpoint struct {
	Method              string
	Path                string
	Tags                []string
	Summary             string
	Description         string
	AuthRequired        bool
	Permissions         []string
	Request             *RequestBody
	Status              int
	Query               []Parameter
	RateLimited         bool
	DownloadContentType string
}

type endpointOption func(*Endpoint)

func endpoint(method, path, tag, summary string, opts ...endpointOption) Endpoint {
	e := Endpoint{
		Method:  method,
		Path:    path,
		Tags:    []string{tag},
		Summary: summary,
		Status:  http.StatusOK,
	}
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

func secured(permissions ...string) endpointOption {
	return func(e *Endpoint) {
		e.AuthRequired = true
		e.Permissions = append([]string(nil), permissions...)
	}
}

func body(schemaRef string, contentTypes ...string) endpointOption {
	return func(e *Endpoint) {
		if len(contentTypes) == 0 {
			contentTypes = []string{"application/json"}
		}
		e.Request = &RequestBody{
			SchemaRef:    schemaRef,
			ContentTypes: append([]string(nil), contentTypes...),
			Required:     true,
		}
	}
}

func optionalBody(schemaRef string, contentTypes ...string) endpointOption {
	return func(e *Endpoint) {
		if len(contentTypes) == 0 {
			contentTypes = []string{"application/json"}
		}
		e.Request = &RequestBody{
			SchemaRef:    schemaRef,
			ContentTypes: append([]string(nil), contentTypes...),
			Required:     false,
		}
	}
}

func created() endpointOption {
	return func(e *Endpoint) {
		e.Status = http.StatusCreated
	}
}

func query(params ...Parameter) endpointOption {
	return func(e *Endpoint) {
		e.Query = append(e.Query, params...)
	}
}

func rateLimited() endpointOption {
	return func(e *Endpoint) {
		e.RateLimited = true
	}
}

func download(contentType string) endpointOption {
	return func(e *Endpoint) {
		e.DownloadContentType = contentType
	}
}

// Endpoints returns the documented API route catalog.
func Endpoints() []Endpoint {
	out := make([]Endpoint, len(endpointCatalog))
	copy(out, endpointCatalog)
	return out
}

// OpenAPIJSON returns the OpenAPI document as pretty JSON.
func OpenAPIJSON(serverURL string) ([]byte, error) {
	return json.MarshalIndent(OpenAPISpec(serverURL), "", "  ")
}

// OpenAPISpec builds the OpenAPI document.
func OpenAPISpec(serverURL string) map[string]any {
	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		serverURL = defaultServerURL
	}

	endpoints := Endpoints()
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path == endpoints[j].Path {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	paths := map[string]any{}
	for _, ep := range endpoints {
		path := RuntimePathToOpenAPI(ep.Path)
		pathItem, ok := paths[path].(map[string]any)
		if !ok {
			pathItem = map[string]any{}
			paths[path] = pathItem
		}
		pathItem[strings.ToLower(ep.Method)] = operationFor(ep)
	}

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "EduWallet Backend API",
			"description": "OpenAPI documentation for the EduWallet backend MVP API.",
			"version":     "1.0.0",
		},
		"servers": []map[string]string{
			{"url": serverURL},
		},
		"tags":  tagDocs(),
		"paths": paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"BearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
			"schemas": schemaComponents(),
		},
	}
}

// RuntimePathToOpenAPI converts Gin route params such as :id to {id}.
func RuntimePathToOpenAPI(path string) string {
	return routeParamPattern.ReplaceAllString(path, `{$1}`)
}

// RouteKey returns a normalized METHOD /openapi/path key for docs coverage tests.
func RouteKey(method, path string) string {
	return strings.ToUpper(method) + " " + RuntimePathToOpenAPI(path)
}

func operationFor(ep Endpoint) map[string]any {
	description := ep.Description
	if len(ep.Permissions) > 0 {
		permissionText := "Requires permission: " + strings.Join(ep.Permissions, ", ") + "."
		if description == "" {
			description = permissionText
		} else {
			description += "\n\n" + permissionText
		}
	}

	op := map[string]any{
		"operationId": operationID(ep.Method, ep.Path),
		"summary":     ep.Summary,
		"tags":        ep.Tags,
		"responses":   responsesFor(ep),
	}
	if description != "" {
		op["description"] = description
	}
	params := append(pathParameters(ep.Path), ep.Query...)
	if len(params) > 0 {
		openAPIParams := make([]map[string]any, 0, len(params))
		for _, param := range params {
			openAPIParams = append(openAPIParams, param.toOpenAPI())
		}
		op["parameters"] = openAPIParams
	}
	if ep.AuthRequired {
		op["security"] = []map[string][]string{{"BearerAuth": []string{}}}
	}
	if ep.Request != nil {
		op["requestBody"] = ep.Request.toOpenAPI()
	}
	return op
}

func (b RequestBody) toOpenAPI() map[string]any {
	content := map[string]any{}
	for _, contentType := range b.ContentTypes {
		content[contentType] = map[string]any{
			"schema": ref(b.SchemaRef),
		}
	}
	out := map[string]any{
		"required": b.Required,
		"content":  content,
	}
	if b.Description != "" {
		out["description"] = b.Description
	}
	return out
}

func (p Parameter) toOpenAPI() map[string]any {
	schema := map[string]any{"type": p.Type}
	if p.Format != "" {
		schema["format"] = p.Format
	}
	if len(p.Enum) > 0 {
		schema["enum"] = p.Enum
	}
	out := map[string]any{
		"name":        p.Name,
		"in":          p.In,
		"required":    p.Required,
		"description": p.Description,
		"schema":      schema,
	}
	return out
}

func responsesFor(ep Endpoint) map[string]any {
	status := ep.Status
	if status == 0 {
		status = http.StatusOK
	}

	responses := map[string]any{
		statusCode(status): successResponse(ep),
	}

	if ep.Method != http.MethodGet || strings.Contains(ep.Path, ":") {
		responses[statusCode(http.StatusBadRequest)] = errorResponse("Validation failed")
	}
	if ep.AuthRequired {
		responses[statusCode(http.StatusUnauthorized)] = errorResponse("Missing, invalid, or expired JWT")
		responses[statusCode(http.StatusForbidden)] = errorResponse("Role or permission check failed")
	}
	if strings.Contains(ep.Path, ":") {
		responses[statusCode(http.StatusNotFound)] = errorResponse("Resource not found")
	}
	if ep.RateLimited {
		responses[statusCode(http.StatusTooManyRequests)] = errorResponse("Rate limit exceeded")
	}
	return responses
}

func successResponse(ep Endpoint) map[string]any {
	if ep.DownloadContentType != "" {
		return map[string]any{
			"description": http.StatusText(ep.Status),
			"content": map[string]any{
				ep.DownloadContentType: map[string]any{
					"schema": map[string]any{
						"type":   "string",
						"format": "binary",
					},
				},
			},
		}
	}
	return map[string]any{
		"description": http.StatusText(ep.Status),
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": ref("APIResponse"),
			},
		},
	}
}

func errorResponse(description string) map[string]any {
	return map[string]any{
		"description": description,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": ref("APIResponse"),
			},
		},
	}
}

func statusCode(status int) string {
	return strconv.Itoa(status)
}

func pathParameters(path string) []Parameter {
	matches := routeParamPattern.FindAllStringSubmatch(path, -1)
	params := make([]Parameter, 0, len(matches))
	for _, match := range matches {
		params = append(params, Parameter{
			Name:        match[1],
			In:          "path",
			Description: "Route identifier.",
			Type:        "string",
			Format:      "uuid",
			Required:    true,
		})
	}
	return params
}

func operationID(method, path string) string {
	normalized := RuntimePathToOpenAPI(path)
	replacer := strings.NewReplacer(
		"/api/v1/", "",
		"/", "_",
		"{", "",
		"}", "",
		"-", "_",
	)
	normalized = strings.Trim(replacer.Replace(normalized), "_")
	return strings.ToLower(method) + "_" + normalized
}

func tagDocs() []map[string]string {
	return []map[string]string{
		{"name": "Docs", "description": "Swagger UI and OpenAPI specification endpoints."},
		{"name": "Health", "description": "Liveness and readiness probes."},
		{"name": "Auth", "description": "Authentication, refresh tokens, tenant selection, and password reset."},
		{"name": "Platform Tenants", "description": "Super admin tenant and branch management."},
		{"name": "Admin Users", "description": "Platform/admin user management."},
		{"name": "Tenant", "description": "Selected tenant profile management."},
		{"name": "Academic Setup", "description": "Academic years, classes, and sections."},
		{"name": "Students", "description": "Students, guardians, links, and imports."},
		{"name": "Billing", "description": "Fee heads, structures, assignments, invoices, dues, and ledgers."},
		{"name": "Payments", "description": "Payment orders, verification, webhooks, offline payments, receipts, and events."},
		{"name": "Operations", "description": "Reminders, dashboard, reports, and exports."},
	}
}

func ref(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func qString(name, description string) Parameter {
	return Parameter{Name: name, In: "query", Description: description, Type: "string"}
}

func qUUID(name, description string) Parameter {
	return Parameter{Name: name, In: "query", Description: description, Type: "string", Format: "uuid"}
}

func qDate(name, description string) Parameter {
	return Parameter{Name: name, In: "query", Description: description, Type: "string", Format: "date"}
}

func qInt(name, description string) Parameter {
	return Parameter{Name: name, In: "query", Description: description, Type: "integer", Format: "int32"}
}

func qEnum(name, description string, values ...string) Parameter {
	return Parameter{Name: name, In: "query", Description: description, Type: "string", Enum: values}
}

func paginationParams() []Parameter {
	return []Parameter{
		qInt("page", "Page number, starting at 1."),
		qInt("page_size", "Items per page."),
		qString("sort_by", "Sort field."),
		qEnum("sort_dir", "Sort direction.", "asc", "desc"),
	}
}

func studentFilterParams() []Parameter {
	params := paginationParams()
	params = append(params,
		qUUID("academic_year_id", "Filter by academic year."),
		qUUID("class_id", "Filter by class."),
		qUUID("section_id", "Filter by section."),
		qString("status", "Filter by student status."),
		qString("search", "Search admission number, name, phone, or email."),
	)
	return params
}

func userFilterParams() []Parameter {
	params := paginationParams()
	params = append(params,
		qEnum("role", "Filter users by role slug.", "admin", "staff", "parents", "student", "super_admin"),
		qString("search", "Search email, first name, or last name."),
	)
	return params
}

func guardianFilterParams() []Parameter {
	params := paginationParams()
	params = append(params,
		qString("search", "Search guardian name, email, or phone."),
		qEnum("linked", "Filter by parent login link status.", "true", "false"),
	)
	return params
}

func paymentFilterParams() []Parameter {
	params := paginationParams()
	params = append(params,
		qUUID("student_id", "Filter by student."),
		qString("status", "Filter by payment status."),
		qString("payment_method", "Filter by payment method."),
		qString("provider", "Filter by payment provider."),
		qDate("from", "Start date."),
		qDate("to", "End date."),
		qString("search", "Search internal ID, gateway IDs, settlement references, or student details."),
	)
	return params
}

func feeAssignmentFilterParams() []Parameter {
	params := paginationParams()
	params = append(params,
		qString("search", "Search fee structure, class, section, or student details."),
		qEnum("assignment_type", "Filter by assignment type.", "class", "section", "student"),
		qEnum("status", "Filter by assignment status.", "active", "inactive", "cancelled"),
		qUUID("fee_structure_id", "Filter by fee structure."),
		qUUID("academic_year_id", "Filter by academic year."),
	)
	return params
}

func receiptFilterParams() []Parameter {
	params := paginationParams()
	params = append(params,
		qUUID("student_id", "Filter by student."),
		qString("status", "Filter by receipt status."),
		qDate("from", "Start date."),
		qDate("to", "End date."),
	)
	return params
}

func parentChildrenFilterParams() []Parameter {
	return []Parameter{
		qString("search", "Search child name or admission number."),
		qInt("page", "Page number, starting at 1."),
		qInt("page_size", "Items per page."),
	}
}

func parentDuesFilterParams() []Parameter {
	return []Parameter{
		qString("search", "Search invoice number."),
		qEnum("status", "Filter invoice status.", "paid", "pending", "partial", "overdue", "failed"),
		qDate("due_from", "Only invoices due on or after this date."),
		qDate("due_to", "Only invoices due on or before this date."),
	}
}

func parentReceiptFilterParams() []Parameter {
	return []Parameter{
		qString("search", "Search receipt number or student name."),
		qEnum("status", "Filter receipt status.", "issued", "cancelled"),
		qDate("from", "Only receipts issued on or after this date."),
		qDate("to", "Only receipts issued on or before this date."),
		qInt("page", "Page number, starting at 1."),
		qInt("page_size", "Items per page."),
	}
}

func reportFilterParams() []Parameter {
	return []Parameter{
		qDate("from", "Start date."),
		qDate("to", "End date."),
		qDate("as_of", "Point-in-time date for due/defaulter reports."),
		qUUID("student_id", "Filter by student."),
		qUUID("class_id", "Filter by class."),
		qUUID("section_id", "Filter by section."),
		qString("payment_method", "Filter by payment method."),
		qString("provider", "Filter by provider."),
	}
}

var endpointCatalog = []Endpoint{
	endpoint(http.MethodGet, "/api/v1/docs", "Docs", "Open Swagger UI"),
	endpoint(http.MethodGet, "/api/v1/docs/api-test", "Docs", "Open API tester guide"),
	endpoint(http.MethodGet, "/api/v1/docs/openapi.json", "Docs", "Download OpenAPI JSON"),
	endpoint(http.MethodGet, "/api/v1/docs/swagger.json", "Docs", "Download Swagger-compatible JSON"),

	endpoint(http.MethodGet, "/api/v1/healthz", "Health", "Liveness probe"),
	endpoint(http.MethodGet, "/api/v1/readyz", "Health", "Readiness probe"),

	endpoint(http.MethodPost, "/api/v1/auth/login", "Auth", "Login", body("LoginRequest"), rateLimited()),
	endpoint(http.MethodPost, "/api/v1/auth/register", "Auth", "Register user", body("RegisterRequest"), created(), rateLimited()),
	endpoint(http.MethodPost, "/api/v1/auth/refresh", "Auth", "Refresh access token", body("RefreshRequest")),
	endpoint(http.MethodPost, "/api/v1/auth/select-tenant", "Auth", "Select tenant context", secured(), body("SelectTenantRequest"), rateLimited()),
	endpoint(http.MethodPost, "/api/v1/auth/logout", "Auth", "Logout", secured()),
	endpoint(http.MethodPost, "/api/v1/auth/forgot-password", "Auth", "Request password reset", body("ForgotPasswordRequest"), rateLimited()),
	endpoint(http.MethodPost, "/api/v1/auth/reset-password", "Auth", "Reset password", body("ResetPasswordRequest"), rateLimited()),

	endpoint(http.MethodPost, "/api/v1/webhooks/razorpay", "Payments", "Process Razorpay webhook", body("RazorpayWebhookRequest"), rateLimited()),

	endpoint(http.MethodPost, "/api/v1/platform/tenants", "Platform Tenants", "Create tenant", secured(), body("CreateTenantRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/platform/tenants", "Platform Tenants", "List tenants", secured(), query(paginationParams()...)),
	endpoint(http.MethodGet, "/api/v1/platform/tenants/:id", "Platform Tenants", "Get tenant", secured()),
	endpoint(http.MethodPatch, "/api/v1/platform/tenants/:id", "Platform Tenants", "Update tenant", secured(), body("GenericPatchRequest")),
	endpoint(http.MethodPost, "/api/v1/platform/tenants/:id/branches", "Platform Tenants", "Create tenant branch", secured(), body("CreateBranchRequest"), created()),

	endpoint(http.MethodPost, "/api/v1/admin/users", "Admin Users", "Create user", secured(), body("CreateUserRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/users", "Admin Users", "List users", secured(), query(userFilterParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/users/:id", "Admin Users", "Get user", secured()),
	endpoint(http.MethodPut, "/api/v1/admin/users/:id", "Admin Users", "Update user", secured(), body("UpdateUserRequest")),
	endpoint(http.MethodDelete, "/api/v1/admin/users/:id", "Admin Users", "Delete user", secured()),

	endpoint(http.MethodGet, "/api/v1/admin/tenant", "Tenant", "Get selected tenant", secured("tenant.read")),
	endpoint(http.MethodPatch, "/api/v1/admin/tenant", "Tenant", "Update selected tenant", secured("tenant.update"), body("GenericPatchRequest")),
	endpoint(http.MethodPost, "/api/v1/admin/tenant/users", "Admin Users", "Create tenant user", secured("users.manage"), body("CreateTenantUserRequest"), created()),

	endpoint(http.MethodPost, "/api/v1/admin/academic-years", "Academic Setup", "Create academic year", secured("academic.manage"), body("CreateAcademicYearRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/academic-years", "Academic Setup", "List academic years", secured("academic.manage"), query(paginationParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/academic-years/:id", "Academic Setup", "Get academic year", secured("academic.manage")),
	endpoint(http.MethodPatch, "/api/v1/admin/academic-years/:id", "Academic Setup", "Update academic year", secured("academic.manage"), body("GenericPatchRequest")),
	endpoint(http.MethodDelete, "/api/v1/admin/academic-years/:id", "Academic Setup", "Delete academic year", secured("academic.manage")),

	endpoint(http.MethodPost, "/api/v1/admin/classes", "Academic Setup", "Create class", secured("academic.manage"), body("CreateClassRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/classes", "Academic Setup", "List classes", secured("academic.manage"), query(paginationParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/classes/:id", "Academic Setup", "Get class", secured("academic.manage")),
	endpoint(http.MethodPatch, "/api/v1/admin/classes/:id", "Academic Setup", "Update class", secured("academic.manage"), body("GenericPatchRequest")),
	endpoint(http.MethodDelete, "/api/v1/admin/classes/:id", "Academic Setup", "Delete class", secured("academic.manage")),

	endpoint(http.MethodPost, "/api/v1/admin/sections", "Academic Setup", "Create section", secured("academic.manage"), body("CreateSectionRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/sections", "Academic Setup", "List sections", secured("academic.manage"), query(paginationParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/sections/:id", "Academic Setup", "Get section", secured("academic.manage")),
	endpoint(http.MethodPatch, "/api/v1/admin/sections/:id", "Academic Setup", "Update section", secured("academic.manage"), body("GenericPatchRequest")),
	endpoint(http.MethodDelete, "/api/v1/admin/sections/:id", "Academic Setup", "Delete section", secured("academic.manage")),

	endpoint(http.MethodPost, "/api/v1/admin/students", "Students", "Create student", secured("students.manage"), body("CreateStudentRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/students", "Students", "List students", secured("students.manage"), query(studentFilterParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/students/:id", "Students", "Get student", secured("students.manage")),
	endpoint(http.MethodPatch, "/api/v1/admin/students/:id", "Students", "Update student", secured("students.manage"), body("GenericPatchRequest")),
	endpoint(http.MethodDelete, "/api/v1/admin/students/:id", "Students", "Delete student", secured("students.manage")),
	endpoint(http.MethodPost, "/api/v1/admin/students/:id/guardians", "Students", "Link guardian to student", secured("students.manage"), body("StudentGuardianRequest"), created()),
	endpoint(http.MethodDelete, "/api/v1/admin/students/:id/guardians/:guardian_id", "Students", "Unlink guardian from student", secured("students.manage")),

	endpoint(http.MethodPost, "/api/v1/admin/guardians", "Students", "Create guardian", secured("guardians.manage"), body("CreateGuardianRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/guardians", "Students", "List guardians", secured("guardians.manage"), query(guardianFilterParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/guardians/:id", "Students", "Get guardian", secured("guardians.manage")),
	endpoint(http.MethodPatch, "/api/v1/admin/guardians/:id", "Students", "Update guardian", secured("guardians.manage"), body("GenericPatchRequest")),
	endpoint(http.MethodDelete, "/api/v1/admin/guardians/:id", "Students", "Delete guardian", secured("guardians.manage")),
	endpoint(http.MethodGet, "/api/v1/admin/guardians/:id/students", "Students", "List students linked to a guardian", secured("guardians.manage")),
	endpoint(http.MethodPost, "/api/v1/admin/guardians/:id/user", "Students", "Link a parent user account to a guardian", secured("guardians.manage"), body("LinkGuardianUserRequest"), created()),
	endpoint(http.MethodDelete, "/api/v1/admin/guardians/:id/user", "Students", "Unlink a parent user account from a guardian", secured("guardians.manage")),
	endpoint(http.MethodGet, "/api/v1/admin/parents", "Students", "List parents (guardian + login + linked students)", secured("guardians.manage"), query(guardianFilterParams()...)),

	endpoint(http.MethodGet, "/api/v1/admin/imports", "Students", "List student import history", secured("imports.manage"), query(paginationParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/imports/students/template", "Students", "Download student import CSV template", secured("imports.manage"), download("text/csv")),
	endpoint(http.MethodPost, "/api/v1/admin/imports/students/preview", "Students", "Preview student import", secured("imports.manage"), optionalBody("StudentImportUploadRequest", "application/json", "text/csv", "multipart/form-data")),
	endpoint(http.MethodPost, "/api/v1/admin/imports/students/commit", "Students", "Commit clean student import preview", secured("imports.manage"), body("StudentImportCommitRequest")),

	endpoint(http.MethodGet, "/api/v1/admin/students/:id/ledger", "Billing", "Get student ledger", secured("fees.manage")),

	endpoint(http.MethodPost, "/api/v1/admin/fee-heads", "Billing", "Create fee head", secured("fees.manage"), body("CreateFeeHeadRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/fee-heads", "Billing", "List fee heads", secured("fees.manage"), query(paginationParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/fee-heads/:id", "Billing", "Get fee head", secured("fees.manage")),
	endpoint(http.MethodPatch, "/api/v1/admin/fee-heads/:id", "Billing", "Update fee head", secured("fees.manage"), body("GenericPatchRequest")),
	endpoint(http.MethodDelete, "/api/v1/admin/fee-heads/:id", "Billing", "Delete fee head", secured("fees.manage")),

	endpoint(http.MethodPost, "/api/v1/admin/fee-structures", "Billing", "Create fee structure", secured("fees.manage"), body("CreateFeeStructureRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/fee-structures", "Billing", "List fee structures", secured("fees.manage"), query(paginationParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/fee-structures/:id", "Billing", "Get fee structure", secured("fees.manage")),
	endpoint(http.MethodPatch, "/api/v1/admin/fee-structures/:id", "Billing", "Update fee structure", secured("fees.manage"), body("GenericPatchRequest")),
	endpoint(http.MethodDelete, "/api/v1/admin/fee-structures/:id", "Billing", "Delete fee structure", secured("fees.manage")),

	endpoint(http.MethodPost, "/api/v1/admin/fee-assignments", "Billing", "Create fee assignment", secured("fees.manage"), body("CreateFeeAssignmentRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/fee-assignments", "Billing", "List fee assignments", secured("fees.manage"), query(feeAssignmentFilterParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/fee-assignments/:id", "Billing", "Get fee assignment", secured("fees.manage")),
	endpoint(http.MethodPatch, "/api/v1/admin/fee-assignments/:id", "Billing", "Update fee assignment", secured("fees.manage"), body("UpdateFeeAssignmentRequest")),
	endpoint(http.MethodDelete, "/api/v1/admin/fee-assignments/:id", "Billing", "Delete fee assignment", secured("fees.manage")),
	endpoint(http.MethodPost, "/api/v1/admin/invoices/generate", "Billing", "Generate invoices", secured("fees.manage"), body("GenerateInvoicesRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/invoices", "Billing", "List invoices", secured("fees.manage"), query(paginationParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/invoices/:id", "Billing", "Get invoice", secured("fees.manage")),
	endpoint(http.MethodGet, "/api/v1/parent/children", "Billing", "List linked children for the authenticated parent", secured(), query(parentChildrenFilterParams()...)),
	endpoint(http.MethodGet, "/api/v1/parent/children/:id/dues", "Billing", "Get parent child dues", secured(), query(parentDuesFilterParams()...)),

	endpoint(http.MethodPost, "/api/v1/parent/payments/orders", "Payments", "Create payment order", secured(), body("CreatePaymentOrderRequest"), created(), rateLimited()),
	endpoint(http.MethodPost, "/api/v1/parent/payments/verify", "Payments", "Verify payment", secured(), body("VerifyPaymentRequest"), rateLimited()),
	endpoint(http.MethodPost, "/api/v1/admin/offline-payments", "Payments", "Create offline payment", secured("payments.manage"), body("CreateOfflinePaymentRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/payments", "Payments", "List payments", secured("payments.manage"), query(paymentFilterParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/payments/:id", "Payments", "Get payment", secured("payments.manage")),
	endpoint(http.MethodGet, "/api/v1/admin/receipts", "Payments", "List receipts", secured("payments.manage"), query(receiptFilterParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/receipts/:id", "Payments", "Get receipt", secured("payments.manage")),
	endpoint(http.MethodGet, "/api/v1/admin/receipts/:id/download", "Payments", "Download receipt PDF", secured("payments.manage"), download("application/pdf")),
	endpoint(http.MethodGet, "/api/v1/parent/receipts", "Payments", "List parent receipts", secured(), query(parentReceiptFilterParams()...)),
	endpoint(http.MethodGet, "/api/v1/parent/receipts/:id/download", "Payments", "Download parent receipt PDF", secured(), download("application/pdf")),
	endpoint(http.MethodGet, "/api/v1/admin/payment-events", "Payments", "List payment events", secured("payments.manage"), query(paymentFilterParams()...)),

	endpoint(http.MethodPost, "/api/v1/admin/reminder-templates", "Operations", "Create reminder template", secured("reminders.manage"), body("CreateReminderTemplateRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/reminder-templates", "Operations", "List reminder templates", secured("reminders.manage"), query(paginationParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/reminder-templates/:id", "Operations", "Get reminder template", secured("reminders.manage")),
	endpoint(http.MethodPatch, "/api/v1/admin/reminder-templates/:id", "Operations", "Update reminder template", secured("reminders.manage"), body("GenericPatchRequest")),
	endpoint(http.MethodDelete, "/api/v1/admin/reminder-templates/:id", "Operations", "Delete reminder template", secured("reminders.manage")),

	endpoint(http.MethodPost, "/api/v1/admin/reminder-rules", "Operations", "Create reminder rule", secured("reminders.manage"), body("CreateReminderRuleRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/reminder-rules", "Operations", "List reminder rules", secured("reminders.manage"), query(paginationParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/reminder-rules/:id", "Operations", "Get reminder rule", secured("reminders.manage")),
	endpoint(http.MethodPatch, "/api/v1/admin/reminder-rules/:id", "Operations", "Update reminder rule", secured("reminders.manage"), body("GenericPatchRequest")),
	endpoint(http.MethodDelete, "/api/v1/admin/reminder-rules/:id", "Operations", "Delete reminder rule", secured("reminders.manage")),

	endpoint(http.MethodPost, "/api/v1/admin/reminders/send", "Operations", "Queue or send reminders", secured("reminders.manage"), body("SendReminderRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/reminder-logs", "Operations", "List reminder logs", secured("reminders.manage"), query(paginationParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/dashboard", "Operations", "Get dashboard", secured("reports.view")),

	endpoint(http.MethodGet, "/api/v1/admin/reports/collections", "Operations", "Collections report", secured("reports.view"), query(reportFilterParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/reports/defaulters", "Operations", "Defaulters report", secured("reports.view"), query(reportFilterParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/reports/dues", "Operations", "Dues report", secured("reports.view"), query(reportFilterParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/reports/fee-heads", "Operations", "Fee-head collection report", secured("reports.view"), query(reportFilterParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/reports/payment-methods", "Operations", "Payment-method report", secured("reports.view"), query(reportFilterParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/reports/offline-payments", "Operations", "Offline payment report", secured("reports.view"), query(reportFilterParams()...)),

	endpoint(http.MethodPost, "/api/v1/admin/exports", "Operations", "Create export", secured("exports.manage"), body("CreateExportRequest"), created()),
	endpoint(http.MethodGet, "/api/v1/admin/exports", "Operations", "List exports", secured("exports.manage"), query(paginationParams()...)),
	endpoint(http.MethodGet, "/api/v1/admin/exports/:id", "Operations", "Get export", secured("exports.manage")),
	endpoint(http.MethodGet, "/api/v1/admin/exports/:id/download", "Operations", "Download export CSV", secured("exports.manage"), download("text/csv")),
}

func schemaComponents() map[string]any {
	return map[string]any{
		"APIResponse": object(nil, map[string]any{
			"success":    boolSchema(),
			"request_id": stringSchema(""),
			"data":       map[string]any{"type": "object", "additionalProperties": true},
			"error":      ref("APIError"),
			"meta":       ref("PaginationMeta"),
		}),
		"APIError": object(nil, map[string]any{
			"code":    stringSchema(""),
			"message": stringSchema(""),
			"details": arrayOf(stringSchema("")),
		}),
		"PaginationMeta": object(nil, map[string]any{
			"page":        intSchema(),
			"page_size":   intSchema(),
			"total":       int64Schema(),
			"total_pages": intSchema(),
		}),
		"GenericPatchRequest": map[string]any{"type": "object", "additionalProperties": true},
		"AddressRequest": object(nil, map[string]any{
			"line1":       stringSchema(""),
			"line2":       stringSchema(""),
			"city":        stringSchema(""),
			"state":       stringSchema(""),
			"postal_code": stringSchema(""),
			"country":     stringSchema(""),
		}),
		"LoginRequest": object([]string{"email", "password"}, map[string]any{
			"email":    stringSchema("email"),
			"password": stringSchema("password"),
		}),
		"RegisterRequest": object([]string{"email", "password", "first_name", "last_name"}, map[string]any{
			"email":      stringSchema("email"),
			"password":   stringSchema("password"),
			"first_name": stringSchema(""),
			"last_name":  stringSchema(""),
		}),
		"RefreshRequest": object([]string{"refresh_token"}, map[string]any{
			"refresh_token": stringSchema(""),
		}),
		"SelectTenantRequest": object([]string{"tenant_id"}, map[string]any{
			"tenant_id": uuidSchema(),
		}),
		"ForgotPasswordRequest": object([]string{"email"}, map[string]any{
			"email": stringSchema("email"),
		}),
		"ResetPasswordRequest": object([]string{"token", "new_password"}, map[string]any{
			"token":        stringSchema(""),
			"new_password": stringSchema("password"),
		}),
		"CreateTenantRequest": object([]string{"name", "slug"}, map[string]any{
			"name":          stringSchema(""),
			"slug":          stringSchema(""),
			"legal_name":    stringSchema(""),
			"domain":        stringSchema("hostname"),
			"contact_email": stringSchema("email"),
			"contact_phone": stringSchema(""),
			"status":        enumSchema("active", "inactive", "trial", "suspended"),
			"address":       ref("AddressRequest"),
			"metadata":      map[string]any{"type": "object", "additionalProperties": true},
			"owner_user_id": uuidSchema(),
			"branch":        ref("CreateBranchRequest"),
		}),
		"CreateBranchRequest": object([]string{"name", "code"}, map[string]any{
			"name":          stringSchema(""),
			"code":          stringSchema(""),
			"contact_email": stringSchema("email"),
			"contact_phone": stringSchema(""),
			"status":        enumSchema("active", "inactive"),
			"address":       ref("AddressRequest"),
			"metadata":      map[string]any{"type": "object", "additionalProperties": true},
		}),
		"CreateUserRequest": object([]string{"email", "password", "first_name", "last_name", "roles"}, map[string]any{
			"email":      stringSchema("email"),
			"password":   stringSchema("password"),
			"first_name": stringSchema(""),
			"last_name":  stringSchema(""),
			"roles":      arrayOf(stringSchema("")),
		}),
		"CreateTenantUserRequest": object([]string{"email", "password", "first_name", "last_name", "role"}, map[string]any{
			"email":      stringSchema("email"),
			"password":   stringSchema("password"),
			"first_name": stringSchema(""),
			"last_name":  stringSchema(""),
			"role":       enumSchema("admin", "staff", "parents", "student"),
		}),
		"UpdateUserRequest": object(nil, map[string]any{
			"email":      stringSchema("email"),
			"first_name": stringSchema(""),
			"last_name":  stringSchema(""),
			"status":     enumSchema("active", "inactive"),
			"roles":      arrayOf(stringSchema("")),
		}),
		"CreateAcademicYearRequest": object([]string{"name", "code", "start_date", "end_date"}, map[string]any{
			"name":       stringSchema(""),
			"code":       stringSchema(""),
			"start_date": stringSchema("date"),
			"end_date":   stringSchema("date"),
			"status":     enumSchema("active", "inactive", "closed"),
			"is_active":  boolSchema(),
			"metadata":   map[string]any{"type": "object", "additionalProperties": true},
		}),
		"CreateClassRequest": object([]string{"name", "code"}, map[string]any{
			"name":       stringSchema(""),
			"code":       stringSchema(""),
			"sort_order": intSchema(),
			"status":     enumSchema("active", "inactive"),
			"metadata":   map[string]any{"type": "object", "additionalProperties": true},
		}),
		"CreateSectionRequest": object([]string{"academic_year_id", "class_id", "name", "code"}, map[string]any{
			"academic_year_id": uuidSchema(),
			"class_id":         uuidSchema(),
			"branch_id":        uuidSchema(),
			"name":             stringSchema(""),
			"code":             stringSchema(""),
			"capacity":         intSchema(),
			"status":           enumSchema("active", "inactive"),
			"metadata":         map[string]any{"type": "object", "additionalProperties": true},
		}),
		"CreateGuardianRequest": object([]string{"name"}, map[string]any{
			"name":                 stringSchema(""),
			"relationship":         stringSchema(""),
			"phone":                stringSchema(""),
			"whatsapp_phone":       stringSchema(""),
			"email":                stringSchema("email"),
			"preferred_language":   stringSchema(""),
			"communication_opt_in": boolSchema(),
			"opt_in_whatsapp":      boolSchema(),
			"address": map[string]any{"oneOf": []any{
				stringSchema(""),
				ref("AddressRequest"),
			}},
			"user_id":  uuidSchema(),
			"metadata": map[string]any{"type": "object", "additionalProperties": true},
		}),
		"GuardianResponse": object([]string{"id", "name"}, map[string]any{
			"id":                   uuidSchema(),
			"name":                 stringSchema(""),
			"phone":                stringSchema(""),
			"whatsapp_phone":       stringSchema(""),
			"email":                stringSchema("email"),
			"relationship":         stringSchema(""),
			"user_id":              uuidSchema(),
			"user_status":          enumSchema("active", "inactive", "invited"),
			"preferred_language":   stringSchema(""),
			"communication_opt_in": boolSchema(),
			"opt_in_whatsapp":      boolSchema(),
			"address":              stringSchema(""),
			"created_at":           map[string]any{"type": "string", "format": "date-time"},
			"updated_at":           map[string]any{"type": "string", "format": "date-time"},
		}),
		"LinkGuardianUserRequest": object([]string{"user_id"}, map[string]any{
			"user_id": uuidSchema(),
		}),
		"GuardianStudentResponse": object(nil, map[string]any{
			"student_id":       uuidSchema(),
			"admission_number": stringSchema(""),
			"first_name":       stringSchema(""),
			"last_name":        stringSchema(""),
			"relationship":     stringSchema(""),
			"is_primary":       boolSchema(),
			"class_name":       stringSchema(""),
			"section_name":     stringSchema(""),
			"status":           enumSchema("active", "inactive", "transferred", "graduated"),
		}),
		"ParentSummaryResponse": object(nil, map[string]any{
			"guardian_id":     uuidSchema(),
			"name":            stringSchema(""),
			"relationship":    stringSchema(""),
			"phone":           stringSchema(""),
			"email":           stringSchema("email"),
			"user_id":         uuidSchema(),
			"user_status":     enumSchema("active", "inactive", "invited"),
			"linked_students": arrayOf(ref("GuardianStudentResponse")),
		}),
		"StudentGuardianRequest": object([]string{"guardian_id"}, map[string]any{
			"guardian_id":  uuidSchema(),
			"relationship": stringSchema(""),
			"is_primary":   boolSchema(),
		}),
		"CreateStudentRequest": object([]string{"academic_year_id", "class_id", "section_id", "admission_number", "first_name"}, map[string]any{
			"academic_year_id":      uuidSchema(),
			"class_id":              uuidSchema(),
			"section_id":            uuidSchema(),
			"branch_id":             uuidSchema(),
			"admission_number":      stringSchema(""),
			"first_name":            stringSchema(""),
			"last_name":             stringSchema(""),
			"roll_number":           stringSchema(""),
			"status":                enumSchema("active", "inactive", "transferred", "graduated"),
			"category":              enumSchema("general", "scholarship", "staff_child", "sibling", "custom"),
			"phone":                 stringSchema(""),
			"email":                 stringSchema("email"),
			"address":               ref("AddressRequest"),
			"opening_balance_paise": int64Schema(),
			"metadata":              map[string]any{"type": "object", "additionalProperties": true},
			"guardians":             arrayOf(ref("StudentGuardianRequest")),
		}),
		"StudentImportUploadRequest": object([]string{"csv"}, map[string]any{
			"filename": stringSchema(""),
			"csv":      stringSchema(""),
			"file":     map[string]any{"type": "string", "format": "binary"},
		}),
		"StudentImportCommitRequest": object([]string{"import_id"}, map[string]any{
			"import_id": uuidSchema(),
		}),
		"CreateFeeHeadRequest": object([]string{"name", "code"}, map[string]any{
			"name":         stringSchema(""),
			"code":         stringSchema(""),
			"description":  stringSchema(""),
			"category":     stringSchema(""),
			"status":       enumSchema("active", "inactive"),
			"taxable":      boolSchema(),
			"tax_rate_bps": intSchema(),
			"metadata":     map[string]any{"type": "object", "additionalProperties": true},
		}),
		"CreateFeeStructureItemRequest": object([]string{"fee_head_id", "amount_paise"}, map[string]any{
			"fee_head_id":  uuidSchema(),
			"name":         stringSchema(""),
			"description":  stringSchema(""),
			"amount_paise": int64Schema(),
			"tax_rate_bps": intSchema(),
			"sort_order":   intSchema(),
			"optional":     boolSchema(),
			"metadata":     map[string]any{"type": "object", "additionalProperties": true},
		}),
		"CreateFeeStructureRequest": object([]string{"academic_year_id", "name", "code", "items"}, map[string]any{
			"academic_year_id":             uuidSchema(),
			"name":                         stringSchema(""),
			"code":                         stringSchema(""),
			"description":                  stringSchema(""),
			"billing_cycle":                enumSchema("one_time", "monthly", "quarterly", "term", "yearly", "custom"),
			"status":                       enumSchema("draft", "active", "inactive", "archived"),
			"allow_partial_payment":        boolSchema(),
			"minimum_partial_amount_paise": int64Schema(),
			"due_day":                      intSchema(),
			"metadata":                     map[string]any{"type": "object", "additionalProperties": true},
			"items":                        arrayOf(ref("CreateFeeStructureItemRequest")),
		}),
		"CreateFeeAssignmentRequest": object([]string{"fee_structure_id", "assignment_type"}, map[string]any{
			"fee_structure_id": uuidSchema(),
			"assignment_type":  enumSchema("class", "section", "student"),
			"academic_year_id": uuidSchema(),
			"class_id":         uuidSchema(),
			"section_id":       uuidSchema(),
			"student_id":       uuidSchema(),
			"effective_from":   stringSchema("date"),
			"effective_until":  stringSchema("date"),
			"status":           enumSchema("active", "inactive", "cancelled"),
			"metadata":         map[string]any{"type": "object", "additionalProperties": true},
		}),
		"UpdateFeeAssignmentRequest": object(nil, map[string]any{
			"fee_structure_id": uuidSchema(),
			"assignment_type":  enumSchema("class", "section", "student"),
			"academic_year_id": uuidSchema(),
			"class_id":         uuidSchema(),
			"section_id":       uuidSchema(),
			"student_id":       uuidSchema(),
			"effective_from":   stringSchema("date"),
			"effective_until":  stringSchema("date"),
			"status":           enumSchema("active", "inactive", "cancelled"),
			"metadata":         map[string]any{"type": "object", "additionalProperties": true},
		}),
		"FeeAssignmentResponse": object(nil, map[string]any{
			"id":                 uuidSchema(),
			"fee_structure_id":   uuidSchema(),
			"fee_structure_name": stringSchema(""),
			"assignment_type":    enumSchema("class", "section", "student"),
			"academic_year_id":   uuidSchema(),
			"academic_year_name": stringSchema(""),
			"class_id":           uuidSchema(),
			"class_name":         stringSchema(""),
			"section_id":         uuidSchema(),
			"section_name":       stringSchema(""),
			"student_id":         uuidSchema(),
			"student_name":       stringSchema(""),
			"status":             enumSchema("active", "inactive", "cancelled"),
			"effective_from":     stringSchema("date"),
			"effective_until":    stringSchema("date"),
			"metadata":           map[string]any{"type": "object", "additionalProperties": true},
			"created_at":         stringSchema("date-time"),
			"updated_at":         stringSchema("date-time"),
		}),
		"GenerateInvoicesRequest": object([]string{"assignment_id"}, map[string]any{
			"assignment_id":        uuidSchema(),
			"issue_date":           stringSchema("date"),
			"due_date":             stringSchema("date"),
			"billing_period_start": stringSchema("date"),
			"billing_period_end":   stringSchema("date"),
			"student_ids":          arrayOf(uuidSchema()),
			"metadata":             map[string]any{"type": "object", "additionalProperties": true},
		}),
		"CreatePaymentOrderRequest": object([]string{"student_id", "invoice_ids"}, map[string]any{
			"student_id":      uuidSchema(),
			"invoice_ids":     arrayOf(uuidSchema()),
			"amount_paise":    int64Schema(),
			"idempotency_key": stringSchema(""),
			"metadata":        map[string]any{"type": "object", "additionalProperties": true},
		}),
		"VerifyPaymentRequest": object([]string{"provider_order_id", "provider_payment_id", "signature"}, map[string]any{
			"provider_order_id":   stringSchema(""),
			"provider_payment_id": stringSchema(""),
			"signature":           stringSchema(""),
			"payment_method":      enumSchema("online", "upi", "card", "netbanking", "wallet", "other"),
			"metadata":            map[string]any{"type": "object", "additionalProperties": true},
		}),
		"RazorpayWebhookRequest": map[string]any{"type": "object", "additionalProperties": true},
		"OfflinePaymentAllocationRequest": object([]string{"invoice_id", "amount_paise"}, map[string]any{
			"invoice_id":   uuidSchema(),
			"amount_paise": int64Schema(),
		}),
		"CreateOfflinePaymentRequest": object([]string{"student_id", "payment_method", "allocations"}, map[string]any{
			"student_id":       uuidSchema(),
			"payment_method":   enumSchema("cash", "cheque", "dd", "bank_transfer", "upi", "other"),
			"allocations":      arrayOf(ref("OfflinePaymentAllocationRequest")),
			"received_on":      stringSchema("date"),
			"reference_number": stringSchema(""),
			"bank_name":        stringSchema(""),
			"instrument_date":  stringSchema("date"),
			"clearance_status": enumSchema("pending", "cleared", "bounced", "cancelled"),
			"remarks":          stringSchema(""),
			"metadata":         map[string]any{"type": "object", "additionalProperties": true},
		}),
		"CreateReminderTemplateRequest": object([]string{"name", "code", "body"}, map[string]any{
			"name":     stringSchema(""),
			"code":     stringSchema(""),
			"channel":  enumSchema("email", "sms", "whatsapp", "in_app"),
			"subject":  stringSchema(""),
			"body":     stringSchema(""),
			"tone":     enumSchema("polite", "formal", "urgent"),
			"status":   enumSchema("active", "inactive", "archived"),
			"metadata": map[string]any{"type": "object", "additionalProperties": true},
		}),
		"CreateReminderRuleRequest": object([]string{"name", "code"}, map[string]any{
			"template_id":     uuidSchema(),
			"name":            stringSchema(""),
			"code":            stringSchema(""),
			"channel":         enumSchema("email", "sms", "whatsapp", "in_app"),
			"trigger_type":    enumSchema("before_due", "on_due", "after_due", "manual"),
			"offset_days":     intSchema(),
			"target_statuses": arrayOf(stringSchema("")),
			"status":          enumSchema("active", "inactive", "archived"),
			"max_attempts":    intSchema(),
			"metadata":        map[string]any{"type": "object", "additionalProperties": true},
		}),
		"SendReminderRequest": object(nil, map[string]any{
			"rule_id":          uuidSchema(),
			"template_id":      uuidSchema(),
			"channel":          enumSchema("email", "sms", "whatsapp", "in_app"),
			"invoice_ids":      arrayOf(uuidSchema()),
			"student_id":       uuidSchema(),
			"class_id":         uuidSchema(),
			"section_id":       uuidSchema(),
			"academic_year_id": uuidSchema(),
			"due_on_or_before": stringSchema("date"),
			"subject":          stringSchema(""),
			"message":          stringSchema(""),
			"process_now":      boolSchema(),
			"metadata":         map[string]any{"type": "object", "additionalProperties": true},
		}),
		"CreateExportRequest": object([]string{"export_type"}, map[string]any{
			"export_type":    enumSchema("collections", "defaulters", "dues", "payment_methods", "fee_heads", "offline_payments", "receipt_register"),
			"format":         enumSchema("csv"),
			"from":           stringSchema("date"),
			"to":             stringSchema("date"),
			"as_of":          stringSchema("date"),
			"student_id":     uuidSchema(),
			"class_id":       uuidSchema(),
			"section_id":     uuidSchema(),
			"payment_method": stringSchema(""),
			"provider":       stringSchema(""),
			"metadata":       map[string]any{"type": "object", "additionalProperties": true},
		}),
	}
}

func object(required []string, properties map[string]any) map[string]any {
	out := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func stringSchema(format string) map[string]any {
	out := map[string]any{"type": "string"}
	if format != "" {
		out["format"] = format
	}
	return out
}

func uuidSchema() map[string]any {
	return stringSchema("uuid")
}

func boolSchema() map[string]any {
	return map[string]any{"type": "boolean"}
}

func intSchema() map[string]any {
	return map[string]any{"type": "integer", "format": "int32"}
}

func int64Schema() map[string]any {
	return map[string]any{"type": "integer", "format": "int64"}
}

func enumSchema(values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}

func arrayOf(item map[string]any) map[string]any {
	return map[string]any{
		"type":  "array",
		"items": item,
	}
}
