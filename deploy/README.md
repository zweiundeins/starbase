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
- [`starbase-metrics.socket`](starbase-metrics.socket): the metrics' listener on 127.0.0.1:8198, which systemd holds and passes to the service (`Sockets=` in the unit). The service's `IPAddressDeny=localhost` would drop Prometheus' connections too; the socket unit has its own filter, which lets localhost reach that one port while Starbase still reaches no local port.
- [`starbase.env.example`](starbase.env.example): its environment, installed as `/etc/starbase/starbase.env`.
- [`starbase.caddy`](starbase.caddy): the site block, for the folder the host's Caddyfile imports: the TLS reverse proxy (HTTP/2, no buffering for the render streams), the JSON access log and the redirect from the first domain.
- [`observability/`](observability): the scrape target, the blackbox probe and the Alloy file, which `setup-host.sh` installs ([Monitoring](#monitoring)).

```sh
sudo useradd --system --home /var/lib/starbase --shell /usr/sbin/nologin starbase
sudo install -d -o starbase -g starbase -m 750 /var/lib/starbase
sudo install -D -m 640 -g starbase starbase.env.example /etc/starbase/starbase.env   # then edit BASE_URL
sudo install -m 644 starbase.service starbase-metrics.socket /etc/systemd/system/ && sudo systemctl daemon-reload
sudo systemctl enable --now starbase-metrics.socket && sudo systemctl enable starbase
sudo install -m 644 starbase.caddy /etc/caddy/sites/
sudo runuser -u caddy -- env HOME=/var/lib/caddy caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
sudo systemctl reload caddy
```

The Caddy block goes in by hand: Caddy asks for the certificate as it loads it, so the domain's DNS records must point at the host first. Validate as Caddy's user: `caddy validate`, run as root, creates the log file root-owned, and Caddy's reload then fails.

**1. Make a deploy key** on your machine:

```sh
ssh-keygen -t ed25519 -N "" -C "github-actions@zweiundeins/starbase" -f starbase-deploy-key
```

**2. Install the deploy script and the restricted user on the server:**

```sh
scp -r deploy libretto.ch:/tmp/starbase-setup
scp starbase-deploy-key.pub libretto.ch:/tmp/starbase-setup/
ssh libretto.ch 'sudo bash /tmp/starbase-setup/setup-host.sh /tmp/starbase-setup/starbase-deploy-key.pub && rm -r /tmp/starbase-setup'
```

It should say `>> ready: CI can deploy as starbase-deploy@…`, then install the monitoring files ([Monitoring](#monitoring)).

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

## Monitoring

The host keeps its own configs for Caddy, Prometheus, Alloy and Grafana, and each loads a folder of files that the apps install: lith's guide [Metrics and monitoring](https://lith.zweiundeins.gmbh/docs/guides/monitoring#drop-in-files-for-a-shared-host) has the folders and the labels. Starbase's:

- [`starbase.caddy`](starbase.caddy) as `/etc/caddy/sites/starbase.caddy`, by hand (step 0).
- [`observability/starbase.metrics.yml`](observability/starbase.metrics.yml) as `/etc/prometheus/apps/starbase.yml`: Prometheus scrapes `127.0.0.1:8198`, labelled `app: starbase` and `slot: prod`, and the host's "lith apps" dashboard and alert rules show Starbase: `starbase_writer_*`, `starbase_live_*`, `starbase_sse_budget_*`, `go_*` and `process_*` (no `db_` or `events_` rows). Prometheus watches the folder.
- [`observability/starbase.targets.yml`](observability/starbase.targets.yml) as `/etc/prometheus/targets/starbase.yml`: a blackbox probe of the home page with the host's `http_2xx_starbase` module, which wants "Starbase" in the body. Prometheus watches the folder.
- [`observability/starbase.alloy`](observability/starbase.alloy) as `/etc/alloy/starbase.alloy`: Caddy's access log as `{job="caddy", site="starbase"}`, through the host's pipeline for access logs.

The metrics listen on loopback only, on a listener of their own: `METRICS_ADDR` in `/etc/starbase/starbase.env`, which `starbase-metrics.socket` holds on the host, never the public socket. The port is in the socket unit, the env and the scrape file; change the three together (`TestMetricsPort`). With the socket unit active, a connection to the port starts Starbase when it is stopped, so stop both: `sudo systemctl stop starbase-metrics.socket starbase`. On a host that ran Starbase before the metrics, install the socket unit and the service unit as in step 0, add `METRICS_ADDR` to the env, then `sudo systemctl daemon-reload && sudo systemctl enable --now starbase-metrics.socket && sudo systemctl restart starbase`.

`setup-host.sh` installs the scrape target, the probe and the Alloy file, each only where its service is installed, reloads Alloy, and puts the old file back if Alloy refuses the new one. Without a key it installs only these, so run it again after changing them:

```sh
scp -r deploy libretto.ch:/tmp/starbase-setup
ssh libretto.ch 'sudo bash /tmp/starbase-setup/setup-host.sh && rm -r /tmp/starbase-setup'
```

## Moving to a new domain

- On the server, change `BASE_URL` in `/etc/starbase/starbase.env`. Change the domain in `starbase.caddy` and `observability/starbase.targets.yml`, and install both again.
- On GitHub, set `STARBASE_URL` (a repository variable) to the new URL. Both the bot and this workflow's final check use it.
- `sudo caddy validate` can leave root-owned log files behind, and then `systemctl reload caddy` fails. Fix them with `sudo sh -c 'chown caddy:caddy /var/log/caddy/*.log'` (the directory isn't readable, so a plain `sudo chown … *.log` fails on the glob).
- Change both together: `BASE_URL` sets the CSP's script origin and the canonical URLs, so pages served under one domain with the other `BASE_URL` load no scripts. Keep the old domain as a permanent redirect (see `starbase.caddy`).
