# Monitor Engine Go

سرویس مانیتورینگ وب‌سایت با **مصرف منابع کم** و **پکیج‌های قابل فروش**.

## اصول

| هدف | پیاده‌سازی |
|-----|------------|
| مرورگر مشترک | یک Browser برای کل سرویس |
| ایزولاسیون | هر درخواست → Context کاملاً تازه |
| همزمانی | درخواست‌ها بدون سقف برنامه‌ای و با Context مستقل اجرا می‌شوند |
| پکیج‌محور | ۲۰ پکیج مستقل قابل انتخاب |
| Docker + Zabbix | HTTP API |

```
Browser (یک‌بار)
  └── هر HTTP request
        └── New Context (تمیز) → Tracker + Collectors → Close
```

## پکیج‌ها (قابل فروش)

| پکیج | محتوا |
|------|--------|
| `availability` | Status، Redirect، DNS/TCP/TLS/TTFB/Load |
| `ssl` | Certificate، Expiry، Issuer، SAN، Wildcard، Self-Signed، TLS، Cipher، HSTS، OCSP |
| `network` | Total/Failed/Success requests، Slow، Largest، Bytes |
| `breakdown` | JS / CSS / Image / Font / XHR-Fetch / Media |
| `assets` | Largest asset، Total page size، >500KB / >1MB |
| `domains` | Unique، Third-party، CDN، External script/image/font |
| `failures` | Broken JS/CSS/Images/Fonts/APIs |
| `console` | Error / Warning / Log / Info counts |
| `exceptions` | TypeError، ReferenceError، SyntaxError، Unhandled Rejection |
| `security` | HSTS، CSP، X-Frame-Options، … + Mixed Content |
| `redirects` | HTTP→HTTPS، Chain، Suspicious domain، Loop |
| `downloads` | Download count، filenames، executable detection |
| `malware` | Heuristic: miner، base64 scripts، obfuscation |
| `page` | Title، Meta description، Canonical، OG، Twitter Cards |
| `seo` | robots.txt، sitemap.xml، noindex، nofollow |
| `tech` | Server، CMS، Framework، CDN، Analytics |
| `performance` | Min/Max/Avg/Median/P95 size، Slowest/Fastest |
| `resources` | Counts by type |
| `thirdparty` | Third-party count/size/failures |

لیست زنده: `GET /collectors`

## اجرا

```bash
go mod tidy
go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6100.0 install chromium
go build -o monitor-server ./cmd/server
./monitor-server
```

`BROWSER_WS_ENDPOINT` can point to a remote Playwright browser server. When it
is empty, the service launches a local Chromium browser as before.

### Docker

The Go service and Playwright browser run in separate containers. The app uses
`BROWSER_WS_ENDPOINT` to connect to the browser service. Build and start both:

```bash
docker compose up --build
```

The browser websocket address defaults to `ws://browser:3000/` in Compose. To
connect the Go app to a browser service at another address, set
`BROWSER_WS_ENDPOINT` on the app container to that service's Playwright
websocket endpoint. Keep the browser endpoint on a trusted private network; it
provides remote control of the browser.

### متغیرهای محیطی

| متغیر | پیش‌فرض |
|-------|---------|
| `LISTEN_ADDR` | `:8080` |
| `BROWSER_WS_ENDPOINT` | *(empty; launch a local browser)* |
| `HEADLESS` | `true` |
| `DEFAULT_TIMEOUT` | `30s` |
| `MAX_CONCURRENT` | *(منسوخ؛ نادیده گرفته می‌شود)* |
| `CHROME_PATH` | — |
| `USER_AGENT` | `MonitorEngine-Go/2.0` |

درخواست‌های مانیتورینگ محدودیت همزمانی داخلی ندارند. هر درخواست یک Context جداگانه می‌سازد؛ تعداد زیاد درخواست‌های همزمان می‌تواند مصرف CPU و حافظه را افزایش دهد.

## API

```bash
# پکیج‌های پیش‌فرض
curl "http://localhost:8080/monitor?domain=pars.host"

# فقط پکیج‌های انتخابی
curl "http://localhost:8080/monitor?domain=pars.host&packages=availability,ssl,security"

curl "http://localhost:8080/monitor?url=https://pars.host&collectors=availability,ssl,network,breakdown"
```

## Zabbix HTTP Agent

- URL: `http://monitor:8080/monitor?domain={HOST.HOST}&packages=availability,ssl,security&timeout=25`
- JSONPath مثال:
  - `$.collectors.availability.data.http.statusCode`
  - `$.collectors.availability.data.timing.ttfb`
  - `$.collectors.ssl.data.certificateExpiryDays`
  - `$.collectors.security.data.headers.hsts`
  - `$.collectors.failures.data.totalFailedResources`
  - `$.collectors.console.data.consoleErrorCount`

## مدل فروش پکیجی

| پلن | پکیج‌ها |
|-----|---------|
| Basic | `availability,ssl,page` |
| Standard | + `network,breakdown,failures,console,security` |
| Pro | + `exceptions,redirects,seo,tech,performance,thirdparty` |
| Full | همه + `malware,downloads,assets,domains,resources` |
# go-monitor-engine
