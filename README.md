# Automium API Gateway

## Setup Secrets

### Git credentials

Edit the **secrets/gitconfig.json** file with your repo configuration.

Create the secrets:
```
kubectl -n openfaas create secret generic secret-git-key --from-file=GitConfig=secrets/gitconfig.json

kubectl -n openfaas-fn create secret generic secret-git-key --from-file=GitConfig=secrets/gitconfig.json
```

### Kubernetes configuration

Get the k8s configuration and edit the secret file **secrets/kubeconfig.json**:

`cat .kube/config | sed -E ':a;N;$!ba;s/\r{0,1}\n/\\n/g' > kubeconfig` 

Create the secrets:
```
kubectl -n openfaas create secret generic secret-kube-key --from-file=KubeConfig=secrets/kubeconfig.json

kubectl -n openfaas-fn create secret generic secret-kube-key --from-file=KubeConfig=secrets/kubeconfig.json
```

### Private Registry [optional]

```
kubectl create secret docker-registry harbor-registry \
--docker-server=<DOCKER_REGISTRY_SERVER> \
--docker-username=<DOCKER_USER> \
--docker-password=<DOCKER_PASSWORD> \
--docker-email=<DOCKER_EMAIL>
```

## Functions

- infraspecs
- infrastatus
- infraservices
- deletespec
- savespec
- applyservice

### Usage

Install OpenFaaS CLI following the official docs: https://docs.openfaas.com/cli/install/.

Set the gateway endpoint exporting the variable OPENFAAS_URL.

#### Deploy a function

In the **functions** folder, run:

```
faas-cli build -f infraspecs.yml  
faas-cli push -f infraspecs.yml   
faas-cli deploy --replace --update=false -f infraspecs.yml
```  

or in a single command:

`faas-cli up -f infraspecs.yml`

#### Test a function

In the **functions** folder, run:

`faas-cli invoke infraspecs`
