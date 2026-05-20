package student_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/student"
)

// ── mock ──────────────────────────────────────────────────────────────────────

type mockRepo struct {
	getByIDFn       func(ctx context.Context, id uuid.UUID) (student.Student, error)
	getByEmailFn    func(ctx context.Context, email string) (student.Student, error)
	listFn          func(ctx context.Context, params student.ListParams) ([]student.Student, int64, error)
	getBatchCodesFn func(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]string, error)
	getEnrollsFn    func(ctx context.Context, id uuid.UUID) ([]student.EnrollmentSummary, error)
	createFn        func(ctx context.Context, id uuid.UUID, req student.CreateStudentRequest) (student.Student, error)
	updateFn        func(ctx context.Context, id uuid.UUID, req student.UpdateStudentRequest) (student.Student, error)
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (student.Student, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return student.Student{}, pgx.ErrNoRows
}
func (m *mockRepo) GetByEmail(ctx context.Context, email string) (student.Student, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return student.Student{}, pgx.ErrNoRows
}
func (m *mockRepo) List(ctx context.Context, params student.ListParams) ([]student.Student, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return nil, 0, nil
}
func (m *mockRepo) GetBatchCodes(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]string, error) {
	if m.getBatchCodesFn != nil {
		return m.getBatchCodesFn(ctx, ids)
	}
	return map[uuid.UUID][]string{}, nil
}
func (m *mockRepo) GetEnrollments(ctx context.Context, id uuid.UUID) ([]student.EnrollmentSummary, error) {
	if m.getEnrollsFn != nil {
		return m.getEnrollsFn(ctx, id)
	}
	return []student.EnrollmentSummary{}, nil
}
func (m *mockRepo) Create(ctx context.Context, id uuid.UUID, req student.CreateStudentRequest) (student.Student, error) {
	if m.createFn != nil {
		return m.createFn(ctx, id, req)
	}
	return student.Student{}, nil
}
func (m *mockRepo) Update(ctx context.Context, id uuid.UUID, req student.UpdateStudentRequest) (student.Student, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, req)
	}
	return student.Student{}, nil
}

func newSvc(repo student.StudentRepository) *student.StudentService {
	return student.NewService(repo)
}

// ── CreateStudent ─────────────────────────────────────────────────────────────

func TestCreateStudent_NilEmail_Success(t *testing.T) {
	id, _ := uuid.NewV7()
	svc := newSvc(&mockRepo{
		createFn: func(_ context.Context, _ uuid.UUID, req student.CreateStudentRequest) (student.Student, error) {
			return student.Student{ID: id, FirstName: req.FirstName, Status: "active"}, nil
		},
	})
	s, err := svc.CreateStudent(context.Background(), student.CreateStudentRequest{
		FirstName: "Rina", LastName: "Kusuma",
	})
	require.NoError(t, err)
	assert.Equal(t, id, s.ID)
}

func TestCreateStudent_EmailTaken_Conflict(t *testing.T) {
	email := "taken@school.com"
	svc := newSvc(&mockRepo{
		getByEmailFn: func(_ context.Context, _ string) (student.Student, error) {
			return student.Student{ID: uuid.New()}, nil
		},
	})
	_, err := svc.CreateStudent(context.Background(), student.CreateStudentRequest{
		FirstName: "Rina", LastName: "Kusuma", Email: &email,
	})
	assert.ErrorIs(t, err, student.ErrEmailConflict)
}

func TestCreateStudent_EmailAvailable_Success(t *testing.T) {
	email := "new@school.com"
	id, _ := uuid.NewV7()
	svc := newSvc(&mockRepo{
		getByEmailFn: func(_ context.Context, _ string) (student.Student, error) {
			return student.Student{}, pgx.ErrNoRows
		},
		createFn: func(_ context.Context, _ uuid.UUID, req student.CreateStudentRequest) (student.Student, error) {
			return student.Student{ID: id, Email: req.Email, Status: "active"}, nil
		},
	})
	s, err := svc.CreateStudent(context.Background(), student.CreateStudentRequest{
		FirstName: "Rina", LastName: "Kusuma", Email: &email,
	})
	require.NoError(t, err)
	require.NotNil(t, s.Email)
	assert.Equal(t, email, *s.Email)
}

// ── GetStudent ────────────────────────────────────────────────────────────────

func TestGetStudent_NotFound(t *testing.T) {
	svc := newSvc(&mockRepo{})
	id, _ := uuid.NewV7()
	_, err := svc.GetStudent(context.Background(), id.String())
	assert.ErrorIs(t, err, student.ErrNotFound)
}

func TestGetStudent_InvalidUUID_NotFound(t *testing.T) {
	svc := newSvc(&mockRepo{})
	_, err := svc.GetStudent(context.Background(), "not-a-uuid")
	assert.ErrorIs(t, err, student.ErrNotFound)
}

