#!/bin/bash
set -e

echo "--> Downloading K3s"
curl -Lo k3s https://github.com/rancher/k3s/releases/download/v1.0.1/k3s && chmod +x k3s

echo "--> Starting K3s"
sudo ./k3s server &
