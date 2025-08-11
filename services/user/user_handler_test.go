package userservice

import (
	"asset/providers"
	"asset/utils"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/golang/mock/gomock"
)

func TestGetEmployeesWithFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	//mock services
	mockService := NewMockUserService(ctrl)
	mockAuth := providers.NewMockAuthMiddlewareService(ctrl)
	mockLogger := providers.NewMockZapLoggerProvider(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

	//inject mock serivces in userhandler
	handler := &UserHandler{
		Service:        mockService,
		Logger:         mockLogger,
		AuthMiddleware: mockAuth,
	}

	managerID := uuid.New()
	contact := "9876543210"

	filteredEmployeesRes := []EmployeeResponseModel{
		{
			ID:             uuid.New().String(),
			Username:       "test user 35",
			Email:          "test.user35@remotestate.com",
			ContactNo:      &contact,
			EmployeeType:   "full_time",
			AssignedAssets: []string{"Laptop", "Mobile"},
		},
		{
			ID:             uuid.New().String(),
			Username:       "test user 36",
			Email:          "test.user36@remotestate.com",
			ContactNo:      &contact,
			EmployeeType:   "intern",
			AssignedAssets: []string{},
		},
	}

	testCases := []struct {
		name               string
		queryParams        string
		systemUserID       string
		authRoles          []string
		authErr            error
		expectServiceCall  bool
		mockServiceReturn  []EmployeeResponseModel
		mockServiceErr     error
		expectedStatusCode int
	}{
		{
			name:               "success, valid role and filter",
			queryParams:        "?page =1&limit=10&search=&type=full_time&role=employee&=",
			systemUserID:       managerID.String(),
			authRoles:          []string{"employee_manager"},
			expectServiceCall:  true,
			mockServiceReturn:  filteredEmployeesRes,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "unauthorized, auth middleware fails",
			authErr:            errors.New("unauthorized"),
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:               "forbidden, unauthorized role",
			systemUserID:       managerID.String(),
			authRoles:          []string{"employee"},
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "internal server error",
			queryParams:        "?search=test",
			systemUserID:       managerID.String(),
			authRoles:          []string{"admin"},
			expectServiceCall:  true,
			mockServiceErr:     errors.New("internal server error"),
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/user/get-employees"+tc.queryParams, nil)
			respRecorder := httptest.NewRecorder()

			//mock middleware test
			if tc.authErr != nil {
				mockAuth.EXPECT().
					GetUserAndRolesFromContext(req).
					Return("", nil, tc.authErr)
			} else {
				mockAuth.EXPECT().
					GetUserAndRolesFromContext(req).
					Return(tc.systemUserID, tc.authRoles, nil)
			}

			//mock service
			if tc.expectServiceCall {
				expectedFilter := EmployeeFilter{
					SearchText:   req.URL.Query().Get("search"),
					IsSearchText: req.URL.Query().Get("search") != "",
					Type:         strings.Split(req.URL.Query().Get("type"), ","),
					Role:         strings.Split(req.URL.Query().Get("role"), ","),
					AssetStatus:  strings.Split(req.URL.Query().Get("asset_status"), ","),
				}
				expectedFilter.Limit, expectedFilter.Offset = utils.GetPageLimitAndOffset(req)

				mockService.EXPECT().
					GetEmployeesWithFilters(gomock.Any(), expectedFilter).
					Return(tc.mockServiceReturn, tc.mockServiceErr)
			}

			//call handler
			handler.GetEmployeesWithFilters(respRecorder, req)

			assert.Equal(t, tc.expectedStatusCode, respRecorder.Code)

			if tc.expectedStatusCode == http.StatusOK {
				var res map[string][]EmployeeResponseModel
				err := jsoniter.NewDecoder(respRecorder.Body).Decode(&res)
				assert.NoError(t, err)
				assert.Equal(t, tc.mockServiceReturn, res["employees"])
			}
		})
	}
}

