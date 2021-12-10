#!/usr/bin/env bash

if [ ! $(command -v "docker") ]; then
    echo "Cannot find or execute docker command"
    exit 1
fi

if [ ! $(command -v "docker-compose") ]; then
    echo "Cannot find or execute docker-compose command"
    exit 1
fi

if [ ! $(command -v "openssl") ]; then
  echo "Cannot find or execute openssl command"
  exit 1
fi

SPINUP_DIR=${SPINUP_DIR:-"${HOME}/.local/spinup"}
SPINUP_VERSION=${SPINUP_VERSION:-"0.2-alpha"}

OS=$(go env GOOS)
ARCH=$(go env GOARCH)
PLATFORM="${OS}-${ARCH}"
API_DL_URL="https://github.com/spinup-host/spinup/releases/download/v${SPINUP_VERSION}/spinup-backend-v${SPINUP_VERSION}-${PLATFORM}.tar.gz"

SPINUP_PACKAGE="spinup-${SPINUP_VERSION}-${OS}-${ARCH}.tar.gz"
TMP_DIR="/tmp/spinup-install"

mkdir -p ${SPINUP_DIR}
mkdir -p ${TMP_DIR}

curl -LSs ${API_DL_URL} -o ${TMP_DIR}/${SPINUP_PACKAGE}
tar xzvf ${TMP_DIR}/${SPINUP_PACKAGE} -C "${TMP_DIR}/"
rm -f ${TMP_DIR}/${SPINUP_PACKAGE}

${TMP_DIR}/spinup-backend version
cp ${TMP_DIR}/spinup-backend ${SPINUP_DIR}/spinup

git clone --depth=1 https://github.com/spinup-host/spinup-dash.git ${TMP_DIR}/spinup-dash
cd ${TMP_DIR}/spinup-dash
npm install --ignore-scripts
npm run build
cp -ar ${TMP_DIR}/spinup-dash/build ${SPINUP_DIR}/spinup-dash

cd ${SPINUP_DIR}
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
EOF

cat >.env <<-EOF
  REACT_APP_CLIENT_ID=abcdefghijk
  REACT_APP_REDIRECT_URI=http://localhost:3000/login
  REACT_APP_GITHUB_SERVER=http://localhost:3000/githubAuth
  REACT_APP_SERVER_URI=http://localhost:3000/createservice
  REACT_APP_LIST_URI=http://localhost:3000/listcluster
EOF

openssl genrsa -out ${SPINUP_DIR}/app.rsa 4096
openssl rsa -in ${SPINUP_DIR}/app.rsa -pubout > ${SPINUP_DIR}/app.rsa.pub

echo "Setup complete! To run spinup from your terminal, add ${SPINUP_DIR} to your shell path"