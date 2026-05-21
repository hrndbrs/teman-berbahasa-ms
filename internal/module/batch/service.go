package batch

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound             = errors.New("batch not found")
	ErrBatchCodeConflict    = errors.New("batch code already in use for this course")
	ErrInvalidInstructor    = errors.New("instructor must be an active teacher")
	ErrInvalidTransition    = errors.New("invalid status transition")
	ErrHasActiveEnrollments = errors.New("cannot delete batch with active enrollments")
)

var validTransitions = map[string]string{
	"upcoming": "ongoing",
	"ongoing":  "completed",
}

func ValidateBatchTransition(current, next string) error {
	expected, ok := validTransitions[current]
	if !ok || expected != next {
		return ErrInvalidTransition
	}
	return nil
}

type Batch struct {
	ID               uuid.UUID
	CourseID         uuid.UUID
	InstructorUserID uuid.UUID
	CreatedByUserID  uuid.UUID
	BatchName        string
	BatchCode        string
	AcademicYear     *string
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type BatchWithDetails struct {
	Batch
	CourseName          string
	CourseCode          string
	InstructorFirstName string
	InstructorLastName  string
	EnrolledCount       int64
}

type StatusUpdateResult struct {
	ID        uuid.UUID
	Status    string
	UpdatedAt time.Time
}

type ListParams struct {
	Page     int
	PerPage  int
	Status   *string
	CourseID *uuid.UUID
	Search   *string
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ListResponse struct {
	Data       []BatchWithDetails
	Pagination PaginationMeta
}

type CreateBatchRequest struct {
	CourseID         uuid.UUID
	InstructorUserID uuid.UUID
	CreatedByUserID  uuid.UUID
	BatchName        string
	BatchCode        string
	AcademicYear     *string
}

type UpdateBatchRequest struct {
	InstructorUserID *uuid.UUID
	BatchName        *string
	BatchCode        *string
	AcademicYear     *string
}

type BatchRepository interface {
	List(ctx context.Context, params ListParams) ([]BatchWithDetails, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (BatchWithDetails, error)
	Create(ctx context.Context, id uuid.UUID, req CreateBatchRequest) (BatchWithDetails, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateBatchRequest) (BatchWithDetails, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (StatusUpdateResult, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountActiveEnrollments(ctx context.Context, batchID uuid.UUID) (int64, error)
	ExistsActiveTeacher(ctx context.Context, instructorID uuid.UUID) (bool, error)
}

type BatchService struct {
	repo BatchRepository
}

func NewService(repo BatchRepository) *BatchService {
	return &BatchService{repo: repo}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func (s *BatchService) List(ctx context.Context, params ListParams) (*ListResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 || params.PerPage > 100 {
		params.PerPage = 20
	}
	batches, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	totalPages := int(total) / params.PerPage
	if int(total)%params.PerPage != 0 {
		totalPages++
	}
	return &ListResponse{
		Data: batches,
		Pagination: PaginationMeta{
			Page:       params.Page,
			PerPage:    params.PerPage,
			Total:      int(total),
			TotalPages: totalPages,
		},
	}, nil
}

func (s *BatchService) GetByID(ctx context.Context, rawID string) (*BatchWithDetails, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrNotFound
	}
	b, err := s.repo.GetByID(ctx, id)
	if isNoRows(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *BatchService) Create(ctx context.Context, req CreateBatchRequest) (*BatchWithDetails, error) {
	req.BatchName = strings.TrimSpace(req.BatchName)
	req.BatchCode = strings.TrimSpace(req.BatchCode)

	ok, err := s.repo.ExistsActiveTeacher(ctx, req.InstructorUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidInstructor
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	b, err := s.repo.Create(ctx, id, req)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrBatchCodeConflict
		}
		return nil, err
	}
	return &b, nil
}

func (s *BatchService) Update(ctx context.Context, rawID string, req UpdateBatchRequest) (*BatchWithDetails, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrNotFound
	}
	_, err = s.repo.GetByID(ctx, id)
	if isNoRows(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if req.InstructorUserID != nil {
		ok, err := s.repo.ExistsActiveTeacher(ctx, *req.InstructorUserID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrInvalidInstructor
		}
	}
	if req.BatchName != nil {
		trimmed := strings.TrimSpace(*req.BatchName)
		req.BatchName = &trimmed
	}
	if req.BatchCode != nil {
		trimmed := strings.TrimSpace(*req.BatchCode)
		req.BatchCode = &trimmed
	}
	b, err := s.repo.Update(ctx, id, req)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrBatchCodeConflict
		}
		return nil, err
	}
	return &b, nil
}

func (s *BatchService) TransitionStatus(ctx context.Context, rawID string, newStatus string) (*StatusUpdateResult, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrNotFound
	}
	current, err := s.repo.GetByID(ctx, id)
	if isNoRows(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := ValidateBatchTransition(current.Status, newStatus); err != nil {
		return nil, err
	}
	result, err := s.repo.UpdateStatus(ctx, id, newStatus)
	if isNoRows(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *BatchService) Delete(ctx context.Context, rawID string) error {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return ErrNotFound
	}
	_, err = s.repo.GetByID(ctx, id)
	if isNoRows(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	count, err := s.repo.CountActiveEnrollments(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrHasActiveEnrollments
	}
	return s.repo.Delete(ctx, id)
}
