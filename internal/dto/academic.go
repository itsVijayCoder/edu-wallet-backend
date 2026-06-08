package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateAcademicYearRequest struct {
	Name      string         `json:"name" binding:"required,min=2,max=120"`
	Code      string         `json:"code" binding:"required,min=2,max=40"`
	StartDate string         `json:"start_date" binding:"required"`
	EndDate   string         `json:"end_date" binding:"required"`
	Status    string         `json:"status" binding:"omitempty,oneof=active inactive closed"`
	IsActive  bool           `json:"is_active"`
	Metadata  map[string]any `json:"metadata"`
}

type UpdateAcademicYearRequest struct {
	Name      *string        `json:"name" binding:"omitempty,min=2,max=120"`
	Code      *string        `json:"code" binding:"omitempty,min=2,max=40"`
	StartDate *string        `json:"start_date"`
	EndDate   *string        `json:"end_date"`
	Status    *string        `json:"status" binding:"omitempty,oneof=active inactive closed"`
	IsActive  *bool          `json:"is_active"`
	Metadata  map[string]any `json:"metadata"`
}

type AcademicYearResponse struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	Name      string         `json:"name"`
	Code      string         `json:"code"`
	StartDate string         `json:"start_date"`
	EndDate   string         `json:"end_date"`
	Status    string         `json:"status"`
	IsActive  bool           `json:"is_active"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type CreateClassRequest struct {
	Name      string         `json:"name" binding:"required,min=1,max=120"`
	Code      string         `json:"code" binding:"required,min=1,max=40"`
	SortOrder int            `json:"sort_order"`
	Status    string         `json:"status" binding:"omitempty,oneof=active inactive"`
	Metadata  map[string]any `json:"metadata"`
}

type UpdateClassRequest struct {
	Name      *string        `json:"name" binding:"omitempty,min=1,max=120"`
	Code      *string        `json:"code" binding:"omitempty,min=1,max=40"`
	SortOrder *int           `json:"sort_order"`
	Status    *string        `json:"status" binding:"omitempty,oneof=active inactive"`
	Metadata  map[string]any `json:"metadata"`
}

type ClassResponse struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	Name      string         `json:"name"`
	Code      string         `json:"code"`
	SortOrder int            `json:"sort_order"`
	Status    string         `json:"status"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type CreateSectionRequest struct {
	AcademicYearID uuid.UUID      `json:"academic_year_id" binding:"required"`
	ClassID        uuid.UUID      `json:"class_id" binding:"required"`
	BranchID       *uuid.UUID     `json:"branch_id"`
	Name           string         `json:"name" binding:"required,min=1,max=120"`
	Code           string         `json:"code" binding:"required,min=1,max=40"`
	Capacity       *int           `json:"capacity"`
	Status         string         `json:"status" binding:"omitempty,oneof=active inactive"`
	Metadata       map[string]any `json:"metadata"`
}

type UpdateSectionRequest struct {
	AcademicYearID *uuid.UUID     `json:"academic_year_id"`
	ClassID        *uuid.UUID     `json:"class_id"`
	BranchID       *uuid.UUID     `json:"branch_id"`
	Name           *string        `json:"name" binding:"omitempty,min=1,max=120"`
	Code           *string        `json:"code" binding:"omitempty,min=1,max=40"`
	Capacity       *int           `json:"capacity"`
	Status         *string        `json:"status" binding:"omitempty,oneof=active inactive"`
	Metadata       map[string]any `json:"metadata"`
}

