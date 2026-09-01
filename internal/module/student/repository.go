package student

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbq "github.com/hrndbrs/teman-berbahasa-ms/internal/db/query"
)

type pgStudentRepository struct {
	q *dbq.Queries
}

func NewRepository(pool *pgxpool.Pool) *pgStudentRepository {
	return &pgStudentRepository{q: dbq.New(pool)}
}

func pgDateToTimePtr(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

func timePtrToPgDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func rowToStudent(row dbq.Student) Student {
	return Student{
		ID:               row.ID,
		FirstName:        row.FirstName,
		LastName:         row.LastName,
		Email:            row.Email,
		Phone:            row.Phone,
		DateOfBirth:      pgDateToTimePtr(row.DateOfBirth),
		Gender:           row.Gender,
		Address:          row.Address,
		ParentName:       row.ParentName,
		ParentPhone:      row.ParentPhone,
		RegistrationDate: row.RegistrationDate.Time,
		Status:           row.Status,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}
}

func (r *pgStudentRepository) GetByID(ctx context.Context, id uuid.UUID) (Student, error) {
	row, err := r.q.GetStudentByID(ctx, id)
	if err != nil {
		return Student{}, err
	}
	return rowToStudent(row), nil
}

func (r *pgStudentRepository) GetByEmail(ctx context.Context, email string) (Student, error) {
	row, err := r.q.GetStudentByEmail(ctx, email)
	if err != nil {
		return Student{}, err
	}
	return rowToStudent(row), nil
}

func (r *pgStudentRepository) List(ctx context.Context, params ListParams) ([]Student, int64, error) {
	rows, err := r.q.ListStudents(ctx, dbq.ListStudentsParams{
		Status:     params.Status,
		Search:     params.Search,
		PageSize:   int32(params.PerPage),
		PageOffset: int32((params.Page - 1) * params.PerPage),
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := r.q.CountStudents(ctx, dbq.CountStudentsParams{
		Status: params.Status,
		Search: params.Search,
	})
	if err != nil {
		return nil, 0, err
	}
	students := make([]Student, len(rows))
	for i, row := range rows {
		students[i] = rowToStudent(row)
	}
	return students, total, nil
}

func (r *pgStudentRepository) GetBatchCodes(ctx context.Context, studentIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	if len(studentIDs) == 0 {
		return map[uuid.UUID][]string{}, nil
	}
	rows, err := r.q.GetStudentBatchCodes(ctx, studentIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID][]string)
	for _, row := range rows {
		result[row.StudentID] = append(result[row.StudentID], row.BatchCode)
	}
	return result, nil
}

func (r *pgStudentRepository) GetEnrollments(ctx context.Context, studentID uuid.UUID) ([]EnrollmentSummary, error) {
	rows, err := r.q.GetStudentEnrollments(ctx, studentID)
	if err != nil {
		return nil, err
	}
	enrollments := make([]EnrollmentSummary, len(rows))
	for i, row := range rows {
		var enrollmentDate time.Time
		if row.EnrollmentDate.Valid {
			enrollmentDate = row.EnrollmentDate.Time
		}
		enrollments[i] = EnrollmentSummary{
			ID:             row.ID,
			Status:         row.Status,
			PaymentStatus:  row.PaymentStatus,
			EnrollmentDate: enrollmentDate,
			Batch: BatchRef{
				ID:        row.BatchID,
				BatchName: row.BatchName,
				BatchCode: row.BatchCode,
				Status:    row.BatchStatus,
			},
			Course: CourseRef{
				ID:         row.CourseID,
				CourseName: row.CourseName,
				CourseCode: row.CourseCode,
				Level:      row.Level,
			},
		}
	}
	return enrollments, nil
}

func (r *pgStudentRepository) Create(ctx context.Context, id uuid.UUID, req CreateStudentRequest) (Student, error) {
	var regDate interface{}
	if req.RegistrationDate != nil {
		regDate = pgtype.Date{Time: *req.RegistrationDate, Valid: true}
	}
	row, err := r.q.CreateStudent(ctx, dbq.CreateStudentParams{
		ID:               id,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		Email:            req.Email,
		Phone:            req.Phone,
		DateOfBirth:      timePtrToPgDate(req.DateOfBirth),
		Gender:           req.Gender,
		Address:          req.Address,
		ParentName:       req.ParentName,
		ParentPhone:      req.ParentPhone,
		RegistrationDate: regDate,
	})
	if err != nil {
		return Student{}, err
	}
	return rowToStudent(row), nil
}

func (r *pgStudentRepository) Update(ctx context.Context, id uuid.UUID, req FullStudentUpdate) (Student, error) {
	row, err := r.q.UpdateStudentFull(ctx, dbq.UpdateStudentFullParams{
		ID:          id,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Phone:       req.Phone,
		DateOfBirth: timePtrToPgDate(req.DateOfBirth),
		Gender:      req.Gender,
		Address:     req.Address,
		ParentName:  req.ParentName,
		ParentPhone: req.ParentPhone,
		Status:      req.Status,
	})
	if err != nil {
		return Student{}, err
	}
	return rowToStudent(row), nil
}
