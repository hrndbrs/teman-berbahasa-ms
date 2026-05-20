package course

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbq "github.com/hrndbrs/teman-berbahasa-ms/internal/db/query"
)

type pgCourseRepository struct {
	q *dbq.Queries
}

func NewRepository(pool *pgxpool.Pool) *pgCourseRepository {
	return &pgCourseRepository{q: dbq.New(pool)}
}

func numericToString(n pgtype.Numeric) *string {
	if !n.Valid {
		return nil
	}
	v, err := n.Value()
	if err != nil || v == nil {
		return nil
	}
	s := fmt.Sprintf("%v", v)
	return &s
}

func stringToNumeric(s *string) pgtype.Numeric {
	if s == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	if err := n.Scan(*s); err != nil {
		return pgtype.Numeric{}
	}
	return n
}

func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int64:
		return n
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func rowToCourse(row dbq.Course) Course {
	return Course{
		ID:            row.ID,
		CourseName:    row.CourseName,
		CourseCode:    row.CourseCode,
		Description:   row.Description,
		Subject:       row.Subject,
		Level:         row.Level,
		SessionCount: row.SessionCount,
		Price:         numericToString(row.Price),
		MaxCapacity:   row.MaxCapacity,
		Status:        row.Status,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
}

func listRowToCourseWithStats(row dbq.CoursesWithStat) CourseWithStats {
	return CourseWithStats{
		Course: Course{
			ID:            row.ID,
			CourseName:    row.CourseName,
			CourseCode:    row.CourseCode,
			Description:   row.Description,
			Subject:       row.Subject,
			Level:         row.Level,
			SessionCount: row.SessionCount,
			Price:         numericToString(row.Price),
			MaxCapacity:   row.MaxCapacity,
			Status:        row.Status,
			CreatedAt:     row.CreatedAt.Time,
			UpdatedAt:     row.UpdatedAt.Time,
		},
		BatchCount:        row.BatchCount,
		OngoingBatchCount: row.OngoingBatchCount,
		EnrolledCount:     toInt64(row.EnrolledCount),
	}
}

func getByIDRowToCourseWithStats(row dbq.GetCourseByIDRow) CourseWithStats {
	return CourseWithStats{
		Course: Course{
			ID:            row.ID,
			CourseName:    row.CourseName,
			CourseCode:    row.CourseCode,
			Description:   row.Description,
			Subject:       row.Subject,
			Level:         row.Level,
			SessionCount: row.SessionCount,
			Price:         numericToString(row.Price),
			MaxCapacity:   row.MaxCapacity,
			Status:        row.Status,
			CreatedAt:     row.CreatedAt.Time,
			UpdatedAt:     row.UpdatedAt.Time,
		},
		BatchCount:        row.BatchCount,
		OngoingBatchCount: row.OngoingBatchCount,
		EnrolledCount:     toInt64(row.EnrolledCount),
	}
}

func (r *pgCourseRepository) List(ctx context.Context, params ListParams) ([]CourseWithStats, int64, error) {
	rows, err := r.q.ListCourses(ctx, dbq.ListCoursesParams{
		Status:     params.Status,
		Level:      params.Level,
		Search:     params.Search,
		PageOffset: int32((params.Page - 1) * params.PerPage),
		PageSize:   int32(params.PerPage),
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := r.q.CountCourses(ctx, dbq.CountCoursesParams{
		Status: params.Status,
		Level:  params.Level,
		Search: params.Search,
	})
	if err != nil {
		return nil, 0, err
	}
	courses := make([]CourseWithStats, len(rows))
	for i, row := range rows {
		courses[i] = listRowToCourseWithStats(row)
	}
	return courses, total, nil
}

func (r *pgCourseRepository) GetByID(ctx context.Context, id uuid.UUID) (CourseWithStats, error) {
	row, err := r.q.GetCourseByID(ctx, id)
	if err != nil {
		return CourseWithStats{}, err
	}
	return getByIDRowToCourseWithStats(row), nil
}

func (r *pgCourseRepository) Create(ctx context.Context, id uuid.UUID, req CreateCourseRequest) (Course, error) {
	row, err := r.q.CreateCourse(ctx, dbq.CreateCourseParams{
		ID:            id,
		CourseName:    req.CourseName,
		CourseCode:    req.CourseCode,
		Description:   req.Description,
		Subject:       req.Subject,
		Level:         req.Level,
		SessionCount: req.SessionCount,
		Price:         stringToNumeric(req.Price),
		MaxCapacity:   req.MaxCapacity,
	})
	if err != nil {
		return Course{}, err
	}
	return rowToCourse(row), nil
}

func (r *pgCourseRepository) Update(ctx context.Context, id uuid.UUID, req UpdateCourseRequest) (Course, error) {
	row, err := r.q.UpdateCourse(ctx, dbq.UpdateCourseParams{
		CourseName:    req.CourseName,
		CourseCode:    req.CourseCode,
		Description:   req.Description,
		Subject:       req.Subject,
		Level:         req.Level,
		SessionCount: req.SessionCount,
		Price:         stringToNumeric(req.Price),
		MaxCapacity:   req.MaxCapacity,
		ID:            id,
	})
	if err != nil {
		return Course{}, err
	}
	return rowToCourse(row), nil
}

func (r *pgCourseRepository) Archive(ctx context.Context, id uuid.UUID) (ArchiveResult, error) {
	row, err := r.q.ArchiveCourse(ctx, id)
	if err != nil {
		return ArchiveResult{}, err
	}
	return ArchiveResult{
		ID:        row.ID,
		Status:    row.Status,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (r *pgCourseRepository) CountOngoingBatches(ctx context.Context, courseID uuid.UUID) (int64, error) {
	return r.q.CountOngoingBatchesByCourse(ctx, courseID)
}
