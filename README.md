# Monitor Engine Go

سرویس مانیتورینگ وب‌سایت با **مصرف منابع کم** و **پکیج‌های قابل فروش**.

## اصول

| هدف | پیاده‌سازی |
|-----|------------|
| کمترین مصرف | یک Browser مشترک برای کل سرویس |
| بدون کش | هر درخواست → Context کاملاً تازه |
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

### Docker

```bash
docker build -f Dockerfile.simple -t monitor-engine-go .
docker run --rm -p 8080:8080 --shm-size=256m monitor-engine-go
```

### متغیرهای محیطی

| متغیر | پیش‌فرض |
|-------|---------|
| `LISTEN_ADDR` | `:8080` |
| `HEADLESS` | `true` |
| `DEFAULT_TIMEOUT` | `30s` |
| `MAX_CONCURRENT` | `4` |
| `CHROME_PATH` | — |
| `USER_AGENT` | `MonitorEngine-Go/2.0` |

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
