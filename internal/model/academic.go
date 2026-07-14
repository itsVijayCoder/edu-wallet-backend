package model

import (
	"time"

	"github.com/google/uuid"
)

type AcademicYear struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	Name      string         `json:"name"`
	Code      string         `json:"code"`
	StartDate time.Time      `json:"start_date"`
	EndDate   time.Time      `json:"end_date"`
	Status    string         `json:"status"`
	IsActive  bool           `json:"is_active"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt *time.Time     `json:"deleted_at,omitempty"`
}

type Class struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	Name      string         `json:"name"`
	Code      string         `json:"code"`
	SortOrder int            `json:"sort_order"`
	Status    string         `json:"status"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt *time.Time     `json:"deleted_at,omitempty"`
}

type Section struct {
	ID             uuid.UUID      `json:"id"`
	TenantID       uuid.UUID      `json:"tenant_id"`
	AcademicYearID uuid.UUID      `json:"academic_year_id"`
	ClassID        uuid.UUID      `json:"class_id"`
	BranchID       *uuid.UUID     `json:"branch_id,omitempty"`
	Name           string         `json:"name"`
	Code           string         `json:"code"`
	Capacity       *int           `json:"capacity,omitempty"`
	Status         string         `json:"status"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      *time.Time     `json:"deleted_at,omitempty"`

	AcademicYear *AcademicYear `json:"academic_year,omitempty"`
	Class        *Class        `json:"class,omitempty"`
}

type Student struct {
	ID                  uuid.UUID      `json:"id"`
	TenantID            uuid.UUID      `json:"tenant_id"`
	AcademicYearID      uuid.UUID      `json:"academic_year_id"`
	ClassID             uuid.UUID      `json:"class_id"`
	SectionID           uuid.UUID      `json:"section_id"`
	BranchID            *uuid.UUID     `json:"branch_id,omitempty"`
	AdmissionNumber     string         `json:"admission_number"`
	FirstName           string         `json:"first_name"`
	LastName            string         `json:"last_name"`
	RollNumber          *string        `json:"roll_number,omitempty"`
	Status              string         `json:"status"`
	Category            string         `json:"category"`
	Phone               *string        `json:"phone,omitempty"`
	Email               *string        `json:"email,omitempty"`
	Address             Address        `json:"address"`
	OpeningBalancePaise int64          `json:"opening_balance_paise"`
	Currency            string         `json:"currency"`
	Metadata            map[string]any `json:"metadata,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           *time.Time     `json:"deleted_at,omitempty"`

	AcademicYear *AcademicYear     `json:"academic_year,omitempty"`
	Class        *Class            `json:"class,omitempty"`
	Section      *Section          `json:"section,omitempty"`
	Guardians    []StudentGuardian `json:"guardians,omitempty"`
}

type Guardian struct {
	ID                 uuid.UUID      `json:"id"`
	TenantID           uuid.UUID      `json:"tenant_id"`
	Name               string         `json:"name"`
	Relationship       string         `json:"relationship"`
	Phone              *string        `json:"phone,omitempty"`
	WhatsAppPhone      *string        `json:"whatsapp_phone,omitempty"`
	Email              *string        `json:"email,omitempty"`
	PreferredLanguage  string         `json:"preferred_language"`
	CommunicationOptIn bool           `json:"communication_opt_in"`
	OptInWhatsApp      bool           `json:"opt_in_whatsapp"`
	Address            Address        `json:"address"`
	UserID             *uuid.UUID     `json:"user_id,omitempty"`
	UserStatus         *string        `json:"user_status,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          *time.Time     `json:"deleted_at,omitempty"`
}

type StudentGuardian struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	StudentID    uuid.UUID `json:"student_id"`
	GuardianID   uuid.UUID `json:"guardian_id"`
	Relationship string    `json:"relationship"`
	IsPrimary    bool      `json:"is_primary"`
	CreatedAt    time.Time `json:"created_at"`
	Guardian     *Guardian `json:"guardian,omitempty"`
}

// GuardianStudent is the reverse projection of StudentGuardian joined with the
// student, class, and section tables so a guardian can be resolved back to the
// students they are responsible for without a second round trip.
type GuardianStudent struct {
	GuardianID      uuid.UUID `json:"guardian_id"`
	StudentID       uuid.UUID `json:"student_id"`
	AdmissionNumber string    `json:"admission_number"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Relationship    string    `json:"relationship"`
	IsPrimary       bool      `json:"is_primary"`
	ClassName       string    `json:"class_name"`
	SectionName     string    `json:"section_name"`
	Status          string    `json:"status"`
}

type Import struct {
	ID             uuid.UUID      `json:"id"`
	TenantID       uuid.UUID      `json:"tenant_id"`
	ImportType     string         `json:"import_type"`
	Status         string         `json:"status"`
	SourceFilename string         `json:"source_filename"`
	TotalRows      int            `json:"total_rows"`
	ValidRows      int            `json:"valid_rows"`
	InvalidRows    int            `json:"invalid_rows"`
	CommittedRows  int            `json:"committed_rows"`
	CreatedBy      *uuid.UUID     `json:"created_by,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CommittedAt    *time.Time     `json:"committed_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type ImportError struct {
	ID        uuid.UUID      `json:"id"`
	ImportID  uuid.UUID      `json:"import_id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	RowNumber int            `json:"row_number"`
	Field     string         `json:"field"`
	Message   string         `json:"message"`
	RawData   map[string]any `json:"raw_data,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type AcademicYearFilter struct {
	Status string
	Search string
}

type ClassFilter struct {
	Status string
	Search string
}

type SectionFilter struct {
	AcademicYearID *uuid.UUID
	ClassID        *uuid.UUID
	Status         string
	Search         string
}

type StudentFilter struct {
	AcademicYearID *uuid.UUID
	ClassID        *uuid.UUID
	SectionID      *uuid.UUID
	Status         string
	Search         string
}

type GuardianFilter struct {
	Search       string
	OnlyLinked   bool
	OnlyUnlinked bool
}

type ImportFilter struct {
	Status     string
	ImportType string
}
