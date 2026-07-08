FROM docker:27-cli

RUN apk add --no-cache --upgrade expat \
  && apk add --no-cache \
  bash \
  ca-certificates \
  git \
  openssh-client \
  py3-pip \
  python3 \
  rsync \
  && pip3 install --break-system-packages --no-cache-dir ansible-core \
  && mkdir -p /root/.ssh

ENV ANSIBLE_FORCE_COLOR=true

COPY entrypoint.sh /usr/local/bin/ansible-entrypoint
RUN chmod +x /usr/local/bin/ansible-entrypoint

WORKDIR /work
ENTRYPOINT ["ansible-entrypoint"]
CMD ["ansible-playbook", "--version"]
