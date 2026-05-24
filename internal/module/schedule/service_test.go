package schedule_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/schedule"
)

// ── mock ──────────────────────────────────────────────────────────────────────

type mockRepo struct {
	getBatchFn              func(ctx context.Context, id uuid.UUID) (schedule.BatchForSchedule, error)
	getCourseSessionCountFn func(ctx context.Context, id uuid.UUID) (int32, error)
	listForCountFn          func(ctx context.Context, batchID uuid.UUID) ([]schedule.ScheduleForCount, error)
	createScheduleFn        func(ctx context.Context, id uuid.UUID, req schedule.CreateScheduleRequest) (schedule.Schedule, error)
	getScheduleByIDFn       func(ctx context.Context, id uuid.UUID) (schedule.Schedule, error)
	listByBatchFn           func(ctx context.Context, batchID uuid.UUID) ([]schedule.Schedule, error)
	updateScheduleFn        func(ctx context.Context, id uuid.UUID, full schedule.FullScheduleUpdate) (schedule.Schedule, error)
	deleteScheduleFn        func(ctx context.Context, id uuid.UUID) error
	getSchedulesForWeekFn   func(ctx context.Context, params schedule.WeeklyParams) ([]schedule.ScheduleForWeek, error)
	getOverridesForWeekFn   func(ctx context.Context, ids []uuid.UUID, ws, we time.Time) ([]schedule.OverrideForWeek, error)
	upsertOverrideFn        func(ctx context.Context, id uuid.UUID, req schedule.CreateOverrideRequest) (schedule.ScheduleOverride, error)
	getOverrideByIDFn       func(ctx context.Context, id uuid.UUID) (schedule.ScheduleOverride, error)
	updateOverrideFn        func(ctx context.Context, id uuid.UUID, req schedule.UpdateOverrideRequest) (schedule.ScheduleOverride, error)
	deleteOverrideFn        func(ctx context.Context, id uuid.UUID) error
}

func (m *mockRepo) GetBatchForSchedule(ctx context.Context, id uuid.UUID) (schedule.BatchForSchedule, error) {
	if m.getBatchFn != nil {
		return m.getBatchFn(ctx, id)
	}
	return schedule.BatchForSchedule{}, schedule.ErrBatchNotFound
}
func (m *mockRepo) GetCourseSessionCount(ctx context.Context, id uuid.UUID) (int32, error) {
	if m.getCourseSessionCountFn != nil {
		return m.getCourseSessionCountFn(ctx, id)
	}
	return 30, nil
}
func (m *mockRepo) ListSchedulesByBatchForCount(ctx context.Context, batchID uuid.UUID) ([]schedule.ScheduleForCount, error) {
	if m.listForCountFn != nil {
		return m.listForCountFn(ctx, batchID)
	}
	return nil, nil
}
func (m *mockRepo) CreateSchedule(ctx context.Context, id uuid.UUID, req schedule.CreateScheduleRequest) (schedule.Schedule, error) {
	if m.createScheduleFn != nil {
		return m.createScheduleFn(ctx, id, req)
	}
	return schedule.Schedule{ID: id}, nil
}
func (m *mockRepo) GetScheduleByID(ctx context.Context, id uuid.UUID) (schedule.Schedule, error) {
	if m.getScheduleByIDFn != nil {
		return m.getScheduleByIDFn(ctx, id)
	}
	return schedule.Schedule{}, schedule.ErrScheduleNotFound
}
func (m *mockRepo) ListSchedulesByBatch(ctx context.Context, batchID uuid.UUID) ([]schedule.Schedule, error) {
	if m.listByBatchFn != nil {
		return m.listByBatchFn(ctx, batchID)
	}
	return nil, nil
}
func (m *mockRepo) UpdateSchedule(ctx context.Context, id uuid.UUID, full schedule.FullScheduleUpdate) (schedule.Schedule, error) {
	if m.updateScheduleFn != nil {
		return m.updateScheduleFn(ctx, id, full)
	}
	return schedule.Schedule{}, nil
}
func (m *mockRepo) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	if m.deleteScheduleFn != nil {
		return m.deleteScheduleFn(ctx, id)
	}
	return nil
}
func (m *mockRepo) GetSchedulesForWeek(ctx context.Context, params schedule.WeeklyParams) ([]schedule.ScheduleForWeek, error) {
	if m.getSchedulesForWeekFn != nil {
		return m.getSchedulesForWeekFn(ctx, params)
	}
	return nil, nil
}
func (m *mockRepo) GetOverridesForWeek(ctx context.Context, ids []uuid.UUID, ws, we time.Time) ([]schedule.OverrideForWeek, error) {
	if m.getOverridesForWeekFn != nil {
		return m.getOverridesForWeekFn(ctx, ids, ws, we)
	}
	return nil, nil
}
func (m *mockRepo) UpsertOverride(ctx context.Context, id uuid.UUID, req schedule.CreateOverrideRequest) (schedule.ScheduleOverride, error) {
	if m.upsertOverrideFn != nil {
		return m.upsertOverrideFn(ctx, id, req)
	}
	return schedule.ScheduleOverride{ID: id}, nil
}
func (m *mockRepo) GetOverrideByID(ctx context.Context, id uuid.UUID) (schedule.ScheduleOverride, error) {
	if m.getOverrideByIDFn != nil {
		return m.getOverrideByIDFn(ctx, id)
	}
	return schedule.ScheduleOverride{}, schedule.ErrOverrideNotFound
}
func (m *mockRepo) UpdateOverride(ctx context.Context, id uuid.UUID, req schedule.UpdateOverrideRequest) (schedule.ScheduleOverride, error) {
	if m.updateOverrideFn != nil {
		return m.updateOverrideFn(ctx, id, req)
	}
	return schedule.ScheduleOverride{}, nil
}
func (m *mockRepo) DeleteOverride(ctx context.Context, id uuid.UUID) error {
	if m.deleteOverrideFn != nil {
		return m.deleteOverrideFn(ctx, id)
	}
	return nil
}

