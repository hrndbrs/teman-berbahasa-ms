package user_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/user"
)

// ── mocks ─────────────────────────────────────────────────────────────────────

type mockRepo struct {
	getUserByIDFullFn              func(ctx context.Context, id uuid.UUID) (user.User, error)
	getUserByEmailFn               func(ctx context.Context, email string) (user.User, error)
	listUsersFn                    func(ctx context.Context, params user.ListParams) ([]user.User, int64, error)
	createUserFn                   func(ctx context.Context, id uuid.UUID, req user.CreateUserRequest, passwordHash string) (user.User, error)
	updateUserFn                   func(ctx context.Context, id uuid.UUID, req user.UpdateUserRequest) (user.User, error)
	insertPasswordResetTokenFn     func(ctx context.Context, userID uuid.UUID, rawToken string, expiresAt time.Time) error
	deletePasswordResetByUserIDFn  func(ctx context.Context, userID uuid.UUID) error
	deleteUserSessionsFn           func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockRepo) GetUserByIDFull(ctx context.Context, id uuid.UUID) (user.User, error) {
	if m.getUserByIDFullFn != nil {
		return m.getUserByIDFullFn(ctx, id)
	}
	return user.User{}, pgx.ErrNoRows
}
func (m *mockRepo) GetUserByEmail(ctx context.Context, email string) (user.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, email)
	}
	return user.User{}, pgx.ErrNoRows
}
func (m *mockRepo) ListUsers(ctx context.Context, params user.ListParams) ([]user.User, int64, error) {
	if m.listUsersFn != nil {
		return m.listUsersFn(ctx, params)
	}
	return nil, 0, nil
}
func (m *mockRepo) CreateUser(ctx context.Context, id uuid.UUID, req user.CreateUserRequest, passwordHash string) (user.User, error) {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, id, req, passwordHash)
	}
	return user.User{}, nil
}
func (m *mockRepo) UpdateUser(ctx context.Context, id uuid.UUID, req user.UpdateUserRequest) (user.User, error) {
	if m.updateUserFn != nil {
		return m.updateUserFn(ctx, id, req)
	}
	return user.User{}, nil
}
func (m *mockRepo) InsertPasswordResetToken(ctx context.Context, userID uuid.UUID, rawToken string, expiresAt time.Time) error {
	if m.insertPasswordResetTokenFn != nil {
		return m.insertPasswordResetTokenFn(ctx, userID, rawToken, expiresAt)
	}
	return nil
}
func (m *mockRepo) DeletePasswordResetTokensByUserID(ctx context.Context, userID uuid.UUID) error {
	if m.deletePasswordResetByUserIDFn != nil {
		return m.deletePasswordResetByUserIDFn(ctx, userID)
	}
	return nil
}
func (m *mockRepo) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	if m.deleteUserSessionsFn != nil {
		return m.deleteUserSessionsFn(ctx, userID)
	}
	return nil
}

type mockEmailSender struct {
	sendInviteFn func(ctx context.Context, toEmail, firstName, rawToken string) error
}

func (m *mockEmailSender) SendInvite(ctx context.Context, toEmail, firstName, rawToken string) error {
	if m.sendInviteFn != nil {
		return m.sendInviteFn(ctx, toEmail, firstName, rawToken)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newSvc(repo user.UserRepository, emailer user.EmailSender) *user.UserService {
	return user.NewService(repo, emailer)
}

// ── CreateUser ────────────────────────────────────────────────────────────────

func TestCreateUser_DuplicateEmail(t *testing.T) {
	svc := newSvc(&mockRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (user.User, error) {
			return user.User{ID: uuid.New()}, nil
		},
	}, &mockEmailSender{})

	_, err := svc.CreateUser(context.Background(), user.CreateUserRequest{
		FirstName: "Ana", LastName: "Budi", Email: "ana@school.com", Role: "teacher",
	})
	assert.ErrorIs(t, err, user.ErrEmailConflict)
}

func TestCreateUser_Success_SendsInvite(t *testing.T) {
	id, _ := uuid.NewV7()
	inviteSent := false

	svc := newSvc(&mockRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (user.User, error) {
			return user.User{}, pgx.ErrNoRows
		},
		createUserFn: func(_ context.Context, _ uuid.UUID, req user.CreateUserRequest, _ string) (user.User, error) {
			return user.User{ID: id, FirstName: req.FirstName, LastName: req.LastName, Email: req.Email, Role: req.Role, Status: "active"}, nil
		},
	}, &mockEmailSender{
		sendInviteFn: func(_ context.Context, _, _, _ string) error {
			inviteSent = true
			return nil
		},
	})

	u, err := svc.CreateUser(context.Background(), user.CreateUserRequest{
		FirstName: "Ana", LastName: "Budi", Email: "ana@school.com", Role: "teacher",
	})
	require.NoError(t, err)
	assert.Equal(t, "Ana", u.FirstName)
	assert.True(t, inviteSent)
}

