## Install KubeArmor

Install KubeArmor using helm

```
helm upgrade --install kubearmor . --set version=<kubearmor-version> -n kube-system

helm upgrade --install kubearmor . --set version=v0.8.0 -n kube-system
```
* [kubearmorrelay](https://github.com/kubearmor/kubearmor-relay-server/).enabled = {true | false} (default: true)

## Verify if all the pods are up and running

```
kubectl get all -n kube-system
```

## Remove KubeArmor

Uninstall KubeArmor using helm

```
helm uninstall kubearmor -n kube-system
```
