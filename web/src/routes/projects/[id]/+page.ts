/**
 * adapter-static: SvelteKit butuh minimal satu entry per rute dinamis saat
 * build. Rute ini butuh sesi/token (client-side), jadi entries mengembalikan
 * satu id placeholder semata untuk memuaskan generator — halaman asli selalu
 * render client-side dengan data /v1 (lihat +page.svelte).
 */
export const entries = () => [{ id: '00000000-0000-0000-0000-000000000000' }];

export const prerender = true;