package schedule_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/schedule"
)

func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func makeWeekSchedule(dayOfWeek string, from, until string) schedule.ScheduleForWeek {
	var eff *time.Time
	if until != "" {
		t := mustDate(until)
		eff = &t
	}
	instrID := uuid.MustParse("019687a2-0000-7000-8000-000000000001")
	batchInstrID := uuid.MustParse("019687a2-0000-7000-8000-000000000002")
	return schedule.ScheduleForWeek{
		ID:                       uuid.MustParse("019687a2-0000-7000-8000-000000000010"),
		BatchID:                  uuid.MustParse("019687a2-0000-7000-8000-000000000020"),
		CourseID:                 uuid.MustParse("019687a2-0000-7000-8000-000000000030"),
		ScheduleInstructorID:     &instrID,
		DayOfWeek:                dayOfWeek,
		StartTime:                "09:00:00",
		EndTime:                  "11:00:00",
		Recurrence:               "weekly",
		EffectiveFrom:            mustDate(from),
		EffectiveUntil:           eff,
		BatchName:                "N5 Spring",
		BatchCode:                "SP26-A",
		BatchInstructorID:        batchInstrID,
		BatchInstructorFirstName: "Batch",
		BatchInstructorLastName:  "Instructor",
		SchedInstructorFirstName: strPtr("Sched"),
		SchedInstructorLastName:  strPtr("Teacher"),
		CourseName:               "JLPT N5",
		CourseCode:               "JP-N5",
	}
}

func strPtr(s string) *string { return &s }

var weekStart = mustDate("2026-05-18")
var weekEnd = mustDate("2026-05-24")

func TestExpandWeek_NoSchedules_ReturnsEmpty(t *testing.T) {
	sessions := schedule.ExpandWeek(nil, nil, weekStart, weekEnd)
	require.Empty(t, sessions)
}

func TestExpandWeek_WeeklySchedule_EmitsMatchingDay(t *testing.T) {
	s := makeWeekSchedule("friday", "2026-05-01", "2026-06-30")
	sessions := schedule.ExpandWeek([]schedule.ScheduleForWeek{s}, nil, weekStart, weekEnd)
	require.Len(t, sessions, 1)
	assert.Equal(t, mustDate("2026-05-22"), sessions[0].Date)
	assert.Equal(t, "scheduled", sessions[0].Status)
	assert.Equal(t, "friday", sessions[0].DayOfWeek)
	assert.Nil(t, sessions[0].OriginalDate)
	assert.Nil(t, sessions[0].Override)
}

func TestExpandWeek_WeeklySchedule_WrongDay_Emits0(t *testing.T) {
	s2 := makeWeekSchedule("wednesday", "2026-05-25", "2026-06-30")
	sessions := schedule.ExpandWeek([]schedule.ScheduleForWeek{s2}, nil, weekStart, weekEnd)
	require.Empty(t, sessions)
}

func TestExpandWeek_EffectiveUntilBeforeWeek_Emits0(t *testing.T) {
	s := makeWeekSchedule("friday", "2026-04-01", "2026-05-15")
	sessions := schedule.ExpandWeek([]schedule.ScheduleForWeek{s}, nil, weekStart, weekEnd)
	require.Empty(t, sessions)
}

func TestExpandWeek_NilEffectiveUntil_EmitsSession(t *testing.T) {
	s := makeWeekSchedule("friday", "2026-01-01", "")
	sessions := schedule.ExpandWeek([]schedule.ScheduleForWeek{s}, nil, weekStart, weekEnd)
	require.Len(t, sessions, 1)
}

func TestExpandWeek_InstructorFallback_ScheduleLevel(t *testing.T) {
	s := makeWeekSchedule("friday", "2026-05-01", "2026-06-30")
	sessions := schedule.ExpandWeek([]schedule.ScheduleForWeek{s}, nil, weekStart, weekEnd)
	require.Len(t, sessions, 1)
	assert.Equal(t, s.ScheduleInstructorID.String(), sessions[0].EffectiveInstructor.ID)
	assert.Equal(t, "Sched", sessions[0].EffectiveInstructor.FirstName)
}

func TestExpandWeek_InstructorFallback_BatchLevel(t *testing.T) {
	s := makeWeekSchedule("friday", "2026-05-01", "2026-06-30")
	s.ScheduleInstructorID = nil
	s.SchedInstructorFirstName = nil
	s.SchedInstructorLastName = nil
	sessions := schedule.ExpandWeek([]schedule.ScheduleForWeek{s}, nil, weekStart, weekEnd)
	require.Len(t, sessions, 1)
	assert.Equal(t, s.BatchInstructorID.String(), sessions[0].EffectiveInstructor.ID)
	assert.Equal(t, "Batch", sessions[0].EffectiveInstructor.FirstName)
}

