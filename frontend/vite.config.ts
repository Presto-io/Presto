import { sveltekit } from '@sveltejs/kit/vite';
import { execSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { defineConfig } from 'vite';
import type { Plugin } from 'vite';

function getGitVersion(): string {
	try {
		return execSync('git describe --tags --abbrev=0 2>/dev/null')
			.toString()
			.trim()
			.replace(/^v/, '');
	} catch {
		return 'dev';
	}
}

function mockServer(): Plugin {
	return {
		name: 'mock-server',
		configureServer(server) {
			server.middlewares.use('/mock', (req, res, next) => {
				const filePath = join(process.cwd(), 'mock', req.url || '');
				try {
					const content = readFileSync(filePath);
					const ext = filePath.split('.').pop();
					const types: Record<string, string> = {
						json: 'application/json',
						md: 'text/markdown',
						svg: 'image/svg+xml',
					};
					res.setHeader('Content-Type', types[ext || ''] || 'application/octet-stream');
					res.end(content);
				} catch {
					next();
				}
			});
		},
	};
}

export default defineConfig({
	plugins: [sveltekit(), mockServer()],
	define: {
		__APP_VERSION__: JSON.stringify(process.env.VERSION || getGitVersion())
	},
	resolve: {
		alias: {
			// Mock @wailsio/runtime for static builds (showcase)
			// The actual runtime is only needed for desktop app
			'@wailsio/runtime': join(process.cwd(), 'src/lib/wails-runtime-stub.ts')
		}
	},
	build: {
		rollupOptions: {
			output: {
				manualChunks(id) {
					if (id.includes('node_modules/codemirror') || id.includes('node_modules/@codemirror/')) {
						return 'vendor-codemirror';
					}
				}
			}
		}
	}
});
