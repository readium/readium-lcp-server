
## build

```
DOCKER_BUILDKIT=1 sudo ./docker/dockerbuild.sh `pwd`

```


## GCR : Google Cloud Registry

 - `gcr.io/lcp-server-337422/github.com/readium/readium-frontend-server:$BRANCH_NAME:COMMIT_HASH`
 - `gcr.io/lcp-server-337422/github.com/readium/readium-lcpserver-server:$BRANCH_NAME:COMMIT_HASH`
 - `gcr.io/lcp-server-337422/github.com/readium/readium-lsdserver-server:$BRANCH_NAME:COMMIT_HASH`

```sh

# docker run -it --rm --platform=linux/amd64 gcr.io/lcp-server-337422/github.com/readium/readium-frontend-server@sha256:968e7daa5febbdf3afe53da1cfa24facc68371831ad85fae96bac21b09307f33


```

