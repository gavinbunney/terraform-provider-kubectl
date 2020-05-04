#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd ${DIR}

export KUBECONFIG="${DIR}/kubeconfig.yaml"
export COMPOSE_PROJECT_NAME=k3s

echo "--> Tearing down k3s in docker-compose"
docker-compose down -v &>/dev/null || true
rm -rf ${KUBECONFIG}
sync; sync;

echo "--> Starting k3s in docker-compose"
docker-compose up -d --build

echo "--> Allow insecure access to registry"
docker exec k3s_node_1 /bin/sh -c 'mkdir -p /etc/rancher/k3s'
docker cp "${DIR}/registries.yaml" k3s_node_1:/etc/rancher/k3s/registries.yaml

echo "--> Wait for k3s kubeconfig file to exist"
while [ ! -s "${KUBECONFIG}" ]  || [ ! -f "${KUBECONFIG}" ]; do sleep 1; done
while ! grep "127.0.0.1" "${KUBECONFIG}" &>/dev/null; do sleep 1; done

HOST_IP=127.0.0.1
if [ -f /.dockerenv ]; then
  HOST_IP="172.17.0.1"
fi

echo "--> Update IP of server to match host ip ${HOST_IP}"
kubectl config set-cluster default --server=https://${HOST_IP}:6443 --kubeconfig ${KUBECONFIG}

TIMEOUT=120
INTERVAL=5
echo "> Waiting for kubectl to make a successful connection to k3s (retrying every ${INTERVAL}s for ${TIMEOUT}s)"
TIMER_START=$SECONDS
COMMAND_RESULT=""
limit=$(( ${TIMEOUT} / ${INTERVAL} ))
count=0
while : ; do
  printf "."

  COMMAND_RESULT=$((kubectl get nodes --kubeconfig ${KUBECONFIG} -o json 2>/dev/null || true) | jq '.items | length')
  [[ "${COMMAND_RESULT}" -gt 0 ]] && printf "\n" && break

  if [[ "${count}" -ge "${limit}" ]]; then
    printf "\n[!] Timeout waiting for connection\n" >&2
    exit 1
  fi

  sleep ${INTERVAL}
  count=$[$count+1]
done

TIMER_DURATION=$(( SECONDS - TIMER_START ))

# restart the node to make sure the registries configuration has been picked up
docker restart k3s_node_1

echo "> Connection established to k3s in ${TIMER_DURATION}s"
