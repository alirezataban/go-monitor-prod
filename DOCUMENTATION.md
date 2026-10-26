# Monitor Engine Go — مستندات کامل

سرویس HTTP برای مانیتورینگ وب‌سایت با مرورگر واقعی (Chromium / Playwright).

طراحی شده برای استفاده در کنار Zabbix (یا هر سیستم مانیتورینگ دیگر) با امکان فروش **پکیج‌محور**.

---

## ۱. معرفی و معماری

### هدف

- بررسی واقعی صفحه با مرورگر (نه فقط HTTP ساده)
- جمع‌آوری متریک‌های Availability، SSL، شبکه، امنیت، SEO و …
- خروجی یکپارچه JSON
- مصرف منابع پایین

### معماری

```
┌─────────────────────────────────────────┐
│           monitor-server (HTTP)         │
│                                         │
│  Browser Chromium  ← یک بار در استارت   │
│         │                               │
│         ▼                               │
│  هر درخواست /monitor                    │
│    → Context جدید (بدون کش/کوکی)        │
│    → Page + Tracker                     │
│    → پکیج‌های درخواستی                  │
│    → Close Context                      │
│    → JSON Response                      │
└─────────────────────────────────────────┘
```

| اصل | توضیح |
|-----|--------|
| Browser مشترک | یک instance برای کل عمر سرویس |
| Context تازه | هر چک ایزوله است؛ کش و کوکی و Storage بین چک‌ها منتقل نمی‌شود |
| محدودیت همزمانی | حداکثر تعداد چک موازی قابل تنظیم است |
| پکیج‌محور | فقط پکیج‌های لازم اجرا می‌شوند |

---

## ۲. نصب و راه‌اندازی

### پیش‌نیاز

- Go 1.22+
- دسترسی به اینترنت برای نصب مرورگر Playwright (一次性)

### ساخت از سورس

```bash
git clone <repo>
cd monitor-engine-go

go mod tidy
go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6100.0 install chromium
# در سرور لینوکس بدون GUI، در صورت نیاز:
# go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6100.0 install --with-deps chromium

go build -o monitor-server ./cmd/server
./monitor-server
```

سرویس به‌صورت پیش‌فرض روی پورت **8080** گوش می‌دهد.

### Docker

```bash
docker build -f Dockerfile.simple -t monitor-engine-go .
docker run --rm -p 8080:8080 --shm-size=256m \
  -e MAX_CONCURRENT=4 \
  -e DEFAULT_TIMEOUT=30s \
  monitor-engine-go
```

> **نکته:** برای Chromium در کانتینر، `--shm-size=256m` (یا بیشتر) توصیه می‌شود.

---

## ۳. تنظیمات سرویس (Environment Variables)

این مقادیر هنگام **استارت** سرویس خوانده می‌شوند.

| متغیر | پیش‌فرض | توضیح |
|-------|---------|--------|
| `LISTEN_ADDR` | `:8080` | آدرس و پورت HTTP |
| `HEADLESS` | `true` | اجرای مرورگر بدون UI |
| `CHROME_PATH` | *(خالی)* | مسیر اجرایی Chrome/Chromium سیستم (اختیاری) |
| `DEFAULT_TIMEOUT` | `30s` | تایم‌اوت پیش‌فرض هر چک (مثلاً `25s`، `1m`) |
| `MAX_CONCURRENT` | `4` | حداکثر تعداد چک همزمان |
| `USER_AGENT` | `MonitorEngine-Go/2.0` | User-Agent پیش‌فرض مرورگر |

### مثال

```bash
export LISTEN_ADDR=:9090
export DEFAULT_TIMEOUT=45s
export MAX_CONCURRENT=2
export CHROME_PATH=/usr/bin/google-chrome
export HEADLESS=true
./monitor-server
```

---

## ۴. API

### ۴.۱. سلامت سرویس

```
GET /health
```

**پاسخ موفق:**
```json
{ "status": "ok", "browser": "ready" }
```

اگر مرورگر آماده نباشد → HTTP `503`.

---

### ۴.۲. لیست پکیج‌ها

```
GET /collectors
```

لیست نام پکیج‌ها، پکیج‌های پیش‌فرض و توضیح کوتاه هر کدام را برمی‌گرداند.

---

### ۴.۳. اجرای مانیتور

```
GET  /monitor
POST /monitor
```

#### پارامترهای Query (مناسب Zabbix HTTP Agent)

