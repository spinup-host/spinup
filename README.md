# SPINUP

An alternative to RDS

### How to run

It requires a bunch of environment variables. You can export them and run using

```
export CF_API_EMAIL=yourcfemail && export CF_API_KEY=yourcfapikey && export CF_ACCOUNT_ID=yourcfaccountid && export CF_ZONE_ID=yourcfzonid && export SPINUP_PROJECT_DIR=/Your/Project/Dir && export ARCHITECTURE=architecture && go run main.go
```

On another terminal create a POST request using
```
curl -X POST http://localhost:8090/createservice \
    -H "Content-Type: application/json" \
    --data '{"name": "postgres","duration": 200,"resource":{"memory": "32MB","storage": 200,"version": {"maj":9,"min":6}},"userid": "replaceme"}'
```

## Endpoints

### Github Auth

- URL

/githubAuth

- Method:

`POST`

- Data Params

```
{
    "code": "githubcode"
}
```

- Success Response:
    - Code: 200
    - Content: `{"login":"","avatar_url":"","name":"","token":""}`

- Error Response:

    - Code: 404 NOT FOUND

        Content: `{ error : "User doesn't exist" }`
    
    OR

    - Code: 401 UNAUTHORIZED

        Content: `{ error : "You are unauthorized to make this request." }`