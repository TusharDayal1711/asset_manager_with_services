package firebaseprovider

import (
	"context"
	firebaseauth "firebase.google.com/go/auth"
)

type FirebaseProvider interface {
	VerifyIDToken(ctx context.Context, idToken string) (*firebaseauth.Token, error)
	GetUserByUID(ctx context.Context, uid string) (*firebaseauth.UserRecord, error)
}
