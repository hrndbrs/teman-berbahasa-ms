package student

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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

var (
	ErrNotFound      = errors.New("student not found")
	ErrEmailConflict = errors.New("email already in use")
)

type Student struct {
	ID               uuid.UUID
	FirstName        string
	LastName         string
	Email            *string
	Phone            *string
	DateOfBirth      *time.Time
	Gender           *string
	Address          *string
	ParentName       *string
	ParentPhone      *string
	RegistrationDate time.Time
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
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
	Level      *string
}

type EnrollmentSummary struct {
	ID             uuid.UUID
	Status         string
	PaymentStatus  string
	EnrollmentDate time.Time
	Batch          BatchRef
	Course         CourseRef
}

type StudentDetail struct {
	Student
	Enrollments []EnrollmentSummary
}

type ListItem struct {
	Student
	EnrolledBatches []string
}

type ListParams struct {
	Page    int
	PerPage int
	Status  *string
	Search  *string
}

type ListResponse struct {
	Data       []ListItem
	Pagination PaginationMeta
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type CreateStudentRequest struct {
	FirstName        string
	LastName         string
	Email            *string
	Phone            *string
	DateOfBirth      *time.Time
	Gender           *string
	Address          *string
	ParentName       *string
	ParentPhone      *string
	RegistrationDate *time.Time
}

type UpdateStudentRequest struct {
	FirstName *string
	LastName  *string
	Status    *string

	Email       *patch.Patchable[string]
	Phone       *patch.Patchable[string]
	DateOfBirth *patch.Patchable[time.Time]
	Gender      *patch.Patchable[string]
	Address     *patch.Patchable[string]
	ParentName  *patch.Patchable[string]
	ParentPhone *patch.Patchable[string]
}

type StudentRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (Student, error)
	GetByEmail(ctx context.Context, email string) (Student, error)
	List(ctx context.Context, params ListParams) ([]Student, int64, error)
	GetBatchCodes(ctx context.Context, studentIDs []uuid.UUID) (map[uuid.UUID][]string, error)
	GetEnrollments(ctx context.Context, studentID uuid.UUID) ([]EnrollmentSummary, error)
	Create(ctx context.Context, id uuid.UUID, req CreateStudentRequest) (Student, error)
	Update(ctx context.Context, id uuid.UUID, req FullStudentUpdate) (Student, error)
}

type StudentService struct {
	repo StudentRepository
}

func NewService(repo StudentRepository) *StudentService {
	return &StudentService{repo: repo}
}

func (s *StudentService) CreateStudent(ctx context.Context, req CreateStudentRequest) (*Student, error) {
	if req.Email != nil {
		_, err := s.repo.GetByEmail(ctx, *req.Email)
		if err == nil {
			return nil, ErrEmailConflict
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	student, err := s.repo.Create(ctx, id, req)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailConflict
		}
		return nil, err
	}
	return &student, nil
}

func (s *StudentService) GetStudent(ctx context.Context, id string) (*StudentDetail, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrNotFound
	}
	student, err := s.repo.GetByID(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	enrollments, err := s.repo.GetEnrollments(ctx, uid)
	if err != nil {
		return nil, err
	}
	if enrollments == nil {
		enrollments = []EnrollmentSummary{}
	}
	return &StudentDetail{Student: student, Enrollments: enrollments}, nil
}

func (s *StudentService) ListStudents(ctx context.Context, params ListParams) (*ListResponse, error) {
	students, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(students))
	for i, st := range students {
		ids[i] = st.ID
	}
	batchCodes, err := s.repo.GetBatchCodes(ctx, ids)
	if err != nil {
		return nil, err
	}

	items := make([]ListItem, len(students))
	for i, st := range students {
		codes := batchCodes[st.ID]
		if codes == nil {
			codes = []string{}
		}
		items[i] = ListItem{Student: st, EnrolledBatches: codes}
	}

	return &ListResponse{
		Data: items,
		Pagination: PaginationMeta{
			Page:       params.Page,
			PerPage:    params.PerPage,
			Total:      int(total),
			TotalPages: pagination.TotalPages(total, params.PerPage),
		},
	}, nil
}

func MergeStudentUpdate(existing Student, req UpdateStudentRequest) FullStudentUpdate {
	u := FullStudentUpdate{
		FirstName:   existing.FirstName,
		LastName:    existing.LastName,
		Email:       existing.Email,
		Phone:       existing.Phone,
		DateOfBirth: existing.DateOfBirth,
		Gender:      existing.Gender,
		Address:     existing.Address,
		ParentName:  existing.ParentName,
		ParentPhone: existing.ParentPhone,
		Status:      existing.Status,
	}
	if req.FirstName != nil {
		u.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		u.LastName = *req.LastName
	}
	if req.Status != nil {
		u.Status = *req.Status
	}
	u.Email = req.Email.ValueOr(u.Email)
	u.Phone = req.Phone.ValueOr(u.Phone)
	u.DateOfBirth = req.DateOfBirth.ValueOr(u.DateOfBirth)
	u.Gender = req.Gender.ValueOr(u.Gender)
	u.Address = req.Address.ValueOr(u.Address)
	u.ParentName = req.ParentName.ValueOr(u.ParentName)
	u.ParentPhone = req.ParentPhone.ValueOr(u.ParentPhone)
	return u
}

type FullStudentUpdate struct {
	FirstName        string
	LastName         string
	Email            *string
	Phone            *string
	DateOfBirth      *time.Time
	Gender           *string
	Address          *string
	ParentName       *string
	ParentPhone      *string
	Status           string
}

func (s *StudentService) UpdateStudent(ctx context.Context, id string, req UpdateStudentRequest) (*Student, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrNotFound
	}
	existing, err := s.repo.GetByID(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if req.Email.Set() && !req.Email.IsNull && req.Email.Value != nil {
		existingEmail := ""
		if existing.Email != nil {
			existingEmail = *existing.Email
		}
		if *req.Email.Value != existingEmail {
			_, err := s.repo.GetByEmail(ctx, *req.Email.Value)
			if err == nil {
				return nil, ErrEmailConflict
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
		}
	}
	merged := MergeStudentUpdate(existing, req)
	student, err := s.repo.Update(ctx, uid, merged)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailConflict
		}
		return nil, err
	}
	return &student, nil
}
