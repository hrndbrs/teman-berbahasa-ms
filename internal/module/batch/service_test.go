package batch_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/batch"
)

// ── mock ──────────────────────────────────────────────────────────────────────

type mockRepo struct {
	listFn                func(ctx context.Context, params batch.ListParams) ([]batch.BatchWithDetails, int64, error)
	getByIDFn             func(ctx context.Context, id uuid.UUID) (batch.BatchWithDetails, error)
	createFn              func(ctx context.Context, id uuid.UUID, req batch.CreateBatchRequest) (batch.BatchWithDetails, error)
	updateFn              func(ctx context.Context, id uuid.UUID, req batch.UpdateBatchRequest) (batch.BatchWithDetails, error)
	updateStatusFn        func(ctx context.Context, id uuid.UUID, status string) (batch.StatusUpdateResult, error)
	deleteFn              func(ctx context.Context, id uuid.UUID) error
	countActiveEnrollFn   func(ctx context.Context, batchID uuid.UUID) (int64, error)
	existsActiveTeacherFn func(ctx context.Context, instructorID uuid.UUID) (bool, error)
}

func (m *mockRepo) List(ctx context.Context, params batch.ListParams) ([]batch.BatchWithDetails, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return nil, 0, nil
}
func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (batch.BatchWithDetails, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return batch.BatchWithDetails{}, pgx.ErrNoRows
}
func (m *mockRepo) Create(ctx context.Context, id uuid.UUID, req batch.CreateBatchRequest) (batch.BatchWithDetails, error) {
	if m.createFn != nil {
		return m.createFn(ctx, id, req)
	}
	return batch.BatchWithDetails{}, nil
}
func (m *mockRepo) Update(ctx context.Context, id uuid.UUID, req batch.UpdateBatchRequest) (batch.BatchWithDetails, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, req)
	}
	return batch.BatchWithDetails{}, nil
}
func (m *mockRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (batch.StatusUpdateResult, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status)
	}
	return batch.StatusUpdateResult{}, nil
}
func (m *mockRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockRepo) CountActiveEnrollments(ctx context.Context, batchID uuid.UUID) (int64, error) {
	if m.countActiveEnrollFn != nil {
		return m.countActiveEnrollFn(ctx, batchID)
	}
	return 0, nil
}
func (m *mockRepo) ExistsActiveTeacher(ctx context.Context, instructorID uuid.UUID) (bool, error) {
	if m.existsActiveTeacherFn != nil {
		return m.existsActiveTeacherFn(ctx, instructorID)
	}
	return false, nil
}

func newSvc(repo batch.BatchRepository) *batch.BatchService {
	return batch.NewService(repo)
}

func fixedID() uuid.UUID {
	id, _ := uuid.Parse("019687a2-0002-7000-8000-000000000001")
	return id
}

func fixedID2() uuid.UUID {
	id, _ := uuid.Parse("019687a2-0002-7000-8000-000000000002")
	return id
}

