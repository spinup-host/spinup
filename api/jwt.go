package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"

	"github.com/spinup-host/spinup/config"
)

func stringToJWT(text string) (string, error) {
	// Declare the expiration time of the token
	// here, we have kept it as 2 days
	log.Println("string to JWTify:", text)
	expirationTime := time.Now().Add(48 * time.Hour)
	// Create the JWT claims, which includes the text and expiry time
	claims := &config.Claims{
		Text: text,
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
			ExpiresAt: expirationTime.Unix(),
		},
	}
	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodPS512, claims)
	// Create the JWT string
	jwt, err := token.SignedString(config.Cfg.SignKey)
	if err != nil {
		return "", err
	}
	return jwt, nil
}

// TODO: vicky to remove this handler after the testing.
func JWT(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	data := query.Get("data")
	if data == "" {
		fmt.Println("data is empty")
	}
	jwtToken, err := stringToJWT(data)
	if err != nil {
		respond(http.StatusInternalServerError, w, map[string]string{
			"message": err.Error(),
		})
		return
	}
	w.Write([]byte(jwtToken))
}

func JWTDecode(w http.ResponseWriter, r *http.Request) {
	jwttoken := r.Header.Get("jwttoken")
	text, err := config.JWTToString(jwttoken)
	if err != nil {
		log.Printf("error jwtdecode %v", err)
		respond(http.StatusInternalServerError, w, map[string]string{
			"message": err.Error()})
		return
	}
	w.Write([]byte(text))
}