| پارامتر | الزامی | پیش‌فرض | توضیح |
|---------|--------|---------|--------|
| `url` | یکی از url یا domain | — | آدرس کامل، مثلاً `https://example.com` |
| `domain` | یکی از url یا domain | — | فقط دامنه؛ سرویس `https://` اضافه می‌کند |
| `packages` | خیر | پکیج‌های پیش‌فرض | لیست پکیج‌ها با کاما |
| `collectors` | خیر | همان packages | نام مستعار packages |
| `timeout` | خیر | از `DEFAULT_TIMEOUT` | ثانیه (حداکثر ۱۲۰) |
| `insecure` | خیر | `false` | `true` / `1` = نادیده گرفتن خطای SSL در مرورگر |
| `ua` | خیر | از `USER_AGENT` | User-Agent سفارشی |
| `wait` | خیر | `load` | `load` \| `domcontentloaded` \| `networkidle` |

#### Body JSON (POST)

```json
{
  "url": "https://pars.host",
  "domain": "pars.host",
  "collectors": ["availability", "ssl", "security"],
  "timeout": 25,
  "insecure": false,
  "userAgent": "MyMonitor/1.0",
  "viewportWidth": 1920,
  "viewportHeight": 1080,
  "waitUntil": "load"
}
```

Query string در صورت وجود، روی body اولویت دارد.

#### مثال‌ها

```bash
# ساده
curl "http://localhost:8080/monitor?domain=pars.host"

# فقط چند پکیج
curl "http://localhost:8080/monitor?domain=pars.host&packages=availability,ssl,page&timeout=25"

# با URL کامل و wait
curl "http://localhost:8080/monitor?url=https://example.com&wait=networkidle&packages=availability,network,breakdown"
```

---

## ۵. ساختار پاسخ JSON

```json
{
  "summary": {
    "success": true,
    "startedAt": 1723300000000,
    "finishedAt": 1723300005200,
    "duration": 5200,
    "error": null
  },
  "url": "https://pars.host",
  "collectors": {
    "availability": {
      "success": true,
      "duration": 12,
      "data": { ... }
    },
    "ssl": {
      "success": true,
      "duration": 85,
      "data": { ... }
    }
  }
}
```

| فیلد | معنی |
|------|------|
| `summary.success` | موفقیت کلی ناوبری/اجرا (خطای fatal نباشد) |
| `summary.duration` | مدت کل چک به میلی‌ثانیه |
| `collectors.<name>.success` | موفقیت همان پکیج |
| `collectors.<name>.duration` | زمان اجرای همان پکیج (ms) |
| `collectors.<name>.data` | داده‌های متریک |
| `collectors.<name>.error` | پیام خطا در صورت شکست پکیج |

حتی اگر یک پکیج fail شود، بقیه اجرا می‌شوند و HTTP معمولاً `200` با JSON کامل برمی‌گردد تا سیستم مانیتورینگ بتواند JSONPath بزند.

---

## ۶. پکیج‌ها و متریک‌ها

پکیج‌های **پیش‌فرض** (اگر `packages` نفرستید):

`availability`, `ssl`, `network`, `breakdown`, `failures`, `console`, `exceptions`, `security`, `page`, `performance`

---

### ۶.۱. `availability` — دسترسی و زمان‌بندی

| مسیر JSON | توضیح |
|-----------|--------|
| `data.http.statusCode` | کد وضعیت HTTP |
| `data.http.statusFamily` | خانواده: `2xx` / `3xx` / `4xx` / `5xx` |
| `data.http.ok` | وضعیت موفق بودن پاسخ |
| `data.http.statusText` | متن وضعیت |
| `data.http.finalUrl` | URL نهایی پس از redirect |
| `data.http.redirectCount` | تعداد redirect |
| `data.http.redirectChain` | زنجیره URLها |
| `data.http.redirectLoop` | تشخیص حلقه redirect |
| `data.http.httpVersion` | پروتکل (مثلاً h2) در صورت وجود |
| `data.timing.dnsLookup` | زمان DNS (ms) |
| `data.timing.tcpConnect` | زمان اتصال TCP (ms) |
| `data.timing.tlsHandshake` | زمان TLS (ms) |
| `data.timing.ttfb` | Time To First Byte (ms) |
| `data.timing.domContentLoaded` | DOMContentLoaded (ms) |
| `data.timing.loadEvent` | Load Event (ms) |
| `data.timing.fullyLoaded` | زمان بارگذاری کامل (ms) |

---

### ۶.۲. `ssl` — گواهی و TLS

