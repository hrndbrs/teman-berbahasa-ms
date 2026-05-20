package course

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
	DurationWeeks *int32
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
	DurationWeeks *int32
	Price         *string
	MaxCapacity   *int32
}

type UpdateCourseRequest struct {
	CourseName    *string
	CourseCode    *string
	Description   *string
	Subject       *string
	Level         *string
	DurationWeeks *int32
	Price         *string
	MaxCapacity   *int32
}

type CourseRepository interface {
	List(ctx context.Context, params ListParams) ([]CourseWithStats, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (CourseWithStats, error)
	Create(ctx context.Context, id uuid.UUID, req CreateCourseRequest) (Course, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateCourseRequest) (Course, error)
	Archive(ctx context.Context, id uuid.UUID) (ArchiveResult, error)
	CountOngoingBatches(ctx context.Context, courseID uuid.UUID) (int64, error)
}

type CourseService struct {
	repo CourseRepository
}

func NewService(repo CourseRepository) *CourseService {
	return &CourseService{repo: repo}
}

func (s *CourseService) List(ctx context.Context, params ListParams) (*ListResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 || params.PerPage > 100 {
		params.PerPage = 20
	}
	courses, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	totalPages := int(total) / params.PerPage
	if int(total)%params.PerPage != 0 {
		totalPages++
	}
	return &ListResponse{
		Data: courses,
		Pagination: PaginationMeta{
			Page:       params.Page,
			PerPage:    params.PerPage,
			Total:      int(total),
			TotalPages: totalPages,
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

func (s *CourseService) Update(ctx context.Context, rawID string, req UpdateCourseRequest) (*Course, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrNotFound
	}
	_, err = s.repo.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if req.CourseName != nil {
		trimmed := strings.TrimSpace(*req.CourseName)
		req.CourseName = &trimmed
	}
	if req.CourseCode != nil {
		trimmed := strings.TrimSpace(*req.CourseCode)
		req.CourseCode = &trimmed
	}
	c, err := s.repo.Update(ctx, id, req)
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
	_, err = s.repo.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	count, err := s.repo.CountOngoingBatches(ctx, id)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrHasOngoingBatches
	}
	result, err := s.repo.Archive(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
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
