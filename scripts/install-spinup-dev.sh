#!/usr/bin/env bash

for req in "docker" "openssl" "npm" "jq";
do
  if [ ! $(command -v "$req") ]; then
      echo "Cannot find or execute '$req' command"
      exit 1
  fi
done

if [ -z "$CLIENT_ID" ]; then
  echo "No value for environment variable CLIENT_ID"
fi

if [ -z "$CLIENT_SECRET" ]; then
  echo "No value for environment variable CLIENT_SECRET"
fi

if [ -z "$SPINUP_API_KEY" ]; then
  echo "No value for environment variable SPINUP_API_KEY. Setting it to default value of spinup"
  SPINUP_API_KEY="spinup"
fi

SPINUP_DIR=${SPINUP_DIR:-"${HOME}/.local/spinup"}

if [ -z "$API_VERSION" ]; then
  echo "Fetching latest Spinup version..."
    SPINUP_VERSION=$(curl --silent "https://api.github.com/repos/spinup-host/spinup/releases" | jq -r 'first | .tag_name')
else
  SPINUP_VERSION=$API_VERSION
fi

if [ -z "$UI_VERSION" ]; then
  echo "Fetching latest Spinup version..."
    SPINUP_UI_VERSION=main
else
  SPINUP_UI_VERSION=$UI_VERSION
fi


OS=$(go env GOOS)
ARCH=$(go env GOARCH)
PLATFORM="${OS}-${ARCH}"

SPINUP_PACKAGE="spinup-${SPINUP_VERSION}-${OS}-${ARCH}.tar.gz"
SPINUP_TMP_DIR="/tmp/spinup-install"

mkdir -p ${SPINUP_DIR}
mkdir -p ${SPINUP_TMP_DIR}

echo "git clone --depth=1 --branch=${SPINUP_VERSION} https://github.com/spinup-host/spinup.git ${SPINUP_TMP_DIR}/spinup-api"
git clone --depth=1 --branch=${SPINUP_VERSION} https://github.com/spinup-host/spinup.git ${SPINUP_TMP_DIR}/spinup-api
cd ${SPINUP_TMP_DIR}/spinup-api
go build -o spinup-backend ./main.go
./spinup-backend version
cp ${SPINUP_TMP_DIR}/spinup-api/spinup-backend ${SPINUP_DIR}/spinup

git clone --depth=1 --branch=${UI_VERSION} https://github.com/spinup-host/spinup-dash.git ${SPINUP_TMP_DIR}/spinup-dash
cd ${SPINUP_TMP_DIR}/spinup-dash
# setup env variables for dashboard's npm build
cat >.env <<-EOF
REACT_APP_CLIENT_ID=${CLIENT_ID}
REACT_APP_REDIRECT_URI=http://localhost:3000/login
REACT_APP_SERVER_URI=http://localhost:4434
REACT_APP_GITHUB_SERVER=http://localhost:4434/githubAuth
REACT_APP_LIST_URI=http://localhost:4434/listcluster
REACT_APP_URL=https://github.com/login/oauth/authorize?client_id=${CLIENT_ID}&redirect_uri=http://localhost:3000/login
EOF
cat .env
npm install --ignore-scripts
npm run build
rm -rf ${SPINUP_DIR}/spinup-dash
cp -a -R ${SPINUP_TMP_DIR}/spinup-dash/build ${SPINUP_DIR}/spinup-dash

cd ${SPINUP_DIR}
# preserve existing config file, or create a new one if none exists

CONFIG_FILE="${SPINUP_DIR}/config.yaml"
if [[ -f "$CONFIG_FILE" ]]; then
  echo "Found existing configuration file at ${CONFIG_FILE}."
else
  cat >config.yaml <<-EOF
  common:
    ports: [
      5432, 5433, 5434, 5435, 5436, 5437
    ]
    db_metric_ports: [
      55432, 55433, 55434, 55435, 55436, 55437
    ]
    architecture: amd64
    projectDir: ${SPINUP_DIR}
    client_id: ${CLIENT_ID}
    client_secret: ${CLIENT_SECRET}
    api_key: ${SPINUP_API_KEY}
EOF
fi

openssl genrsa -out ${SPINUP_DIR}/app.rsa 4096
openssl rsa -in ${SPINUP_DIR}/app.rsa -pubout > ${SPINUP_DIR}/app.rsa.pub

rm -rf ${SPINUP_TMP_DIR}
echo "Setup complete! To run spinup from your terminal, add ${SPINUP_DIR} to your shell path"