| مسیر JSON | توضیح |
|-----------|--------|
| `data.sslExists` | وجود SSL |
| `data.certificateExpiryDays` | روز باقی‌مانده تا انقضا |
| `data.certificateIssuer` | صادرکننده |
| `data.subject` | Subject |
| `data.sans` | لیست SAN |
| `data.wildcardCertificate` | آیا wildcard است |
| `data.selfSigned` | آیا self-signed است |
| `data.tlsVersion` | نسخه TLS |
| `data.cipherSuite` | Cipher Suite |
| `data.hstsEnabled` | فعال بودن HSTS (از هدر پاسخ) |
| `data.ocspStapling` | وجود OCSP stapling |
| `data.valid` / `data.expired` | وضعیت اعتبار زمانی |
| `data.notBefore` / `data.notAfter` | بازه اعتبار (RFC3339) |

> این پکیج با dial مستقیم TLS کار می‌کند و حتی در صورت خطای ناوبری مرورگر قابل اجراست.

---

### ۶.۳. `network` — خلاصه درخواست‌های شبکه

| مسیر JSON | توضیح |
|-----------|--------|
| `data.totalRequests` | تعداد کل درخواست‌ها |
| `data.failedRequests` | تعداد ناموفق |
| `data.successfulRequests` | تعداد موفق |
| `data.redirectRequests` | درخواست‌های 3xx |
| `data.averageResponseTime` | میانگین مدت پاسخ (ms) |
| `data.slowRequestsCount` | تعداد درخواست‌های کند (>3s) |
| `data.totalDownloadedBytes` | حجم دانلود تقریبی |
| `data.largestResponse` | بزرگ‌ترین پاسخ (url, size, type) |

---

### ۶.۴. `breakdown` — تفکیک نوع منبع

| مسیر JSON | توضیح |
|-----------|--------|
| `data.javascript.count` / `totalSize` / `failedCount` | اسکریپت‌ها |
| `data.css.*` | استایل‌ها |
| `data.images.*` + `largestImage` | تصاویر |
| `data.fonts.*` | فونت‌ها |
| `data.xhrFetch.apiRequestCount` / `failedApiCount` | XHR و Fetch |
| `data.media.*` | رسانه |

---

### ۶.۵. `assets` — حجم و دارایی‌ها

| مسیر JSON | توضیح |
|-----------|--------|
| `data.totalPageSize` | حجم تقریبی کل صفحه (بایت) |
| `data.averageAssetSize` | میانگین حجم |
| `data.numberOfAssets` | تعداد منابع |
| `data.numberOfAssetsOver1MB` | منابع بزرگ‌تر از ۱MB |
| `data.numberOfAssetsOver500KB` | منابع بزرگ‌تر از ۵۰۰KB |
| `data.largestAssetName` / `Url` / `Type` / `Size` | بزرگ‌ترین دارایی |

---

### ۶.۶. `domains` — تحلیل دامنه

| مسیر JSON | توضیح |
|-----------|--------|
| `data.uniqueDomainsCount` | تعداد دامنه‌های یکتا |
| `data.thirdPartyDomainsCount` | دامنه‌های third-party |
| `data.cdnDomainsCount` | دامنه‌های مشکوک به CDN |
| `data.externalScriptDomains` | دامنه‌های اسکریپت خارجی |
| `data.externalImageDomains` | دامنه‌های تصویر خارجی |
| `data.externalFontDomains` | دامنه‌های فونت خارجی |
| `data.mainHost` | هاست اصلی صفحه |

---

### ۶.۷. `failures` — منابع خراب

| مسیر JSON | توضیح |
|-----------|--------|
| `data.brokenJsFiles` | لیست JSهای fail |
| `data.brokenCssFiles` | CSSهای fail |
| `data.brokenImages` | تصاویر fail |
| `data.brokenFonts` | فونت‌های fail |
| `data.brokenApis` | APIهای fail |
| `data.totalFailedResources` | جمع کل |
| `data.counts` | شمارش به تفکیک نوع |

---

### ۶.۸. `console` — کنسول مرورگر

| مسیر JSON | توضیح |
|-----------|--------|
| `data.consoleErrorCount` | تعداد error |
| `data.consoleWarningCount` | تعداد warning |
| `data.consoleLogCount` | تعداد log |
| `data.consoleInfoCount` | تعداد info |
| `data.total` | جمع پیام‌ها |
| `data.messages` | نمونه پیام‌ها (محدود) |

---

### ۶.۹. `exceptions` — خطاهای JavaScript

| مسیر JSON | توضیح |
|-----------|--------|
| `data.typeErrorCount` | TypeError |
| `data.referenceErrorCount` | ReferenceError |
| `data.syntaxErrorCount` | SyntaxError |
| `data.unhandledPromiseRejections` | Unhandled Promise |
| `data.totalJsExceptions` | جمع |
| `data.exceptionMessages` | متن خطاها (محدود) |