func TestExpandWeek_RescheduleNewDateInWeek_EmitsRescheduled(t *testing.T) {
	s := makeWeekSchedule("friday", "2026-05-01", "2026-06-30")
	newDate := mustDate("2026-05-20")
	o := schedule.OverrideForWeek{
		ID:           uuid.New(),
		ScheduleID:   s.ID,
		OriginalDate: mustDate("2026-05-22"),
		OverrideType: "reschedule",
		NewDate:      &newDate,
	}
	sessions := schedule.ExpandWeek([]schedule.ScheduleForWeek{s}, []schedule.OverrideForWeek{o}, weekStart, weekEnd)
	require.Len(t, sessions, 1)
	assert.Equal(t, mustDate("2026-05-20"), sessions[0].Date)
	origDate := mustDate("2026-05-22")
	assert.Equal(t, &origDate, sessions[0].OriginalDate)
	assert.Equal(t, "rescheduled", sessions[0].Status)
	require.NotNil(t, sessions[0].Override)
}

func TestExpandWeek_RescheduleNewDateOutOfWeek_Emits0(t *testing.T) {
	s := makeWeekSchedule("friday", "2026-05-01", "2026-06-30")
	newDate := mustDate("2026-05-25")
	o := schedule.OverrideForWeek{
		ID:           uuid.New(),
		ScheduleID:   s.ID,
		OriginalDate: mustDate("2026-05-22"),
		OverrideType: "reschedule",
		NewDate:      &newDate,
	}
	sessions := schedule.ExpandWeek([]schedule.ScheduleForWeek{s}, []schedule.OverrideForWeek{o}, weekStart, weekEnd)
	require.Empty(t, sessions)
}

func TestExpandWeek_IncomingReschedule_EmitsSession(t *testing.T) {
	s := makeWeekSchedule("friday", "2026-05-01", "2026-06-30")
	newDate := mustDate("2026-05-20")
	o := schedule.OverrideForWeek{
		ID:           uuid.New(),
		ScheduleID:   s.ID,
		OriginalDate: mustDate("2026-05-15"),
		OverrideType: "reschedule",
		NewDate:      &newDate,
	}
	sessions := schedule.ExpandWeek([]schedule.ScheduleForWeek{s}, []schedule.OverrideForWeek{o}, weekStart, weekEnd)
	require.Len(t, sessions, 2)
	dates := []string{sessions[0].Date.Format("2006-01-02"), sessions[1].Date.Format("2006-01-02")}
	assert.Contains(t, dates, "2026-05-20")
	assert.Contains(t, dates, "2026-05-22")
}

func TestExpandWeek_InstructorChange_EmitsInstructorChanged(t *testing.T) {
	s := makeWeekSchedule("friday", "2026-05-01", "2026-06-30")
	newInstrID := uuid.MustParse("019687a2-0000-7000-8000-000000000099")
	o := schedule.OverrideForWeek{
		ID:                     uuid.New(),
		ScheduleID:             s.ID,
		OriginalDate:           mustDate("2026-05-22"),
		OverrideType:           "instructor_change",
		NewInstructorID:        &newInstrID,
		NewInstructorFirstName: strPtr("Guest"),
		NewInstructorLastName:  strPtr("Teacher"),
	}
	sessions := schedule.ExpandWeek([]schedule.ScheduleForWeek{s}, []schedule.OverrideForWeek{o}, weekStart, weekEnd)
	require.Len(t, sessions, 1)
	assert.Equal(t, mustDate("2026-05-22"), sessions[0].Date)
	assert.Equal(t, "instructor_changed", sessions[0].Status)
	assert.Equal(t, newInstrID.String(), sessions[0].EffectiveInstructor.ID)
	assert.Equal(t, "Guest", sessions[0].EffectiveInstructor.FirstName)
}

func TestExpandWeek_MultipleSchedules_SortedByDateThenTime(t *testing.T) {
	monSched := makeWeekSchedule("monday", "2026-05-01", "2026-06-30")
	monSched.StartTime = "14:00:00"
	friSched := makeWeekSchedule("friday", "2026-05-01", "2026-06-30")
	friSched.ID = uuid.MustParse("019687a2-0000-7000-8000-000000000011")
	sessions := schedule.ExpandWeek(
		[]schedule.ScheduleForWeek{friSched, monSched},
		nil, weekStart, weekEnd,
	)
	require.Len(t, sessions, 2)
	assert.True(t, sessions[0].Date.Before(sessions[1].Date), "monday before friday")
}

func TestCountExpandedSessions_Weekly(t *testing.T) {
	s := schedule.ScheduleForCount{
		Recurrence:     "weekly",
		DayOfWeek:      "monday",
		EffectiveFrom:  mustDate("2026-03-02"),
		EffectiveUntil: timePtr(mustDate("2026-03-30")),
	}
	assert.Equal(t, 5, schedule.CountExpandedSessions(s))
}

func TestCountExpandedSessions_OneTime(t *testing.T) {
	s := schedule.ScheduleForCount{
		Recurrence:     "one-time",
		DayOfWeek:      "tuesday",
		EffectiveFrom:  mustDate("2026-05-19"),
		EffectiveUntil: timePtr(mustDate("2026-05-19")),
	}
	assert.Equal(t, 1, schedule.CountExpandedSessions(s))
}

func TestCountExpandedSessions_NilUntil_ReturnsZero(t *testing.T) {
	s := schedule.ScheduleForCount{
		Recurrence:    "weekly",
		DayOfWeek:     "monday",
		EffectiveFrom: mustDate("2026-01-01"),
	}
	assert.Equal(t, 0, schedule.CountExpandedSessions(s))
}

func timePtr(t time.Time) *time.Time { return &t }
