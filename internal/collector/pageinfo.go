package collector

type PageInfo struct{}

func (c *PageInfo) Name() string { return "page" }

func (c *PageInfo) Collect(ctx *Context) (interface{}, error) {
	result, err := ctx.Page.Evaluate(`() => {
		const meta = (name) => {
			const el = document.querySelector('meta[name="'+name+'"]') ||
			           document.querySelector('meta[property="'+name+'"]');
			return el ? el.getAttribute('content') : null;
		};
		const robots = meta('robots');
		const canonical = document.querySelector('link[rel="canonical"]');
		const ogTitle = meta('og:title');
		const twCard = meta('twitter:card');
		return {
			title: document.title || null,
			metaDescription: meta('description'),
			canonicalUrl: canonical ? canonical.href : null,
			robotsMeta: robots,
			openGraphExists: !!(ogTitle || meta('og:image') || meta('og:type')),
			twitterCardsExists: !!twCard
		};
	}`)
	if err != nil {
		return nil, err
	}
	return result, nil
}
