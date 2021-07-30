# SPINUP

An open source alternative to [AWS RDS](https://aws.amazon.com/rds/), [Cloud SQL](https://cloud.google.com/sql). 

## Arhictecture

The idea is simple. Spinup creates multiple containers through docker-compose. 
Spinup can be deployed anywhere. Only requirement is docker-compose. It can run on anywhere [Digital Ocean droplet](https://www.digitalocean.com/products/droplets/), [Azure Compute](https://azure.microsoft.com/en-us/product-categories/compute/), [Oracle Compute](https://www.oracle.com/cloud/compute/), [Raspberry Pi](https://www.raspberrypi.org/) etc. 

We are currently using Github Authentication. We should be able to support other authentication methods.

Currently we only support Postgres dbms, but we should be able to support other open source databases like [MySQL](https://www.mysql.com/), [MariaDB](https://mariadb.org/) etc.

![architecture](architecture.jpeg)
### How to run

It requires a bunch of environment variables. You can export them and run using

```
export SPINUP_PROJECT_DIR=/Your/Project/Dir && export ARCHITECTURE=architecture && go run main.go
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

### Create Service

- URL

/createservice

- Method:

`POST`

- Data Params

```
{
    "code": "githubcode"
    "name": "postgres",
    "duration": 200,
    "resource":
        {
            "memory": "32MB",
            "storage": 200,
            "version": 
                {"maj":9,"min":6}
        },
    "userid": "replaceme"
}
```

- Success Response:
    - Code: 200

- Error Response:

    - Code: 500 INTERNALSERVER ERROR