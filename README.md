# snutt-proxy

Reverse proxy for SNU sugang syllabus pages (강의계획서).
sugang.snu.ac.kr redirects to its main page unless `Referer` is under `*.snu.ac.kr`,
so this proxy forwards requests with the required `Referer` attached, using the same
paths as the upstream so the page works without HTML rewriting.

## Routes

- `GET/POST /sugang/cc/{action}`: only `cc1XX(ajax)?.action`
- `GET /kor/**`, `/adm/**`: static assets. Paths with a `..` segment are 404. 2xx and 304 responses without `Cache-Control` get `Cache-Control: public, max-age=86400`.
- `GET /healthz`: not written to the access log

Other paths and actions are 404. `GET` routes also accept `HEAD`. A listed path with another method is 405. `/kor` and `/adm` redirect to `/kor/` and `/adm/`. Cookies are stripped in both directions. A request whose client disconnects before the upstream responds is logged with status 499.

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