// ── GetUser ───────────────────────────────────────────────────────────────────

func TestGetUser_OwnRecord(t *testing.T) {
	id, _ := uuid.NewV7()
	svc := newSvc(&mockRepo{
		getUserByIDFullFn: func(_ context.Context, _ uuid.UUID) (user.User, error) {
			return user.User{ID: id, Email: "ana@school.com", Role: "teacher", Status: "active"}, nil
		},
	}, &mockEmailSender{})

	u, err := svc.GetUser(context.Background(), id.String(), "teacher", id.String())
	require.NoError(t, err)
	assert.Equal(t, id, u.ID)
}

func TestGetUser_OtherUser_NonAdmin_Forbidden(t *testing.T) {
	callerID, _ := uuid.NewV7()
	targetID, _ := uuid.NewV7()

	svc := newSvc(&mockRepo{}, &mockEmailSender{})

	_, err := svc.GetUser(context.Background(), callerID.String(), "teacher", targetID.String())
	assert.ErrorIs(t, err, user.ErrForbidden)
}

func TestGetUser_AdminGetsOther(t *testing.T) {
	callerID, _ := uuid.NewV7()
	targetID, _ := uuid.NewV7()

	svc := newSvc(&mockRepo{
		getUserByIDFullFn: func(_ context.Context, _ uuid.UUID) (user.User, error) {
			return user.User{ID: targetID, Role: "teacher", Status: "active"}, nil
		},
	}, &mockEmailSender{})

	u, err := svc.GetUser(context.Background(), callerID.String(), "admin", targetID.String())
	require.NoError(t, err)
	assert.Equal(t, targetID, u.ID)
}

func TestGetUser_NotFound(t *testing.T) {
	id, _ := uuid.NewV7()
	svc := newSvc(&mockRepo{
		getUserByIDFullFn: func(_ context.Context, _ uuid.UUID) (user.User, error) {
			return user.User{}, pgx.ErrNoRows
		},
	}, &mockEmailSender{})

	_, err := svc.GetUser(context.Background(), id.String(), "admin", id.String())
	assert.ErrorIs(t, err, user.ErrNotFound)
}

// ── UpdateUser ────────────────────────────────────────────────────────────────

func TestUpdateUser_RoleChange_KillsSessions(t *testing.T) {
	id, _ := uuid.NewV7()
	sessionKilled := false
	newRole := "admin"

	svc := newSvc(&mockRepo{
		getUserByIDFullFn: func(_ context.Context, _ uuid.UUID) (user.User, error) {
			return user.User{ID: id, Role: "teacher", Status: "active"}, nil
		},
		updateUserFn: func(_ context.Context, _ uuid.UUID, _ user.UpdateUserRequest) (user.User, error) {
			return user.User{ID: id, Role: "admin", Status: "active"}, nil
		},
		deleteUserSessionsFn: func(_ context.Context, _ uuid.UUID) error {
			sessionKilled = true
			return nil
		},
	}, &mockEmailSender{})

	_, err := svc.UpdateUser(context.Background(), id.String(), user.UpdateUserRequest{Role: &newRole})
	require.NoError(t, err)
	assert.True(t, sessionKilled)
}

