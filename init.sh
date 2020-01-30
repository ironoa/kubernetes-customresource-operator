operator-sdk build ironoa/customresource-operator:v0.0.2
docker push ironoa/customresource-operator:v0.0.2
kubectl create -f deploy/operator.yaml
kubectl create -f deploy/crds/cache.example.com_v1alpha1_customresource_cr.yaml