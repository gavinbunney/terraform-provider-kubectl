#/bin/bash
set -e

# Adapted from: https://github.com/LiliC/travis-minikube/blob/minikube-26-kube-1.10/.travis.yml

export CHANGE_MINIKUBE_NONE_USER=true

echo "--> Downloading minikube"
# Make root mounted as rshared to fix kube-dns issues.
sudo mount --make-rshared /
# Download kubectl, which is a requirement for using minikube.
curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.12.0/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
# Download minikube.
curl -Lo minikube https://storage.googleapis.com/minikube/releases/v0.30.0/minikube-linux-amd64 && chmod +x minikube && sudo mv minikube /usr/local/bin/

echo "--> Starting minikube"
sudo minikube start --vm-driver=none --bootstrapper=kubeadm --kubernetes-version=v1.12.0
# Fix permissions issue in AzurePipelines
sudo chmod --recursive 777 $HOME/.minikube
sudo chmod --recursive 777 $HOME/.kube
# Fix the kubectl context, as it's often stale.
minikube update-context

echo "--> Waiting for cluster to be usable"
# Wait for Kubernetes to be up and ready.
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; until kubectl get nodes -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1; done
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; until kubectl -n kube-system get pods -lcomponent=kube-addon-manager -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1;echo "waiting for kube-addon-manager to be available"; kubectl get pods --all-namespaces; done
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; until kubectl -n kube-system get pods -lk8s-app=kube-dns -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1;echo "waiting for kube-dns to be available"; kubectl get pods --all-namespaces; done


echo "--> Get cluster details to check its running"
kubectl cluster-info

echo "--> Setup support for external IPs in LoadBalancer services"
# See workaround details here: https://github.com/elsonrodriguez/minikube-lb-patch
kubectl run minikube-lb-patch --replicas=1 --image=elsonrodriguez/minikube-lb-patch:0.1 --namespace=kube-system