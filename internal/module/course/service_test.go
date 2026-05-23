package course_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/course"
)

// ── mock ──────────────────────────────────────────────────────────────────────

type mockRepo struct {
	listFn    func(ctx context.Context, params course.ListParams) ([]course.CourseWithStats, int64, error)
	getByIDFn func(ctx context.Context, id uuid.UUID) (course.CourseWithStats, error)
	createFn  func(ctx context.Context, id uuid.UUID, req course.CreateCourseRequest) (course.Course, error)
	updateFullFn func(ctx context.Context, id uuid.UUID, req course.FullCourseUpdate) (course.Course, error)
	archiveFn func(ctx context.Context, id uuid.UUID) (course.ArchiveResult, error)
}

func (m *mockRepo) List(ctx context.Context, params course.ListParams) ([]course.CourseWithStats, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return nil, 0, nil
}
func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (course.CourseWithStats, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return course.CourseWithStats{}, pgx.ErrNoRows
}
func (m *mockRepo) Create(ctx context.Context, id uuid.UUID, req course.CreateCourseRequest) (course.Course, error) {
	if m.createFn != nil {
		return m.createFn(ctx, id, req)
	}
	return course.Course{}, nil
}
func (m *mockRepo) UpdateFull(ctx context.Context, id uuid.UUID, req course.FullCourseUpdate) (course.Course, error) {
	if m.updateFullFn != nil {
		return m.updateFullFn(ctx, id, req)
	}
	return course.Course{}, nil
}
func (m *mockRepo) Archive(ctx context.Context, id uuid.UUID) (course.ArchiveResult, error) {
	if m.archiveFn != nil {
		return m.archiveFn(ctx, id)
	}
	return course.ArchiveResult{}, nil
}

func newSvc(repo course.CourseRepository) *course.CourseService {
	return course.NewService(repo)
}