func TestPublicRegisterThroughFirebase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := NewMockUserService(ctrl)
	mockLogger := providers.NewMockZapLoggerProvider(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewExample()).AnyTimes()

	handler := &UserHandler{
		Service: mockService,
		Logger:  mockLogger,
	}

	mockToken := "mock_firebase_token"

	successResp := &FirebaseRegistrationResponse{
		UserID:      uuid.New(),
		FirebaseUID: "firebase_uid_123454232323",
	}

	testCases := []struct {
		name                    string
		authHeader              string
		mockServiceResponseBody *FirebaseRegistrationResponse
		mockServiceErr          error
		expectedStatusCode      int
		expectedMessage         string
	}{
		{
			name:                    "success",
			authHeader:              "Bearer " + mockToken,
			mockServiceResponseBody: successResp,
			expectedStatusCode:      http.StatusCreated,
			expectedMessage:         "register through firebase successful",
		},
		{
			name:               "missing Authorization header",
			authHeader:         "",
			expectedStatusCode: http.StatusUnauthorized,
			expectedMessage:    "unauthorized",
		},
		{
			name:               "invalid firebase token",
			authHeader:         "Bearer " + mockToken,
			mockServiceErr:     errors.New("invalid firebase token"),
			expectedStatusCode: http.StatusUnauthorized,
			expectedMessage:    "unauthorized",
		},
		{
			name:               "user already exists",
			authHeader:         "Bearer " + mockToken,
			mockServiceErr:     errors.New("user already exists"),
			expectedStatusCode: http.StatusConflict,
			expectedMessage:    "user already exists",
		},
		{
			name:               "internal service error",
			authHeader:         "Bearer " + mockToken,
			mockServiceErr:     errors.New("unexpected failure"),
			expectedStatusCode: http.StatusInternalServerError,
			expectedMessage:    "registration failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/auth/firebase-register", nil)
			req.Header.Set("Authorization", tc.authHeader)
			res := httptest.NewRecorder()

			if tc.authHeader != "" {
				mockService.
					EXPECT().
					FirebaseUserRegistration(gomock.Any(), mockToken).
					Return(tc.mockServiceResponseBody, tc.mockServiceErr)
			}

			handler.PublicRegisterThroughFirebase(res, req)

			assert.Equal(t, tc.expectedStatusCode, res.Code)

			var body map[string]interface{}
			_ = jsoniter.NewDecoder(res.Body).Decode(&body)

			if tc.expectedStatusCode == http.StatusCreated {
				assert.Equal(t, tc.expectedMessage, body["message"])
				assert.Equal(t, successResp.UserID.String(), body["userId"])
				assert.Equal(t, successResp.FirebaseUID, body["firebaseUID"])
			} else {
				assert.Equal(t, tc.expectedMessage, body["message"])
			}
		})
	}
}

func TestPublicRegister(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name                string
		reqBody             PublicUserReq
		mockServiceProvider func(mockUserService *MockUserService)
		expectedStatusCode  int
	}{
		{
			name: "successful",
			reqBody: PublicUserReq{
				Email: "test.user27@remotestate.com",
			},
			mockServiceProvider: func(mockUserService *MockUserService) {
				mockUserService.EXPECT().
					PublicRegister(gomock.Any(), PublicUserReq{
						Email: "test.user27@remotestate.com",
					}).
					Return(uuid.New(), "User created", nil)
			},
			expectedStatusCode: http.StatusCreated,
		},
		{
			name: "invalid input 400 Bad Request",
			reqBody: PublicUserReq{
				Email: "",
			},
			mockServiceProvider: func(mockUserService *MockUserService) {
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "service failure",
			reqBody: PublicUserReq{
				Email: "test.user27@remotestate.com",
			},
			mockServiceProvider: func(mockUserService *MockUserService) {
				mockUserService.EXPECT().
					PublicRegister(gomock.Any(), PublicUserReq{
						Email: "test.user27@remotestate.com",
					}).
					Return(uuid.Nil, "", fmt.Errorf("failed to create user status code 400"))
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			//mock useservice
			mockUserService := NewMockUserService(ctrl)

			mockLogger := providers.NewMockZapLoggerProvider(ctrl)
			mockLogger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
			tc.mockServiceProvider(mockUserService)

			handler := &UserHandler{
				Service: mockUserService,
				Logger:  mockLogger,
			}

			body, err := jsoniter.Marshal(tc.reqBody)
			assert.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			//call the handler
			handler.PublicRegister(w, req)

			//check assertion
			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tc.expectedStatusCode, res.StatusCode)
		})
	}
}

