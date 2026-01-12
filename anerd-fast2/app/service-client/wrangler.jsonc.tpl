{
	"name": "__CONTEXT_NAME__",
	"main": ".svelte-kit/cloudflare/_worker.js",
    "compatibility_date": "2025-09-23",
	"workers_dev": true,
	"preview_urls": true,
	"build": {
		"command": "npm run build"
	},
	"routes": [
		{
			"pattern": "__CLIENT_DOMAIN__",
			"custom_domain": true
		}
	],
	"assets": {
		"binding": "ASSETS",
        "directory": ".svelte-kit/cloudflare"
	}
}
