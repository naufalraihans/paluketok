import { dashboard } from '#lib/mock/dashboard';

/**
 * Prerender butuh daftar rute [id] yang diketahui saat build. Mock pakai nama
 * kartu proyek dari dashboard; nanti diganti data dari API.
 * (SvelteKit 3: `EntryGenerator` tidak diekspor; tipkan manual.)
 */
export const entries = () =>
	dashboard.list.cards.map((p) => ({ id: p.name.toLowerCase().replace(/\s+/g, '-') }));

export const prerender = true;
