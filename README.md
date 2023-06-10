# SPINUP

An open source alternative to [AWS RDS](https://aws.amazon.com/rds/), [Cloud SQL](https://cloud.google.com/sql). 

## Architecture

The idea is simple. Spinup creates multiple containers through docker-compose. 
Spinup can be deployed anywhere. System requirements are Go and docker-compose. Once this [issue](https://github.com/spinup-host/spinup/issues/45) has been fixed then you don't need to have Go installed. It can run on [Digital Ocean droplet](https://www.digitalocean.com/products/droplets/), [Azure Compute](https://azure.microsoft.com/en-us/product-categories/compute/), [Oracle Compute](https://www.oracle.com/cloud/compute/), [Raspberry Pi](https://www.raspberrypi.org/) etc. 

We are currently using Github Authentication. We should be able to support other authentication methods.

Currently we only support Postgres dbms, but we should be able to support other open source databases like [MySQL](https://www.mysql.com/), [MariaDB](https://mariadb.org/) etc.

![architecture](architecture.png)
## Installation
### Linux
To get started with Spinup on Linux:
- Run the installation script using the command below:
```bash
bash < <(curl -s https://raw.githubusercontent.com/spinup-host/spinup/main/scripts/install-spinup.sh)
```
- Add the Spinup installation directory (default is `$HOME/.local/spinup`) to your shell PATH to use Spinup from your terminal.
- Start the Spinup servers (both API and frontend) by running:
```bash
spinup start
```
- (Optional) If you want to use Github Authentication, create a project in [github](https://github.com). Next, export the project's client ID and Secret as environment variables
```bash
export CLIENT_ID=<GITHUB_CLIENT_ID>
export CLIENT_SECRET=<GITHUB_CLIENT_SECRET>
```

- (Optional) You can change the API_KEY value using enviornment variable. Default value is `spinup`
```bash
export SPINUP_API_KEY=<YOUR_API_KEY>
```

### Docker
Spinup works by starting other (database and related) services in docker, but you can test-run Spinup itself as a docker container.
To do this:
- Clone this repository and enter into the directory.
- Build the Spinup docker image using the `Dockerfile` in the root folder e.g., with:
   ```
  docker build -t spinup/spinup .`
  ```
- Mount the `docker.sock` from your host into the Spinup container and start it as shown below:
    ```
  docker run -v /var/run/docker.sock:/var/run/docker.sock -v ./path/to/config.yaml:/root/.local/spinup/config.yaml -p 4434:4434 id/spinup:latest start --api-only
    ```
  The command above will start Spinup in API-only mode.

NB: Mounting `docker.sock` gives Spinup root access to your host machine.

## Monitoring
With monitoring enabled, Spinup will automatically setup monitoring services (Prometheus, Postgres Exporter, and Grafana) 
for you on startup. Every new database you add will automatically be added to postgres exporter for scraping and its metrics exposed in Prometheus/Grafana.

To enable monitoring, set the `common.monitoring` field in your spinup config file to `true` (or add it if it doesn't exist).
*Note that `common.monitoring` above implies the `monitoring` field in the `common` section in the YAML file*
When monitoring is enabled, the monitoring services are started as follows:
- Prometheus - http://localhost:9090
- Grafana - http://localhost:9091
- Postgres Exporter - http://localhost:9187

Visit http://localhost:9091/explore to explore the provisioned Prometheus in Grafana,

## Backups
### Others

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
**Create a config.yaml file**

```
common:
  ports: [
    5432, 5433, 5434, 5435, 5436, 5437
  ]
  architecture: amd64
  projectDir: <PROJECT_DIR>
  client_id: <CLIENT_ID>
  client_secret: <CLIENT_SECRET>
  api_key: <API_KEY> //if not using github authentication
  monitoring: true
 ```
 **To run spinup**
 
 ```go run main.go```

#### Authentication
We use JWT for verification. You need to have a private and public key that you can create using OpenSSL:

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
            "name": "localtest",
            "username": "spinup",
            "password": "spinup",
            "memory": 6000, //In Megabytes
            "cpu": 2
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
The API documentation is autogenerated and saved in `openapi/swagger.yaml`. To view the documentaion on Swagger UI, run the command below from the project directory:
```bash
make serve-docs
```