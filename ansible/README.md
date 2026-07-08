# Readium LCP VPS deployment

Ansible deploys a hardened Debian 13 VPS with Docker containers behind Caddy:

- `front-prod` -> `https://front-prod.example.org`
- `front-test` -> `https://front-test.example.org`
- SSH hardening, UFW, fail2ban, journald retention, Caddy logs, Docker log rotation, node exporter
- local image build/test, then `docker save` + remote `docker load`

## Windows control host

Do not run Ansible from native Windows Python. Ansible is not supported as a
native Windows control node and can fail with errors such as:

```text
OSError: [WinError 1] Incorrect function
```

Use WSL, Linux, macOS, or run Ansible directly through Docker.

### Recommended SSH setup from WSL

Do not bridge a native Windows SSH agent into Linux. Use WSL as the Linux control
host and load a dedicated deployment key into the WSL OpenSSH agent.

```sh
sudo apt update
sudo apt install -y openssh-client
```

Create or import a deployment key in WSL, then load it in the WSL agent:

```sh
mkdir -p ~/.ssh
chmod 700 ~/.ssh
ssh-keygen -t ed25519 -f ~/.ssh/lcp_vps_deploy -C "lcp-vps-ansible"
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/lcp_vps_deploy
ssh-add -l
```

Add `~/.ssh/lcp_vps_deploy.pub` to the VPS `debian` user's
`~/.ssh/authorized_keys`, then verify SSH from WSL:

```sh
ssh debian@152.228.141.110
```

Then either run Ansible from WSL:

```sh
ANSIBLE_CONFIG=ansible/ansible.cfg ansible-playbook -i ansible/inventory.yml ansible/playbooks/provision.yml
```

Or keep the Ansible control host sandboxed in Docker from WSL:

```sh
docker build -f ansible/Dockerfile -t readium-lcp-ansible:local ansible

docker run --rm -it \
  -e ANSIBLE_CONFIG=/work/ansible/ansible.cfg \
  -e SSH_AUTH_SOCK=/ssh-agent \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$PWD:/work" \
  -v "$SSH_AUTH_SOCK:/ssh-agent" \
  -w /work \
  readium-lcp-ansible:local \
  ansible-playbook -i ansible/inventory.yml ansible/playbooks/provision.yml
```

Native Windows Docker with a Windows agent socket is not recommended. If you
need to run from native Windows anyway, use the private-key mount form:

```powershell
docker build -f ansible/Dockerfile -t readium-lcp-ansible:local ansible

docker run --rm -it `
  -e ANSIBLE_CONFIG=/work/ansible/ansible.cfg `
  -v /var/run/docker.sock:/var/run/docker.sock `
  -v "${PWD}:/work" `
  -v "C:\path\to\private_key:/ssh/id_key:ro" `
  -w /work `
  readium-lcp-ansible:local `
  ansible-playbook -i ansible/inventory.yml ansible/playbooks/provision.yml
```

## Setup

```sh
cp ansible/inventory.example.yml ansible/inventory.yml
```

Edit `ansible/inventory.yml` with the VPS IP and DNS names.

Ansible needs the matching private key, not the `.pub` file. Use `ssh-agent` or
set `ansible_ssh_private_key_file` in `ansible/inventory.yml`.

## Provision the VPS

```sh
ANSIBLE_CONFIG=ansible/ansible.cfg ansible-playbook -i ansible/inventory.yml ansible/playbooks/provision.yml
```

## Build and deploy an image

Deploy test:

```sh
ANSIBLE_CONFIG=ansible/ansible.cfg ansible-playbook -i ansible/inventory.yml ansible/playbooks/build-and-update.yml \
  -e deploy_instance=front-test
```

Deploy prod, with secrets in `./_`:

```sh
ANSIBLE_CONFIG=ansible/ansible.cfg ansible-playbook -i ansible/inventory.yml ansible/playbooks/build-and-update.yml \
  -e deploy_instance=front-prod \
  -e lcp_prod_secrets_dir=./_
```

By default, `build-and-update.yml` tags the image with the current git short SHA.
You can also pass the tag explicitly:

```sh
ANSIBLE_CONFIG=ansible/ansible.cfg ansible-playbook -i ansible/inventory.yml ansible/playbooks/build-and-update.yml \
  -e deploy_instance=front-test \
  -e lcp_image_tag="$(git rev-parse --short HEAD)"
```

The image build runs from the playbook on the local Ansible control host with
`python3 docker/ci-build.py`, then the image is transferred with `docker save`
and `docker load`.

Ansible buffers command output. During the build, follow the local log file shown
by the playbook:

```sh
tail -f .ansible-artifacts/logs/front-test-<tag>.build.log
```

## Runtime environment

Each container receives `/etc/readium-lcp-server/<instance>.env`. It only contains
deployment-specific overrides; defaults stay in `docker/Dockerfile`. Public URLs
are derived from the instance domain by default.

Override any value per instance with an `env` block in `ansible/inventory.yml`:

```yaml
lcp_instances:
  front-test:
    env:
      FRONTEND_PROVIDER_URI: https://front-test.example.org
      LCP_UPDATE_AUTH_USER: adm_username
      LCP_UPDATE_AUTH_PASS: adm_password
```

## Maintenance

Status `front-test`:

```sh
ANSIBLE_CONFIG=ansible/ansible.cfg ansible-playbook -i ansible/inventory.yml ansible/playbooks/maintenance.yml \
  -e maintenance_action=status \
  -e deploy_instance=front-test
```

Then logs:

```sh
ANSIBLE_CONFIG=ansible/ansible.cfg ansible-playbook -i ansible/inventory.yml ansible/playbooks/maintenance.yml \
  -e maintenance_action=logs \
  -e deploy_instance=front-test
```

Reset one volume manually:

```sh
ANSIBLE_CONFIG=ansible/ansible.cfg ansible-playbook -i ansible/inventory.yml ansible/playbooks/maintenance.yml \
  -e maintenance_action=reset-volume \
  -e deploy_instance=front-test
```

Show service and container logs:

```sh
ANSIBLE_CONFIG=ansible/ansible.cfg ansible-playbook -i ansible/inventory.yml ansible/playbooks/maintenance.yml \
  -e maintenance_action=logs \
  -e deploy_instance=front-test
```

Prune unused images:

```sh
ANSIBLE_CONFIG=ansible/ansible.cfg ansible-playbook -i ansible/inventory.yml ansible/playbooks/maintenance.yml \
  -e maintenance_action=image-prune
```

Scheduled volume resets are configured by `lcp_volume_reset_cron` in the inventory.
This is destructive: it stops the service, removes the named Docker volume, recreates
it, and starts the service again.
