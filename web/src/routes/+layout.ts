// adapter-static: halaman statis diprerender jadi HTML nyata.
// Rute dinamis /projects/[id] TIDAK diprerender (id-nya baru diketahui saat
// runtime) — rute itu dilayani SPA fallback /200.html dan render di klien.
//
// Catatan: `prerender` tidak bisa berupa fungsi di sini karena SvelteKit
// meng-clone konfigurasi ke worker build dan fungsi gagal di-clone
// (DataCloneError). Penolakan rute dinamis diatur di src/routes/projects/[id]/+page.ts
// lewat `export const prerender = false`, dengan `handleMissingId` di vite.config.
export const prerender = true;
