package collector

import (
	"strings"
)

type Tech struct{}

func (c *Tech) Name() string { return "tech" }

func (c *Tech) Collect(ctx *Context) (interface{}, error) {
	server, powered, cache := "", "", ""
	if ctx.Response != nil {
		server, _ = ctx.Response.HeaderValue("server")
		powered, _ = ctx.Response.HeaderValue("x-powered-by")
		cache, _ = ctx.Response.HeaderValue("x-cache")
		if cache == "" {
			cache, _ = ctx.Response.HeaderValue("cf-cache-status")
		}
		if cache == "" {
			cache, _ = ctx.Response.HeaderValue("cache-control")
		}
	}

	result, err := ctx.Page.Evaluate(`() => {
		const html = document.documentElement.innerHTML.slice(0, 200000);
		const scripts = [...document.scripts].map(s => s.src || '').join(' ');
		const gens = document.querySelector('meta[name="generator"]');
		const generator = gens ? gens.content : null;

		const has = (s) => html.includes(s) || scripts.includes(s);

		let cms = null;
		if (has('/wp-content/') || has('wordpress')) cms = 'WordPress';
		else if (has('drupal')) cms = 'Drupal';
		else if (has('joomla')) cms = 'Joomla';
		else if (generator) cms = generator;

		let framework = null;
		if (has('react') || has('__NEXT_DATA__') || has('_next/')) framework = 'React/Next';
		else if (has('ng-version') || has('angular')) framework = 'Angular';
		else if (has('vue') || has('__VUE__')) framework = 'Vue';
		else if (has('laravel')) framework = 'Laravel';

		let jsFw = null;
		if (typeof window.jQuery !== 'undefined') jsFw = 'jQuery';
		else if (typeof window.React !== 'undefined') jsFw = 'React';
		else if (typeof window.Vue !== 'undefined') jsFw = 'Vue';
		else if (typeof window.angular !== 'undefined') jsFw = 'Angular';

		let analytics = [];
		if (has('google-analytics') || has('gtag') || has('G-') || has('UA-')) analytics.push('Google Analytics');
		if (has('googletagmanager')) analytics.push('Google Tag Manager');
		if (has('clarity.ms')) analytics.push('Microsoft Clarity');
		if (has('hotjar')) analytics.push('Hotjar');
		if (has('facebook.net') || has('fbevents')) analytics.push('Facebook Pixel');

		let cdn = null;
		if (has('cloudflare')) cdn = 'Cloudflare';
		else if (has('cloudfront')) cdn = 'CloudFront';
		else if (has('fastly')) cdn = 'Fastly';
		else if (has('jsdelivr')) cdn = 'jsDelivr';

		return { generator, cms, framework, jsFramework: jsFw, analytics, cdn };
	}`)
	if err != nil {
		result = map[string]interface{}{}
	}

	out := map[string]interface{}{
		"webServer":     server,
		"poweredBy":     powered,
		"cacheHeaders":  cache,
	}
	if m, ok := result.(map[string]interface{}); ok {
		for k, v := range m {
			out[k] = v
		}
	}

	// normalize empty strings
	for _, k := range []string{"webServer", "poweredBy", "cacheHeaders"} {
		if s, ok := out[k].(string); ok && strings.TrimSpace(s) == "" {
			out[k] = nil
		}
	}
	return out, nil
}
