import { defineConfig } from 'vitepress'

export default defineConfig({
	base: '/gateway/',
	title: 'gateway',
	description: 'Go library for creating application gateways with Kubernetes CRDs',
	themeConfig: {
		logo: '/logo.png',
		outline: [2, 4],
		nav: [
			{
				text: 'Getting Started',
				link: '/guide/getting-started',
			}
		],
		sidebar: [
			{
				text: 'Guide',
				items: [
					{ text: 'Introduction', link: '/' },
					{ text: 'Getting Started', link: '/guide/getting-started' },
				],
			},
			{
				text: 'Reference',
				items: [
					{ text: 'Gateway CRD', link: '/reference/crd' },
					{ text: 'Go API', link: '/reference/api' },
				],
			},
			{
				text: 'Contributing',
				items: [
					{
						text: "Guideline",
						link: '/CONTRIBUTING.md',
					},
					{
						text: "Code of conduct",
						link: '/CODE_OF_CONDUCT.md',
					},
					{
						text: "Security guidelines",
						link: '/SECURITY.md',
					},
				],
			},
		],
		editLink: {
			pattern: 'https://github.com/foomo/gateway/edit/main/docs/:path',
			text: 'Suggest changes to this page',
		},
		search: {
			provider: 'local',
		},
		footer: {
			message: 'Made with ♥ <a href="https://www.foomo.org">foomo</a> by <a href="https://www.bestbytes.com">bestbytes</a>',
		},
		socialLinks: [
			{
				icon: 'github',
				link: 'https://github.com/foomo/gateway',
			},
		],
	},
	head: [
		['meta', { name: 'theme-color', content: '#ffffff' }],
		['link', { rel: 'icon', href: '/logo.png' }],
		['meta', { name: 'author', content: 'foomo by bestbytes' }],
		['meta', { property: 'og:title', content: 'foomo/gateway' }],
		[
			'meta',
			{
				property: 'og:image',
				content: 'https://github.com/foomo/gateway/blob/main/docs/public/banner.png?raw=true',
			},
		],
		[
			'meta',
			{
				property: 'og:description',
				content: 'Go library for creating application gateways with Kubernetes CRDs',
			},
		],
		['meta', { name: 'twitter:card', content: 'summary_large_image' }],
		[
			'meta',
			{
				name: 'twitter:image',
				content: 'https://github.com/foomo/gateway/blob/main/docs/public/banner.png?raw=true',
			},
		],
		[
			'meta',
			{
				name: 'viewport',
				content: 'width=device-width, initial-scale=1.0, viewport-fit=cover',
			},
		],
	],
	markdown: {
		theme: {
			dark: 'one-dark-pro',
			light: 'github-light',
		}
	},
	sitemap: {
		hostname: 'https://foomo.github.io/gateway',
	},
	ignoreDeadLinks: true,
})
