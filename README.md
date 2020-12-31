# Hands on dex

Examples using [dex](https://dexidp.io) in K8s with the connectors:

- mock
- LDAP

The example is more or less a K8s variant of the [example from the official dex documentation](https://dexidp.io/docs/connectors/ldap/#getting-started)

## prerequsites

The examples assume that you have access to an k8s cluster with a Kong Ingress Controller listening on a public IP `11.22.33.44`. Check your ip e.g. if your kong is installed via helm with the release `kong` in namespace `kong` with `kubectl -n kong get svc kong-kong-proxy -o json | jq -r .status.loadBalancer.ingress[0].ip`. You need to adapt the configurations for this IP accordingly.

We are working on the namespace `auth` using the following tools:

- `kubectl` with plugin `ns`
- `kustomize`
- `helm`

```bash
kubectl create namespace auth
kubectl-ns auth
```

## mock connector

```bash
# install dex with mock connector config
helm upgrade dex banzaicloud-stable/dex -f dex/dex-mock-values.yaml --install

# install dex-clientapp
kubectl apply -k dex-clientapp/v1
```

Browse to `http://dex-clientapp.11.22.33.44.xip.io/` and log in for client-id `dex-clientapp` and connector ID `mock`.
The dex-clientapp requests at least access for the scopes "openid", "profile", "email" and "offline access" if the provider does not support any additional scopes ( which is the case for the mock connector). Now, dex redirects you to the dex-clientapp which shows you the content of the Openconnect-ID token.


## ldap connector


### install open ldap server

```bash
kubectl apply -k ldap/l1
# check LDAP content
LDAP_POD=$(kubectl get pod -l app.kubernetes.io/name=openldap-1 -o=name | head -1)
kubectl exec -i -t $LDAP_POD -- bash
ldapsearch -x -b "dc=example,dc=org" -H ldap://localhost:389 -D "cn=admin,dc=example,dc=org" -w admin
```

### install dex with ldap connector

```bash
# install dex with mock connector config
helm upgrade dex banzaicloud-stable/dex -f dex/dex-ldap-values.yaml --install

# install dex-clientapp
kubectl apply -k dex-clientapp/v1
```

Browse to `http://dex-clientapp.11.22.33.44.xip.io/` and log in for client-id `dex-clientapp`, additional scopes `groups` and connector ID `ldap`. Authenticate with  user `john` and password `bar` and you get the user attributes and his group memberships (`admins`, `developer`).
