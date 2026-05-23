package enrollment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	ipagination "github.com/hrndbrs/teman-berbahasa-ms/internal/pagination"
)

var (
	ErrNotFound                    = errors.New("enrollment not found")
	ErrBatchNotFound               = errors.New("batch not found")
	ErrStudentNotFound             = errors.New("student not found")
	ErrBatchCompleted              = errors.New("cannot enroll into a completed batch")
	ErrCapacityFull                = errors.New("batch is at full capacity")
	ErrDuplicate                   = errors.New("student already enrolled in this batch")
	ErrInvalidTransition           = errors.New("invalid status transition")
	ErrFinalGradeRequiresCompleted = errors.New("final grade can only be set on completed enrollments")
)

// EnrollmentCurrent holds the locked DB state used for transition validation inside Update.
type EnrollmentCurrent struct {
	Status        string
	PaymentStatus string
	FinalGrade    *string
}

var validStatusTransitions = map[string][]string{
	"enrolled": {"dropped", "completed"},
}

var validPaymentTransitions = map[string][]string{
	"pending": {"partial", "paid"},
	"partial": {"paid"},
}

func validateStatusTransition(current, next string) error {
	if current == next {
		return nil
	}
	allowed, ok := validStatusTransitions[current]
	if !ok {
		return ErrInvalidTransition
	}
	for _, s := range allowed {
		if s == next {
			return nil
		}
	}
	return ErrInvalidTransition
}

func validatePaymentTransition(current, next string) error {
	if current == next {
		return nil
	}
	allowed, ok := validPaymentTransitions[current]
	if !ok {
		return ErrInvalidTransition
	}
	for _, s := range allowed {
		if s == next {
			return nil
		}
	}
	return ErrInvalidTransition
}

type StudentRef struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	Email     *string
}

type BatchRef struct {
	ID        uuid.UUID
	BatchName string
	BatchCode string
	Status    string
}

type CourseRef struct {
	ID         uuid.UUID
	CourseName string
	CourseCode string
}

type Enrollment struct {
	ID             uuid.UUID
	Status         string
	PaymentStatus  string
	FinalGrade     *string
	EnrollmentDate time.Time
	Student        StudentRef
	Batch          BatchRef
	Course         CourseRef
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ListParams struct {
	Page          int
	PerPage       int
	BatchID       *uuid.UUID
	StudentID     *uuid.UUID
	Status        *string
	PaymentStatus *string
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ListResponse struct {
	Data       []Enrollment
	Pagination PaginationMeta
}

type CreateRequest struct {
	StudentID uuid.UUID
	BatchID   uuid.UUID
}

type UpdateRequest struct {
	Status        *string
	PaymentStatus *string
	FinalGrade    *string
}

type EnrollmentRepository interface {
	List(ctx context.Context, params ListParams) ([]Enrollment, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (Enrollment, error)
	Create(ctx context.Context, id uuid.UUID, req CreateRequest) (Enrollment, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest, validate func(EnrollmentCurrent) error) (Enrollment, error)
}

type EnrollmentService struct {
	repo EnrollmentRepository
}

func NewService(repo EnrollmentRepository) *EnrollmentService {
	return &EnrollmentService{repo: repo}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "40001"
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func (s *EnrollmentService) List(ctx context.Context, params ListParams) (*ListResponse, error) {
	rows, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	return &ListResponse{
		Data: rows,
		Pagination: PaginationMeta{
			Page:       params.Page,
			PerPage:    params.PerPage,
			Total:      int(total),
			TotalPages: ipagination.TotalPages(total, params.PerPage),
		},
	}, nil
}

func (s *EnrollmentService) GetByID(ctx context.Context, rawID string) (*Enrollment, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrNotFound
	}
	e, err := s.repo.GetByID(ctx, id)
	if isNoRows(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *EnrollmentService) Create(ctx context.Context, req CreateRequest) (*Enrollment, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	e, err := s.repo.Create(ctx, id, req)
	if err != nil && isSerializationFailure(err) {
		time.Sleep(50 * time.Millisecond)
		id, err = uuid.NewV7()
		if err != nil {
			return nil, err
		}
		e, err = s.repo.Create(ctx, id, req)
	}
	if err != nil {
		switch {
		case errors.Is(err, ErrBatchNotFound):
			return nil, ErrBatchNotFound
		case errors.Is(err, ErrStudentNotFound):
			return nil, ErrStudentNotFound
		case errors.Is(err, ErrBatchCompleted):
			return nil, ErrBatchCompleted
		case errors.Is(err, ErrCapacityFull):
			return nil, ErrCapacityFull
		case isUniqueViolation(err):
			return nil, ErrDuplicate
		}
		return nil, err
	}
	return &e, nil
}

func (s *EnrollmentService) Update(ctx context.Context, rawID string, req UpdateRequest) (*Enrollment, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrNotFound
	}

	validate := func(cur EnrollmentCurrent) error {
		if req.FinalGrade != nil && cur.Status != "completed" {
			return ErrFinalGradeRequiresCompleted
		}
		if req.Status != nil {
			if err := validateStatusTransition(cur.Status, *req.Status); err != nil {
				return err
			}
		}
		if req.PaymentStatus != nil {
			if err := validatePaymentTransition(cur.PaymentStatus, *req.PaymentStatus); err != nil {
				return err
			}
		}
		return nil
	}

	e, err := s.repo.Update(ctx, id, req, validate)
	if isNoRows(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}