---

### ۶.۱۰. `security` — هدرهای امنیتی و محتوای ناامن

| مسیر JSON | توضیح |
|-----------|--------|
| `data.headers.hsts` | وجود HSTS |
| `data.headers.csp` | وجود CSP |
| `data.headers.xFrameOptions` | وجود X-Frame-Options |
| `data.headers.xContentTypeOptions` | nosniff |
| `data.headers.referrerPolicy` | وجود Referrer-Policy |
| `data.headers.permissionsPolicy` | وجود Permissions-Policy |
| `data.headers.xXssProtection` | وجود X-XSS-Protection |
| `data.contentSecurity.mixedContentDetected` | تشخیص Mixed Content |
| `data.contentSecurity.mixedContentCount` | تعداد |
| `data.contentSecurity.mixedContentUrls` | نمونه URLها |
| `data.contentSecurity.insecureHttpResources` | منابع HTTP روی صفحه HTTPS |

---

### ۶.۱۱. `redirects` — ریدایرکت

| مسیر JSON | توضیح |
|-----------|--------|
| `data.httpToHttpsRedirect` | ریدایرکت HTTP→HTTPS |
| `data.redirectChain` | زنجیره |
| `data.redirectCount` | تعداد |
| `data.redirectToSuspiciousDomain` | ریدایرکت به دامنه مشکوک |
| `data.redirectLoop` | حلقه |

---

### ۶.۱۲. `downloads` — دانلود

| مسیر JSON | توضیح |
|-----------|--------|
| `data.downloadAttemptCount` | تعداد تلاش دانلود |
| `data.downloadedFileNames` | نام فایل‌ها |
| `data.downloadedFileTypes` | پسوندها |
| `data.executableDownloadDetection` | تشخیص دانلود اجرایی |

---

### ۶.۱۳. `malware` — شاخص‌های مشکوک (Heuristic)

| مسیر JSON | توضیح |
|-----------|--------|
| `data.suspiciousRedirect` | ریدایرکت مشکوک |
| `data.cryptominerScripts` | اسکریپت‌های مشکوک به miner |
| `data.base64InjectedScripts` | اسکریپت‌های data/base64 |
| `data.obfuscatedScriptsDetection` | نشانه‌های obfuscation |
| `data.phishingDomains` | دامنه‌های مشکوک |
| `data.note` | یادآوری: فقط heuristic است |

> این پکیج جایگزین آنتی‌ویروس یا threat intelligence تخصصی نیست.

---

### ۶.۱۴. `page` — اطلاعات صفحه

| مسیر JSON | توضیح |
|-----------|--------|
| `data.title` | عنوان صفحه |
| `data.metaDescription` | meta description |
| `data.canonicalUrl` | canonical |
| `data.robotsMeta` | محتوای meta robots |
| `data.openGraphExists` | وجود Open Graph |
| `data.twitterCardsExists` | وجود Twitter Card |

---

### ۶.۱۵. `seo` — پایه SEO

| مسیر JSON | توضیح |
|-----------|--------|
| `data.robotsTxtExists` | وجود `/robots.txt` |
| `data.sitemapXmlExists` | وجود `/sitemap.xml` |
| `data.noindex` | noindex در meta |
| `data.nofollow` | nofollow در meta |

---

### ۶.۱۶. `tech` — تشخیص فناوری (تقریبی)

| مسیر JSON | توضیح |
|-----------|--------|
| `data.webServer` | هدر Server |
| `data.poweredBy` | X-Powered-By |
| `data.cacheHeaders` | وضعیت/هدر کش |
| `data.cms` | CMS احتمالی (مثلاً WordPress) |
| `data.framework` | فریم‌ورک احتمالی |
| `data.jsFramework` | کتابخانه JS در runtime |
| `data.analytics` | لیست ابزارهای analytics |
| `data.cdn` | CDN احتمالی |

> تشخیص بر اساس هدر و الگوهای HTML/JS است و ۱۰۰٪ قطعی نیست.

---

### ۶.۱۷. `performance` — آمار اندازه و سرعت منابع

| مسیر JSON | توضیح |
|-----------|--------|
| `data.minResourceSize` | حداقل حجم |
| `data.maxResourceSize` | حداکثر حجم |
| `data.averageResourceSize` | میانگین |
| `data.medianResourceSize` | میانه |
| `data.p95ResourceSize` | صدک ۹۵ |
| `data.slowestRequest` | کندترین درخواست |
| `data.fastestRequest` | سریع‌ترین درخواست |

---

### ۶.۱۸. `resources` — شمارش بر اساس نوع

