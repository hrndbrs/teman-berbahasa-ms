package enrollment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbq "github.com/hrndbrs/teman-berbahasa-ms/internal/db/query"
)

type pgEnrollmentRepository struct {
	pool *pgxpool.Pool
	q    *dbq.Queries
}

func NewRepository(pool *pgxpool.Pool) *pgEnrollmentRepository {
	return &pgEnrollmentRepository{pool: pool, q: dbq.New(pool)}
}

func toPgtypeUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func rowToEnrollment(row dbq.GetEnrollmentByIDRow) Enrollment {
	var enrollmentDate time.Time
	if row.EnrollmentDate.Valid {
		enrollmentDate = row.EnrollmentDate.Time
	}
	return Enrollment{
		ID:             row.ID,
		Status:         row.Status,
		PaymentStatus:  row.PaymentStatus,
		FinalGrade:     row.FinalGrade,
		EnrollmentDate: enrollmentDate,
		Student: StudentRef{
			ID:        row.StudentID,
			FirstName: row.StudentFirstName,
			LastName:  row.StudentLastName,
			Email:     row.StudentEmail,
		},
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
		},
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

func listRowToEnrollment(row dbq.ListEnrollmentsRow) Enrollment {
	var enrollmentDate time.Time
	if row.EnrollmentDate.Valid {
		enrollmentDate = row.EnrollmentDate.Time
	}
	return Enrollment{
		ID:             row.ID,
		Status:         row.Status,
		PaymentStatus:  row.PaymentStatus,
		FinalGrade:     row.FinalGrade,
		EnrollmentDate: enrollmentDate,
		Student: StudentRef{
			ID:        row.StudentID,
			FirstName: row.StudentFirstName,
			LastName:  row.StudentLastName,
			Email:     row.StudentEmail,
		},
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
		},
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

func (r *pgEnrollmentRepository) List(ctx context.Context, params ListParams) ([]Enrollment, int64, error) {
	offset := int32((params.Page - 1) * params.PerPage)

	rows, err := r.q.ListEnrollments(ctx, dbq.ListEnrollmentsParams{
		BatchID:       toPgtypeUUID(params.BatchID),
		StudentID:     toPgtypeUUID(params.StudentID),
		Status:        params.Status,
		PaymentStatus: params.PaymentStatus,
		PageOffset:    offset,
		PageSize:      int32(params.PerPage),
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := r.q.CountEnrollments(ctx, dbq.CountEnrollmentsParams{
		BatchID:       toPgtypeUUID(params.BatchID),
		StudentID:     toPgtypeUUID(params.StudentID),
		Status:        params.Status,
		PaymentStatus: params.PaymentStatus,
	})
	if err != nil {
		return nil, 0, err
	}

	result := make([]Enrollment, len(rows))
	for i, row := range rows {
		result[i] = listRowToEnrollment(row)
	}
	return result, total, nil
}

func (r *pgEnrollmentRepository) GetByID(ctx context.Context, id uuid.UUID) (Enrollment, error) {
	row, err := r.q.GetEnrollmentByID(ctx, id)
	if err != nil {
		return Enrollment{}, err
	}
	return rowToEnrollment(row), nil
}

func (r *pgEnrollmentRepository) Create(ctx context.Context, id uuid.UUID, req CreateRequest) (Enrollment, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Enrollment{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Advisory lock serializes enrollment creation per batch.
	// hashtext returns int4; cast to bigint for pg_advisory_xact_lock.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1)::bigint)", req.BatchID.String()); err != nil {
		return Enrollment{}, err
	}

	txq := r.q.WithTx(tx)

	batch, err := txq.GetBatchForEnrollment(ctx, req.BatchID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Enrollment{}, ErrBatchNotFound
		}
		return Enrollment{}, err
	}
	if batch.Status == "completed" {
		return Enrollment{}, ErrBatchCompleted
	}

	if batch.MaxCapacity != nil {
		count, err := txq.CountActiveEnrollments(ctx, req.BatchID)
		if err != nil {
			return Enrollment{}, err
		}
		if count >= int64(*batch.MaxCapacity) {
			return Enrollment{}, ErrCapacityFull
		}
	}

	_, err = txq.CreateEnrollment(ctx, dbq.CreateEnrollmentParams{
		ID:        id,
		StudentID: req.StudentID,
		BatchID:   req.BatchID,
		CourseID:  batch.CourseID,
	})
	if err != nil {
		if isFKViolation(err) {
			// batch FK already confirmed above; only student FK can fire here
			return Enrollment{}, ErrStudentNotFound
		}
		return Enrollment{}, err
	}

	// Fetch full enrollment with joins inside the transaction — no post-commit round-trip.
	row, err := txq.GetEnrollmentByID(ctx, id)
	if err != nil {
		return Enrollment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Enrollment{}, err
	}

	return rowToEnrollment(row), nil
}

// txGetCurrent locks the enrollment row for update and returns its current transition-relevant state.
func (r *pgEnrollmentRepository) txGetCurrent(ctx context.Context, tx pgx.Tx, id uuid.UUID) (EnrollmentCurrent, error) {
	var cur EnrollmentCurrent
	err := tx.QueryRow(ctx,
		"SELECT status, payment_status, final_grade FROM enrollments WHERE id = $1 FOR UPDATE NOWAIT",
		id,
	).Scan(&cur.Status, &cur.PaymentStatus, &cur.FinalGrade)
	if err != nil {
		return EnrollmentCurrent{}, err
	}
	return cur, nil
}

func (r *pgEnrollmentRepository) Update(ctx context.Context, id uuid.UUID, req UpdateRequest, validate func(EnrollmentCurrent) error) (Enrollment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Enrollment{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	cur, err := r.txGetCurrent(ctx, tx, id)
	if err != nil {
		return Enrollment{}, err
	}

	if err := validate(cur); err != nil {
		return Enrollment{}, err
	}

	finalGrade := cur.FinalGrade
	if req.FinalGrade.Set() {
		if req.FinalGrade.IsNull {
			finalGrade = nil
		} else {
			finalGrade = req.FinalGrade.Value
		}
	}

	_, err = tx.Exec(ctx,
		`UPDATE enrollments SET
			status        = COALESCE($1::text, status),
			payment_status = COALESCE($2::text, payment_status),
			final_grade   = $3::text,
			updated_at    = NOW()
		WHERE id = $4`,
		req.Status, req.PaymentStatus, finalGrade, id,
	)
	if err != nil {
		return Enrollment{}, err
	}

	// Fetch full enrollment with joins inside the transaction.
	txq := r.q.WithTx(tx)
	row, err := txq.GetEnrollmentByID(ctx, id)
	if err != nil {
		return Enrollment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Enrollment{}, err
	}

	return rowToEnrollment(row), nil
}
