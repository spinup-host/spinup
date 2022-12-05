package api

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"log"
	"net/http"
)

type user struct {
	Username  string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	Name      string `json:"name"`
	Token     string `json:"token"`
	JWTToken  string `json:"jwttoken"`
}

type githubAccess struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type GithubAuthHandler struct {
	clientID     string
	clientSecret string
	privateKey   *rsa.PrivateKey
}

func NewGithubAuthHandler(key *rsa.PrivateKey, clientID, clientSecret string) GithubAuthHandler {
	return GithubAuthHandler{
		clientID:     clientID,
		clientSecret: clientSecret,
		privateKey:   key,
	}
}

func (g GithubAuthHandler) GithubAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		respond(http.StatusMethodNotAllowed, w, map[string]string{
			"message": "Invalid Method",
		})
		return
	}
	type userAuth struct {
		Code string `json:"code"`
	}
	log.Println("req::", r.Body)
	var ua userAuth
	// TODO: format this to include best practices https://www.alexedwards.net/blog/how-to-properly-parse-a-json-request-body
	err := json.NewDecoder(r.Body).Decode(&ua)
	if err != nil {
		respond(http.StatusBadRequest, w, map[string]string{
			"message": err.Error(),
		})
		return
	}
	log.Println("ua", ua)

	log.Println("req::", r.Body)
	requestBodyMap := map[string]string{"client_id": g.clientID, "client_secret": g.clientSecret, "code": ua.Code}
	requestBodyJSON, err := json.Marshal(requestBodyMap)
	if err != nil {
		log.Printf("ERROR: marshalling github auth %v", requestBodyMap)
		respond(http.StatusInternalServerError, w, map[string]string{
			"message": "Internal Server Error",
		})
		return
	}
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(requestBodyJSON))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Access-Control-Allow-Origin", "*")
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		respond(http.StatusInternalServerError, w, map[string]string{
			"message": err.Error(),
		})
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Printf("ERROR: likely user code is invalid")
		// http.Error(w, "ERROR: likely user code is invalid", http.StatusInternalServerError)
		// return
	}
	var ghAccessToken githubAccess
	err = json.NewDecoder(res.Body).Decode(&ghAccessToken)
	if err != nil {
		respond(http.StatusInternalServerError, w, map[string]string{
			"message": err.Error(),
		})
		return
	}
	githubUserDataURL := "https://api.github.com/user"
	req, err = http.NewRequest("GET", githubUserDataURL, nil)
	req.Header.Add("Authorization", "token "+ghAccessToken.AccessToken)
	client = http.Client{}
	res, err = client.Do(req)
	if err != nil {
		respond(http.StatusInternalServerError, w, map[string]string{
			"message": err.Error(),
		})
		return
	}
	defer res.Body.Close()
	// TODO: Do we need token field? Token is meant for the backend to communicate to Github
	var u user
	err = json.NewDecoder(res.Body).Decode(&u)
	if err != nil {
		respond(http.StatusInternalServerError, w, map[string]string{
			"message": err.Error(),
		})
		return
	}
	JWTToken, err := stringToJWT(g.privateKey, u.Username)
	if err != nil {
		respond(http.StatusInternalServerError, w, map[string]string{
			"message": err.Error(),
		})
		return
	}
	u.JWTToken = JWTToken
	userJSON, err := json.Marshal(u)
	if err != nil {
		respond(http.StatusInternalServerError, w, map[string]string{
			"message": err.Error(),
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(userJSON)
}
