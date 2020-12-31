# dex client app

## build and release

```bash
cd src
docker build -t alitari/dex-clientapp .
docker login
docker push alitari/dex-clientapp
```

## start local

```bash
docker run -it alitari/dex-clientapp --help
```

## deploy on k8s

```bash

kubectl apply -k base
```


