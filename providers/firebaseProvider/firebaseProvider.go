package firebaseprovider

import (
	"asset/providers"
	"context"
	"errors"
	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

type firebaseService struct {
	client *firebaseauth.Client
	//app    *firebase.App
}

func NewFirebaseProvider(serviceAccountJSON []byte) (providers.FirebaseProvider, error) {
	opt := option.WithCredentialsJSON(serviceAccountJSON)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, err
	}

	authClient, err := app.Auth(context.Background())
	if err != nil {
		return nil, err
	}

	return &firebaseService{client: authClient}, nil
}

func (f *firebaseService) VerifyIDToken(ctx context.Context, idToken string) (*firebaseauth.Token, error) {
	return f.client.VerifyIDToken(ctx, idToken)
}

func (f *firebaseService) GetUserByUID(ctx context.Context, uid string) (*firebaseauth.UserRecord, error) {
	return f.client.GetUser(ctx, uid)
}

func (f *firebaseService) GetUserByEmail(ctx context.Context, email string) (*firebaseauth.UserRecord, error) {
	return f.client.GetUserByEmail(ctx, email)
}

func (f *firebaseService) CreateUser(ctx context.Context, email string) (*firebaseauth.UserRecord, error) {
	params := new(firebaseauth.UserToCreate)
	if email != "" {
		params = params.Email(email)
	}
	userRecords, err := f.client.CreateUser(context.Background(), params)
	if err != nil {
		return nil, err
	}
	return userRecords, nil
}

func (f *firebaseService) DeleteAuthUser(ctx context.Context, uid string) error {
	return f.client.DeleteUser(ctx, uid)
}

func (f *firebaseService) GetEmailFromUID(ctx context.Context, uid string) (*firebaseauth.UserRecord, error) {
	return f.client.GetUser(ctx, uid)
}

func (f *firebaseService) CustomToken(ctx context.Context, uid string) (string, error) {
	return "", errors.New("testing...")
}

func (f *firebaseService) GetAuthUserID(ctx context.Context, email string) (string, error) {
	user, err := f.client.GetUserByEmail(ctx, email)
	if err != nil {
		logrus.Errorf("GetAuthUserID: error getting User: %v", err)
		return "", err
	}
	return user.UID, nil
}
