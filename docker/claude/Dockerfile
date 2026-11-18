FROM debian:trixie-slim

ARG USERNAME
ARG UID
ARG GLAB_VERSION=1.129.0

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl git jq zsh less procps ripgrep \
      chromium postgresql-client \
      iptables ipset dnsutils util-linux \
    && curl -fsSL -o /tmp/glab.deb "https://gitlab.com/gitlab-org/cli/-/releases/v${GLAB_VERSION}/downloads/glab_${GLAB_VERSION}_linux_amd64.deb" \
    && dpkg -i /tmp/glab.deb && rm /tmp/glab.deb \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u ${UID} -s /bin/zsh ${USERNAME} \
    && mkdir -p /home/${USERNAME}/.claude /home/${USERNAME}/.local/share/pnpm /home/${USERNAME}/.config /home/${USERNAME}/.cache \
    && touch /home/${USERNAME}/.zshrc \
    && chown -R ${UID}:${UID} /home/${USERNAME}

ENV HOME=/home/${USERNAME} \
    USER=${USERNAME} \
    DCLAUDE_UID=${UID} \
    SHELL=/bin/zsh \
    PATH=/home/${USERNAME}/.fnm/aliases/default/bin:/home/${USERNAME}/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    DISABLE_AUTOUPDATER=1

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
