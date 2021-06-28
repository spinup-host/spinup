package api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func GithubAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	type userAuth struct {
		Code string `json:"stupid"`
	}

	type githubAuth struct {
		clientID     string
		clientSecret string
		code         string
	}
	var ua userAuth
	// TODO: format this to include best practices https://www.alexedwards.net/blog/how-to-properly-parse-a-json-request-body
	err := json.NewDecoder(r.Body).Decode(&ua)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("ua::", ua)
	clientID, ok := os.LookupEnv("CLIENT_ID")
	if !ok {
		log.Fatalf("FATAL: getting environment variable CLIENT_ID")
	}
	clientSecret, ok := os.LookupEnv("CLIENT_SECRET")
	if !ok {
		log.Fatalf("FATAL: getting environment variable CLIENT_SECRET")
	}
	githubAccessTokenURL := "https://github.com/login/oauth/access_token"
	ghAuth := githubAuth{
		clientID:     clientID,
		clientSecret: clientSecret,
		code:         ua.Code,
	}
	ghAuthJSON, err := json.Marshal(ghAuth)
	if err != nil {
		log.Printf("ERROR: marshalling github auth %v", ghAuth)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	res, err := http.Post(githubAccessTokenURL, "application/json", bytes.NewReader(ghAuthJSON))
	// MYSTERY: for some reason error value is nill even though the res status code is 404
	// To catch that case, I have added a if gate checking the response statuscode
	// I am expecting the error value to be non nill when the Post request doesn't succeed
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Printf("ERROR: likely user code is invalid")
		http.Error(w, "ERROR: likely user code is invalid", http.StatusInternalServerError)
		return
	}
	var ghAccessToken string
	err = json.NewDecoder(res.Body).Decode(&ghAccessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	githubUserDataURL := "https://api.github.com/user"
	req, err := http.NewRequest("GET", githubUserDataURL, nil)
	req.Header.Add("Authorization: token ", ghAccessToken)
	client := http.Client{}
	res, err = client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()
	// TODO: Do we need token field? Token is meant for the backend to communicate to Github
	type User struct {
		username  string
		avatarURL string
		name      string
		token     string
	}
	var user User
	json.NewDecoder(res.Body).Decode(&user)
	userJSON, err := json.Marshal(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(userJSON)
}
