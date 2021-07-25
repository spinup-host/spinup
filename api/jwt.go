package api

import (
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
)

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Create a struct that will be encoded to a JWT.
// We add jwt.StandardClaims as an embedded type, to provide fields like expiry time
type claims struct {
	Text string `json:"text"`
	jwt.StandardClaims
}

func stringToJWT(text string) (string, error) {
	// Declare the expiration time of the token
	// here, we have kept it as 2 days
	expirationTime := time.Now().Add(48 * time.Hour)
	// Create the JWT claims, which includes the text and expiry time
	claims := &claims{
		Text: text,
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
			ExpiresAt: expirationTime.Unix(),
		},
	}
	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodPS512, claims)
	// Create the JWT string
	jwt, err := token.SignedString(signKey)
	if err != nil {
		return "", err
	}
	return jwt, nil
}

// TODO: vicky to remove this handler after the testing
func JWT(w http.ResponseWriter, r *http.Request) {
	plaintext := r.Header.Get("plaintext")
	jwtToken, err := stringToJWT(plaintext)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(jwtToken))
}

func JWTDecode(w http.ResponseWriter, r *http.Request) {
	jwttoken := r.Header.Get("jwttoken")
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		return verifyKey, nil
	}
	claims := &claims{}
	token, err := jwt.ParseWithClaims(jwttoken, claims, keyFunc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !token.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Write([]byte(claims.Text))
}
