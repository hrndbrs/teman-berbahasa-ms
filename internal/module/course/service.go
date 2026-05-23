package course

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/patch"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/pagination"
)

var (
	ErrNotFound           = errors.New("course not found")
	ErrCourseCodeConflict = errors.New("course code already in use")
	ErrHasOngoingBatches  = errors.New("cannot archive a course with ongoing batches")
)

type Course struct {
	ID            uuid.UUID
	CourseName    string
	CourseCode    string
	Description   *string
	Subject       *string
	Level         *string
	SessionCount *int32
	Price         *string
	MaxCapacity   *int32
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CourseWithStats struct {
	Course
	BatchCount        int64
	OngoingBatchCount int64
	EnrolledCount     int64
}

type ArchiveResult struct {
	ID        uuid.UUID
	Status    string
	UpdatedAt time.Time
}

type ListParams struct {
	Page    int
	PerPage int
	Status  *string
	Level   *string
	Search  *string
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ListResponse struct {
	Data       []CourseWithStats
	Pagination PaginationMeta
}

type CreateCourseRequest struct {
	CourseName    string
	CourseCode    string
	Description   *string
	Subject       *string
	Level         *string
	SessionCount *int32
	Price         *string
	MaxCapacity   *int32
}

type UpdateCourseRequest struct {
	CourseName *string
	CourseCode *string
	Status     *string

	Description  *patch.Patchable[string]
	Subject      *patch.Patchable[string]
	Level        *patch.Patchable[string]
	SessionCount *patch.Patchable[int32]
	Price        *patch.Patchable[string]
	MaxCapacity  *patch.Patchable[int32]
}

type FullCourseUpdate struct {
	CourseName   string
	CourseCode   string
	Description  *string
	Subject      *string
	Level        *string
	SessionCount *int32
	Price        *string
	MaxCapacity  *int32
	Status       string
}

type CourseRepository interface {
	List(ctx context.Context, params ListParams) ([]CourseWithStats, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (CourseWithStats, error)
	Create(ctx context.Context, id uuid.UUID, req CreateCourseRequest) (Course, error)
	UpdateFull(ctx context.Context, id uuid.UUID, req FullCourseUpdate) (Course, error)
	Archive(ctx context.Context, id uuid.UUID) (ArchiveResult, error)
}

type CourseService struct {
	repo CourseRepository
}

func NewService(repo CourseRepository) *CourseService {
	return &CourseService{repo: repo}
}

func (s *CourseService) List(ctx context.Context, params ListParams) (*ListResponse, error) {
	courses, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	return &ListResponse{
		Data: courses,
		Pagination: PaginationMeta{
			Page:       params.Page,
			PerPage:    params.PerPage,
			Total:      int(total),
			TotalPages: pagination.TotalPages(total, params.PerPage),
		},
	}, nil
}

func (s *CourseService) GetByID(ctx context.Context, rawID string) (*CourseWithStats, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrNotFound
	}
	c, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *CourseService) Create(ctx context.Context, req CreateCourseRequest) (*Course, error) {
	req.CourseName = strings.TrimSpace(req.CourseName)
	req.CourseCode = strings.TrimSpace(req.CourseCode)

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	c, err := s.repo.Create(ctx, id, req)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrCourseCodeConflict
		}
		return nil, err
	}
	return &c, nil
}

func MergeCourseUpdate(existing CourseWithStats, req UpdateCourseRequest) FullCourseUpdate {
	u := FullCourseUpdate{
		CourseName:   existing.CourseName,
		CourseCode:   existing.CourseCode,
		Description:  existing.Description,
		Subject:      existing.Subject,
		Level:        existing.Level,
		SessionCount: existing.SessionCount,
		Price:        existing.Price,
		MaxCapacity:  existing.MaxCapacity,
		Status:       existing.Status,
	}
	if req.CourseName != nil {
		trimmed := strings.TrimSpace(*req.CourseName)
		u.CourseName = trimmed
	}
	if req.CourseCode != nil {
		trimmed := strings.TrimSpace(*req.CourseCode)
		u.CourseCode = trimmed
	}
	if req.Status != nil {
		u.Status = *req.Status
	}
	u.Description = req.Description.ValueOr(u.Description)
	u.Subject = req.Subject.ValueOr(u.Subject)
	u.Level = req.Level.ValueOr(u.Level)
	u.SessionCount = req.SessionCount.ValueOr(u.SessionCount)
	u.Price = req.Price.ValueOr(u.Price)
	u.MaxCapacity = req.MaxCapacity.ValueOr(u.MaxCapacity)
	return u
}

func (s *CourseService) Update(ctx context.Context, rawID string, req UpdateCourseRequest) (*Course, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrNotFound
	}
	existing, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	merged := MergeCourseUpdate(existing, req)
	c, err := s.repo.UpdateFull(ctx, id, merged)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrCourseCodeConflict
		}
		return nil, err
	}
	return &c, nil
}

func (s *CourseService) Archive(ctx context.Context, rawID string) (*ArchiveResult, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrNotFound
	}
	result, err := s.repo.Archive(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrHasOngoingBatches
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
