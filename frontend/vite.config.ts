import { sveltekit } from '@sveltejs/kit/vite';
import adapter from '@sveltejs/adapter-static';
import { defineConfig } from 'vite';

const apiTarget = process.env.YU_API ?? 'http://127.0.0.1:8080';

export default defineConfig({
	plugins: [
		sveltekit({
			adapter: adapter({ fallback: 'index.html', strict: false })
		})
	],
	server: {
		proxy: {
			'/api': { target: apiTarget, changeOrigin: false },
			'/collect.js': { target: apiTarget, changeOrigin: false }
		}
	}
});
