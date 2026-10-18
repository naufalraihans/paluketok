/**
 * Render markdown balasan model jadi HTML yang aman.
 *
 * Isi dokumen/produk ini adalah DATA pengguna (brief, artefak, nama proyek) dan
 * balasan model menyisipkannya. Model bisa saja menyalin teks dari brief yang
 * berisi HTML jahat, jadi hasil `marked` TIDAK boleh langsung dipakai
 * `{@html ...}`.
 *
 * AH biasa: aturan `markdown` dan `html` dari DOMPurify, tapi ditulis ulang.
 * Sejak aturan `stripHtml` model dilarang mengeluarkan tag, jadi TIDAK ADA
 * tag mentah yang perlu dipertahankan: pixel `tag: null` (hapus semua tag),
 * lalu kode inline `dibuat sendiri` di escape. Sisa yang lolos cuma teks, dan
 * teks tidak bisa jalan. Jadi XSS tertutup tanpa daftar putih.
 *
 * Harga yang dibayar: markdown yang butuh tag HTML mentah tidak akan tampil
 * (mis. `<details>`, `<span class>`). Untuk balasan chat, itu justru disengaja.
 */
import { marked } from 'marked';

marked.setOptions({
	gfm: true,
	breaks: true // satu baris baru = <br>, seperti chat pada umumnya
});

/**
 * escapeHtml: buat blok kode dan jalur `code` inline.
 *
 * tag di atas kode ini tidak boleh di-escape, karena `<` di dalam kode memang
 * harus tampil sebagai `<`.
 */
function escapeHtml(s: string): string {
	return s
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;')
		.replace(/'/g, '&#39;');
}

/**
 * Tag yang boleh lolos. Semua di sini berasal dari `marked`, tidak ada yang
 * dari input pengguna.
 */
const ALLOWED_TAGS = new Set([
	'p', 'br', 'hr',
	'strong', 'em', 'del', 'code', 'pre',
	'ul', 'ol', 'li',
	'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
	'blockquote', 'a',
	'table', 'thead', 'tbody', 'tr', 'th', 'td'
]);

/** Atribut: hanya href di <a>, dan hanya skema aman. */
function safeHref(raw: string): string | null {
	const url = raw.trim();
	if (/^(https?:|mailto:|#|\/)/i.test(url)) return url;
	return null; // javascript:, data:, vbscript:, dst.
}

/**
 * mdToHtml: markdown -> HTML bersih.
 */
export function mdToHtml(md: string): string {
	if (!md) return '';
	const html = marked.parse(md, { async: false }) as string;

	// Buang SETIAP tag di luar daftar putih beserta isinya untuk tag berbahaya,
	// lalu sisakan tag yang diizinkan tanpa atribut apa pun.
	return html
		.replace(/<(script|style|iframe|object|embed|form|svg|math)\b[\s\S]*?<\/\1\s*>/gi, '')
		.replace(/<(script|style|iframe|object|embed|form|svg|math)\b[^>]*\/?>/gi, '')
		.replace(/<\/?([a-zA-Z][a-zA-Z0-9]*)\b([^>]*)>/g, (_m, name: string, attrs: string) => {
			const tag = name.toLowerCase();
			if (!ALLOWED_TAGS.has(tag)) return '';
			if (tag !== 'a') return `<${_m.slice(1, 2) === '/' ? '/' : ''}${tag}>`;
			const href = /href\s*=\s*["']([^"']*)["']/i.exec(attrs);
			const safe = href ? safeHref(href[1]) : null;
			if (!safe) return '';
			const open = _m.slice(0, 2) === '</' ? false : true;
			return open
				? `<a href="${escapeHtml(safe)}" target="_blank" rel="noopener noreferrer nofollow">`
				: '</a>';
		});
}
