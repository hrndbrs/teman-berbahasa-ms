package course

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbq "github.com/hrndbrs/teman-berbahasa-ms/internal/db/query"
)

type pgCourseRepository struct {
	pool *pgxpool.Pool
	q    *dbq.Queries
}

func NewRepository(pool *pgxpool.Pool) *pgCourseRepository {
	return &pgCourseRepository{pool: pool, q: dbq.New(pool)}
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
	sc := row.SessionCount
	return Course{
		ID:            row.ID,
		CourseName:    row.CourseName,
		CourseCode:    row.CourseCode,
		Description:   row.Description,
		Subject:       row.Subject,
		Level:         row.Level,
		SessionCount: &sc,
		Price:         numericToString(row.Price),
		MaxCapacity:   row.MaxCapacity,
		Status:        row.Status,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
}

func scanCourseRow(row pgx.Row) (Course, error) {
	var c Course
	var createdAt, updatedAt pgtype.Timestamptz
	var price pgtype.Numeric
	err := row.Scan(
		&c.ID, &c.CourseName, &c.CourseCode, &c.Description, &c.Subject,
		&c.Level, &c.SessionCount, &price, &c.MaxCapacity,
		&c.Status, &createdAt, &updatedAt,
	)
	if err != nil {
		return Course{}, err
	}
	c.Price = numericToString(price)
	c.CreatedAt = createdAt.Time
	c.UpdatedAt = updatedAt.Time
	return c, nil
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
	sc := row.SessionCount
	return CourseWithStats{
		Course: Course{
			ID:            row.ID,
			CourseName:    row.CourseName,
			CourseCode:    row.CourseCode,
			Description:   row.Description,
			Subject:       row.Subject,
			Level:         row.Level,
			SessionCount: &sc,
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

func (r *pgCourseRepository) UpdateFull(ctx context.Context, id uuid.UUID, req FullCourseUpdate) (Course, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE courses SET
			course_name    = $1,
			course_code    = $2,
			description    = $3,
			subject        = $4,
			level          = $5,
			session_count = $6,
			price          = $7,
			max_capacity   = $8,
			updated_at     = NOW()
		WHERE id = $9
		RETURNING id, course_name, course_code, description, subject, level, session_count, price, max_capacity, status, created_at, updated_at`,
		req.CourseName, req.CourseCode, req.Description, req.Subject,
		req.Level, req.SessionCount, stringToNumeric(req.Price), req.MaxCapacity, id,
	)
	return scanCourseRow(row)
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


