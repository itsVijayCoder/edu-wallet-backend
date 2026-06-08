package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/itsVijayCoder/edu-wallet-backend/internal/apperror"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/dto"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/model"
	"github.com/itsVijayCoder/edu-wallet-backend/internal/service"
)

const maxImportUploadBytes = 5 * 1024 * 1024

type AcademicHandler struct {
	academicSvc service.AcademicService
}

func NewAcademicHandler(academicSvc service.AcademicService) *AcademicHandler {
	return &AcademicHandler{academicSvc: academicSvc}
}

func (h *AcademicHandler) CreateAcademicYear(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.CreateAcademicYearRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.academicSvc.CreateAcademicYear(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *AcademicHandler) ListAcademicYears(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	filter := model.AcademicYearFilter{
		Status: c.Query("status"),
		Search: c.Query("search"),
	}
	result, err := h.academicSvc.ListAcademicYears(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *AcademicHandler) GetAcademicYear(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.academicSvc.GetAcademicYear(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *AcademicHandler) UpdateAcademicYear(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	id, ok := paramUUID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateAcademicYearRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.academicSvc.UpdateAcademicYear(c.Request.Context(), actorID, tenantID, id, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *AcademicHandler) DeleteAcademicYear(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	if err := h.academicSvc.DeleteAcademicYear(c.Request.Context(), actorID, tenantID, id); err != nil {
		HandleError(c, err)
		return
	}
	RespondMessage(c, "academic year deleted")
}

func (h *AcademicHandler) CreateClass(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.CreateClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.academicSvc.CreateClass(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *AcademicHandler) ListClasses(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	filter := model.ClassFilter{Status: c.Query("status"), Search: c.Query("search")}
	result, err := h.academicSvc.ListClasses(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *AcademicHandler) GetClass(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.academicSvc.GetClass(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *AcademicHandler) UpdateClass(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	id, ok := paramUUID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.academicSvc.UpdateClass(c.Request.Context(), actorID, tenantID, id, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *AcademicHandler) DeleteClass(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	if err := h.academicSvc.DeleteClass(c.Request.Context(), actorID, tenantID, id); err != nil {
		HandleError(c, err)
		return
	}
	RespondMessage(c, "class deleted")
}

func (h *AcademicHandler) CreateSection(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.CreateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.academicSvc.CreateSection(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *AcademicHandler) ListSections(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	academicYearID, ok := queryUUID(c, "academic_year_id")
	if !ok {
		return
	}
	classID, ok := queryUUID(c, "class_id")
	if !ok {
		return
	}
	filter := model.SectionFilter{
		AcademicYearID: academicYearID,
		ClassID:        classID,
		Status:         c.Query("status"),
		Search:         c.Query("search"),
	}
	result, err := h.academicSvc.ListSections(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *AcademicHandler) GetSection(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.academicSvc.GetSection(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *AcademicHandler) UpdateSection(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	id, ok := paramUUID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.academicSvc.UpdateSection(c.Request.Context(), actorID, tenantID, id, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *AcademicHandler) DeleteSection(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	if err := h.academicSvc.DeleteSection(c.Request.Context(), actorID, tenantID, id); err != nil {
		HandleError(c, err)
		return
	}
	RespondMessage(c, "section deleted")
}

func (h *AcademicHandler) CreateGuardian(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.CreateGuardianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.academicSvc.CreateGuardian(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *AcademicHandler) ListGuardians(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	result, err := h.academicSvc.ListGuardians(c.Request.Context(), tenantID, model.GuardianFilter{Search: c.Query("search")}, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *AcademicHandler) GetGuardian(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.academicSvc.GetGuardian(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *AcademicHandler) UpdateGuardian(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	id, ok := paramUUID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateGuardianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.academicSvc.UpdateGuardian(c.Request.Context(), actorID, tenantID, id, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *AcademicHandler) DeleteGuardian(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	if err := h.academicSvc.DeleteGuardian(c.Request.Context(), actorID, tenantID, id); err != nil {
		HandleError(c, err)
		return
	}
	RespondMessage(c, "guardian deleted")
}

func (h *AcademicHandler) CreateStudent(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.academicSvc.CreateStudent(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *AcademicHandler) ListStudents(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	academicYearID, ok := queryUUID(c, "academic_year_id")
	if !ok {
		return
	}
	classID, ok := queryUUID(c, "class_id")
	if !ok {
		return
	}
	sectionID, ok := queryUUID(c, "section_id")
	if !ok {
		return
	}
	filter := model.StudentFilter{
		AcademicYearID: academicYearID,
		ClassID:        classID,
		SectionID:      sectionID,
		Status:         c.Query("status"),
		Search:         c.Query("search"),
	}
	result, err := h.academicSvc.ListStudents(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func (h *AcademicHandler) GetStudent(c *gin.Context) {
	tenantID, id, ok := currentTenantAndParamID(c, "id")
	if !ok {
		return
	}
	resp, err := h.academicSvc.GetStudent(c.Request.Context(), tenantID, id)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *AcademicHandler) UpdateStudent(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	id, ok := paramUUID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.academicSvc.UpdateStudent(c.Request.Context(), actorID, tenantID, id, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *AcademicHandler) DeleteStudent(c *gin.Context) {
	actorID, tenantID, id, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	if err := h.academicSvc.DeleteStudent(c.Request.Context(), actorID, tenantID, id); err != nil {
		HandleError(c, err)
		return
	}
	RespondMessage(c, "student deleted")
}

func (h *AcademicHandler) LinkStudentGuardian(c *gin.Context) {
	actorID, tenantID, studentID, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	var req dto.StudentGuardianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.academicSvc.LinkStudentGuardian(c.Request.Context(), actorID, tenantID, studentID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *AcademicHandler) UnlinkStudentGuardian(c *gin.Context) {
	actorID, tenantID, studentID, ok := currentActorTenantAndParamID(c, "id")
	if !ok {
		return
	}
	guardianID, ok := paramUUID(c, "guardian_id")
	if !ok {
		return
	}
	resp, err := h.academicSvc.UnlinkStudentGuardian(c.Request.Context(), actorID, tenantID, studentID, guardianID)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *AcademicHandler) StudentImportTemplate(c *gin.Context) {
	c.Header("Content-Disposition", `attachment; filename="student_import_template.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(h.academicSvc.StudentImportTemplate()))
}

func (h *AcademicHandler) PreviewStudentImport(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	filename, payload, err := readCSVUpload(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	resp, err := h.academicSvc.PreviewStudentImport(c.Request.Context(), actorID, tenantID, filename, payload)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondCreated(c, resp)
}

func (h *AcademicHandler) CommitStudentImport(c *gin.Context) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return
	}
	var req dto.StudentImportCommitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, extractValidationErrors(err))
		return
	}
	resp, err := h.academicSvc.CommitStudentImport(c.Request.Context(), actorID, tenantID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondOK(c, resp)
}

func (h *AcademicHandler) ListImports(c *gin.Context) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return
	}
	filter := model.ImportFilter{
		Status:     c.Query("status"),
		ImportType: c.Query("import_type"),
	}
	result, err := h.academicSvc.ListImports(c.Request.Context(), tenantID, filter, dto.ExtractPagination(c))
	if err != nil {
		HandleError(c, err)
		return
	}
	RespondPaginated(c, result.Data, result.Page, result.PageSize, result.Total, result.TotalPages)
}

func currentActorAndTenant(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	actorID, err := currentUserID(c)
	if err != nil {
		HandleError(c, err)
		return uuid.Nil, uuid.Nil, false
	}
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return uuid.Nil, uuid.Nil, false
	}
	return actorID, tenantID, true
}

func currentTenantAndParamID(c *gin.Context, name string) (uuid.UUID, uuid.UUID, bool) {
	tenantID, err := currentTenantID(c)
	if err != nil {
		HandleError(c, err)
		return uuid.Nil, uuid.Nil, false
	}
	id, ok := paramUUID(c, name)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, id, true
}

func currentActorTenantAndParamID(c *gin.Context, name string) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	actorID, tenantID, ok := currentActorAndTenant(c)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	id, ok := paramUUID(c, name)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	return actorID, tenantID, id, true
}

func paramUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		RespondValidationError(c, []string{"invalid UUID format for " + name})
		return uuid.Nil, false
	}
	return id, true
}

func queryUUID(c *gin.Context, name string) (*uuid.UUID, bool) {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return nil, true
	}
	id, err := uuid.Parse(value)
	if err != nil {
		RespondValidationError(c, []string{"invalid UUID format for " + name})
		return nil, false
	}
	return &id, true
}

func readCSVUpload(c *gin.Context) (string, []byte, error) {
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			return "", nil, apperror.New("IMPORT_FILE_REQUIRED", "CSV file is required", http.StatusBadRequest)
		}
		defer file.Close()
		payload, err := readLimited(file)
		if err != nil {
			return "", nil, err
		}
		return header.Filename, payload, nil
	}

	if strings.Contains(contentType, "application/json") {
		var req dto.StudentImportUploadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return "", nil, apperror.ErrValidationFailed
		}
		return req.Filename, []byte(req.CSV), nil
	}

	payload, err := readLimited(c.Request.Body)
	if err != nil {
		return "", nil, err
	}
	return c.Query("filename"), payload, nil
}

func readLimited(r io.Reader) ([]byte, error) {
	limited := io.LimitReader(r, maxImportUploadBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(payload) > maxImportUploadBytes {
		return nil, apperror.New("IMPORT_TOO_LARGE", "CSV import file is too large", http.StatusRequestEntityTooLarge)
	}
	return payload, nil
}
