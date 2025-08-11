package userservice

import (
	"context"
	"errors"
	"testing"

	"asset/providers"

	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestGetDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockUserRepository(ctrl)
	mockLogger := providers.NewMockZapLoggerProvider(ctrl)

	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

	service := &userServiceStruct{
		repo:   mockRepo,
		logger: mockLogger,
	}

	ctx := context.Background()
	userID := uuid.New()
	contact := "1234567890"
	typ := "employee"

	expectedDashboard := UserDashboardRes{
		ID:        userID.String(),
		Username:  "testuser",
		Email:     "test.user40@remotestate.com",
		ContactNo: &contact,
		Type:      &typ,
		Roles:     []string{"employee"},
		AssignedAssets: []AssetDetails{
			{ID: userID, Type: "Laptop"},
		},
	}

	t.Run("success", func(t *testing.T) {
		mockRepo.EXPECT().
			GetUserDashboardById(ctx, userID).
			Return(expectedDashboard, nil)

		dashboard, err := service.GetDashboard(ctx, userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedDashboard, dashboard)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo.EXPECT().
			GetUserDashboardById(ctx, userID).
			Return(UserDashboardRes{}, errors.New("db error"))

		_, err := service.GetDashboard(ctx, userID)

		assert.Error(t, err)
	})

}

