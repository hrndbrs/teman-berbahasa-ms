package batch

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbq "github.com/hrndbrs/teman-berbahasa-ms/internal/db/query"
)

type pgBatchRepository struct {
	pool *pgxpool.Pool
	q    *dbq.Queries
}

func NewRepository(pool *pgxpool.Pool) *pgBatchRepository {
	return &pgBatchRepository{pool: pool, q: dbq.New(pool)}
}

func scanBatchRow(row pgx.Row) (BatchWithDetails, error) {
	var b BatchWithDetails
	var createdAt, updatedAt pgtype.Timestamptz
	var academicYear *string
	err := row.Scan(
		&b.ID, &b.CourseID, &b.InstructorUserID, &b.CreatedByUserID,
		&b.BatchName, &b.BatchCode, &academicYear,
		&b.Status, &createdAt, &updatedAt,
		&b.CourseName, &b.CourseCode,
		&b.InstructorFirstName, &b.InstructorLastName,
		&b.EnrolledCount,
	)
	if err != nil {
		return BatchWithDetails{}, err
	}
	b.AcademicYear = academicYear
	b.CreatedAt = createdAt.Time
	b.UpdatedAt = updatedAt.Time
	return b, nil
}

func toPgtypeUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func rowToBatchWithDetails(row dbq.BatchesWithStat) BatchWithDetails {
	return BatchWithDetails{
		Batch: Batch{
			ID:               row.ID,
			CourseID:         row.CourseID,
			InstructorUserID: row.InstructorUserID,
			CreatedByUserID:  row.CreatedByUserID,
			BatchName:        row.BatchName,
			BatchCode:        row.BatchCode,
			AcademicYear:     row.AcademicYear,
			Status:           row.Status,
			CreatedAt:        row.CreatedAt.Time,
			UpdatedAt:        row.UpdatedAt.Time,
		},
		CourseName:          row.CourseName,
		CourseCode:          row.CourseCode,
		InstructorFirstName: row.InstructorFirstName,
		InstructorLastName:  row.InstructorLastName,
		EnrolledCount:       row.EnrolledCount,
	}
}

func (r *pgBatchRepository) List(ctx context.Context, params ListParams) ([]BatchWithDetails, int64, error) {
	offset := int32((params.Page - 1) * params.PerPage)
	rows, err := r.q.ListBatches(ctx, dbq.ListBatchesParams{
		Status:     params.Status,
		CourseID:   toPgtypeUUID(params.CourseID),
		Search:     params.Search,
		PageOffset: offset,
		PageSize:   int32(params.PerPage),
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := r.q.CountBatches(ctx, dbq.CountBatchesParams{
		Status:   params.Status,
		CourseID: toPgtypeUUID(params.CourseID),
		Search:   params.Search,
	})
	if err != nil {
		return nil, 0, err
	}
	batches := make([]BatchWithDetails, len(rows))
	for i, row := range rows {
		batches[i] = rowToBatchWithDetails(row)
	}
	return batches, total, nil
}

func (r *pgBatchRepository) GetByID(ctx context.Context, id uuid.UUID) (BatchWithDetails, error) {
	row, err := r.q.GetBatchByID(ctx, id)
	if err != nil {
		return BatchWithDetails{}, err
	}
	return rowToBatchWithDetails(row), nil
}

const createBatchReturning = `WITH ins AS (
  INSERT INTO batches (id, course_id, instructor_user_id, created_by_user_id, batch_name, batch_code, status, academic_year)
  VALUES ($1, $2, $3, $4, $5, $6, 'upcoming', $7)
  RETURNING id
)
SELECT b.id, b.course_id, b.instructor_user_id, b.created_by_user_id,
       b.batch_name, b.batch_code, b.academic_year, b.status,
       b.created_at, b.updated_at,
       b.course_name, b.course_code,
       b.instructor_first_name, b.instructor_last_name,
       b.enrolled_count
FROM batches_with_stats b JOIN ins ON b.id = ins.id`

const updateBatchReturning = `WITH upd AS (
  UPDATE batches SET
    instructor_user_id = $2::uuid,
    batch_name         = $3::text,
    batch_code         = $4::text,
    academic_year      = $5::text,
    updated_at         = NOW()
  WHERE id = $1
  RETURNING id
)
SELECT b.id, b.course_id, b.instructor_user_id, b.created_by_user_id,
       b.batch_name, b.batch_code, b.academic_year, b.status,
       b.created_at, b.updated_at,
       b.course_name, b.course_code,
       b.instructor_first_name, b.instructor_last_name,
       b.enrolled_count
FROM batches_with_stats b JOIN upd ON b.id = upd.id`

func (r *pgBatchRepository) Create(ctx context.Context, id uuid.UUID, req CreateBatchRequest) (BatchWithDetails, error) {
	row := r.pool.QueryRow(ctx, createBatchReturning,
		id, req.CourseID, req.InstructorUserID, req.CreatedByUserID,
		req.BatchName, req.BatchCode, req.AcademicYear,
	)
	return scanBatchRow(row)
}

func (r *pgBatchRepository) Update(ctx context.Context, id uuid.UUID, req FullBatchUpdate) (BatchWithDetails, error) {
	row := r.pool.QueryRow(ctx, updateBatchReturning,
		id,
		req.InstructorUserID,
		req.BatchName,
		req.BatchCode,
		req.AcademicYear,
	)
	return scanBatchRow(row)
}

func (r *pgBatchRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (StatusUpdateResult, error) {
	row, err := r.q.UpdateBatchStatus(ctx, dbq.UpdateBatchStatusParams{
		Status: status,
		ID:     id,
	})
	if err != nil {
		return StatusUpdateResult{}, err
	}
	return StatusUpdateResult{
		ID:        row.ID,
		Status:    row.Status,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (r *pgBatchRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.q.DeleteBatchIfEmpty(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		_, existErr := r.q.GetBatchByID(ctx, id)
		if errors.Is(existErr, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return ErrHasActiveEnrollments
	}
	return err
}

func (r *pgBatchRepository) ExistsActiveTeacher(ctx context.Context, instructorID uuid.UUID) (bool, error) {
	return r.q.ExistsActiveTeacher(ctx, instructorID)
}
