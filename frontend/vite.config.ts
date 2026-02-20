import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
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
