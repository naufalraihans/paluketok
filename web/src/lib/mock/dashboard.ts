import type { DashboardContent } from './dashboard-types';

/**
 * Isi /projects, disalin dari design/web/dasrhboard.html (template beku).
 * Belum memanggil API (Fase 9 langkah 35). Semua angka dan teks = data contoh
 * persis seperti mockup; menggantinya nanti = tugas langkah 36.
 */
export const dashboard: DashboardContent = {
	meta: { title: 'PaluKetok · Daftar Proyek' },
	breadcrumb: { root: 'Proyek', current: 'Semua proyek saya' },
	notice:
		'PaluKetok Demo adalah proyek contoh dengan brief toko online, bukan jenis sistem PaluKetok. Semua kartu di atas adalah data contoh.',
	chip: 'Data contoh',
	credits: '8 kredit',
	topupLabel: 'Top-up',
	createLabel: '+ Buat proyek baru',
	sidebar: {
		brand: 'PaluKetok',
		studio: 'Workspace Studio',
		activeLabel: 'Proyek Aktif',
		activeProject: 'PaluKetok Demo',
		sectionLabel: 'Rancangan Proyek',
		items: [
			{ label: 'Daftar proyek', icon: 'folder', href: '#', active: true },
			{ label: 'Run berjalan', icon: 'refresh', href: '#' },
			{ label: 'Dokumen · PRD', icon: 'document', href: '#' },
			{ label: 'Model data · ERD', icon: 'database', href: '#' },
			{ label: 'Kontrak OpenAPI', icon: 'code', href: '#' },
			{ label: 'Role access', icon: 'users', href: '#' },
			{ label: 'Proteksi', icon: 'shield', href: '#' },
			{ label: 'Struktur & prototipe', icon: 'eye', href: '#' },
			{ label: 'Keputusan JEV', icon: 'bolt', href: '#' },
			{ label: 'Billing / Top-up', icon: 'credit-card', href: '#' }
		],
		user: { name: 'Naufal', meta: 'free · 8 kredit · data contoh' }
	},
	hero: {
		kicker: 'PALUKETOK / DAFTAR PROYEK',
		heading: 'Rancanganmu, dalam satu tempat.',
		body: 'Kelola brief, stack, dan dokumen planning dari proyek milikmu sendiri.'
	},
	metrics: [
		{ value: '3', label: 'Proyek saya', note: 'Data contoh' },
		{ value: '1', label: 'Siap · ready', note: 'Data contoh' },
		{ value: '1', label: 'Draf · draft', note: 'Data contoh' },
		{ value: '1', label: 'Arsip · archived', note: 'Data contoh' }
	],
	list: {
		heading: 'Semua proyek saya',
		count: '3 proyek · data contoh',
		cards: [
			{
				name: 'PaluKetok Demo',
				brief: 'Website toko online sederhana, fitur lengkap',
				stack: 'Svelte + TypeScript / Go / Supabase / Midtrans',
				status: 'ready',
				meta: '6 artefak tersedia · pemilik Naufal',
				action: 'Buka proyek'
			},
			{
				name: 'API internal',
				brief: 'Rencana API internal untuk pengelolaan data aplikasi.',
				stack: 'Svelte + TypeScript / Go / Supabase / Midtrans',
				status: 'draft',
				meta: 'Belum ada run · pemilik Naufal',
				action: 'Buka proyek'
			},
			{
				name: 'Demo arsip',
				brief: 'Contoh rancangan lama yang disimpan sebagai arsip.',
				stack: 'Svelte + TypeScript / Go / Supabase / Midtrans',
				status: 'archived',
				meta: 'Diarsipkan · pemilik Naufal',
				action: 'Lihat arsip'
			}
		]
	},
	pipeline: {
		heading: 'Dari brief ke planning',
		body: 'Buat proyek tanpa biaya kredit. Saat brief siap, mulai generate untuk menyusun enam dokumen secara berurutan.',
		chain: 'prd → erd → openapi → roles → security → structure',
		action: 'Buat proyek baru'
	},
	balance: {
		heading: 'Saldo kredit',
		value: '8',
		meta: 'Data contoh · tier free',
		note: '1 run = 1 kredit. Regenerate satu tahap = 1 kredit.',
		action: 'Top-up kredit'
	},
	states: {
		heading: 'State halaman',
		note: 'Contoh tampilan alternatif · bukan data aktif',
		items: [
			{
				badge: 'EMPTY',
				tone: 'slate',
				title: 'Belum ada proyek',
				body: '0 proyek. Mulai dengan nama dan brief kebutuhan software.',
				action: 'Buat proyek baru'
			},
			{
				badge: 'LOADING',
				tone: 'blue',
				title: 'Memuat daftar proyek...',
				body: '',
				action: ''
			},
			{
				badge: 'ERROR',
				tone: 'rose',
				title: 'Permintaan tidak berhasil',
				body: 'Terjadi kesalahan atau terlalu banyak permintaan. Coba muat ulang daftar proyek.',
				action: 'Muat ulang daftar'
			}
		],
		loadingNote: 'Tunggu hingga permintaan selesai.'
	}
};