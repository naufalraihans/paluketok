import type { SummaryContent } from './summary-types';

/**
 * Kerangka halaman /projects/[id] disalin dari design/web/ringkasanProyek.html
 * (template beku). Halaman mengisinya dengan data live + nilai netral.
 *
 * ATURAN: tidak boleh ada angka, nama orang, atau dokumen yang dikarang di
 * sini. Nilai contoh dari mockup (mis. "12 Halaman website", "Dimas", "F-04")
 * sudah dibuang: semuanya tampil sebagai hasil analisis padahal cuma
 * placeholder, dan di halaman proyek pengguna angka palsu lebih buruk daripada
 * kolom kosong.
 *
 * Blok yang isinya bisa dihitung dari artefak (metrik, benang merah, proteksi)
 * dihitung di halaman; yang tidak bisa dihitung tetap di sini dengan teks
 * netral.
 */

/** Placeholder seragam untuk nilai yang tidak bisa dihitung dari artefak. */
const NONE = '—';

export const summary: SummaryContent = {
	meta: {
		title: 'PaluKetok - Ringkasan Proyek',
		bodyClass: 'bg-[#F8F9FA] text-[#1E293B] antialiased flex h-screen overflow-hidden text-sm'
	},
	breadcrumb: { root: 'Proyek', project: '', chip: '' },
	pageHeader: {
		eyebrow: '',
		title: '',
		subtitle: ''
	},
	overview: {
		monogram: '',
		name: '',
		status: '',
		description: '',
		facts: [],
		readiness: {
			label: 'Progres generate',
			percent: 0,
			note: ''
		}
	},
	metrics: [
		{ value: NONE, label: 'Dokumen', note: '', icon: 'document', kind: 'prd' },
		{ value: NONE, label: 'Entitas data', note: '', icon: 'db-simple', kind: 'erd' },
		{ value: NONE, label: 'Endpoint', note: '', icon: 'code', kind: 'openapi' },
		{ value: NONE, label: 'Kontrol proteksi', note: '', icon: 'shield', kind: 'security' }
	],
	thread: {
		title: 'Dokumen terbaru',
		// Halaman keterkaitan antar dokumen belum ada; label, bukan link mati.
		link: '',
		subtitle: 'Versi terakhir tiap tahap untuk proyek ini.',
		nodes: [],
		connected: '',
		pending: ''
	},
	security: {
		title: 'Proteksi sejak awal',
		body: 'Belum ada dokumen proteksi untuk proyek ini.',
		items: [],
		link: ''
	},
	review: {
		title: 'Aktivitas dokumen',
		counter: '',
		feedTitle: 'Versi terakhir',
		activities: [],
		feedLink: ''
	},
	footerLeft: '',
	footerRight: ''
};

/** Label kotak progres di sidebar: teks tetap, angkanya dari halaman. */
export const summarySidebar = {
	milestone: { label: 'Progres generate' }
};
