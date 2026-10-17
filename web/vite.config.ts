import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

/**
 * SvelteKit 3: konfigurasi Kit lewat plugin `sveltekit(...)` di sini,
 * bukan svelte.config.js. adapter-static: FE di-serve Go (server/) tanpa proses
 * Node di VPS. prerender dinyatakan di src/routes/+layout.ts.
 */
export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			adapter: adapter({
				pages: 'build',
				assets: 'build',
				// Rute dinamis (/projects/[id]) tidak bisa diprerender: id-nya
				// tak diketahui saat build. Tanpa fallback, nginx jatuh ke
				// /index.html sehingga halaman detail render landing page.
				fallback: '200.html',
				precompress: false,
				strict: true
			}),
			prerender: {
				// Mockup punya anchor tanpa id tujuan (#mulai/#masuk/#contoh): tidak ada
				// section-nya di landingPage.html. Perilaku sama: link mati, tidak diubah.
				handleMissingId: 'ignore'
			}
		})
	]
});
