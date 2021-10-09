# SPINUP

An open source alternative to [AWS RDS](https://aws.amazon.com/rds/), [Cloud SQL](https://cloud.google.com/sql). 

## Arhictecture

The idea is simple. Spinup creates multiple containers through docker-compose. 
Spinup can be deployed anywhere. System requirements are Go and docker-compose. Once this [issue](https://github.com/spinup-host/spinup/issues/45) has been fixed then you don't need to have Go installed. It can run on [Digital Ocean droplet](https://www.digitalocean.com/products/droplets/), [Azure Compute](https://azure.microsoft.com/en-us/product-categories/compute/), [Oracle Compute](https://www.oracle.com/cloud/compute/), [Raspberry Pi](https://www.raspberrypi.org/) etc. 

We are currently using Github Authentication. We should be able to support other authentication methods.

Currently we only support Postgres dbms, but we should be able to support other open source databases like [MySQL](https://www.mysql.com/), [MariaDB](https://mariadb.org/) etc.

![architecture](architecture.jpeg)
### How to run

#### Requirement for JWT
We use JWT for verification. You need to have a private and public key that you can create using OpenSSL:

**To create a private key**
```
visi@visis-MacBook-Pro spinup % openssl genrsa -out /${SPINUP_PROJECT_DIR}/app.rsa 4096 
Generating RSA private key, 4096 bit long modulus
...++
...................++
e is 65537 (0x10001)
```

**To create a public key**
```
visi@visis-MacBook-Pro spinup % openssl rsa -in /${SPINUP_PROJECT_DIR}/app.rsa -pubout > /${SPINUP_PROJECT_DIR}/app.rsa.pub
writing RSA key
```

It requires a bunch of environment variables. You can export them and run using.

```
export SPINUP_PROJECT_DIR=${SPINUP_PROJECT_DIR} && export ARCHITECTURE=${ARCHITECTRE} && export CLIENT_ID=${CLIENT_ID} && export CLIENT_SECRET=${CLIENT_SECRET} && go run main.go
```

* SPINUP_PROJECT_DIR - The project directory which stores config and data files.
* ARCHITECTURE - What architecture that your system is.
    valid values: arm32v7, amd64
* CLIENT_ID - Github client id
* CLIENT_SECRET - Github client secret

On another terminal you can start the [dash](https://github.com/spinup-host/spinup-dash) to access the backend.

To check the API endpoint:
```
curl -X POST http://localhost:4434/createservice \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer reaplaceyourtokenhere" \
    --data '{
        "userId": "viggy28",
        "db": {
            "type": "postgres",
            "name": "localtest"
            },
        "version": {"maj":9,"min":6}
        }'
```

Once you created a cluster, you can connect using psql or any other postgres client

```
visi@visis-MacBook-Pro ~ % psql -h localhost -U postgres -p <port>
Password for user postgres:
psql (9.6.18, server 13.3 (Debian 13.3-1.pgdg100+1))
WARNING: psql major version 9.6, server major version 13.
         Some psql features might not work.
Type "help" for help.

postgres=# \dt
```

# API Documentation
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
`/createservice` - Creates a new database service. Only Postgres databases are supported at the moment
#### Authentication Header
`Bearer <GITHUB_TOKEN>`

#### Method:

`POST`

#### Data Params
- `userId=[String]`: GitHub username of the user.
- `version=[Object]`: Preferred database version. It must have the following properties:
    - `maj=[Integer]`: Database major version. E.g for Postgres:9.6, the value here is 9.
    - `min=[Integer]`: Database major version. E.g for Postgres:9.6, the value here is 6.
- `db=[Object]`: A JSON object describing the properties of the database to be created. The properties are:
  - `type=[String]`: Type of the database service to be created. Only `postgres` is supported for now.
  - `name=[String]`: Name of the database service
  - `monitoring=[String]`: Optional. Must be either of `enable` or `disable` when present. When enabled, it turns on prometheus metrics for the database
    on port 9187.
```json
{
  "userId": "viggy28",
  "db": {
    "type": "postgres",
    "name": "localtest",
    "monitoring": "enable"
  },
  "version": {
    "maj": 9,
    "min": 6
  }
}
```

#### Success Response:
    Code: 200

#### Error Response:
    Code: 500 INTERNALSERVER ERROR

### JWT
- URL

/jwt?data=replaceme

- Method:

`GET`

- Success Response:
    - Code: 200
    - Content: `{jwtofreplaceme}`
