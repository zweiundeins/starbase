#!/usr/bin/env bash
#
# One-time host setup for CI deploys (see deploy/README.md). Run as root on
# the server, with the public half of the CI deploy key (a .pub file):
#
#   sudo bash setup-host.sh starbase-deploy-key.pub
#
# It installs /usr/local/bin/starbase-deploy and creates the `starbase-deploy`
# user. That user's only key is forced to `sudo -n /usr/local/bin/starbase-deploy -`
# (no shell, no forwarding), and sudo allows exactly that command. Rerunning
# replaces the key.
set -euo pipefail
KEYFILE="${1:?usage: setup-host.sh <public key file>}"
[[ -f "$KEYFILE" ]] || { echo "!! no such file: $KEYFILE" >&2; exit 2; }
PUBKEY=$(tr -d '\r' < "$KEYFILE" | sed '/^$/d')
[[ "$PUBKEY" == ssh-* && "$PUBKEY" != *$'\n'* ]] || { echo "!! that does not look like one public key" >&2; exit 2; }
HERE=$(cd "$(dirname "$0")" && pwd)

install -m 0755 -o root -g root "$HERE/starbase-deploy" /usr/local/bin/starbase-deploy
id starbase-deploy >/dev/null 2>&1 ||
	useradd --system --create-home --home-dir /var/lib/starbase-deploy --shell /bin/sh starbase-deploy
install -d -m 0700 -o starbase-deploy -g starbase-deploy /var/lib/starbase-deploy/.ssh
printf 'restrict,command="sudo -n /usr/local/bin/starbase-deploy -" %s\n' "$PUBKEY" > /var/lib/starbase-deploy/.ssh/authorized_keys
chown starbase-deploy: /var/lib/starbase-deploy/.ssh/authorized_keys
chmod 0600 /var/lib/starbase-deploy/.ssh/authorized_keys

echo "starbase-deploy ALL=(root) NOPASSWD: /usr/local/bin/starbase-deploy -" > /etc/sudoers.d/starbase-deploy.tmp
chmod 0440 /etc/sudoers.d/starbase-deploy.tmp
visudo -cqf /etc/sudoers.d/starbase-deploy.tmp
mv -f /etc/sudoers.d/starbase-deploy.tmp /etc/sudoers.d/starbase-deploy
echo ">> ready: CI can deploy as starbase-deploy@$(hostname)"