func makeBatchWithDetails() batch.BatchWithDetails {
	return batch.BatchWithDetails{
		Batch: batch.Batch{
			ID:               fixedID(),
			CourseID:         fixedID2(),
			InstructorUserID: fixedID2(),
			CreatedByUserID:  fixedID2(),
			BatchName:        "N5 Spring Morning",
			BatchCode:        "SP26-A",
			Status:           "upcoming",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		CourseName:          "JLPT N5 Foundations",
		CourseCode:          "JP-N5",
		InstructorFirstName: "Taro",
		InstructorLastName:  "Tanaka",
		EnrolledCount:       0,
	}
}

// ── ValidateBatchTransition ───────────────────────────────────────────────────

func TestValidateBatchTransition_ValidTransitions(t *testing.T) {
	assert.NoError(t, batch.ValidateBatchTransition("upcoming", "ongoing"))
	assert.NoError(t, batch.ValidateBatchTransition("ongoing", "completed"))
}

func TestValidateBatchTransition_InvalidTransitions(t *testing.T) {
	cases := []struct{ from, to string }{
		{"upcoming", "completed"},
		{"ongoing", "upcoming"},
		{"completed", "ongoing"},
		{"completed", "upcoming"},
	}
	for _, c := range cases {
		err := batch.ValidateBatchTransition(c.from, c.to)
		assert.ErrorIs(t, err, batch.ErrInvalidTransition, "from=%s to=%s", c.from, c.to)
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestList_ReturnsPaginatedResults(t *testing.T) {
	svc := newSvc(&mockRepo{
		listFn: func(_ context.Context, _ batch.ListParams) ([]batch.BatchWithDetails, int64, error) {
			return []batch.BatchWithDetails{makeBatchWithDetails()}, 1, nil
		},
	})
	resp, err := svc.List(context.Background(), batch.ListParams{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, 1, resp.Pagination.Total)
	assert.Equal(t, 1, resp.Pagination.TotalPages)
}

func TestList_DefaultsPaginationParams(t *testing.T) {
	var captured batch.ListParams
	svc := newSvc(&mockRepo{
		listFn: func(_ context.Context, params batch.ListParams) ([]batch.BatchWithDetails, int64, error) {
			captured = params
			return nil, 0, nil
		},
	})
	_, err := svc.List(context.Background(), batch.ListParams{Page: 0, PerPage: 0})
	require.NoError(t, err)
	assert.Equal(t, 1, captured.Page)
	assert.Equal(t, 20, captured.PerPage)
}

func TestList_CapsPerPageAt100(t *testing.T) {
	var captured batch.ListParams
	svc := newSvc(&mockRepo{
		listFn: func(_ context.Context, params batch.ListParams) ([]batch.BatchWithDetails, int64, error) {
			captured = params
			return nil, 0, nil
		},
	})
	_, err := svc.List(context.Background(), batch.ListParams{Page: 1, PerPage: 999})
	require.NoError(t, err)
	assert.Equal(t, 20, captured.PerPage)
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestGetByID_ReturnsBatchWithDetails(t *testing.T) {
	expected := makeBatchWithDetails()
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (batch.BatchWithDetails, error) {
			return expected, nil
		},
	})
	got, err := svc.GetByID(context.Background(), fixedID().String())
	require.NoError(t, err)
	assert.Equal(t, expected.BatchName, got.BatchName)
	assert.Equal(t, "JP-N5", got.CourseCode)
	assert.Equal(t, int64(0), got.EnrolledCount)
}

func TestGetByID_NotFound(t *testing.T) {
	svc := newSvc(&mockRepo{})
	_, err := svc.GetByID(context.Background(), fixedID().String())
	assert.ErrorIs(t, err, batch.ErrNotFound)
}

func TestGetByID_InvalidUUID(t *testing.T) {
	svc := newSvc(&mockRepo{})
	_, err := svc.GetByID(context.Background(), "not-a-uuid")
	assert.ErrorIs(t, err, batch.ErrNotFound)
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestCreate_HappyPath(t *testing.T) {
	svc := newSvc(&mockRepo{
		existsActiveTeacherFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		createFn: func(_ context.Context, id uuid.UUID, req batch.CreateBatchRequest) (batch.BatchWithDetails, error) {
			b := makeBatchWithDetails()
			b.ID = id
			b.BatchName = req.BatchName
			b.BatchCode = req.BatchCode
			return b, nil
		},
	})
	got, err := svc.Create(context.Background(), batch.CreateBatchRequest{
		CourseID:         fixedID2(),
		InstructorUserID: fixedID2(),
		CreatedByUserID:  fixedID2(),
		BatchName:        "N5 Spring Morning",
		BatchCode:        "SP26-A",
	})
	require.NoError(t, err)
	assert.Equal(t, "N5 Spring Morning", got.BatchName)
	assert.Equal(t, "SP26-A", got.BatchCode)
}

func TestCreate_InvalidInstructor(t *testing.T) {
	svc := newSvc(&mockRepo{
		existsActiveTeacherFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	})
	_, err := svc.Create(context.Background(), batch.CreateBatchRequest{
		CourseID:         fixedID2(),
		InstructorUserID: fixedID2(),
		CreatedByUserID:  fixedID2(),
		BatchName:        "Test",
		BatchCode:        "T26-A",
	})
	assert.ErrorIs(t, err, batch.ErrInvalidInstructor)
}

func TestCreate_DuplicateBatchCode(t *testing.T) {
	svc := newSvc(&mockRepo{
		existsActiveTeacherFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		createFn: func(_ context.Context, _ uuid.UUID, _ batch.CreateBatchRequest) (batch.BatchWithDetails, error) {
			return batch.BatchWithDetails{}, &pgconn.PgError{Code: "23505"}
		},
	})
	_, err := svc.Create(context.Background(), batch.CreateBatchRequest{
		CourseID:         fixedID2(),
		InstructorUserID: fixedID2(),
		CreatedByUserID:  fixedID2(),
		BatchName:        "Dup",
		BatchCode:        "DUP",
	})
	assert.ErrorIs(t, err, batch.ErrBatchCodeConflict)
}

func TestCreate_TrimsWhitespace(t *testing.T) {
	var captured batch.CreateBatchRequest
	svc := newSvc(&mockRepo{
		existsActiveTeacherFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		createFn: func(_ context.Context, _ uuid.UUID, req batch.CreateBatchRequest) (batch.BatchWithDetails, error) {
			captured = req
			return makeBatchWithDetails(), nil
		},
	})
	_, err := svc.Create(context.Background(), batch.CreateBatchRequest{
		CourseID:         fixedID2(),
		InstructorUserID: fixedID2(),
		CreatedByUserID:  fixedID2(),
		BatchName:        "  N5 Spring  ",
		BatchCode:        "  SP26-A  ",
	})
	require.NoError(t, err)
	assert.Equal(t, "N5 Spring", captured.BatchName)
	assert.Equal(t, "SP26-A", captured.BatchCode)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestUpdate_HappyPath(t *testing.T) {
	newName := "Updated Name"
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (batch.BatchWithDetails, error) {
			return makeBatchWithDetails(), nil
		},
		updateFn: func(_ context.Context, _ uuid.UUID, req batch.UpdateBatchRequest) (batch.BatchWithDetails, error) {
			b := makeBatchWithDetails()
			b.BatchName = *req.BatchName
			return b, nil
		},
	})
	got, err := svc.Update(context.Background(), fixedID().String(), batch.UpdateBatchRequest{BatchName: &newName})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", got.BatchName)
}

func TestUpdate_NotFound(t *testing.T) {
	svc := newSvc(&mockRepo{})
	_, err := svc.Update(context.Background(), fixedID().String(), batch.UpdateBatchRequest{})
	assert.ErrorIs(t, err, batch.ErrNotFound)
}

func TestUpdate_InvalidUUID(t *testing.T) {
	svc := newSvc(&mockRepo{})
	_, err := svc.Update(context.Background(), "not-a-uuid", batch.UpdateBatchRequest{})
	assert.ErrorIs(t, err, batch.ErrNotFound)
}

func TestUpdate_InvalidInstructor(t *testing.T) {
	instrID := fixedID2()
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (batch.BatchWithDetails, error) {
			return makeBatchWithDetails(), nil
		},
		existsActiveTeacherFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	})
	_, err := svc.Update(context.Background(), fixedID().String(), batch.UpdateBatchRequest{InstructorUserID: &instrID})
	assert.ErrorIs(t, err, batch.ErrInvalidInstructor)
}

func TestUpdate_DuplicateBatchCode(t *testing.T) {
	code := "EXISTS"
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (batch.BatchWithDetails, error) {
			return makeBatchWithDetails(), nil
		},
		updateFn: func(_ context.Context, _ uuid.UUID, _ batch.UpdateBatchRequest) (batch.BatchWithDetails, error) {
			return batch.BatchWithDetails{}, &pgconn.PgError{Code: "23505"}
		},
	})
	_, err := svc.Update(context.Background(), fixedID().String(), batch.UpdateBatchRequest{BatchCode: &code})
	assert.ErrorIs(t, err, batch.ErrBatchCodeConflict)
}