type SectionResponse struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	AcademicYearID uuid.UUID       `json:"academic_year_id"`
	ClassID        uuid.UUID       `json:"class_id"`
	BranchID       *uuid.UUID      `json:"branch_id,omitempty"`
	Name           string          `json:"name"`
	Code           string          `json:"code"`
	Capacity       *int            `json:"capacity,omitempty"`
	Status         string          `json:"status"`
	Metadata       map[string]any  `json:"metadata,omitempty"`
	AcademicYear   *LookupResponse `json:"academic_year,omitempty"`
	Class          *LookupResponse `json:"class,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type LookupResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Code string    `json:"code"`
}

type CreateGuardianRequest struct {
	Name               string         `json:"name" binding:"required,min=2,max=160"`
	Relationship       string         `json:"relationship" binding:"omitempty,max=60"`
	Phone              *string        `json:"phone"`
	WhatsAppPhone      *string        `json:"whatsapp_phone"`
	Email              *string        `json:"email" binding:"omitempty,email"`
	PreferredLanguage  string         `json:"preferred_language" binding:"omitempty,max=40"`
	CommunicationOptIn *bool          `json:"communication_opt_in"`
	Address            AddressRequest `json:"address"`
	Metadata           map[string]any `json:"metadata"`
}

type UpdateGuardianRequest struct {
	Name               *string         `json:"name" binding:"omitempty,min=2,max=160"`
	Relationship       *string         `json:"relationship" binding:"omitempty,max=60"`
	Phone              *string         `json:"phone"`
	WhatsAppPhone      *string         `json:"whatsapp_phone"`
	Email              *string         `json:"email" binding:"omitempty,email"`
	PreferredLanguage  *string         `json:"preferred_language" binding:"omitempty,max=40"`
	CommunicationOptIn *bool           `json:"communication_opt_in"`
	Address            *AddressRequest `json:"address"`
	Metadata           map[string]any  `json:"metadata"`
}

type GuardianResponse struct {
	ID                 uuid.UUID       `json:"id"`
	TenantID           uuid.UUID       `json:"tenant_id"`
	Name               string          `json:"name"`
	Relationship       string          `json:"relationship"`
	Phone              *string         `json:"phone,omitempty"`
	WhatsAppPhone      *string         `json:"whatsapp_phone,omitempty"`
	Email              *string         `json:"email,omitempty"`
	PreferredLanguage  string          `json:"preferred_language"`
	CommunicationOptIn bool            `json:"communication_opt_in"`
	Address            AddressResponse `json:"address"`
	Metadata           map[string]any  `json:"metadata,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type StudentGuardianRequest struct {
	GuardianID   uuid.UUID `json:"guardian_id" binding:"required"`
	Relationship string    `json:"relationship" binding:"omitempty,max=60"`
	IsPrimary    bool      `json:"is_primary"`
}

type CreateStudentRequest struct {
	AcademicYearID      uuid.UUID                `json:"academic_year_id" binding:"required"`
	ClassID             uuid.UUID                `json:"class_id" binding:"required"`
	SectionID           uuid.UUID                `json:"section_id" binding:"required"`
	BranchID            *uuid.UUID               `json:"branch_id"`
	AdmissionNumber     string                   `json:"admission_number" binding:"required,min=1,max=80"`
	FirstName           string                   `json:"first_name" binding:"required,min=1,max=100"`
	LastName            string                   `json:"last_name" binding:"omitempty,max=100"`
	RollNumber          *string                  `json:"roll_number" binding:"omitempty,max=40"`
	Status              string                   `json:"status" binding:"omitempty,oneof=active inactive transferred graduated"`
	Category            string                   `json:"category" binding:"omitempty,oneof=general scholarship staff_child sibling custom"`
	Phone               *string                  `json:"phone"`
	Email               *string                  `json:"email" binding:"omitempty,email"`
	Address             AddressRequest           `json:"address"`
	OpeningBalancePaise int64                    `json:"opening_balance_paise"`
	Metadata            map[string]any           `json:"metadata"`
	Guardians           []StudentGuardianRequest `json:"guardians"`
}

