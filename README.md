# snutt-proxy

Reverse proxy for SNU sugang syllabus pages (강의계획서).
sugang.snu.ac.kr redirects to its main page unless `Referer` is under `*.snu.ac.kr`,
so this proxy forwards requests with the required `Referer` attached, using the same
paths as the upstream so the page works without HTML rewriting.

## Routes

- `GET/POST /sugang/cc/{action}` — only `cc1XX(ajax)?.action`
- `GET /kor/**`, `/adm/**` — static assets
- `GET /healthz`

Anything else is 404. Cookies are stripped in both directions.

## Development

```sh
go run .          # :8080, override with PORT
go test ./...
```

## Deployment

Pushing to `main` builds and pushes
`yny.ocir.io/ax1dvc8vmenm/snutt-prod/snutt-proxy:<run number>`.
Manifests are in [waffle-world-oci](https://github.com/wafflestudio/waffle-world-oci)
under `argocd/snutt-prod/snutt-proxy/`. Served at `https://snutt-proxy.wafflestudio.com`.
