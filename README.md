# SPINUP

An open source alternative to [AWS RDS](https://aws.amazon.com/rds/), [Cloud SQL](https://cloud.google.com/sql). 

## Arhictecture

The idea is simple. Spinup creates multiple containers through docker-compose. 
Spinup can be deployed anywhere. Only requirement is docker-compose. It can run on anywhere [Digital Ocean droplet](https://www.digitalocean.com/products/droplets/), [Azure Compute](https://azure.microsoft.com/en-us/product-categories/compute/), [Oracle Compute](https://www.oracle.com/cloud/compute/), [Raspberry Pi](https://www.raspberrypi.org/) etc. 

We are currently using Github Authentication. We should be able to support other authentication methods.

Currently we only support Postgres dbms, but we should be able to support other open source databases like [MySQL](https://www.mysql.com/), [MariaDB](https://mariadb.org/) etc.

![architecture](architecture.jpeg)
### How to run

It requires a bunch of environment variables. You can export them and run using. Also change the port to a higher number if you don't run the program as `root` user.

```
export SPINUP_PROJECT_DIR=/tmp/spinuplocal && export ARCHITECTURE=amd64 && export CF_AUTHORIZATION_TOKEN=replaceme  && export CF_ZONE_ID=replaceme && export CLIENT_ID=replaceme && export CLIENT_SECRET=replaceme && go run main.go
```

* SPINUP_PROJECT_DIR - The project directory which stores config and data files.
* ARCHITECTURE - What architecture that your system is.
valid values: arm32v7, amd64
* CF_AUTHORIZATION_TOKEN - Cloudflare authorization token for manipulating DNS records
* CF_ZONE_ID - Cloudflare zone id
* CLIENT_ID - Github client id
* CLIENT_SECRET - Github client secret

You need to have a private and public key that you can create using OpenSSL:

**To create a private key**
```
visi@visis-MacBook-Pro spinup % openssl genrsa -out /tmp/spinuplocal/app.rsa 4096 
Generating RSA private key, 4096 bit long modulus
...++
...................++
e is 65537 (0x10001)
```

**To create a public key**
```
visi@visis-MacBook-Pro spinup % openssl rsa -in /tmp/spinuplocal/app.rsa -pubout > /tmp/spinuplocal/app.rsa.pub
writing RSA key
```

On another terminal create a POST request using
```
curl -X POST http://localhost:8000/createservice \
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

- URL

/jwt?data=replaceme

- Method:

`GET`

- Success Response:
    - Code: 200
    - Content: `{jwtofreplaceme}`
