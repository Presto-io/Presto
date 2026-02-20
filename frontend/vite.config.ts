import { sveltekit } from '@sveltejs/kit/vite';
import { execSync } from 'node:child_process';
import { defineConfig } from 'vite';

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

export default defineConfig({
	plugins: [sveltekit()],
	define: {
		__APP_VERSION__: JSON.stringify(process.env.VERSION || getGitVersion())
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
