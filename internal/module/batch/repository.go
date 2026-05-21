package batch

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbq "github.com/hrndbrs/teman-berbahasa-ms/internal/db/query"
)

type pgBatchRepository struct {
	q *dbq.Queries
}

func NewRepository(pool *pgxpool.Pool) *pgBatchRepository {
	return &pgBatchRepository{q: dbq.New(pool)}
}

func toPgtypeDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func fromPgtypeDate(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
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
			StartDate:        fromPgtypeDate(row.StartDate),
			EndDate:          fromPgtypeDate(row.EndDate),
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

func (r *pgBatchRepository) Create(ctx context.Context, id uuid.UUID, req CreateBatchRequest) (BatchWithDetails, error) {
	_, err := r.q.CreateBatch(ctx, dbq.CreateBatchParams{
		ID:               id,
		CourseID:         req.CourseID,
		InstructorUserID: req.InstructorUserID,
		CreatedByUserID:  req.CreatedByUserID,
		BatchName:        req.BatchName,
		BatchCode:        req.BatchCode,
		StartDate:        toPgtypeDate(req.StartDate),
		EndDate:          toPgtypeDate(req.EndDate),
		AcademicYear:     req.AcademicYear,
	})
	if err != nil {
		return BatchWithDetails{}, err
	}
	return r.GetByID(ctx, id)
}

func (r *pgBatchRepository) Update(ctx context.Context, id uuid.UUID, req UpdateBatchRequest) (BatchWithDetails, error) {
	_, err := r.q.UpdateBatch(ctx, dbq.UpdateBatchParams{
		InstructorUserID: toPgtypeUUID(req.InstructorUserID),
		BatchName:        req.BatchName,
		BatchCode:        req.BatchCode,
		StartDate:        toPgtypeDate(req.StartDate),
		EndDate:          toPgtypeDate(req.EndDate),
		AcademicYear:     req.AcademicYear,
		ID:               id,
	})
	if err != nil {
		return BatchWithDetails{}, err
	}
	return r.GetByID(ctx, id)
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
	return r.q.DeleteBatch(ctx, id)
}

func (r *pgBatchRepository) CountActiveEnrollments(ctx context.Context, batchID uuid.UUID) (int64, error) {
	return r.q.CountActiveEnrollments(ctx, batchID)
}

func (r *pgBatchRepository) ExistsActiveTeacher(ctx context.Context, instructorID uuid.UUID) (bool, error) {
	return r.q.ExistsActiveTeacher(ctx, instructorID)
}
