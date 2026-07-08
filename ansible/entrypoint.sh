#!/bin/sh
set -eu

mkdir -p /root/.ssh
chmod 700 /root/.ssh

if [ -f /ssh/id_key ]; then
  cp /ssh/id_key /root/.ssh/id_ansible
  chmod 600 /root/.ssh/id_ansible
  export ANSIBLE_PRIVATE_KEY_FILE=/root/.ssh/id_ansible
fi

if [ -f /ssh/known_hosts ] && [ ! -e /root/.ssh/known_hosts ]; then
  ln -s /ssh/known_hosts /root/.ssh/known_hosts
fi

exec "$@"
