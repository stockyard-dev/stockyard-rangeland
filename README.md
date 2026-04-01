# Stockyard Rangeland

**Web analytics.** Privacy-first pageview tracking. No cookies, no personal data, GDPR compliant. One script tag, full dashboard. Single binary, no external dependencies.

Part of the [Stockyard](https://stockyard.dev) suite of self-hosted developer tools.

## Quick Start

```bash
curl -sfL https://stockyard.dev/install/rangeland | sh
rangeland

# Or with Docker
docker run -p 8840:8840 -v rangeland-data:/data ghcr.io/stockyard-dev/stockyard-rangeland:latest
```

Dashboard at [http://localhost:8840/ui](http://localhost:8840/ui)

## Usage

```bash
# 1. Add your site
curl -X POST http://localhost:8840/api/sites \
  -H "Content-Type: application/json" \
  -d '{"domain":"example.com"}'

# 2. Add the script tag to your site
# <script defer data-domain="example.com" src="http://localhost:8840/js/script.js"></script>

# 3. View analytics
curl http://localhost:8840/api/sites/{id}/overview?days=7
```

## How It Works

The tracking script (~800 bytes) sends a single POST per pageview. No cookies, no localStorage, no fingerprinting. Visitor counts use a daily-rotating SHA-256 hash of IP + User-Agent — impossible to trace back to individuals.

## API

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/sites | Add site |
| GET | /api/sites | List sites |
| DELETE | /api/sites/{id} | Remove site |
| GET | /api/sites/{id}/overview | Summary stats |
| GET | /api/sites/{id}/timeseries | Daily views/visitors |
| GET | /api/sites/{id}/pages | Top pages |
| GET | /api/sites/{id}/referrers | Top referrers |
| GET | /api/sites/{id}/browsers | Browser breakdown |
| GET | /api/sites/{id}/devices | Device breakdown |
| GET | /api/sites/{id}/countries | Country breakdown |
| GET | /api/sites/{id}/os | OS breakdown |
| GET | /api/sites/{id}/realtime | Visitors in last 5 min |
| POST | /api/event | Record pageview |
| GET | /js/script.js | Tracking script |

All analytics endpoints accept `?days=N` (default 7).

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8840 | HTTP port |
| DATA_DIR | ./data | SQLite data directory |
| RETENTION_DAYS | 30 | Pageview retention |
| RANGELAND_LICENSE_KEY | | Pro license key |

## Free vs Pro

| Feature | Free | Pro ($4.99/mo) |
|---------|------|----------------|
| Sites | 1 | Unlimited |
| Pageviews/month | 10,000 | Unlimited |
| Retention | 7 days | 1 year |
| Real-time visitors | ✓ | ✓ |
| Data export | — | ✓ |
| API access | — | ✓ |

## License

Apache 2.0 — see [LICENSE](LICENSE).
