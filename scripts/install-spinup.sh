#!/usr/bin/env bash

for req in "docker" "docker-compose" "openssl" "npm";
do
  if [ ! $(command -v "$req") ]; then
      echo "Cannot find or execute '$req' command"
      exit 1
  fi
done

SPINUP_DIR=${SPINUP_DIR:-"${HOME}/.local/spinup"}
SPINUP_VERSION=$(curl --silent "https://api.github.com/repos/spinup-host/spinup/releases" | jq -r 'first | .tag_name')

OS=$(go env GOOS)
ARCH=$(go env GOARCH)
PLATFORM="${OS}-${ARCH}"
API_DL_URL="https://github.com/spinup-host/spinup/releases/download/${SPINUP_VERSION}/spinup-backend-${SPINUP_VERSION}-${PLATFORM}.tar.gz"

SPINUP_PACKAGE="spinup-${SPINUP_VERSION}-${OS}-${ARCH}.tar.gz"
SPINUP_TMP_DIR="/tmp/spinup-install"

mkdir -p ${SPINUP_DIR}
mkdir -p ${SPINUP_TMP_DIR}

curl -LSs ${API_DL_URL} -o ${SPINUP_TMP_DIR}/${SPINUP_PACKAGE}
tar xzvf ${SPINUP_TMP_DIR}/${SPINUP_PACKAGE} -C "${SPINUP_TMP_DIR}/"
rm -f ${SPINUP_TMP_DIR}/${SPINUP_PACKAGE}

${SPINUP_TMP_DIR}/spinup-backend version
cp ${SPINUP_TMP_DIR}/spinup-backend ${SPINUP_DIR}/spinup

git clone --depth=1 https://github.com/spinup-host/spinup-dash.git ${SPINUP_TMP_DIR}/spinup-dash
cd ${SPINUP_TMP_DIR}/spinup-dash
npm install --ignore-scripts
npm run build
cp -a -R ${SPINUP_TMP_DIR}/spinup-dash/build ${SPINUP_DIR}/spinup-dash

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

rm -rf ${SPINUP_TMP_DIR}
echo "Setup complete! To run spinup from your terminal, add ${SPINUP_DIR} to your shell path"