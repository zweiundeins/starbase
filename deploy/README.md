# Deploying

`main` deploys itself: when CI passes on a push to `main`, the [Deploy workflow](../.github/workflows/deploy.yml) builds a linux/amd64 binary and pipes it over SSH into `starbase-deploy` on the server. You can also start it by hand with "Run workflow".

On the server, [`starbase-deploy`](starbase-deploy):

1. checks it's an ELF binary that reports a version. It runs `--version` via `systemd-run` as the unprivileged `starbase` user, never as root, with no network, a read-only system and a 10 s limit.
2. swaps it into `/opt/starbase/starbase` and restarts the `starbase` unit
3. waits up to 30 s for `/healthz` to answer `ok <new version>` (over the Unix socket)
4. if that fails, puts the running binary back and exits non-zero, so the workflow fails

The last good binary is kept as `starbase.prev`. `sudo starbase-deploy rollback` switches to it.

## What CI can do on the host

CI logs in as the `starbase-deploy` user with a key restricted to one command:

```
restrict,command="sudo -n /usr/local/bin/starbase-deploy -" ssh-ed25519 …
```

That means no shell, no port forwarding, and no other commands. Sudo allows exactly `/usr/local/bin/starbase-deploy -` for that user. The worst a leaked key can do is deploy a binary that passes the checks and runs as `starbase`, which is what a merge to `main` does anyway.

## One-time setup

**0. The service itself** (skip it if Starbase already runs there). The files are in this folder:

- [`starbase.service`](starbase.service): the systemd unit. It runs as the `starbase` user, capped at 384 MB, and can only write `/var/lib/starbase`. It listens on a Unix socket and is denied all of localhost (`IPAddressDeny=localhost`, except the DNS stub). On a shared host, that keeps it away from other services' local ports, such as Caddy's admin API.
- [`starbase.env.example`](starbase.env.example): its environment, installed as `/etc/starbase/starbase.env`.
- [`Caddyfile.snippet`](Caddyfile.snippet): the TLS reverse proxy (HTTP/2, no buffering for the render streams).

```sh
sudo useradd --system --home /var/lib/starbase --shell /usr/sbin/nologin starbase
sudo install -d -o starbase -g starbase -m 750 /var/lib/starbase
sudo install -D -m 640 -g starbase starbase.env.example /etc/starbase/starbase.env   # then edit BASE_URL
sudo install -m 644 starbase.service /etc/systemd/system/ && sudo systemctl daemon-reload && sudo systemctl enable starbase
# append Caddyfile.snippet to the Caddyfile, then: sudo caddy validate … && sudo systemctl reload caddy
```

`caddy validate`, run as root, creates the log file root-owned, and Caddy's reload then fails. Chown it to Caddy's user before reloading.

**1. Make a deploy key** on your machine:

```sh
ssh-keygen -t ed25519 -N "" -C "github-actions@zweiundeins/starbase" -f starbase-deploy-key
```

**2. Install the deploy script and the restricted user on the server:**

```sh
scp deploy/starbase-deploy deploy/setup-host.sh starbase-deploy-key.pub libretto.ch:/tmp/
ssh libretto.ch 'sudo bash /tmp/setup-host.sh /tmp/starbase-deploy-key.pub && rm /tmp/setup-host.sh /tmp/starbase-deploy /tmp/starbase-deploy-key.pub'
```

It should end with `>> ready: CI can deploy as starbase-deploy@…`.

**3. Give GitHub the key and the host.** The job's `if` runs before environment variables are loaded, so the host and its fingerprint are repository variables. The private key is a secret of the `production` environment, which only `main` may use.

```sh
gh api -X PUT repos/zweiundeins/starbase/environments/production \
  --input - <<<'{"deployment_branch_policy": {"protected_branches": false, "custom_branch_policies": true}}'
gh api -X POST repos/zweiundeins/starbase/environments/production/deployment-branch-policies -f name=main -f type=branch
gh secret set DEPLOY_SSH_KEY --env production -R zweiundeins/starbase < starbase-deploy-key
ssh-keyscan -t ed25519 179.237.86.212 2>/dev/null | gh variable set DEPLOY_KNOWN_HOSTS -R zweiundeins/starbase
gh variable set DEPLOY_HOST -R zweiundeins/starbase --body 179.237.86.212   # this switches deploys on
rm starbase-deploy-key starbase-deploy-key.pub
```

**4. Try it:** `gh workflow run deploy.yml -R zweiundeins/starbase`, then `gh run watch -R zweiundeins/starbase`.

To pause deploys, delete the `DEPLOY_HOST` variable. To revoke CI's access, delete `/var/lib/starbase-deploy/.ssh/authorized_keys` on the server.

## Moving to a new domain

- On the server, change `BASE_URL` in `/etc/starbase/starbase.env` and the Caddy site block.
- On GitHub, set `STARBASE_URL` (a repository variable) to the new URL. Both the bot and this workflow's final check use it.
- `sudo caddy validate` can leave root-owned log files behind, and then `systemctl reload caddy` fails. Fix them with `sudo sh -c 'chown caddy:caddy /var/log/caddy/*.log'` (the directory isn't readable, so a plain `sudo chown … *.log` fails on the glob).
- Change both together: `BASE_URL` sets the CSP's script origin and the canonical URLs, so pages served under one domain with the other `BASE_URL` load no scripts. Keep the old domain as a permanent redirect (see `Caddyfile.snippet`).
