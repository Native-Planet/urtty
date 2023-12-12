import { sveltekit } from '@sveltejs/kit/vite';

/** @type {import('vite').UserConfig} */
const config = {
	plugins: [sveltekit()]
};

config["server"] = {
	proxy: {
	'^/session.js': {
		target: 'http://127.0.0.1:8080/',
		changeOrigin: true
	},
	'^/spider/.*': {
		target: 'http://127.0.0.1:8080/',
		changeOrigin: true
	},
	'^/~/.*': {
		target: 'http://127.0.0.1:8080/',
		changeOrigin: true
	}
	},
	cors: true
}  

export default config;