func newSvc(r schedule.ScheduleRepository) *schedule.ScheduleService {
	return schedule.NewService(r)
}

var (
	batchID  = uuid.MustParse("019687a2-0000-7000-8000-000000000001")
	courseID = uuid.MustParse("019687a2-0000-7000-8000-000000000002")
	instrID  = uuid.MustParse("019687a2-0000-7000-8000-000000000003")
	schedID  = uuid.MustParse("019687a2-0000-7000-8000-000000000004")
	overrID  = uuid.MustParse("019687a2-0000-7000-8000-000000000005")
	userID   = uuid.MustParse("019687a2-0000-7000-8000-000000000006")
)

func fixedBatch() schedule.BatchForSchedule {
	return schedule.BatchForSchedule{ID: batchID, CourseID: courseID, InstructorUserID: instrID}
}

func fixedSchedule() schedule.Schedule {
	return schedule.Schedule{
		ID:            schedID,
		BatchID:       batchID,
		CourseID:      courseID,
		DayOfWeek:     "friday",
		StartTime:     "09:00:00",
		EndTime:       "11:00:00",
		Recurrence:    "weekly",
		EffectiveFrom: mustDate("2026-03-03"),
		EffectiveUntil: timePtr(mustDate("2026-06-27")),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func baseCreateReq() schedule.CreateScheduleRequest {
	return schedule.CreateScheduleRequest{
		BatchID:         batchID,
		DayOfWeek:       "friday",
		StartTime:       "09:00:00",
		EndTime:         "11:00:00",
		Recurrence:      "weekly",
		EffectiveFrom:   mustDate("2026-03-03"),
		EffectiveUntil:  timePtr(mustDate("2026-06-27")),
		CreatedByUserID: userID,
	}
}

// ── CreateSchedule ────────────────────────────────────────────────────────────

func TestCreateSchedule_HappyPath(t *testing.T) {
	svc := newSvc(&mockRepo{
		getBatchFn: func(_ context.Context, _ uuid.UUID) (schedule.BatchForSchedule, error) {
			return fixedBatch(), nil
		},
		getCourseSessionCountFn: func(_ context.Context, _ uuid.UUID) (int32, error) {
			return 30, nil
		},
		listForCountFn: func(_ context.Context, _ uuid.UUID) ([]schedule.ScheduleForCount, error) {
			return nil, nil // no existing sessions
		},
		createScheduleFn: func(_ context.Context, id uuid.UUID, _ schedule.CreateScheduleRequest) (schedule.Schedule, error) {
			s := fixedSchedule()
			s.ID = id
			return s, nil
		},
	})
	got, err := svc.CreateSchedule(context.Background(), baseCreateReq())
	require.NoError(t, err)
	assert.Equal(t, courseID, got.CourseID) // course_id derived from batch
}

func TestCreateSchedule_BatchNotFound(t *testing.T) {
	svc := newSvc(&mockRepo{})
	_, err := svc.CreateSchedule(context.Background(), baseCreateReq())
	assert.ErrorIs(t, err, schedule.ErrBatchNotFound)
}

func TestCreateSchedule_SessionCapExceeded(t *testing.T) {
	svc := newSvc(&mockRepo{
		getBatchFn: func(_ context.Context, _ uuid.UUID) (schedule.BatchForSchedule, error) {
			return fixedBatch(), nil
		},
		getCourseSessionCountFn: func(_ context.Context, _ uuid.UUID) (int32, error) {
			return 10, nil
		},
		listForCountFn: func(_ context.Context, _ uuid.UUID) ([]schedule.ScheduleForCount, error) {
			return nil, nil
		},
	})
	req := baseCreateReq()
	req.EffectiveFrom = mustDate("2026-03-06")
	req.EffectiveUntil = timePtr(mustDate("2026-07-03"))
	_, err := svc.CreateSchedule(context.Background(), req)
	assert.ErrorIs(t, err, schedule.ErrSessionCapExceeded)
}

func TestCreateSchedule_OneTime_SetsEffectiveUntil(t *testing.T) {
	var captured schedule.CreateScheduleRequest
	svc := newSvc(&mockRepo{
		getBatchFn: func(_ context.Context, _ uuid.UUID) (schedule.BatchForSchedule, error) {
			return fixedBatch(), nil
		},
		getCourseSessionCountFn: func(_ context.Context, _ uuid.UUID) (int32, error) { return 30, nil },
		listForCountFn:          func(_ context.Context, _ uuid.UUID) ([]schedule.ScheduleForCount, error) { return nil, nil },
		createScheduleFn: func(_ context.Context, id uuid.UUID, req schedule.CreateScheduleRequest) (schedule.Schedule, error) {
			captured = req
			s := fixedSchedule()
			s.ID = id
			return s, nil
		},
	})
	req := baseCreateReq()
	req.Recurrence = "one-time"
	req.EffectiveFrom = mustDate("2026-05-22")
	req.EffectiveUntil = nil
	_, err := svc.CreateSchedule(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, captured.EffectiveUntil)
	assert.Equal(t, mustDate("2026-05-22"), *captured.EffectiveUntil)
}

func TestCreateSchedule_SkipCapCheck_WhenOpenEnded(t *testing.T) {
	svc := newSvc(&mockRepo{
		getBatchFn: func(_ context.Context, _ uuid.UUID) (schedule.BatchForSchedule, error) {
			return fixedBatch(), nil
		},
		getCourseSessionCountFn: func(_ context.Context, _ uuid.UUID) (int32, error) { return 0, nil },
		listForCountFn:          func(_ context.Context, _ uuid.UUID) ([]schedule.ScheduleForCount, error) { return nil, nil },
		createScheduleFn: func(_ context.Context, id uuid.UUID, _ schedule.CreateScheduleRequest) (schedule.Schedule, error) {
			s := fixedSchedule()
			s.ID = id
			s.EffectiveUntil = nil
			return s, nil
		},
	})
	req := baseCreateReq()
	req.EffectiveUntil = nil
	_, err := svc.CreateSchedule(context.Background(), req)
	require.NoError(t, err)
}

// ── DeleteSchedule ────────────────────────────────────────────────────────────

func TestDeleteSchedule_HappyPath(t *testing.T) {
	svc := newSvc(&mockRepo{
		deleteScheduleFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	})
	assert.NoError(t, svc.DeleteSchedule(context.Background(), schedID.String()))
}

func TestDeleteSchedule_NotFound(t *testing.T) {
	svc := newSvc(&mockRepo{
		deleteScheduleFn: func(_ context.Context, _ uuid.UUID) error { return schedule.ErrScheduleNotFound },
	})
	assert.ErrorIs(t, svc.DeleteSchedule(context.Background(), schedID.String()), schedule.ErrScheduleNotFound)
}

func TestDeleteSchedule_InvalidUUID(t *testing.T) {
	svc := newSvc(&mockRepo{})
	assert.ErrorIs(t, svc.DeleteSchedule(context.Background(), "bad"), schedule.ErrScheduleNotFound)
}

// ── CreateOverride ────────────────────────────────────────────────────────────

func TestCreateOverride_HappyPath_Reschedule(t *testing.T) {
	svc := newSvc(&mockRepo{
		getScheduleByIDFn: func(_ context.Context, _ uuid.UUID) (schedule.Schedule, error) {
			s := fixedSchedule()
			s.BatchID = batchID
			return s, nil
		},
		getBatchFn: func(_ context.Context, _ uuid.UUID) (schedule.BatchForSchedule, error) {
			return fixedBatch(), nil
		},
		upsertOverrideFn: func(_ context.Context, id uuid.UUID, _ schedule.CreateOverrideRequest) (schedule.ScheduleOverride, error) {
			return schedule.ScheduleOverride{ID: id}, nil
		},
	})
	newDate := mustDate("2026-05-20")
	req := schedule.CreateOverrideRequest{
		ScheduleID:      schedID,
		OriginalDate:    mustDate("2026-05-22"),
		OverrideType:    "reschedule",
		NewDate:         &newDate,
		CreatedByUserID: userID,
	}
	_, err := svc.CreateOverride(context.Background(), req, "admin")
	require.NoError(t, err)
}

func TestCreateOverride_OriginalDateWrongDay_ReturnsError(t *testing.T) {
	svc := newSvc(&mockRepo{
		getScheduleByIDFn: func(_ context.Context, _ uuid.UUID) (schedule.Schedule, error) {
			return fixedSchedule(), nil
		},
		getBatchFn: func(_ context.Context, _ uuid.UUID) (schedule.BatchForSchedule, error) {
			return fixedBatch(), nil
		},
	})
	req := schedule.CreateOverrideRequest{
		ScheduleID:      schedID,
		OriginalDate:    mustDate("2026-05-21"), // thursday, NOT friday
		OverrideType:    "reschedule",
		NewDate:         timePtr(mustDate("2026-05-20")),
		CreatedByUserID: userID,
	}
	_, err := svc.CreateOverride(context.Background(), req, "admin")
	assert.ErrorIs(t, err, schedule.ErrOriginalDateOutOfRange)
}

func TestCreateOverride_RescheduleMissingNewDate(t *testing.T) {
	svc := newSvc(&mockRepo{
		getScheduleByIDFn: func(_ context.Context, _ uuid.UUID) (schedule.Schedule, error) {
			return fixedSchedule(), nil
		},
		getBatchFn: func(_ context.Context, _ uuid.UUID) (schedule.BatchForSchedule, error) {
			return fixedBatch(), nil
		},
	})
	req := schedule.CreateOverrideRequest{
		ScheduleID:      schedID,
		OriginalDate:    mustDate("2026-05-22"),
		OverrideType:    "reschedule",
		NewDate:         nil,
		CreatedByUserID: userID,
	}
	_, err := svc.CreateOverride(context.Background(), req, "admin")
	assert.ErrorIs(t, err, schedule.ErrRescheduleMissingDate)
}

func TestCreateOverride_InstructorChange_WithNewDate_ReturnsError(t *testing.T) {
	svc := newSvc(&mockRepo{
		getScheduleByIDFn: func(_ context.Context, _ uuid.UUID) (schedule.Schedule, error) {
			return fixedSchedule(), nil
		},
		getBatchFn: func(_ context.Context, _ uuid.UUID) (schedule.BatchForSchedule, error) {
			return fixedBatch(), nil
		},
	})
	bad := mustDate("2026-05-20")
	req := schedule.CreateOverrideRequest{
		ScheduleID:      schedID,
		OriginalDate:    mustDate("2026-05-22"),
		OverrideType:    "instructor_change",
		NewDate:         &bad,
		CreatedByUserID: userID,
	}
	_, err := svc.CreateOverride(context.Background(), req, "admin")
	assert.ErrorIs(t, err, schedule.ErrInstructorChangeInvalid)
}

func TestCreateOverride_TeacherForbidden_WrongBatch(t *testing.T) {
	otherInstrID := uuid.New()
	svc := newSvc(&mockRepo{
		getScheduleByIDFn: func(_ context.Context, _ uuid.UUID) (schedule.Schedule, error) {
			return fixedSchedule(), nil
		},
		getBatchFn: func(_ context.Context, _ uuid.UUID) (schedule.BatchForSchedule, error) {
			b := fixedBatch()
			b.InstructorUserID = otherInstrID
			return b, nil
		},
	})
	newDate := mustDate("2026-05-20")
	req := schedule.CreateOverrideRequest{
		ScheduleID:      schedID,
		OriginalDate:    mustDate("2026-05-22"),
		OverrideType:    "reschedule",
		NewDate:         &newDate,
		CreatedByUserID: userID,
	}
	_, err := svc.CreateOverride(context.Background(), req, "teacher")
	assert.ErrorIs(t, err, schedule.ErrForbidden)
}

func TestCreateOverride_TeacherAllowed_OwnBatch(t *testing.T) {
	svc := newSvc(&mockRepo{
		getScheduleByIDFn: func(_ context.Context, _ uuid.UUID) (schedule.Schedule, error) {
			return fixedSchedule(), nil
		},
		getBatchFn: func(_ context.Context, _ uuid.UUID) (schedule.BatchForSchedule, error) {
			b := fixedBatch()
			b.InstructorUserID = userID
			return b, nil
		},
		upsertOverrideFn: func(_ context.Context, id uuid.UUID, _ schedule.CreateOverrideRequest) (schedule.ScheduleOverride, error) {
			return schedule.ScheduleOverride{ID: id}, nil
		},
	})
	newDate := mustDate("2026-05-20")
	req := schedule.CreateOverrideRequest{
		ScheduleID:      schedID,
		OriginalDate:    mustDate("2026-05-22"),
		OverrideType:    "reschedule",
		NewDate:         &newDate,
		CreatedByUserID: userID,
	}
	_, err := svc.CreateOverride(context.Background(), req, "teacher")
	require.NoError(t, err)
}
