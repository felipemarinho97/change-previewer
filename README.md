# change-previewer
A tool for assisting the code review process through ephemeral deploys

# Usage


Deploy a montor with the preferred configuration for your `namespace`. Example:

```bash
echo "apiVersion: change-previewer.com.github.felipemarinho97/v1
kind: KamenevMonitor
metadata:
  name: kamenevmonitor-sample
  namespace: default # change this to your namespace where you want to deploy the monitor
spec:
  maxLifeTime: 1800 # in seconds (30 minutes)
" | kubectl apply -f -
```


Just anotate your kubernetes manifests with the following annotation:

```yaml
metadata:
  annotations:
    com.github.felipemarinho97.change-previewer/enabled": "true"
```

# How it works

The change-previewer will watch for changes in the kubernetes resources and will schedule a deletion of the resource after the time specified in the annotation `com.github.felipemarinho97.change-previewer/timeout` (default: 30 minutes).

# Installation

Add the kamenev controller to your cluster:

```bash
kubectl apply -f kamenev/kamenev.yaml
```

This will provide the kamenev controller with the necessary permissions to watch for changes in the kubernetes resources. Check if the controller is running:

```bash
kubectl get all -n kamenev-system
```

Check if the CRD was created:

```bash
kubectl get crd | grep "change-previewer"
```
