#!/usr/bin/env bash

# read SPINUP_DIR from env (or default to $HOME/.local/spinup)
# download go-release into $SPINUP_DIR/spinup
# download spinup-dash into $SPINUP_DIR/dashboard/
# download and possibly populate config.example.yaml into config.yaml
# prompt to add $SPINUP_DIR to path

if [ ! $(command -v "docker") ]; then
    echo "Cannot find or execute docker command"
    exit 1
fi

if [ ! $(command -v "docker-compose") ]; then
    echo "Cannot find or execute docker-compose command"
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

#${TMP_DIR}/spinup-backend -v
cp ${TMP_DIR}/spinup-backend ${SPINUP_DIR}/spinup

git clone --depth=1 https://github.com/spinup-host/spinup-dash/ ${TMP_DIR}/spinup-dash
cd ${TMP_DIR}/spinup-dash
npm install --ignore-scripts
cp -ar ${TMP_DIR} ${SPINUP_DIR}/spinup-dash

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

echo "Setup complete! To run spinup from your terminal, add ${SPINUP_DIR} to your shell path"