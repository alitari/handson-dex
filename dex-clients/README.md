# dex clients

## build and release

```bash
# dex-clientapp
cd src/dex-clientapp
docker build -t alitari/dex-clientapp .
docker login
docker push alitari/dex-clientapp
# dex-oidconsumer
cd src/dex-client-oidconsumer
docker build -t alitari/dex-client-oidconsumer .
docker login
docker push alitari/dex-client-oidconsumer
```

## start local

```bash
# dex-clientapp
docker run -it alitari/dex-clientapp --help
# dex-oidconsumer
docker run -it alitari/dex-client-oidconsumer --help


```

## deploy on k8s

```bash
# dex-clientapp
kubectl apply -k dex-clientapp
# dex-oidconsumer
kubectl apply -k dex-client-oidconsumer
```


