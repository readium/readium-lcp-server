
## build

```
./docker/dockerbuild.sh `pwd`
```

```
DOCKER_BUILDKIT=1 sudo ./docker/dockerbuild.sh `pwd`
```


to build only the master container (lcp+lsd+frontend) : 
```
./docker/dockerbuild.sh `pwd` master
```

## RUN

```
./docker/dockerrun.sh
```

