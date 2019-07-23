# K8s operator for Copybird CRD

Simple operator for Copybird CRD 

## Local Development
Install operator-framework first: 
```
mkdir -p $GOPATH/src/github.com/operator-framework
cd $GOPATH/src/github.com/operator-framework
git clone https://github.com/operator-framework/operator-sdk
cd operator-sdk
make 
make install
```

To Update CRD change the CopybirdSpec and CopybirdStatus Go structs:
```
type CopybirdSpec struct {
...
}
type CopybirdStatus struct {
...
}
```

Each time we make a change in these structures, we need to run the operator-sdk generate k8s command to update the `pkg/apis/copybird/v1alpha1/zz_generated.deepcopy.go` file accordingly.


## Build & Deploy
To build and deploy image for operator run:
```
operator-sdk build <your image name>
docker push <your image name>
```
Then change the image name in the corresponding field in `deploy/operator.yaml` as well. 

With working `kubectl` command do the following:

```
# Setup Service Account
kubectl create -f deploy/service_account.yaml

# Setup RBAC
kubectl create -f deploy/role.yaml
kubectl create -f deploy/role_binding.yaml

# Setup the CRD
kubectl create -f deploy/crds/copybird_v1alpha1_copybird_crd.yaml

# Deploy the copybird-operator
kubectl create -f deploy/operator.yaml
```

And then deploy Copybird CR with: 
```
kubectl create -f deploy/crds/copybird_v1alpha1_copybird_cr.yaml
```