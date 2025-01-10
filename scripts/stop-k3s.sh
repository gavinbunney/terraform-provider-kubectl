#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd ${DIR}

export COMPOSE_PROJECT_NAME=k3s
export ARCH=$(uname -m | tr '[:upper:]' '[:lower:]')
export DOCKER_DEFAULT_PLATFORM=linux/${ARCH}

export K3S_VERSION=${K3S_VERSION:-v1.32.0-k3s1}

echo "--> Stopping k3s in docker-compose"
docker-compose down -v
rm -rf kubeconfig.yaml
