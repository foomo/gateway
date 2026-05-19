# Gateway Example

Minimal demo of `foomo/gateway` routing — no Docker, no Kubernetes required.

## Run

```bash
go run main.go
```

Everything runs in a single process:

| Service            | Role                                      |
|--------------------|-------------------------------------------|
| Mock contentserver | Resolves URIs to mime types               |
| Page frontend      | Handles `application/x-sandbox-page`      |
| News frontend      | Handles `application/x-sandbox-news`      |
| Error frontend     | Handles 404 / 500 (`errorFrontend: true`) |

## Try it

```bash
# known page URI
curl -i http://localhost:8080/

# another page
curl -i http://localhost:8080/about

# news article (different mime type → different frontend)
curl -i http://localhost:8080/news/article-1

# unknown URI → routed to error frontend
curl -i http://localhost:8080/missing

# built-in gateway endpoints
curl -i http://localhost:8080/robots.txt
curl -i http://localhost:8080/sitemap.xml
curl -i http://localhost:8080/gateway/status
```

## What to look for

- `X-Content-Id` and `X-Mime-Type` headers are set on the proxied request — the frontend reads them to fetch page
  payload and site context in parallel.
- `X-Error-Code` is set when routing to the error frontend.
- `/sitemap.xml`, `/robots.txt`, and `/gateway/status` are handled by the gateway itself.

## Content tree

The mock contentserver (`contentTree` in `main.go`) defines the known URIs and their mime types. Edit it to try
different routing scenarios.

In a real project the contentserver is a separate service loaded from your CMS. The gateway is unaware of mime type
values — it routes whatever string the contentserver returns.