func TestUpdateEmployee(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockUserRepository(ctrl)
	mockLogger := providers.NewMockZapLoggerProvider(ctrl)

	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

	service := &userServiceStruct{
		repo:   mockRepo,
		logger: mockLogger,
	}

	ctx := context.Background()
	managerID := uuid.New()
	employeeID := uuid.New()

	tests := []struct {
		name             string
		req              UpdateEmployeeReq
		mockRepoBehavior func()
		expectError      bool
	}{
		{
			name: "success",
			req: UpdateEmployeeReq{
				UserID:    employeeID,
				Username:  "test user41",
				Email:     "test.user41@example.com",
				ContactNo: "9876543210",
			},
			mockRepoBehavior: func() {
				mockRepo.EXPECT().UpdateEmployeeInfo(ctx, gomock.Any(), managerID).
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "repository error",
			req: UpdateEmployeeReq{
				UserID:    employeeID,
				Username:  "test user41",
				Email:     "test.user41@example.com",
				ContactNo: "1234567890",
			},
			mockRepoBehavior: func() {
				mockRepo.EXPECT().
					UpdateEmployeeInfo(ctx, gomock.Any(), managerID).
					Return(errors.New("db error"))
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockRepoBehavior()

			err := service.UpdateEmployee(ctx, tc.req, managerID)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	userID := uuid.New()
	userEmail := "test.user@remotestate.com"
	userUID := "firebase-uid"

	tests := []struct {
		name             string
		managerRole      string
		setupMocks       func(repo *MockUserRepository, firebase *providers.MockFirebaseProvider)
		expectedErrorMsg string
	}{
		{
			name:        "success delete by admin",
			managerRole: "admin",
			setupMocks: func(repo *MockUserRepository, firebase *providers.MockFirebaseProvider) {
				repo.EXPECT().GetUserRoleById(ctx, userID).Return("employee", nil)
				repo.EXPECT().GetEmailByUserID(ctx, userID).Return(userEmail, nil)
				firebase.EXPECT().GetUserByEmail(ctx, userEmail).Return(&firebaseauth.UserRecord{
					UserInfo: &firebaseauth.UserInfo{
						UID: userUID,
					},
				}, nil)
				firebase.EXPECT().DeleteAuthUser(ctx, userUID).Return(nil)
				repo.EXPECT().DeleteUserByID(ctx, userID).Return(nil)
			},
			expectedErrorMsg: "",
		},

		{
			name:        "unauthorized user",
			managerRole: "employee",
			setupMocks: func(repo *MockUserRepository, firebase *providers.MockFirebaseProvider) {
				repo.EXPECT().GetUserRoleById(ctx, userID).Return("admin", nil)
			},
			expectedErrorMsg: "only admin can delete admin or manager roles",
		},
		{
			name:        "failed to get user role",
			managerRole: "admin",
			setupMocks: func(repo *MockUserRepository, firebase *providers.MockFirebaseProvider) {
				repo.EXPECT().GetUserRoleById(ctx, userID).Return("", errors.New("db error"))
			},
			expectedErrorMsg: "db error",
		},
		{
			name:        "failed to get user email",
			managerRole: "admin",
			setupMocks: func(repo *MockUserRepository, firebase *providers.MockFirebaseProvider) {
				repo.EXPECT().GetUserRoleById(ctx, userID).Return("employee", nil)
				repo.EXPECT().GetEmailByUserID(ctx, userID).Return("", errors.New("user not found"))
			},
			expectedErrorMsg: "failed to get user email from user table",
		},
		{
			name:        "firebase user not found",
			managerRole: "admin",
			setupMocks: func(repo *MockUserRepository, firebase *providers.MockFirebaseProvider) {
				repo.EXPECT().GetUserRoleById(ctx, userID).Return("employee", nil)
				repo.EXPECT().GetEmailByUserID(ctx, userID).Return(userEmail, nil)
				firebase.EXPECT().GetUserByEmail(ctx, userEmail).Return(nil, errors.New("not found"))
			},
			expectedErrorMsg: "failed to get user UID from firebase user table",
		},
		{
			name:        "firebase delete failure",
			managerRole: "admin",
			setupMocks: func(repo *MockUserRepository, firebase *providers.MockFirebaseProvider) {
				repo.EXPECT().GetUserRoleById(ctx, userID).Return("employee", nil)
				repo.EXPECT().GetEmailByUserID(ctx, userID).Return(userEmail, nil)
				firebase.EXPECT().GetUserByEmail(ctx, userEmail).Return(&firebaseauth.UserRecord{
					UserInfo: &firebaseauth.UserInfo{
						UID: userUID,
					},
				}, nil)
				firebase.EXPECT().DeleteAuthUser(ctx, userUID).Return(errors.New("firebase error"))
			},
			expectedErrorMsg: "failed to delete auth user from firebase",
		},
		{
			name:        "repo delete failure",
			managerRole: "admin",
			setupMocks: func(repo *MockUserRepository, firebase *providers.MockFirebaseProvider) {
				repo.EXPECT().GetUserRoleById(ctx, userID).Return("employee", nil)
				repo.EXPECT().GetEmailByUserID(ctx, userID).Return(userEmail, nil)
				firebase.EXPECT().GetUserByEmail(ctx, userEmail).Return(&firebaseauth.UserRecord{
					UserInfo: &firebaseauth.UserInfo{
						UID: userUID,
					},
				}, nil)
				firebase.EXPECT().DeleteAuthUser(ctx, userUID).Return(nil)
				repo.EXPECT().DeleteUserByID(ctx, userID).Return(errors.New("delete failed"))
			},
			expectedErrorMsg: "delete failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockUserRepository(ctrl)
			mockFirebase := providers.NewMockFirebaseProvider(ctrl)
			mockLogger := providers.NewMockZapLoggerProvider(ctrl)
			mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

			tc.setupMocks(mockRepo, mockFirebase)

			service := &userServiceStruct{
				repo:     mockRepo,
				logger:   mockLogger,
				firebase: mockFirebase,
			}

			err := service.DeleteUser(ctx, userID, tc.managerRole)

			if tc.expectedErrorMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrorMsg)
			}
		})
	}
}

func TestUserLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	email := "test.user41@remotestate.com"
	userID := uuid.New()
	role := "employee"
	accessToken := "Access_Token"
	refreshToken := "Refresh_Token"

	tests := []struct {
		name         string
		req          PublicUserReq
		mockSetups   func(repo *MockUserRepository, authMiddleware *providers.MockAuthMiddlewareService)
		expectSucess bool
	}{
		{
			name: "user login success",
			req:  PublicUserReq{Email: email},
			mockSetups: func(repo *MockUserRepository, authMiddleware *providers.MockAuthMiddlewareService) {
				repo.EXPECT().GetUserByEmail(ctx, email).Return(userID, nil)
				repo.EXPECT().GetUserRoleById(ctx, userID).Return(role, nil)
				authMiddleware.EXPECT().GenerateJWT(userID.String(), []string{role}).Return(accessToken, nil)
				authMiddleware.EXPECT().GenerateRefreshToken(userID.String()).Return(refreshToken, nil)
			},
			expectSucess: true,
		},
		{
			name: "user email not found",
			req:  PublicUserReq{Email: email},
			mockSetups: func(repo *MockUserRepository, authMiddleware *providers.MockAuthMiddlewareService) {
				repo.EXPECT().GetUserByEmail(ctx, email).Return(uuid.Nil, errors.New("user not found"))
			},
			expectSucess: false,
		},
		{
			name: "user role not found",
			req:  PublicUserReq{Email: email},
			mockSetups: func(repo *MockUserRepository, authMiddleware *providers.MockAuthMiddlewareService) {
				repo.EXPECT().GetUserByEmail(ctx, email).Return(userID, nil)
				repo.EXPECT().GetUserRoleById(ctx, userID).Return("", errors.New("user role not found"))
			},
			expectSucess: false,
		},
		{
			name: "failed to generate access token",
			req:  PublicUserReq{Email: email},
			mockSetups: func(repo *MockUserRepository, authMiddleware *providers.MockAuthMiddlewareService) {
				repo.EXPECT().GetUserByEmail(ctx, email).Return(userID, nil)
				repo.EXPECT().GetUserRoleById(ctx, userID).Return(role, nil)
				authMiddleware.EXPECT().GenerateJWT(userID.String(), []string{role}).Return("", errors.New("failed to generate access token"))
			},
			expectSucess: false,
		},
		{
			name: "failed to generate refresh token",
			req:  PublicUserReq{Email: email},
			mockSetups: func(repo *MockUserRepository, authMiddleware *providers.MockAuthMiddlewareService) {
				repo.EXPECT().GetUserByEmail(ctx, email).Return(userID, nil)
				repo.EXPECT().GetUserRoleById(ctx, userID).Return(role, nil)
				authMiddleware.EXPECT().GenerateJWT(userID.String(), []string{role}).Return(accessToken, nil)
				authMiddleware.EXPECT().GenerateRefreshToken(userID.String()).Return("", errors.New("failed to generate refresh token"))
			},
			expectSucess: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockUserRepository(ctrl)
			mockAuthMiddleware := providers.NewMockAuthMiddlewareService(ctrl)
			mockLogger := providers.NewMockZapLoggerProvider(ctrl)
			mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

			tc.mockSetups(mockRepo, mockAuthMiddleware)

			service := &userServiceStruct{
				repo:           mockRepo,
				AuthMiddleware: mockAuthMiddleware,
				logger:         mockLogger,
			}

			_, _, _, err := service.UserLogin(ctx, tc.req)

			if tc.expectSucess {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