func TestGetStudent_Success_WithEnrollments(t *testing.T) {
	id, _ := uuid.NewV7()
	batchID, _ := uuid.NewV7()
	courseID, _ := uuid.NewV7()
	enrollID, _ := uuid.NewV7()

	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (student.Student, error) {
			return student.Student{ID: id, FirstName: "Rina", Status: "active"}, nil
		},
		getEnrollsFn: func(_ context.Context, _ uuid.UUID) ([]student.EnrollmentSummary, error) {
			return []student.EnrollmentSummary{{
				ID:     enrollID,
				Status: "enrolled",
				Batch:  student.BatchRef{ID: batchID, BatchCode: "B001"},
				Course: student.CourseRef{ID: courseID, CourseCode: "MTK01"},
			}}, nil
		},
	})

	detail, err := svc.GetStudent(context.Background(), id.String())
	require.NoError(t, err)
	assert.Equal(t, id, detail.ID)
	assert.Len(t, detail.Enrollments, 1)
	assert.Equal(t, "B001", detail.Enrollments[0].Batch.BatchCode)
}

// ── ListStudents ──────────────────────────────────────────────────────────────

func TestListStudents_PaginationMeta(t *testing.T) {
	id1, _ := uuid.NewV7()
	id2, _ := uuid.NewV7()
	students := []student.Student{
		{ID: id1, FirstName: "A", Status: "active"},
		{ID: id2, FirstName: "B", Status: "active"},
	}
	svc := newSvc(&mockRepo{
		listFn: func(_ context.Context, _ student.ListParams) ([]student.Student, int64, error) {
			return students, 2, nil
		},
	})
	resp, err := svc.ListStudents(context.Background(), student.ListParams{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, 2, resp.Pagination.Total)
	assert.Equal(t, 1, resp.Pagination.TotalPages)
}

func TestListStudents_BatchCodesMerged(t *testing.T) {
	id1, _ := uuid.NewV7()
	id2, _ := uuid.NewV7()
	students := []student.Student{
		{ID: id1, FirstName: "A", Status: "active"},
		{ID: id2, FirstName: "B", Status: "active"},
	}
	svc := newSvc(&mockRepo{
		listFn: func(_ context.Context, _ student.ListParams) ([]student.Student, int64, error) {
			return students, 2, nil
		},
		getBatchCodesFn: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]string, error) {
			return map[uuid.UUID][]string{
				id1: {"B001", "B002"},
			}, nil
		},
	})
	resp, err := svc.ListStudents(context.Background(), student.ListParams{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, []string{"B001", "B002"}, resp.Data[0].EnrolledBatches)
	assert.Equal(t, []string{}, resp.Data[1].EnrolledBatches)
}

// ── UpdateStudent ─────────────────────────────────────────────────────────────

func TestUpdateStudent_NotFound(t *testing.T) {
	id, _ := uuid.NewV7()
	svc := newSvc(&mockRepo{})
	newName := "Rani"
	_, err := svc.UpdateStudent(context.Background(), id.String(), student.UpdateStudentRequest{FirstName: &newName})
	assert.ErrorIs(t, err, student.ErrNotFound)
}

func TestUpdateStudent_EmailConflict(t *testing.T) {
	id, _ := uuid.NewV7()
	original := "original@school.com"
	taken := "taken@school.com"
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (student.Student, error) {
			return student.Student{ID: id, Email: &original, Status: "active"}, nil
		},
		getByEmailFn: func(_ context.Context, _ string) (student.Student, error) {
			return student.Student{ID: uuid.New()}, nil
		},
	})
	_, err := svc.UpdateStudent(context.Background(), id.String(), student.UpdateStudentRequest{Email: &taken})
	assert.ErrorIs(t, err, student.ErrEmailConflict)
}

func TestUpdateStudent_NilEmail_NoCheck(t *testing.T) {
	id, _ := uuid.NewV7()
	newName := "Rani"
	emailCheckCalled := false
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (student.Student, error) {
			return student.Student{ID: id, Status: "active"}, nil
		},
		getByEmailFn: func(_ context.Context, _ string) (student.Student, error) {
			emailCheckCalled = true
			return student.Student{}, pgx.ErrNoRows
		},
		updateFn: func(_ context.Context, _ uuid.UUID, _ student.UpdateStudentRequest) (student.Student, error) {
			return student.Student{ID: id, Status: "active"}, nil
		},
	})
	_, err := svc.UpdateStudent(context.Background(), id.String(), student.UpdateStudentRequest{FirstName: &newName})
	require.NoError(t, err)
	assert.False(t, emailCheckCalled)
}

func TestUpdateStudent_Success(t *testing.T) {
	id, _ := uuid.NewV7()
	newStatus := "graduated"
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (student.Student, error) {
			return student.Student{ID: id, Status: "active"}, nil
		},
		updateFn: func(_ context.Context, _ uuid.UUID, req student.UpdateStudentRequest) (student.Student, error) {
			return student.Student{ID: id, Status: *req.Status, UpdatedAt: time.Now()}, nil
		},
	})
	s, err := svc.UpdateStudent(context.Background(), id.String(), student.UpdateStudentRequest{Status: &newStatus})
	require.NoError(t, err)
	assert.Equal(t, "graduated", s.Status)
}
