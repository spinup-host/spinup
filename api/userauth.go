package api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
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
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	type userAuth struct {
		Code string `json:"code"`
	}
	log.Println("inside githubauth::")
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("ua", ua)
	clientID, ok := os.LookupEnv("CLIENT_ID")
	if !ok {
		log.Fatalf("FATAL: getting environment variable CLIENT_ID")
	}
	clientSecret, ok := os.LookupEnv("CLIENT_SECRET")
	if !ok {
		log.Fatalf("FATAL: getting environment variable CLIENT_SECRET")
	}
	log.Println("req::", r.Body)
	requestBodyMap := map[string]string{"client_id": clientID, "client_secret": clientSecret, "code": ua.Code}
	requestBodyJSON, err := json.Marshal(requestBodyMap)
	if err != nil {
		log.Printf("ERROR: marshalling github auth %v", requestBodyMap)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(requestBodyJSON))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Access-Control-Allow-Origin", "*")
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	githubUserDataURL := "https://api.github.com/user"
	req, err = http.NewRequest("GET", githubUserDataURL, nil)
	req.Header.Add("Authorization", "token "+ghAccessToken.AccessToken)
	client = http.Client{}
	res, err = client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()
	// TODO: Do we need token field? Token is meant for the backend to communicate to Github
	var u user
	err = json.NewDecoder(res.Body).Decode(&u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	JWTToken, err := stringToJWT(u.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u.JWTToken = JWTToken
	userJSON, err := json.Marshal(u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(userJSON)
}
