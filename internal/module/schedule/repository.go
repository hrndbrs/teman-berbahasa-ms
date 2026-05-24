package schedule

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbq "github.com/hrndbrs/teman-berbahasa-ms/internal/db/query"
)

type pgScheduleRepository struct {
	pool *pgxpool.Pool
	q    *dbq.Queries
}

func NewRepository(pool *pgxpool.Pool) *pgScheduleRepository {
	return &pgScheduleRepository{pool: pool, q: dbq.New(pool)}
}

// ── pgtype helpers ────────────────────────────────────────────────────────────

func toPgtypeDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func toPgtypeDatePtr(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func fromPgtypeDate(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}

func fromPgtypeDatePtr(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

func toPgtypeTime(s string) pgtype.Time {
	for _, layout := range []string{"15:04:05", "15:04"} {
		t, err := time.Parse(layout, s)
		if err == nil {
			us := int64(t.Hour())*3600*1e6 + int64(t.Minute())*60*1e6 + int64(t.Second())*1e6
			return pgtype.Time{Microseconds: us, Valid: true}
		}
	}
	return pgtype.Time{}
}

func toPgtypeTimePtr(s *string) pgtype.Time {
	if s == nil {
		return pgtype.Time{}
	}
	return toPgtypeTime(*s)
}

func fromPgtypeTime(t pgtype.Time) string {
	if !t.Valid {
		return ""
	}
	d := time.Duration(t.Microseconds) * time.Microsecond
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	sec := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, sec)
}

func fromPgtypeTimePtr(t pgtype.Time) *string {
	if !t.Valid {
		return nil
	}
	s := fromPgtypeTime(t)
	return &s
}

func toPgtypeUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func fromPgtypeUUID(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}

// ── Schedule CRUD ────────────────────────────────────────────────────────────

func (r *pgScheduleRepository) GetBatchForSchedule(ctx context.Context, batchID uuid.UUID) (BatchForSchedule, error) {
	row, err := r.q.GetBatchForSchedule(ctx, batchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return BatchForSchedule{}, ErrBatchNotFound
	}
	if err != nil {
		return BatchForSchedule{}, err
	}
	return BatchForSchedule{
		ID:               row.ID,
		CourseID:         row.CourseID,
		InstructorUserID: row.InstructorUserID,
	}, nil
}

func (r *pgScheduleRepository) GetCourseSessionCount(ctx context.Context, courseID uuid.UUID) (int32, error) {
	count, err := r.q.GetCourseSessionCount(ctx, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return count, err
}

func (r *pgScheduleRepository) ListSchedulesByBatchForCount(ctx context.Context, batchID uuid.UUID) ([]ScheduleForCount, error) {
	rows, err := r.q.ListSchedulesByBatchForCount(ctx, batchID)
	if err != nil {
		return nil, err
	}
	result := make([]ScheduleForCount, len(rows))
	for i, row := range rows {
		result[i] = ScheduleForCount{
			ID:             row.ID,
			Recurrence:     row.Recurrence,
			DayOfWeek:      row.DayOfWeek,
			EffectiveFrom:  fromPgtypeDate(row.EffectiveFrom),
			EffectiveUntil: fromPgtypeDatePtr(row.EffectiveUntil),
		}
	}
	return result, nil
}

func (r *pgScheduleRepository) CreateSchedule(ctx context.Context, id uuid.UUID, req CreateScheduleRequest) (Schedule, error) {
	row, err := r.q.CreateSchedule(ctx, dbq.CreateScheduleParams{
		ID:               id,
		BatchID:          req.BatchID,
		CourseID:         req.CourseID,
		InstructorUserID: toPgtypeUUID(req.InstructorUserID),
		DayOfWeek:        req.DayOfWeek,
		StartTime:        toPgtypeTime(req.StartTime),
		EndTime:          toPgtypeTime(req.EndTime),
		Room:             req.Room,
		Recurrence:       req.Recurrence,
		EffectiveFrom:    toPgtypeDate(req.EffectiveFrom),
		EffectiveUntil:   toPgtypeDatePtr(req.EffectiveUntil),
	})
	if err != nil {
		return Schedule{}, err
	}
	return rowToSchedule(row), nil
}

func rowToSchedule(row dbq.Schedule) Schedule {
	return Schedule{
		ID:               row.ID,
		BatchID:          row.BatchID,
		CourseID:         row.CourseID,
		InstructorUserID: fromPgtypeUUID(row.InstructorUserID),
		DayOfWeek:        row.DayOfWeek,
		StartTime:        fromPgtypeTime(row.StartTime),
		EndTime:          fromPgtypeTime(row.EndTime),
		Room:             row.Room,
		Recurrence:       row.Recurrence,
		EffectiveFrom:    fromPgtypeDate(row.EffectiveFrom),
		EffectiveUntil:   fromPgtypeDatePtr(row.EffectiveUntil),
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}
}

func (r *pgScheduleRepository) GetScheduleByID(ctx context.Context, id uuid.UUID) (Schedule, error) {
	row, err := r.q.GetScheduleByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Schedule{}, ErrScheduleNotFound
	}
	if err != nil {
		return Schedule{}, err
	}
	return rowToSchedule(row), nil
}

func (r *pgScheduleRepository) ListSchedulesByBatch(ctx context.Context, batchID uuid.UUID) ([]Schedule, error) {
	rows, err := r.q.ListSchedulesByBatch(ctx, batchID)
	if err != nil {
		return nil, err
	}
	result := make([]Schedule, len(rows))
	for i, row := range rows {
		result[i] = rowToSchedule(row)
	}
	return result, nil
}

func (r *pgScheduleRepository) UpdateSchedule(ctx context.Context, id uuid.UUID, full FullScheduleUpdate) (Schedule, error) {
	row, err := r.q.UpdateSchedule(ctx, dbq.UpdateScheduleParams{
		ID:               id,
		InstructorUserID: toPgtypeUUID(full.InstructorUserID),
		DayOfWeek:        full.DayOfWeek,
		StartTime:        toPgtypeTime(full.StartTime),
		EndTime:          toPgtypeTime(full.EndTime),
		Room:             full.Room,
		Recurrence:       full.Recurrence,
		EffectiveFrom:    toPgtypeDate(full.EffectiveFrom),
		EffectiveUntil:   toPgtypeDatePtr(full.EffectiveUntil),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Schedule{}, ErrScheduleNotFound
	}
	if err != nil {
		return Schedule{}, err
	}
	return rowToSchedule(row), nil
}

func (r *pgScheduleRepository) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	err := r.q.DeleteSchedule(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrScheduleNotFound
	}
	return err
}

// ── Weekly calendar ───────────────────────────────────────────────────────────

func (r *pgScheduleRepository) GetSchedulesForWeek(ctx context.Context, params WeeklyParams) ([]ScheduleForWeek, error) {
	weekEnd := params.WeekStart.AddDate(0, 0, 6)
	rows, err := r.q.ListSchedulesForWeek(ctx, dbq.ListSchedulesForWeekParams{
		WeekEnd:   toPgtypeDate(weekEnd),
		WeekStart: toPgtypeDate(params.WeekStart),
		CourseID:  toPgtypeUUID(params.CourseID),
		BatchID:   toPgtypeUUID(params.BatchID),
		Level:     params.Level,
	})
	if err != nil {
		return nil, err
	}
	result := make([]ScheduleForWeek, len(rows))
	for i, row := range rows {
		result[i] = listWeekRowToScheduleForWeek(row)
	}
	return result, nil
}

func listWeekRowToScheduleForWeek(row dbq.ListSchedulesForWeekRow) ScheduleForWeek {
	return ScheduleForWeek{
		ID:                       row.ID,
		BatchID:                  row.BatchID,
		CourseID:                 row.CourseID,
		ScheduleInstructorID:     fromPgtypeUUID(row.InstructorUserID),
		DayOfWeek:                row.DayOfWeek,
		StartTime:                fromPgtypeTime(row.StartTime),
		EndTime:                  fromPgtypeTime(row.EndTime),
		Room:                     row.Room,
		Recurrence:               row.Recurrence,
		EffectiveFrom:            fromPgtypeDate(row.EffectiveFrom),
		EffectiveUntil:           fromPgtypeDatePtr(row.EffectiveUntil),
		BatchName:                row.BatchName,
		BatchCode:                row.BatchCode,
		BatchInstructorID:        row.BatchInstructorID,
		BatchInstructorFirstName: row.BatchInstructorFirstName,
		BatchInstructorLastName:  row.BatchInstructorLastName,
		SchedInstructorFirstName: row.SchedInstructorFirstName,
		SchedInstructorLastName:  row.SchedInstructorLastName,
		CourseName:               row.CourseName,
		CourseCode:               row.CourseCode,
		CourseLevel:              row.CourseLevel,
	}
}

func (r *pgScheduleRepository) GetOverridesForWeek(ctx context.Context, scheduleIDs []uuid.UUID, weekStart, weekEnd time.Time) ([]OverrideForWeek, error) {
	rows, err := r.q.GetOverridesByScheduleIDs(ctx, dbq.GetOverridesByScheduleIDsParams{
		ScheduleIds: scheduleIDs,
		WeekStart:   toPgtypeDate(weekStart),
		WeekEnd:     toPgtypeDate(weekEnd),
	})
	if err != nil {
		return nil, err
	}
	result := make([]OverrideForWeek, len(rows))
	for i, row := range rows {
		result[i] = overrideWeekRowToOverrideForWeek(row)
	}
	return result, nil
}

func overrideWeekRowToOverrideForWeek(row dbq.GetOverridesByScheduleIDsRow) OverrideForWeek {
	return OverrideForWeek{
		ID:                     row.ID,
		ScheduleID:             row.ScheduleID,
		OriginalDate:           fromPgtypeDate(row.OriginalDate),
		OverrideType:           row.OverrideType,
		NewDate:                fromPgtypeDatePtr(row.NewDate),
		NewStartTime:           fromPgtypeTimePtr(row.NewStartTime),
		NewEndTime:             fromPgtypeTimePtr(row.NewEndTime),
		NewRoom:                row.NewRoom,
		NewInstructorID:        fromPgtypeUUID(row.NewInstructorUserID),
		NewInstructorFirstName: row.NewInstructorFirstName,
		NewInstructorLastName:  row.NewInstructorLastName,
		Reason:                 row.Reason,
		CreatedByUserID:        row.CreatedByUserID,
		CreatedAt:              row.CreatedAt.Time,
	}
}

// ── Override CRUD ─────────────────────────────────────────────────────────────

func (r *pgScheduleRepository) UpsertOverride(ctx context.Context, id uuid.UUID, req CreateOverrideRequest) (ScheduleOverride, error) {
	row, err := r.q.UpsertScheduleOverride(ctx, dbq.UpsertScheduleOverrideParams{
		ID:                  id,
		ScheduleID:          req.ScheduleID,
		OriginalDate:        toPgtypeDate(req.OriginalDate),
		OverrideType:        req.OverrideType,
		NewDate:             toPgtypeDatePtr(req.NewDate),
		NewStartTime:        toPgtypeTimePtr(req.NewStartTime),
		NewEndTime:          toPgtypeTimePtr(req.NewEndTime),
		NewRoom:             req.NewRoom,
		NewInstructorUserID: toPgtypeUUID(req.NewInstructorUserID),
		Reason:              req.Reason,
		CreatedByUserID:     req.CreatedByUserID,
	})
	if err != nil {
		return ScheduleOverride{}, err
	}
	return rowToOverride(row), nil
}

func rowToOverride(row dbq.ScheduleOverride) ScheduleOverride {
	return ScheduleOverride{
		ID:                  row.ID,
		ScheduleID:          row.ScheduleID,
		OriginalDate:        fromPgtypeDate(row.OriginalDate),
		OverrideType:        row.OverrideType,
		NewDate:             fromPgtypeDatePtr(row.NewDate),
		NewStartTime:        fromPgtypeTimePtr(row.NewStartTime),
		NewEndTime:          fromPgtypeTimePtr(row.NewEndTime),
		NewRoom:             row.NewRoom,
		NewInstructorUserID: fromPgtypeUUID(row.NewInstructorUserID),
		Reason:              row.Reason,
		CreatedByUserID:     row.CreatedByUserID,
		CreatedAt:           row.CreatedAt.Time,
	}
}

func (r *pgScheduleRepository) GetOverrideByID(ctx context.Context, id uuid.UUID) (ScheduleOverride, error) {
	row, err := r.q.GetOverrideByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ScheduleOverride{}, ErrOverrideNotFound
	}
	if err != nil {
		return ScheduleOverride{}, err
	}
	o := rowToOverride(dbq.ScheduleOverride{
		ID:                  row.ID,
		ScheduleID:          row.ScheduleID,
		OriginalDate:        row.OriginalDate,
		OverrideType:        row.OverrideType,
		NewDate:             row.NewDate,
		NewStartTime:        row.NewStartTime,
		NewEndTime:          row.NewEndTime,
		NewRoom:             row.NewRoom,
		NewInstructorUserID: row.NewInstructorUserID,
		Reason:              row.Reason,
		CreatedByUserID:     row.CreatedByUserID,
		CreatedAt:           row.CreatedAt,
	})
	o.NewInstructorFirstName = row.NewInstructorFirstName
	o.NewInstructorLastName = row.NewInstructorLastName
	return o, nil
}

func (r *pgScheduleRepository) UpdateOverride(ctx context.Context, id uuid.UUID, req UpdateOverrideRequest) (ScheduleOverride, error) {
	row, err := r.q.UpdateScheduleOverride(ctx, dbq.UpdateScheduleOverrideParams{
		ID:                  id,
		NewDate:             toPgtypeDatePtr(req.NewDate),
		NewStartTime:        toPgtypeTimePtr(req.NewStartTime),
		NewEndTime:          toPgtypeTimePtr(req.NewEndTime),
		NewRoom:             req.NewRoom,
		NewInstructorUserID: toPgtypeUUID(req.NewInstructorUserID),
		Reason:              req.Reason,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ScheduleOverride{}, ErrOverrideNotFound
	}
	if err != nil {
		return ScheduleOverride{}, err
	}
	return rowToOverride(row), nil
}

func (r *pgScheduleRepository) DeleteOverride(ctx context.Context, id uuid.UUID) error {
	err := r.q.DeleteScheduleOverride(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOverrideNotFound
	}
	return err
}