func TestUserLoginHandler(t *testing.T) {
	tests := []struct {
		name                 string
		reqBody              PublicUserReq
		mockServiceProvider  func(mockUserService *MockUserService)
		expectedStatusCode   int
		expectResponseFields map[string]bool
	}{
		{
			name: "login successful ",
			reqBody: PublicUserReq{
				Email: "test.user27@remotestate.com",
			},
			mockServiceProvider: func(mockUserService *MockUserService) {
				mockUserService.EXPECT().
					UserLogin(gomock.Any(), PublicUserReq{
						Email: "test.user27@remotestate.com",
					}).
					Return(uuid.New(), "access_token", "refresh_token", nil)
			},
			expectedStatusCode: http.StatusOK,
			expectResponseFields: map[string]bool{
				"user_id":       true,
				"access_token":  true,
				"refresh_token": true,
			},
		},
		{
			name: "Invalid input",
			reqBody: PublicUserReq{
				Email: "",
			},
			mockServiceProvider: func(mockUserService *MockUserService) {

			},
			expectedStatusCode:   http.StatusBadRequest,
			expectResponseFields: map[string]bool{},
		},
		{
			name: "Service failure",
			reqBody: PublicUserReq{
				Email: "test.user27@remotestate.com",
			},
			mockServiceProvider: func(mockUserService *MockUserService) {
				mockUserService.EXPECT().
					UserLogin(gomock.Any(), PublicUserReq{Email: "test.user27@remotestate.com"}).
					Return(uuid.Nil, "", "", fmt.Errorf("login failed"))
			},
			expectedStatusCode:   http.StatusUnauthorized,
			expectResponseFields: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := NewMockUserService(ctrl)
			mockLogger := providers.NewMockZapLoggerProvider(ctrl)
			mockLogger.EXPECT().GetLogger().Return(zap.NewExample()).AnyTimes()

			tt.mockServiceProvider(mockService)

			handler := &UserHandler{
				Service: mockService,
				Logger:  mockLogger,
			}

			reqBytes, _ := jsoniter.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(reqBytes))
			req.Header.Set("Content-Type", "application/json")
			resRecorder := httptest.NewRecorder()

			handler.UserLogin(resRecorder, req)

			assert.Equal(t, tt.expectedStatusCode, resRecorder.Code)

			if tt.expectedStatusCode == http.StatusOK {
				var result map[string]interface{}
				err := jsoniter.Unmarshal(resRecorder.Body.Bytes(), &result)
				assert.NoError(t, err)

				for field := range tt.expectResponseFields {
					assert.Contains(t, result, field)
				}
			}
		})
	}
}

func TestGoogleAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	idToken := "uid-token"

	tests := []struct {
		name               string
		idToken            string
		mockService        func(service *MockUserService, logger *providers.MockZapLoggerProvider)
		expectedStatusCode int
		expectUserID       uuid.UUID
		expectAccessToken  string
		expectRefreshToken string
	}{
		{
			name:    "Success",
			idToken: idToken,
			mockService: func(service *MockUserService, logger *providers.MockZapLoggerProvider) {
				logger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
				service.EXPECT().
					GoogleAuth(gomock.Any(), idToken).
					Return(uuid.New(), "access_token", "refresh_token", nil)
			},
			expectedStatusCode: http.StatusOK,
			expectAccessToken:  "access_token",
			expectRefreshToken: "refresh_token",
		},
		{
			name:    "Missing Authorization Header",
			idToken: "",
			mockService: func(service *MockUserService, logger *providers.MockZapLoggerProvider) {
				logger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:    "Invalid Token",
			idToken: idToken,
			mockService: func(service *MockUserService, logger *providers.MockZapLoggerProvider) {
				logger.EXPECT().GetLogger().Return(zap.NewNop()).AnyTimes()

				service.EXPECT().
					GoogleAuth(gomock.Any(), idToken).
					Return(uuid.Nil, "", "", errors.New("invalid token"))
			},
			expectedStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockService := NewMockUserService(ctrl)
			mockLogger := providers.NewMockZapLoggerProvider(ctrl)

			tc.mockService(mockService, mockLogger)

			handler := &UserHandler{
				Service: mockService,
				Logger:  mockLogger,
			}

			req := httptest.NewRequest(http.MethodPost, "/api/user/google-auth", bytes.NewReader(nil))
			if tc.idToken != "" {
				req.Header.Set("Authorization", "Bearer "+tc.idToken)
			}
			ResponseRecorder := httptest.NewRecorder()

			handler.GoogleAuth(ResponseRecorder, req)

			assert.Equal(t, tc.expectedStatusCode, ResponseRecorder.Code)

			if tc.expectedStatusCode == http.StatusOK {
				var resp map[string]interface{}
				err := jsoniter.Unmarshal(ResponseRecorder.Body.Bytes(), &resp)
				assert.NoError(t, err)

				assert.Equal(t, tc.expectAccessToken, resp["access_token"])
				assert.Equal(t, tc.expectRefreshToken, resp["refresh_token"])
				assert.NotEmpty(t, resp["user_id"])
			}
		})
	}
}

func TestRegisterEmployeeByManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := NewMockUserService(ctrl)
	mockAuth := providers.NewMockAuthMiddlewareService(ctrl)
	mockLogger := providers.NewMockZapLoggerProvider(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewExample()).AnyTimes()

	handler := &UserHandler{
		Service:        mockService,
		Logger:         mockLogger,
		AuthMiddleware: mockAuth,
	}

	managerID := uuid.New()
	userID := uuid.New()

	testCases := []struct {
		name               string
		requestBody        ManagerRegisterReq
		systemUserID       string
		authRoles          []string
		authErr            error
		expectServiceCall  bool
		serviceReturnID    uuid.UUID
		serviceErr         error
		expectedStatusCode int
	}{
		{
			name: "success",
			requestBody: ManagerRegisterReq{
				Username: "test User28", Email: "test.user28@remotestate.com", ContactNo: "784948585", Type: "full_time",
			},
			systemUserID:       managerID.String(),
			authRoles:          []string{"employee_manager"},
			expectServiceCall:  true,
			serviceReturnID:    userID,
			expectedStatusCode: http.StatusCreated,
		},
		{
			name: "unauthorized user",
			requestBody: ManagerRegisterReq{
				Username: "test user 29", Email: "test.user29@remotestate.com", ContactNo: "12345678908976567892intern", Type: "full_time",
			},
			systemUserID:       managerID.String(),
			authRoles:          []string{"employee"},
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name: "invalid req body",
			requestBody: ManagerRegisterReq{
				Email: "test.user30@gmail.com", ContactNo: "3456789765",
			},
			systemUserID:       managerID.String(),
			authRoles:          []string{"admin"},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "auth middleware error",
			requestBody: ManagerRegisterReq{
				Username: "test user31", Email: "test.user31@remotestate.com", ContactNo: "8976545678", Type: "full_time",
			},
			authErr:            errors.New("unauthorized"),
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name: "failed to create user",
			requestBody: ManagerRegisterReq{
				Username: "test user32", Email: "test.user32@remotestate.com", ContactNo: "8976567829", Type: "intern",
			},
			systemUserID:       managerID.String(),
			authRoles:          []string{"admin"},
			expectServiceCall:  true,
			serviceErr:         errors.New("failed to create user"),
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := jsoniter.Marshal(tc.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/user/register-employee", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			ResponseRecorder := httptest.NewRecorder()

			if tc.authErr != nil {
				mockAuth.EXPECT().GetUserAndRolesFromContext(req).
					Return("", nil, tc.authErr)
			} else {
				mockAuth.EXPECT().GetUserAndRolesFromContext(req).
					Return(tc.systemUserID, tc.authRoles, nil)
			}

			if tc.expectServiceCall {
				mockService.EXPECT().
					RegisterEmployeeByManager(gomock.Any(), tc.requestBody, gomock.Any()).
					Return(tc.serviceReturnID, tc.serviceErr)
			}

			handler.RegisterEmployeeByManager(ResponseRecorder, req)

			assert.Equal(t, tc.expectedStatusCode, ResponseRecorder.Code)

			if ResponseRecorder.Code == http.StatusCreated {
				var res map[string]interface{}
				err := jsoniter.NewDecoder(ResponseRecorder.Body).Decode(&res)
				assert.NoError(t, err)
				assert.Contains(t, res, "user UUID")
			}
		})
	}
}

func TestDeleteUserHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := NewMockUserService(ctrl)
	mockAuth := providers.NewMockAuthMiddlewareService(ctrl)
	mockLogger := providers.NewMockZapLoggerProvider(ctrl)
	mockLogger.EXPECT().GetLogger().Return(zap.NewExample()).AnyTimes()

	handler := &UserHandler{
		Service:        mockService,
		Logger:         mockLogger,
		AuthMiddleware: mockAuth,
	}

	userID := uuid.New()

	testCases := []struct {
		name               string
		queryUserID        string
		systemUserID       string
		authRoles          []string
		authErr            error
		expectServiceCall  bool
		serviceErr         error
		expectedStatusCode int
	}{
		{
			name:               "successfully deletes user",
			queryUserID:        userID.String(),
			systemUserID:       uuid.New().String(),
			authRoles:          []string{"admin"},
			expectServiceCall:  true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "missing user_id query param",
			queryUserID:        "",
			systemUserID:       uuid.New().String(),
			authRoles:          []string{"admin"},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "invalid user_id format",
			queryUserID:        "invalid-uuid",
			systemUserID:       uuid.New().String(),
			authRoles:          []string{"admin"},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "unauthorized due to role",
			queryUserID:        userID.String(),
			systemUserID:       uuid.New().String(),
			authRoles:          []string{"employee"},
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "unauthorized due to missing context",
			queryUserID:        userID.String(),
			authErr:            errors.New("unauthorized"),
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:               "internal error deleting user",
			queryUserID:        userID.String(),
			systemUserID:       uuid.New().String(),
			authRoles:          []string{"admin"},
			expectServiceCall:  true,
			serviceErr:         errors.New("delete failed"),
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/user/delete?user_id="+tc.queryUserID, nil)
			res := httptest.NewRecorder()

			if tc.authErr != nil {
				mockAuth.EXPECT().GetUserAndRolesFromContext(req).
					Return("", nil, tc.authErr)
			} else {
				mockAuth.EXPECT().GetUserAndRolesFromContext(req).
					Return(tc.systemUserID, tc.authRoles, nil)
			}

			if tc.expectServiceCall {
				mockService.EXPECT().
					DeleteUser(gomock.Any(), gomock.Any(), tc.authRoles[0]).
					Return(tc.serviceErr)
			}

			handler.DeleteUser(res, req)

			assert.Equal(t, tc.expectedStatusCode, res.Code)

			if res.Code == http.StatusOK {
				var body map[string]string
				err := jsoniter.NewDecoder(res.Body).Decode(&body)
				assert.NoError(t, err)
				assert.Equal(t, "user deleted successfully", body["message"])
			}
		})
	}
}
