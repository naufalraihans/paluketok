import type { SummaryContent } from './summary-types';

/**
 * Isi /projects/[id], disalin dari design/web/ringkasanProyek.html (template
 * beku). Belum memanggil API (Fase 9 langkah 35).
 */
export const summary: SummaryContent = {
	meta: {
		title: 'PaluKetok - Ruang Ramu Blueprint Overview',
		bodyClass: 'bg-[#F8F9FA] text-[#1E293B] antialiased flex h-screen overflow-hidden text-sm'
	},
	breadcrumb: { root: 'Proyek', project: 'Ruang Ramu', chip: 'MVP · v0.8' },
	savedAt: 'Tersimpan · 09.41',
	collaborators: [
		{ initials: 'NA', class: 'bg-[#E2E8F0] text-[#475569]' },
		{ initials: 'DP', class: 'bg-[#CBD5E1] text-[#475569]' },
		{ initials: 'FS', class: 'bg-[#94A3B8] text-white' }
	],
	shareLabel: 'Bagikan',
	exportLabel: 'Ekspor blueprint',
	pageHeader: {
		eyebrow: 'Ruang Ramu / Ringkasan',
		title: 'Rancangan yang saling terhubung.',
		subtitle: 'Dari kebutuhan pertama sampai keputusan teknis, dalam satu benang merah.'
	},
	overview: {
		monogram: 'Rr.',
		name: 'Ruang Ramu',
		status: 'Dalam perencanaan',
		description: 'Platform reservasi workshop kreatif di Jakarta.',
		facts: ['Web responsif', 'Pemilik: Nadia Ayu', 'Target rilis: 30 Nov 2026'],
		readiness: {
			label: 'Kesiapan blueprint',
			percent: 72,
			note: 'Menuju review rancangan · 14 Okt'
		}
	},
	metrics: [
		{
			value: '12',
			label: 'Halaman website',
			note: '9 siap · 3 dalam rancangan',
			icon: 'layout-right'
		},
		{ value: '6', label: 'Entitas data', note: '7 relasi sudah dipetakan', icon: 'db-simple' },
		{
			value: '18',
			label: 'Kebutuhan produk',
			note: '14 disetujui · 4 perlu review',
			icon: 'document'
		},
		{
			value: '8/10',
			label: 'Kontrol proteksi',
			note: '2 keputusan masih terbuka',
			icon: 'shield'
		}
	],
	thread: {
		title: 'Benang merah rancangan',
		link: 'Lihat keterkaitan ↗',
		subtitle: 'Satu kebutuhan, ditelusuri sampai halaman, data, dan proteksinya.',
		nodes: [
			{
				tag: 'PRD · F-04',
				title: 'Reservasi workshop',
				note: 'Pilih jadwal, pesan tempat',
				icon: 'document'
			},
			{
				tag: 'HALAMAN · P-05',
				title: 'Detail workshop',
				note: '/workshop/:slug',
				icon: 'layout-right',
				mono: true
			},
			{
				tag: 'ENTITAS · E-04',
				title: 'bookings',
				note: 'user_id → session_id',
				icon: 'db-simple',
				mono: true
			},
			{
				tag: 'PROTEKSI · S-03',
				title: 'Akses berbasis peran',
				note: 'Peserta hanya lihat miliknya',
				icon: 'shield'
			}
		],
		connected: '14 kebutuhan terhubung',
		pending: '4 kebutuhan belum memiliki halaman'
	},
	security: {
		title: 'Proteksi sejak awal',
		body: 'Rencana kontrol untuk akun, reservasi, dan data peserta. Belum merupakan audit keamanan.',
		items: [
			{ label: 'RBAC peserta & admin', status: 'Direncanakan', ok: true },
			{ label: 'Validasi input & rate limit', status: 'Direncanakan', ok: true },
			{ label: 'Kebijakan retensi data', status: 'Perlu keputusan', ok: false },
			{ label: 'Pemulihan & backup', status: 'Perlu keputusan', ok: false }
		],
		link: 'Tinjau 10 kontrol proteksi →'
	},
	review: {
		title: 'Sebelum review berikutnya',
		counter: '3 terbuka',
		tasks: [
			{ label: 'Tetapkan retensi data peserta', tag: 'Proteksi · S-09', assignee: 'Dimas' },
			{ label: 'Lengkapi alur pembatalan reservasi', tag: 'Struktur · P-09', assignee: 'Nadia' },
			{ label: 'Setujui batas kapasitas per sesi', tag: 'PRD · F-07', assignee: 'Farah' }
		],
		feedTitle: 'Catatan perjalanan',
		activities: [
			{ time: '09.32', text: 'Dimas memperbarui relasi bookings' },
			{ time: '09.15', text: 'Nadia menautkan F-04 ke Detail workshop' },
			{ time: 'Kemarin', text: 'Farah menyetujui tujuan produk' }
		],
		feedLink: 'Semua aktivitas ↗'
	},
	footerLeft: 'RR-2026 / BLUEPRINT 008',
	footerRight: 'Rancangan yang jelas, sebelum satu baris kode.'
};

/** Item navigasi sidebar ringkasan (bagian kiri). */
export const summaryNav = [
	{ label: 'Ringkasan', icon: 'sitemap', active: true },
	{ label: 'Struktur & prototipe', icon: 'layers-row', count: '12' },
	{ label: 'Model data · ERD', icon: 'db-simple', count: '6' },
	{ label: 'Dokumen · PRD', icon: 'document', count: '18' },
	{ label: 'Proteksi', icon: 'shield', count: '8/10' },
	{ label: 'Aktivitas proyek', icon: 'clock' },
	{ label: 'Referensi & aset', icon: 'folder-open' }
];

/** Sidebar bawah: milestone berikutnya + profil. */
export const summarySidebar = {
	milestone: {
		label: 'Milestone Berikutnya',
		title: 'Review rancangan',
		when: '14 Okt 2026 · 10.00 WIB'
	},
	user: { initials: 'NA', name: 'Nadia Ayu', role: 'Product designer' }
};
