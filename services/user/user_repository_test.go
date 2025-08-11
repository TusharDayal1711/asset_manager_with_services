package userservice

import (
	"asset/providers"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetUserByEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	email := "test.user@remotestate.com"
	userID := uuid.MustParse("5f62831e-44c5-46c4-bede-0d5e3253cc16")

	tests := []struct {
		name           string
		mockSetup      func(mock sqlmock.Sqlmock)
		inputEmail     string
		expectedUserID uuid.UUID
		expectedErr    error
	}{
		{
			name:       "successfully retrieves user by email",
			inputEmail: email,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(userID)
				mock.ExpectQuery(`SELECT id FROM users WHERE email = \$1 AND archived_at IS NULL$`).
					WithArgs(email).
					WillReturnRows(rows)
			},
			expectedUserID: userID,
			expectedErr:    nil,
		},
		{
			name:       "user not found",
			inputEmail: email,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id FROM users WHERE email = \$1 AND archived_at IS NULL$`).
					WithArgs(email).
					WillReturnError(sql.ErrNoRows)
			},
			expectedUserID: uuid.Nil,
			expectedErr:    sql.ErrNoRows,
		},
		{
			name:       "query error",
			inputEmail: email,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id FROM users WHERE email = \$1 AND archived_at IS NULL$`).
					WithArgs(email).
					WillReturnError(errors.New("db error"))
			},
			expectedUserID: uuid.Nil,
			expectedErr:    errors.New("db error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")

			mockLogger := providers.NewMockZapLoggerProvider(ctrl)
			mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

			repo := &PostgresUserRepository{
				DB:     sqlxDB,
				Logger: mockLogger,
			}

			tc.mockSetup(mock)

			gotUserID, err := repo.GetUserByEmail(ctx, tc.inputEmail)
			assert.Equal(t, tc.expectedUserID, gotUserID)

			if tc.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetUserDashboardById(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	contactNo := "1234567890"
	userType := "manager"
	expectedUser := UserDashboardRes{
		ID:        userID.String(),
		Username:  "testuser",
		Email:     "test.user30@remotestate.com",
		ContactNo: &contactNo,
		Type:      &userType,
		Roles:     []string{"employee"},
		AssignedAssets: []AssetDetails{
			{
				ID:       uuid.New(),
				Brand:    "Lenovo",
				Model:    "Thinkpad",
				SerialNo: "LN1234567",
				Type:     "Laptop",
				Status:   "assigned",
				OwnedBy:  "company",
			},
		},
	}

	t.Run("successful", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "postgres")
		cacheData, _ := jsoniter.Marshal(expectedUser)

		mockLogger := providers.NewMockZapLoggerProvider(ctrl)
		mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

		mockRedis := providers.NewMockRedisProvider(ctrl)
		mockRedis.EXPECT().Get(ctx, "user:dashboard:"+userID.String()).Return(string(cacheData), nil)

		repo := &PostgresUserRepository{
			DB:     sqlxDB,
			Logger: mockLogger,
			Redis:  mockRedis,
		}

		result, err := repo.GetUserDashboardById(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser.ID, result.ID)
		assert.Equal(t, expectedUser.Username, result.Username)
		assert.Equal(t, expectedUser.Email, result.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success, fetch from DB", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		mockLogger := providers.NewMockZapLoggerProvider(ctrl)
		mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

		mockRedis := providers.NewMockRedisProvider(ctrl)
		mockRedis.EXPECT().Get(ctx, "user:dashboard:"+userID.String()).Return("", errors.New("user not found"))
		mockRedis.EXPECT().Set(ctx, "user:dashboard:"+userID.String(), gomock.Any(), 5*time.Minute).Return(nil)

		rowsUser := sqlmock.NewRows([]string{"id", "username", "email", "contact_no", "type"}).
			AddRow(userID.String(), expectedUser.Username, expectedUser.Email, contactNo, userType)

		rowsRoles := sqlmock.NewRows([]string{"role"}).AddRow("employee")

		rowsAssets := sqlmock.NewRows([]string{"id", "brand", "model", "serial_no", "type", "status", "owned_by"}).
			AddRow(expectedUser.AssignedAssets[0].ID, "Lenovo", "Thinkpad", "LN1234567", "Laptop", "assigned", "company")

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT u.id, u.username, u.email, u.contact_no, ut.type`).
			WithArgs(userID).WillReturnRows(rowsUser)
		mock.ExpectQuery(`SELECT role FROM user_roles`).
			WithArgs(userID).WillReturnRows(rowsRoles)
		mock.ExpectQuery(`SELECT a.id, a.brand, a.model, a.serial_no, a.type, a.status, a.owned_by`).
			WithArgs(userID).WillReturnRows(rowsAssets)
		mock.ExpectCommit()

		repo := &PostgresUserRepository{
			DB:     sqlxDB,
			Logger: mockLogger,
			Redis:  mockRedis,
		}

		result, err := repo.GetUserDashboardById(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser.Username, result.Username)
		assert.Equal(t, expectedUser.Email, result.Email)
		assert.Equal(t, *expectedUser.ContactNo, *result.ContactNo)
		assert.Equal(t, expectedUser.Type, result.Type)
		assert.Equal(t, "employee", result.Roles[0])
		assert.Equal(t, expectedUser.AssignedAssets[0].Brand, result.AssignedAssets[0].Brand)
		assert.Equal(t, expectedUser.AssignedAssets[0].Model, result.AssignedAssets[0].Model)
		assert.Equal(t, expectedUser.AssignedAssets[0].SerialNo, result.AssignedAssets[0].SerialNo)

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestDeleteUserByID(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("successfully deletes user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		mockLogger := providers.NewMockZapLoggerProvider(ctrl)
		mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT count\(\*\) FROM asset_assign`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectExec(`UPDATE users SET archived_at = now()`).
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`UPDATE user_roles SET archived_at = now()`).
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`UPDATE user_type SET archived_at = now()`).
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		repo := &PostgresUserRepository{
			DB:     sqlxDB,
			Logger: mockLogger,
		}

		err = repo.DeleteUserByID(ctx, userID)
		assert.NoError(t, err)
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("fail, user has assigned assets", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		mockLogger := providers.NewMockZapLoggerProvider(ctrl)
		mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT count\(\*\) FROM asset_assign`).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		mock.ExpectRollback()

		repo := &PostgresUserRepository{
			DB:     sqlxDB,
			Logger: mockLogger,
		}

		err = repo.DeleteUserByID(ctx, userID)
		assert.EqualError(t, err, "cannot delete user, still have asset assigned")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fail - error while checking assigned assets", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()
		sqlxDB := sqlx.NewDb(db, "postgres")

		mockLogger := providers.NewMockZapLoggerProvider(ctrl)
		mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT count\(\*\) FROM asset_assign`).
			WithArgs(userID).
			WillReturnError(errors.New("db error"))

		mock.ExpectRollback()

		repo := &PostgresUserRepository{
			DB:     sqlxDB,
			Logger: mockLogger,
		}

		err = repo.DeleteUserByID(ctx, userID)
		assert.Contains(t, err.Error(), "failed to check asset assignment")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed, DB transaction faield", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "postgres")

		mock.ExpectBegin().WillReturnError(errors.New("begin tx failed"))

		mockLogger := providers.NewMockZapLoggerProvider(ctrl)
		mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

		repo := &PostgresUserRepository{
			DB:     sqlxDB,
			Logger: mockLogger,
		}
		err = repo.DeleteUserByID(ctx, userID)
		assert.EqualError(t, err, "failed to start transaction: begin tx failed")

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestCreateNewEmployee(t *testing.T) {
	ctx := context.Background()
	managerID := uuid.New()
	newUserID := uuid.New()
	req := ManagerRegisterReq{
		Username:  "test user32",
		Email:     "test.user32@remotestate.com",
		ContactNo: "78772883",
		Type:      "full_time",
	}

	type testCase struct {
		name                string
		mockSetup           func(mock sqlmock.Sqlmock)
		expectedErrContains string
	}

	testCases := []testCase{
		{
			name: "successfully creates new employee ",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO users \(username, email, contact_no, created_by\)`).
					WithArgs(req.Username, req.Email, req.ContactNo, managerID).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(newUserID))

				mock.ExpectExec(`INSERT INTO user_type \(user_id, type, created_by\)`).
					WithArgs(newUserID, req.Type, managerID).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec(`INSERT INTO user_roles \(user_id, role, created_by\)`).
					WithArgs(newUserID, managerID).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectCommit()
			},
		},
		{
			name: "failed, error inserting user into users table",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO users \(username, email, contact_no, created_by\)`).
					WithArgs(req.Username, req.Email, req.ContactNo, managerID).
					WillReturnError(errors.New("insert error"))
				mock.ExpectRollback()
			},
			expectedErrContains: "failed to insert employee",
		},
		{
			name: "failed, error inserting into user_type",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO users \(username, email, contact_no, created_by\)`).
					WithArgs(req.Username, req.Email, req.ContactNo, managerID).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(newUserID))

				mock.ExpectExec(`INSERT INTO user_type \(user_id, type, created_by\)`).
					WithArgs(newUserID, req.Type, managerID).
					WillReturnError(errors.New("type insert error"))
				mock.ExpectRollback()
			},
			expectedErrContains: "failed to insert employee type",
		},
		{
			name: "failed, error inserting into user_roles",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO users \(username, email, contact_no, created_by\)`).
					WithArgs(req.Username, req.Email, req.ContactNo, managerID).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(newUserID))

				mock.ExpectExec(`INSERT INTO user_type \(user_id, type, created_by\)`).
					WithArgs(newUserID, req.Type, managerID).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec(`INSERT INTO user_roles \(user_id, role, created_by\)`).
					WithArgs(newUserID, managerID).
					WillReturnError(errors.New("role insert error"))
				mock.ExpectRollback()
			},
			expectedErrContains: "failed to insert employee role",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			tc.mockSetup(mock)

			tx, err := sqlxDB.BeginTxx(ctx, nil)
			require.NoError(t, err)

			mockLogger := providers.NewMockZapLoggerProvider(ctrl)
			mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

			repo := &PostgresUserRepository{
				DB:     sqlxDB,
				Logger: mockLogger,
			}

			id, err := repo.CreateNewEmployee(ctx, tx, req, managerID)

			if tc.expectedErrContains != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrContains)
				assert.Equal(t, uuid.Nil, id)
				_ = tx.Rollback()
			} else {
				assert.NoError(t, err)
				assert.Equal(t, newUserID, id)
				err := tx.Commit()
				require.NoError(t, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestInsertUserRole(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	role := "employee"
	createdBy := uuid.New()

	type testCase struct {
		name          string
		mockSetup     func(mock sqlmock.Sqlmock)
		expectSuccess bool
	}

	testCases := []testCase{
		{
			name: "successfully inserts user role",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO user_roles \(role, user_id, created_by\)`).
					WithArgs(role, userID, createdBy).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectSuccess: true,
		},
		{
			name: "fails to insert user role",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO user_roles \(role, user_id, created_by\)`).
					WithArgs(role, userID, createdBy).
					WillReturnError(errors.New("db error"))
				mock.ExpectRollback()
			},
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "postgres")
			tc.mockSetup(mock)

			tx, err := sqlxDB.BeginTxx(ctx, nil)
			require.NoError(t, err)

			mockLogger := providers.NewMockZapLoggerProvider(ctrl)
			mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

			repo := &PostgresUserRepository{
				DB:     sqlxDB,
				Logger: mockLogger,
			}

			err = repo.InsertUserRole(ctx, tx, userID, role, createdBy)

			if tc.expectSuccess {
				assert.NoError(t, err)
				tx.Commit()
			} else {
				assert.Error(t, err)
				_ = tx.Rollback()
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
