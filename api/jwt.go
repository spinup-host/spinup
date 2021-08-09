package api

import (
	"errors"
	"fmt"
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
	log.Println("string to JWTify:", text)
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

func JWTToString(tokenString string) (string, error) {
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		return verifyKey, nil
	}
	claims := &claims{}
	log.Println("JWT to string:", tokenString)
	token, err := jwt.ParseWithClaims(tokenString, claims, keyFunc)
	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", errors.New("invalid token")
	}
	log.Println("claims", claims.Text)
	return claims.Text, nil
}

// TODO: vicky to remove this handler after the testing
func JWT(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	data := query.Get("data")
	if data == "" {
		fmt.Println("data is empty")
	}
	jwtToken, err := stringToJWT(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(jwtToken))
}

func JWTDecode(w http.ResponseWriter, r *http.Request) {
	jwttoken := r.Header.Get("jwttoken")
	text, err := JWTToString(jwttoken)
	if err != nil {
		log.Printf("error jwtdecode %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(text))
}
