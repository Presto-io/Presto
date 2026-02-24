import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({
			pages: 'build',
			assets: 'build',
			fallback: 'index.html'
		}),
		prerender: {
			entries: [
				'*',
				'/showcase/editor-gongwen',
				'/showcase/editor-jiaoan',
				'/showcase/batch',
				'/showcase/templates',
				'/showcase/drop',
				'/showcase/hero',
				'/showcase/editor',
				'/showcase/store-templates'
			]
		}
	},
	preprocess: vitePreprocess()
};

export default config;