func TestUpdateUser_StatusInactive_KillsSessions(t *testing.T) {
	id, _ := uuid.NewV7()
	sessionKilled := false
	inactive := "inactive"

	svc := newSvc(&mockRepo{
		getUserByIDFullFn: func(_ context.Context, _ uuid.UUID) (user.User, error) {
			return user.User{ID: id, Role: "teacher", Status: "active"}, nil
		},
		updateUserFn: func(_ context.Context, _ uuid.UUID, _ user.UpdateUserRequest) (user.User, error) {
			return user.User{ID: id, Role: "teacher", Status: "inactive"}, nil
		},
		deleteUserSessionsFn: func(_ context.Context, _ uuid.UUID) error {
			sessionKilled = true
			return nil
		},
	}, &mockEmailSender{})

	_, err := svc.UpdateUser(context.Background(), id.String(), user.UpdateUserRequest{Status: &inactive})
	require.NoError(t, err)
	assert.True(t, sessionKilled)
}

func TestUpdateUser_NameChange_NoSessionKill(t *testing.T) {
	id, _ := uuid.NewV7()
	sessionKilled := false
	newName := "Rani"

	svc := newSvc(&mockRepo{
		getUserByIDFullFn: func(_ context.Context, _ uuid.UUID) (user.User, error) {
			return user.User{ID: id, Role: "teacher", Status: "active"}, nil
		},
		updateUserFn: func(_ context.Context, _ uuid.UUID, _ user.UpdateUserRequest) (user.User, error) {
			return user.User{ID: id, Role: "teacher", Status: "active"}, nil
		},
		deleteUserSessionsFn: func(_ context.Context, _ uuid.UUID) error {
			sessionKilled = true
			return nil
		},
	}, &mockEmailSender{})

	_, err := svc.UpdateUser(context.Background(), id.String(), user.UpdateUserRequest{FirstName: &newName})
	require.NoError(t, err)
	assert.False(t, sessionKilled)
}

func TestUpdateUser_DuplicateEmail(t *testing.T) {
	id, _ := uuid.NewV7()
	newEmail := "taken@school.com"

	svc := newSvc(&mockRepo{
		getUserByIDFullFn: func(_ context.Context, _ uuid.UUID) (user.User, error) {
			return user.User{ID: id, Email: "original@school.com", Role: "teacher", Status: "active"}, nil
		},
		getUserByEmailFn: func(_ context.Context, _ string) (user.User, error) {
			return user.User{ID: uuid.New()}, nil
		},
	}, &mockEmailSender{})

	_, err := svc.UpdateUser(context.Background(), id.String(), user.UpdateUserRequest{Email: &newEmail})
	assert.ErrorIs(t, err, user.ErrEmailConflict)
}

func TestUpdateUser_NotFound(t *testing.T) {
	id, _ := uuid.NewV7()
	newName := "Rani"

	svc := newSvc(&mockRepo{
		getUserByIDFullFn: func(_ context.Context, _ uuid.UUID) (user.User, error) {
			return user.User{}, pgx.ErrNoRows
		},
	}, &mockEmailSender{})

	_, err := svc.UpdateUser(context.Background(), id.String(), user.UpdateUserRequest{FirstName: &newName})
	assert.ErrorIs(t, err, user.ErrNotFound)
}

// ── ListUsers ─────────────────────────────────────────────────────────────────

func TestListUsers_ReturnsPaginatedResults(t *testing.T) {
	id1, _ := uuid.NewV7()
	id2, _ := uuid.NewV7()
	users := []user.User{
		{ID: id1, Email: "a@school.com", Role: "admin", Status: "active"},
		{ID: id2, Email: "b@school.com", Role: "teacher", Status: "active"},
	}

	svc := newSvc(&mockRepo{
		listUsersFn: func(_ context.Context, _ user.ListParams) ([]user.User, int64, error) {
			return users, 2, nil
		},
	}, &mockEmailSender{})

	resp, err := svc.ListUsers(context.Background(), user.ListParams{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, 2, resp.Pagination.Total)
	assert.Equal(t, 1, resp.Pagination.TotalPages)
}
