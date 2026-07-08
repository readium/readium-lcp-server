
## QuickStart

To build and run you have to go to the root directory

```
cd ..
```

To build the master container (lcp+lsd+frontend) :

```
./docker/dockerbuild.sh `pwd` master
```

To run it :

```
./docker/dockerrun.sh
```


go to http://127.0.0.1:8080/frontend

## Local Docker CI build

The local CI runner builds `docker/Dockerfile` and checks that the generated image
contains the three server binaries. It does not push anything to a registry.

The build logic is a Python script executed inside Docker. The host only needs
Docker; Go, Node, Python, and the Docker CLI used by the build flow are isolated
from the host.

Build and test the test image:

```
./docker/ci-local-build.sh test
```

This creates:

```
readium-lcp-server-test:latest
```

Production build, with secrets in `./_`:

```
./_
  libuserkey.a
  userkey.h
  user_key.go
  cert-edrlab.pem
  privkey-edrlab.pem
```

```
./docker/ci-local-build.sh prod ./_
```

This creates:

```
readium-lcp-server-prod:latest
```

The wrapper itself only starts a Python container and runs the build script:

```
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$PWD:/workspace" \
  -w /workspace \
  docker:27-cli \
  sh -lc 'apk add --no-cache ca-certificates python3 >/dev/null && python3 docker/ci-build.py "$@"' \
  -- test
```

For a direct `docker run` on each host, adapt only the workspace mount syntax:

- Bash/macOS/Linux: `-v "$PWD:/workspace"`
- PowerShell: `-v "${PWD}:/workspace"`
- Windows cmd: `-v "%cd%:/workspace"`

Custom image name:

```
IMAGE_NAME=my-private-repo/readium-lcp-server-test ./docker/ci-local-build.sh test
```

Useful variables:

- `IMAGE_NAME`: repository name without tag, for example `org/image`.
- `IMAGE_REF`: full image reference; overrides `IMAGE_NAME` and `TAG`.
- `CI_DOCKER_IMAGE`: Docker CLI image used by the wrapper, defaults to `docker:27-cli`.
- `TAG`: image tag, defaults to `latest`.
- `PLATFORM`: Docker build platform, defaults to `linux/amd64`; use `local` to let Docker choose the host platform.
- `RUN_IMAGE_TEST`: set to `false` to skip the image smoke test.
- `LOGGING_SLACK_TOKEN` and `LOGGING_SLACK_CHANNEL`: forwarded as Docker build args. When a Slack token is provided without a channel, the Jenkins test channel `C07KFNG59PF` is used.

Deployment can select the image with an environment variable:

```
IMAGE_NAME=readium-lcp-server-test:latest ./docker/dockerrun.sh
```