func fixedID() uuid.UUID {
	id, _ := uuid.Parse("019687a2-0002-7000-8000-000000000001")
	return id
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestList_ReturnsPaginatedResults(t *testing.T) {
	svc := newSvc(&mockRepo{
		listFn: func(_ context.Context, _ course.ListParams) ([]course.CourseWithStats, int64, error) {
			return []course.CourseWithStats{
				{Course: course.Course{ID: fixedID(), CourseName: "JLPT N5", CourseCode: "JP-N5", Status: "active"}},
			}, 1, nil
		},
	})

	resp, err := svc.List(context.Background(), course.ListParams{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, 1, resp.Pagination.Total)
	assert.Equal(t, 1, resp.Pagination.TotalPages)
}

func TestList_PassesThroughPaginationParams(t *testing.T) {
	var capturedParams course.ListParams
	svc := newSvc(&mockRepo{
		listFn: func(_ context.Context, params course.ListParams) ([]course.CourseWithStats, int64, error) {
			capturedParams = params
			return nil, 0, nil
		},
	})

	_, err := svc.List(context.Background(), course.ListParams{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, capturedParams.Page)
	assert.Equal(t, 20, capturedParams.PerPage)
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestGetByID_ReturnsCourseWithStats(t *testing.T) {
	expected := course.CourseWithStats{
		Course:            course.Course{ID: fixedID(), CourseName: "JLPT N5", Status: "active"},
		BatchCount:        3,
		OngoingBatchCount: 2,
		EnrolledCount:     14,
	}
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (course.CourseWithStats, error) {
			return expected, nil
		},
	})

	got, err := svc.GetByID(context.Background(), fixedID().String())
	require.NoError(t, err)
	assert.Equal(t, expected.CourseName, got.CourseName)
	assert.Equal(t, int64(3), got.BatchCount)
	assert.Equal(t, int64(14), got.EnrolledCount)
}

func TestGetByID_NotFound(t *testing.T) {
	svc := newSvc(&mockRepo{})

	_, err := svc.GetByID(context.Background(), fixedID().String())
	assert.ErrorIs(t, err, course.ErrNotFound)
}

func TestGetByID_InvalidUUID(t *testing.T) {
	svc := newSvc(&mockRepo{})

	_, err := svc.GetByID(context.Background(), "not-a-uuid")
	assert.ErrorIs(t, err, course.ErrNotFound)
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestCreate_HappyPath(t *testing.T) {
	svc := newSvc(&mockRepo{
		createFn: func(_ context.Context, id uuid.UUID, req course.CreateCourseRequest) (course.Course, error) {
			return course.Course{
				ID:         id,
				CourseName: req.CourseName,
				CourseCode: req.CourseCode,
				Status:     "active",
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}, nil
		},
	})

	c, err := svc.Create(context.Background(), course.CreateCourseRequest{
		CourseName: "JLPT N5 Foundations",
		CourseCode: "JP-N5",
	})
	require.NoError(t, err)
	assert.Equal(t, "JLPT N5 Foundations", c.CourseName)
	assert.Equal(t, "active", c.Status)
}

func TestCreate_DuplicateCourseCode(t *testing.T) {
	svc := newSvc(&mockRepo{
		createFn: func(_ context.Context, _ uuid.UUID, _ course.CreateCourseRequest) (course.Course, error) {
			return course.Course{}, course.ErrCourseCodeConflict
		},
	})

	_, err := svc.Create(context.Background(), course.CreateCourseRequest{
		CourseName: "Dup",
		CourseCode: "DUP",
	})
	assert.ErrorIs(t, err, course.ErrCourseCodeConflict)
}

func TestCreate_TrimsCourseCode(t *testing.T) {
	var capturedReq course.CreateCourseRequest
	svc := newSvc(&mockRepo{
		createFn: func(_ context.Context, _ uuid.UUID, req course.CreateCourseRequest) (course.Course, error) {
			capturedReq = req
			return course.Course{CourseName: req.CourseName, CourseCode: req.CourseCode, Status: "active"}, nil
		},
	})

	_, err := svc.Create(context.Background(), course.CreateCourseRequest{
		CourseName: "  JLPT N5  ",
		CourseCode: "  JP-N5  ",
	})
	require.NoError(t, err)
	assert.Equal(t, "JLPT N5", capturedReq.CourseName)
	assert.Equal(t, "JP-N5", capturedReq.CourseCode)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestUpdate_PartialUpdate(t *testing.T) {
	newName := "Updated Name"
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (course.CourseWithStats, error) {
			return course.CourseWithStats{Course: course.Course{ID: fixedID(), Status: "active"}}, nil
		},
		updateFullFn: func(_ context.Context, _ uuid.UUID, req course.FullCourseUpdate) (course.Course, error) {
			return course.Course{ID: fixedID(), CourseName: req.CourseName, Status: "active"}, nil
		},
	})

	got, err := svc.Update(context.Background(), fixedID().String(), course.UpdateCourseRequest{
		CourseName: &newName,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", got.CourseName)
}

func TestUpdate_NotFound(t *testing.T) {
	svc := newSvc(&mockRepo{})

	_, err := svc.Update(context.Background(), fixedID().String(), course.UpdateCourseRequest{})
	assert.ErrorIs(t, err, course.ErrNotFound)
}

func TestUpdate_DuplicateCourseCode(t *testing.T) {
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (course.CourseWithStats, error) {
			return course.CourseWithStats{Course: course.Course{ID: fixedID(), Status: "active"}}, nil
		},
		updateFullFn: func(_ context.Context, _ uuid.UUID, _ course.FullCourseUpdate) (course.Course, error) {
			return course.Course{}, course.ErrCourseCodeConflict
		},
	})

	code := "EXISTS"
	_, err := svc.Update(context.Background(), fixedID().String(), course.UpdateCourseRequest{CourseCode: &code})
	assert.ErrorIs(t, err, course.ErrCourseCodeConflict)
}

func TestUpdate_InvalidUUID(t *testing.T) {
	svc := newSvc(&mockRepo{})

	_, err := svc.Update(context.Background(), "not-a-uuid", course.UpdateCourseRequest{})
	assert.ErrorIs(t, err, course.ErrNotFound)
}

func TestUpdate_TrimsCourseCode(t *testing.T) {
	var capturedReq course.FullCourseUpdate
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (course.CourseWithStats, error) {
			return course.CourseWithStats{Course: course.Course{ID: fixedID(), Status: "active"}}, nil
		},
		updateFullFn: func(_ context.Context, _ uuid.UUID, req course.FullCourseUpdate) (course.Course, error) {
			capturedReq = req
			return course.Course{ID: fixedID(), Status: "active"}, nil
		},
	})

	name := "  Updated  "
	code := "  JP-N5  "
	_, err := svc.Update(context.Background(), fixedID().String(), course.UpdateCourseRequest{
		CourseName: &name,
		CourseCode: &code,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated", capturedReq.CourseName)
	assert.Equal(t, "JP-N5", capturedReq.CourseCode)
}

// ── Archive ───────────────────────────────────────────────────────────────────

func TestArchive_HappyPath(t *testing.T) {
	svc := newSvc(&mockRepo{
		archiveFn: func(_ context.Context, id uuid.UUID) (course.ArchiveResult, error) {
			return course.ArchiveResult{ID: id, Status: "archived", UpdatedAt: time.Now()}, nil
		},
	})

	result, err := svc.Archive(context.Background(), fixedID().String())
	require.NoError(t, err)
	assert.Equal(t, "archived", result.Status)
}

func TestArchive_HasOngoingBatches(t *testing.T) {
	svc := newSvc(&mockRepo{
		archiveFn: func(_ context.Context, _ uuid.UUID) (course.ArchiveResult, error) {
			return course.ArchiveResult{}, pgx.ErrNoRows
		},
	})

	_, err := svc.Archive(context.Background(), fixedID().String())
	assert.ErrorIs(t, err, course.ErrHasOngoingBatches)
}

func TestArchive_NotFound(t *testing.T) {
	svc := newSvc(&mockRepo{
		archiveFn: func(_ context.Context, _ uuid.UUID) (course.ArchiveResult, error) {
			return course.ArchiveResult{}, pgx.ErrNoRows
		},
	})

	_, err := svc.Archive(context.Background(), fixedID().String())
	assert.ErrorIs(t, err, course.ErrHasOngoingBatches)
}