func TestUpdate_TrimsWhitespace(t *testing.T) {
	var captured batch.UpdateBatchRequest
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (batch.BatchWithDetails, error) {
			return makeBatchWithDetails(), nil
		},
		updateFn: func(_ context.Context, _ uuid.UUID, req batch.UpdateBatchRequest) (batch.BatchWithDetails, error) {
			captured = req
			return makeBatchWithDetails(), nil
		},
	})
	name := "  Trimmed Name  "
	code := "  SP26-B  "
	_, err := svc.Update(context.Background(), fixedID().String(), batch.UpdateBatchRequest{BatchName: &name, BatchCode: &code})
	require.NoError(t, err)
	assert.Equal(t, "Trimmed Name", *captured.BatchName)
	assert.Equal(t, "SP26-B", *captured.BatchCode)
}

// ── TransitionStatus ──────────────────────────────────────────────────────────

func TestTransitionStatus_UpcomingToOngoing(t *testing.T) {
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (batch.BatchWithDetails, error) {
			b := makeBatchWithDetails()
			b.Status = "upcoming"
			return b, nil
		},
		updateStatusFn: func(_ context.Context, id uuid.UUID, status string) (batch.StatusUpdateResult, error) {
			return batch.StatusUpdateResult{ID: id, Status: status, UpdatedAt: time.Now()}, nil
		},
	})
	result, err := svc.TransitionStatus(context.Background(), fixedID().String(), "ongoing")
	require.NoError(t, err)
	assert.Equal(t, "ongoing", result.Status)
}

