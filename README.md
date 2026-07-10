# snutt-proxy

Reverse proxy that makes SNU sugang syllabus pages (강의계획서) accessible from SNUTT clients.

## Why

`sugang.snu.ac.kr` redirects to its main page unless the `Referer` header is under `*.snu.ac.kr`.
This applies to the syllabus page itself and to the ajax endpoints that load its tab contents,
and browsers cannot set `Referer` themselves, so the previous libproxy-based workaround broke
when SNU tightened the check. This proxy forwards requests with the required `Referer` attached.

Paths are mounted identically to the upstream so the page works without any HTML rewriting:
its root-relative asset links (`/kor/...`, `/adm/...`) and ajax calls (`/sugang/cc/...`)
resolve against this host and are forwarded as-is.

## Routes

| Route | Methods | Purpose |
|---|---|---|
| `/sugang/cc/{action}` (only `cc1XX(ajax)?.action`) | GET, POST | syllabus page, tab data ajax, excel download |
| `/kor/**`, `/adm/**` | GET | static assets |
| `/healthz` | GET | health check |

Anything else returns 404. Cookies are stripped in both directions.
Static asset responses get `Cache-Control: public, max-age=86400` when upstream sends none.

## Development

```sh
go run .          # listens on :8080 (override with PORT)
go test ./...
```

With Nix: `nix develop` for a shell with the Go toolchain, or `nix run` to build and run.

## Deployment

Pushing to `main` builds a linux/arm64 image and pushes it to OCIR as
`yny.ocir.io/ax1dvc8vmenm/snutt-{dev,prod}/snutt-proxy:<run number>`.

Kubernetes manifests live in
[waffle-world-oci](https://github.com/wafflestudio/waffle-world-oci) under
`argocd/snutt-{dev,prod}/snutt-proxy/`; bump the image tag there to release.

Served at `https://snutt-proxy.wafflestudio.com` (prod) and
`https://snutt-proxy-dev.wafflestudio.com` (dev). The snutt API returns syllabus links
pointing at these hosts from `GET /v1/course_books/official`.
