package api

import (
	"crypto/rsa"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"

	"github.com/spinup-host/spinup/config"
)

func stringToJWT(key *rsa.PrivateKey, text string) (string, error) {
	// Declare the expiration time of the token
	// here, we have kept it as 2 days
	expirationTime := time.Now().Add(48 * time.Hour)
	// Create the JWT claims, which includes the text and expiry time
	claims := &Claims{
		Text: text,
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
			ExpiresAt: expirationTime.Unix(),
		},
	}
	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodPS512, claims)
	// Create the JWT string
	signedToken, err := token.SignedString(key)
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func ValidateToken(appConfig config.Configuration, authHeader string) (string, error) {
	splitToken := strings.Split(authHeader, "Bearer ")
	if len(splitToken) < 2 {
		return "", fmt.Errorf("cannot validate empty token")
	}
	reqToken := splitToken[1]
	userID, err := JWTToString(appConfig.VerifyKey, reqToken)
	if err != nil {
		return "", err
	}

	if userID == "" {
		return "", errors.New("user ID cannot be blank")
	}
	return userID, nil
}

// Claims is a struct that will be encoded to a JWT.
// We add jwt.StandardClaims as an embedded type, to provide fields like expiry time
type Claims struct {
	Text string `json:"text"`
	jwt.StandardClaims
}

func ValidateUser(appConfig config.Configuration, authHeader string, apiKeyHeader string) (string, error) {
	if authHeader == "" && apiKeyHeader == "" {
		return "", errors.New("no authorization keys found")
	}

	if apiKeyHeader != "" {
		if err := ValidateApiKey(appConfig, apiKeyHeader); err != nil {
			log.Printf("error validating api-key %v", apiKeyHeader)
			return "", errors.New("error validating api-key")
		} else {
			return "testuser", nil
		}
	}

	if authHeader != "" {
		if userId, err := ValidateToken(appConfig, authHeader); err != nil {
			log.Printf("error validating token %v", authHeader)
			return "", errors.New("error validating token")
		} else {
			return userId, nil
		}
	}

	return "testuser", errors.New("could not validate authentication headers")
}

func ValidateApiKey(appConfig config.Configuration, apiKeyHeader string) error {
	if apiKeyHeader != appConfig.Common.ApiKey {
		return errors.New("invalid api key")
	}
	return nil
}

func JWTToString(publicKey *rsa.PublicKey, tokenString string) (string, error) {
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		return publicKey, nil
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, keyFunc)
	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", errors.New("invalid token")
	}
	return claims.Text, nil
}
