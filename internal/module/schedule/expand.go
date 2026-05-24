package schedule

import (
	"sort"
	"time"

	"github.com/google/uuid"
)

var weekdayNames = map[time.Weekday]string{
	time.Monday:    "monday",
	time.Tuesday:   "tuesday",
	time.Wednesday: "wednesday",
	time.Thursday:  "thursday",
	time.Friday:    "friday",
	time.Saturday:  "saturday",
	time.Sunday:    "sunday",
}

func weekdayName(w time.Weekday) string { return weekdayNames[w] }

func inWeek(d, weekStart, weekEnd time.Time) bool {
	return !d.Before(weekStart) && !d.After(weekEnd)
}

// CountExpandedSessions counts how many sessions a schedule slot produces.
// Returns 0 for open-ended slots (effective_until == nil) — skip enforcement.
func CountExpandedSessions(s ScheduleForCount) int {
	if s.EffectiveUntil == nil {
		return 0
	}
	if s.Recurrence == "one-time" {
		return 1
	}
	count := 0
	for d := s.EffectiveFrom; !d.After(*s.EffectiveUntil); d = d.AddDate(0, 0, 1) {
		if weekdayName(d.Weekday()) == s.DayOfWeek {
			count++
		}
	}
	return count
}

type overrideKey struct {
	scheduleID   uuid.UUID
	originalDate time.Time
}

// ExpandWeek expands recurring schedules into concrete sessions for [weekStart, weekEnd].
// weekStart must be a Monday. Overrides alter which sessions appear and how.
func ExpandWeek(schedules []ScheduleForWeek, overrides []OverrideForWeek, weekStart, weekEnd time.Time) []Session {
	overrideMap := make(map[overrideKey]*OverrideForWeek, len(overrides))
	for i := range overrides {
		o := &overrides[i]
		overrideMap[overrideKey{o.ScheduleID, o.OriginalDate}] = o
	}

	type incomingEntry struct {
		override *OverrideForWeek
		sched    *ScheduleForWeek
	}
	schedByID := make(map[uuid.UUID]*ScheduleForWeek, len(schedules))
	for i := range schedules {
		s := &schedules[i]
		schedByID[s.ID] = s
	}
	var incoming []incomingEntry
	for i := range overrides {
		o := &overrides[i]
		if o.OverrideType != "reschedule" || o.NewDate == nil {
			continue
		}
		if inWeek(*o.NewDate, weekStart, weekEnd) && !inWeek(o.OriginalDate, weekStart, weekEnd) {
			if s, ok := schedByID[o.ScheduleID]; ok {
				incoming = append(incoming, incomingEntry{o, s})
			}
		}
	}

	var sessions []Session

	for i := range schedules {
		s := &schedules[i]
		for d := weekStart; !d.After(weekEnd); d = d.AddDate(0, 0, 1) {
			if d.Before(s.EffectiveFrom) {
				continue
			}
			if s.EffectiveUntil != nil && d.After(*s.EffectiveUntil) {
				continue
			}
			if weekdayName(d.Weekday()) != s.DayOfWeek {
				continue
			}
			o := overrideMap[overrideKey{s.ID, d}]
			switch {
			case o == nil:
				sessions = append(sessions, buildSession(s, d, nil, "scheduled"))
			case o.OverrideType == "reschedule" && o.NewDate != nil && inWeek(*o.NewDate, weekStart, weekEnd):
				orig := d
				sess := buildSession(s, *o.NewDate, &orig, "rescheduled")
				sess.Override = toOverrideDomain(o)
				sessions = append(sessions, sess)
			case o.OverrideType == "reschedule":
				continue
			case o.OverrideType == "instructor_change":
				sess := buildSession(s, d, nil, "instructor_changed")
				sess.EffectiveInstructor = resolveOverrideInstructor(s, o)
				sess.Override = toOverrideDomain(o)
				sessions = append(sessions, sess)
			}
		}
	}

	for _, entry := range incoming {
		orig := entry.override.OriginalDate
		sess := buildSession(entry.sched, *entry.override.NewDate, &orig, "rescheduled")
		sess.Override = toOverrideDomain(entry.override)
		sessions = append(sessions, sess)
	}

	sort.Slice(sessions, func(i, j int) bool {
		di, dj := sessions[i].Date, sessions[j].Date
		if di.Equal(dj) {
			return sessions[i].StartTime < sessions[j].StartTime
		}
		return di.Before(dj)
	})

	return sessions
}

func buildSession(s *ScheduleForWeek, date time.Time, originalDate *time.Time, status string) Session {
	return Session{
		ScheduleID:          s.ID,
		Date:                date,
		OriginalDate:        originalDate,
		DayOfWeek:           weekdayName(date.Weekday()),
		StartTime:           s.StartTime,
		EndTime:             s.EndTime,
		Room:                s.Room,
		Status:              status,
		EffectiveInstructor: resolveScheduleInstructor(s),
		Batch: BatchRef{
			ID:        s.BatchID.String(),
			BatchName: s.BatchName,
			BatchCode: s.BatchCode,
		},
		Course: CourseRef{
			ID:         s.CourseID.String(),
			CourseName: s.CourseName,
			CourseCode: s.CourseCode,
			Level:      s.CourseLevel,
		},
	}
}

func resolveScheduleInstructor(s *ScheduleForWeek) InstructorRef {
	if s.ScheduleInstructorID != nil && s.SchedInstructorFirstName != nil {
		return InstructorRef{
			ID:        s.ScheduleInstructorID.String(),
			FirstName: *s.SchedInstructorFirstName,
			LastName:  derefStr(s.SchedInstructorLastName),
		}
	}
	return InstructorRef{
		ID:        s.BatchInstructorID.String(),
		FirstName: s.BatchInstructorFirstName,
		LastName:  s.BatchInstructorLastName,
	}
}

func resolveOverrideInstructor(s *ScheduleForWeek, o *OverrideForWeek) InstructorRef {
	if o.NewInstructorID != nil && o.NewInstructorFirstName != nil {
		return InstructorRef{
			ID:        o.NewInstructorID.String(),
			FirstName: *o.NewInstructorFirstName,
			LastName:  derefStr(o.NewInstructorLastName),
		}
	}
	return resolveScheduleInstructor(s)
}

func toOverrideDomain(o *OverrideForWeek) *ScheduleOverride {
	return &ScheduleOverride{
		ID:                  o.ID,
		ScheduleID:          o.ScheduleID,
		OriginalDate:        o.OriginalDate,
		OverrideType:        o.OverrideType,
		NewDate:             o.NewDate,
		NewStartTime:        o.NewStartTime,
		NewEndTime:          o.NewEndTime,
		NewRoom:             o.NewRoom,
		NewInstructorUserID: o.NewInstructorID,
		Reason:              o.Reason,
		CreatedByUserID:     o.CreatedByUserID,
		CreatedAt:           o.CreatedAt,
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
