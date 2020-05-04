#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd ${DIR}

export COMPOSE_PROJECT_NAME=k3s

echo "--> Stopping k3s in docker-compose"
docker-compose down -v
rm -rf kubeconfig.yaml