| مسیر JSON | توضیح |
|-----------|--------|
| `data.scriptCount` | تعداد script |
| `data.stylesheetCount` | stylesheet |
| `data.imageCount` | image |
| `data.fontCount` | font |
| `data.fetchCount` / `xhrCount` | fetch / xhr |
| `data.mediaCount` | media |
| `data.documentCount` | document |
| `data.otherCount` | سایر |

---

### ۶.۱۹. `thirdparty` — شخص ثالث

| مسیر JSON | توضیح |
|-----------|--------|
| `data.thirdPartyRequestCount` | تعداد درخواست third-party |
| `data.thirdPartySize` | حجم تقریبی |
| `data.thirdPartyFailures` | تعداد fail |
| `data.slowThirdPartyDomains` | دامنه‌های کند |

---

## ۷. مدل استفاده پکیج‌محور (فروش)

از پنل مانیتورینگ فقط پکیج‌های مجاز مشتری را در `packages` بفرستید:

| پلن پیشنهادی | packages |
|--------------|----------|
| **Basic** | `availability,ssl,page` |
| **Standard** | Basic + `network,breakdown,failures,console,security` |
| **Pro** | Standard + `exceptions,redirects,seo,tech,performance,thirdparty` |
| **Full** | همه پکیج‌ها |

هر پکیج مستقل است؛ اضافه/کم کردن آن فقط همان بخش از متریک‌ها را تغییر می‌دهد و روی معماری سرویس اثر ندارد.

---

## ۸. یکپارچه‌سازی با Zabbix

### HTTP Agent Item

- **Type:** HTTP agent  
- **URL:**  
  `http://MONITOR_HOST:8080/monitor?domain={HOST.HOST}&packages=availability,ssl,security&timeout=25`
- **Request type:** GET  
- **Type of information:** Text  
- **Timeout:** کمی بیشتر از timeout سرویس (مثلاً ۳۰–۴۰ ثانیه)

### Preprocessing با JSONPath (نمونه)

| متریک | JSONPath |
|-------|----------|
| HTTP Status | `$.collectors.availability.data.http.statusCode` |
| TTFB | `$.collectors.availability.data.timing.ttfb` |
| Load time | `$.collectors.availability.data.timing.fullyLoaded` |
| SSL days left | `$.collectors.ssl.data.certificateExpiryDays` |
| SSL valid | `$.collectors.ssl.data.valid` |
| HSTS | `$.collectors.security.data.headers.hsts` |
| Failed resources | `$.collectors.failures.data.totalFailedResources` |
| Console errors | `$.collectors.console.data.consoleErrorCount` |
| Success کلی | `$.summary.success` |

### Dependent Items

یک Item متنی اصلی برای کل JSON، و چند Dependent Item با JSONPath برای هر متریک عددی/بولین.

---

## ۹. محدودیت‌ها و نکات عملی

1. **مصرف حافظه:** هر Context جدا است؛ با `MAX_CONCURRENT` تعداد موازی را محدود کنید.
2. **بدون کش بین چک‌ها:** عمداً Context تازه ساخته می‌شود تا نتیجه شبیه «اولین بازدید» باشد.
3. **timeout حداکثر ۱۲۰ ثانیه** در سطح درخواست API.
4. **`malware` فقط heuristic** است؛ برای امنیت سازمانی به ابزار تخصصی نیاز دارید.
5. **`tech` تقریبی** است و ممکن است false positive/negative داشته باشد.
6. در Docker حتماً `--shm-size` مناسب تنظیم شود.
7. برای سایت‌های سنگین، `wait=networkidle` ممکن است طولانی‌تر شود؛ در صورت نیاز `load` یا `domcontentloaded` استفاده کنید.

---

## ۱۰. عیب‌یابی سریع

| مشکل | بررسی |
|------|--------|
| `browser not ready` | لاگ استارت سرویس؛ نصب بودن Chromium Playwright |
| تایم‌اوت زیاد | `timeout` درخواست و `DEFAULT_TIMEOUT` |
| خطای SSL در مرورگر | `insecure=true` (فقط برای تست) |
| تعداد زیاد fail همزمان | کاهش `MAX_CONCURRENT` |
| پاسخ خالی / 503 | `GET /health` |

---

## ۱۱. خلاصه Endpointها

| Method | Path | کاربرد |
|--------|------|--------|
| GET | `/` | معرفی سرویس |
| GET | `/health` | سلامت |
| GET | `/collectors` | لیست پکیج‌ها |
| GET/POST | `/monitor` | اجرای مانیتور |

---

*نسخه مستند منطبق با Monitor Engine Go (معماری Browser مشترک + پکیج‌های قابل انتخاب).*