type UpdateStudentRequest struct {
	AcademicYearID      *uuid.UUID               `json:"academic_year_id"`
	ClassID             *uuid.UUID               `json:"class_id"`
	SectionID           *uuid.UUID               `json:"section_id"`
	BranchID            *uuid.UUID               `json:"branch_id"`
	AdmissionNumber     *string                  `json:"admission_number" binding:"omitempty,min=1,max=80"`
	FirstName           *string                  `json:"first_name" binding:"omitempty,min=1,max=100"`
	LastName            *string                  `json:"last_name" binding:"omitempty,max=100"`
	RollNumber          *string                  `json:"roll_number" binding:"omitempty,max=40"`
	Status              *string                  `json:"status" binding:"omitempty,oneof=active inactive transferred graduated"`
	Category            *string                  `json:"category" binding:"omitempty,oneof=general scholarship staff_child sibling custom"`
	Phone               *string                  `json:"phone"`
	Email               *string                  `json:"email" binding:"omitempty,email"`
	Address             *AddressRequest          `json:"address"`
	OpeningBalancePaise *int64                   `json:"opening_balance_paise"`
	Metadata            map[string]any           `json:"metadata"`
	Guardians           []StudentGuardianRequest `json:"guardians"`
}

type StudentResponse struct {
	ID                  uuid.UUID                 `json:"id"`
	TenantID            uuid.UUID                 `json:"tenant_id"`
	AcademicYearID      uuid.UUID                 `json:"academic_year_id"`
	ClassID             uuid.UUID                 `json:"class_id"`
	SectionID           uuid.UUID                 `json:"section_id"`
	BranchID            *uuid.UUID                `json:"branch_id,omitempty"`
	AdmissionNumber     string                    `json:"admission_number"`
	FirstName           string                    `json:"first_name"`
	LastName            string                    `json:"last_name"`
	RollNumber          *string                   `json:"roll_number,omitempty"`
	Status              string                    `json:"status"`
	Category            string                    `json:"category"`
	Phone               *string                   `json:"phone,omitempty"`
	Email               *string                   `json:"email,omitempty"`
	Address             AddressResponse           `json:"address"`
	OpeningBalancePaise int64                     `json:"opening_balance_paise"`
	Currency            string                    `json:"currency"`
	Metadata            map[string]any            `json:"metadata,omitempty"`
	AcademicYear        *LookupResponse           `json:"academic_year,omitempty"`
	Class               *LookupResponse           `json:"class,omitempty"`
	Section             *LookupResponse           `json:"section,omitempty"`
	Guardians           []StudentGuardianResponse `json:"guardians,omitempty"`
	CreatedAt           time.Time                 `json:"created_at"`
	UpdatedAt           time.Time                 `json:"updated_at"`
}

type StudentGuardianResponse struct {
	GuardianID   uuid.UUID         `json:"guardian_id"`
	Relationship string            `json:"relationship"`
	IsPrimary    bool              `json:"is_primary"`
	Guardian     *GuardianResponse `json:"guardian,omitempty"`
}

type StudentImportUploadRequest struct {
	Filename string `json:"filename"`
	CSV      string `json:"csv" binding:"required"`
}

type StudentImportCommitRequest struct {
	ImportID uuid.UUID `json:"import_id" binding:"required"`
}

type ImportErrorResponse struct {
	RowNumber int               `json:"row_number"`
	Field     string            `json:"field"`
	Message   string            `json:"message"`
	RawData   map[string]string `json:"raw_data,omitempty"`
}

type StudentImportPreviewResponse struct {
	ImportID    uuid.UUID             `json:"import_id"`
	TotalRows   int                   `json:"total_rows"`
	ValidRows   int                   `json:"valid_rows"`
	InvalidRows int                   `json:"invalid_rows"`
	Errors      []ImportErrorResponse `json:"errors,omitempty"`
	Template    []string              `json:"template"`
}

type StudentImportCommitResponse struct {
	ImportID      uuid.UUID `json:"import_id"`
	CommittedRows int       `json:"committed_rows"`
}

type ImportResponse struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	ImportType     string     `json:"import_type"`
	Status         string     `json:"status"`
	SourceFilename string     `json:"source_filename"`
	TotalRows      int        `json:"total_rows"`
	ValidRows      int        `json:"valid_rows"`
	InvalidRows    int        `json:"invalid_rows"`
	CommittedRows  int        `json:"committed_rows"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	CommittedAt    *time.Time `json:"committed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
