package api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/spinup-host/spinup/config"
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

func GithubAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		respond(http.StatusMethodNotAllowed, w, map[string]string{
			"message": "Invalid Method",
		})
		return
	}
	type userAuth struct {
		Code string `json:"code"`
	}
	type githubAuth struct {
		Code         string `json:"code"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
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
	clientID := config.Cfg.Common.ClientID

	clientSecret := config.Cfg.Common.ClientSecret

	log.Println("req::", r.Body)
	requestBodyMap := map[string]string{"client_id": clientID, "client_secret": clientSecret, "code": ua.Code}
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
	JWTToken, err := stringToJWT(u.Username)
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

func AltAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		respond(http.StatusMethodNotAllowed, w, map[string]string{
			"message": "Invalid Method",
		})
		return
	}
	apiKeyHeader := r.Header.Get("x-api-key")
	_, err := config.ValidateUser("", apiKeyHeader)

	response := map[string]string{}
	var code int
	if err != nil {
		response["message"] = err.Error()
		code = http.StatusUnauthorized
	} else {
		response["message"] = "valid API key"
		code = http.StatusOK
	}
	respond(code, w, response)
	return
}
