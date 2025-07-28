package firebaseprovider

import (
	"asset/providers"
	"context"
	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

type firebaseService struct {
	client *firebaseauth.Client
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