func TestTransitionStatus_OngoingToCompleted(t *testing.T) {
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (batch.BatchWithDetails, error) {
			b := makeBatchWithDetails()
			b.Status = "ongoing"
			return b, nil
		},
		updateStatusFn: func(_ context.Context, id uuid.UUID, status string) (batch.StatusUpdateResult, error) {
			return batch.StatusUpdateResult{ID: id, Status: status, UpdatedAt: time.Now()}, nil
		},
	})
	result, err := svc.TransitionStatus(context.Background(), fixedID().String(), "completed")
	require.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
}

func TestTransitionStatus_InvalidTransition(t *testing.T) {
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (batch.BatchWithDetails, error) {
			b := makeBatchWithDetails()
			b.Status = "upcoming"
			return b, nil
		},
	})
	_, err := svc.TransitionStatus(context.Background(), fixedID().String(), "completed")
	assert.ErrorIs(t, err, batch.ErrInvalidTransition)
}

func TestTransitionStatus_NotFound(t *testing.T) {
	svc := newSvc(&mockRepo{})
	_, err := svc.TransitionStatus(context.Background(), fixedID().String(), "ongoing")
	assert.ErrorIs(t, err, batch.ErrNotFound)
}

func TestTransitionStatus_InvalidUUID(t *testing.T) {
	svc := newSvc(&mockRepo{})
	_, err := svc.TransitionStatus(context.Background(), "not-a-uuid", "ongoing")
	assert.ErrorIs(t, err, batch.ErrNotFound)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestDelete_HappyPath(t *testing.T) {
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (batch.BatchWithDetails, error) {
			return makeBatchWithDetails(), nil
		},
		countActiveEnrollFn: func(_ context.Context, _ uuid.UUID) (int64, error) {
			return 0, nil
		},
		deleteFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	})
	assert.NoError(t, svc.Delete(context.Background(), fixedID().String()))
}

func TestDelete_NotFound(t *testing.T) {
	svc := newSvc(&mockRepo{})
	assert.ErrorIs(t, svc.Delete(context.Background(), fixedID().String()), batch.ErrNotFound)
}

func TestDelete_InvalidUUID(t *testing.T) {
	svc := newSvc(&mockRepo{})
	assert.ErrorIs(t, svc.Delete(context.Background(), "not-a-uuid"), batch.ErrNotFound)
}

func TestDelete_HasActiveEnrollments(t *testing.T) {
	svc := newSvc(&mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (batch.BatchWithDetails, error) {
			return makeBatchWithDetails(), nil
		},
		countActiveEnrollFn: func(_ context.Context, _ uuid.UUID) (int64, error) {
			return 3, nil
		},
	})
	assert.ErrorIs(t, svc.Delete(context.Background(), fixedID().String()), batch.ErrHasActiveEnrollments)
}
