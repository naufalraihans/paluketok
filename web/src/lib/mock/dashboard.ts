import type { DashboardContent } from './dashboard-types';

/**
 * Kerangka /projects. Sejak Fase 10 halaman ini live: nama proyek, metrik,
 * kredit, dan sidebar user di-override di routes/projects/+page.svelte.
 * Nilai di sini hanya dipakai saat belum login / sebelum data tiba — jadi
 * ditulis netral, bukan "data contoh" (semua teks data contoh sudah dibuang).
 */
export const dashboard: DashboardContent = {
	meta: { title: 'PaluKetok · Daftar Proyek' },
	breadcrumb: { root: 'Proyek', current: 'Semua proyek saya' },
	notice: 'Masuk untuk melihat proyek milikmu.',
	chip: 'Belum masuk',
	credits: '0 kredit',
	topupLabel: 'Top-up',
	createLabel: '+ Buat proyek baru',
	sidebar: {
		brand: 'PaluKetok',
		studio: 'Workspace Studio',
		activeLabel: 'Proyek Aktif',
		activeProject: 'Belum dipilih',
		sectionLabel: 'Rancangan Proyek',
		items: [
			{ label: 'Daftar proyek', icon: 'folder', href: '/projects', active: true },
			{ label: 'Run berjalan', icon: 'refresh', href: '/projects' },
			{ label: 'Dokumen · PRD', icon: 'document', href: '/projects' },
			{ label: 'Model data · ERD', icon: 'database', href: '/projects' },
			{ label: 'Kontrak OpenAPI', icon: 'code', href: '/projects' },
			{ label: 'Role access', icon: 'users', href: '/projects' },
			{ label: 'Proteksi', icon: 'shield', href: '/projects' },
			{ label: 'Struktur & prototipe', icon: 'eye', href: '/projects' },
			{ label: 'Keputusan JEV', icon: 'bolt', href: '/projects' },
			{ label: 'Billing / Top-up', icon: 'credit-card', href: '/billing' }
		],
		// ponytail: meta diisi runtime oleh /projects (DashSidebar), fallback ini hanya sebelum data live tiba.
		user: { name: 'Pengguna', meta: 'memuat…' }
	},
	hero: {
		kicker: 'PALUKETOK / DAFTAR PROYEK',
		heading: 'Rancanganmu, dalam satu tempat.',
		body: 'Kelola brief, stack, dan dokumen planning dari proyek milikmu sendiri.'
	},
	// Fallback sebelum login; halaman live menimpanya dengan hitungan nyata.
	metrics: [
		{ value: '—', label: 'Proyek saya', note: 'Belum masuk' },
		{ value: '—', label: 'Siap · ready', note: 'Belum masuk' },
		{ value: '—', label: 'Draf · draft', note: 'Belum masuk' },
		{ value: '—', label: 'Arsip · archived', note: 'Belum masuk' }
	],
	list: {
		heading: 'Semua proyek saya',
		count: 'Belum ada proyek',
		// Fallback sebelum login; kartu nyata datang dari GET /v1/projects.
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
		body: 'Buat proyek tanpa biaya kredit. Saat brief siap, AI menyusun susunan tahapnya lebih dulu, kamu tinjau sebentar, lalu mulai generate.',
		chain: '',
		action: 'Buat proyek baru'
	},
	balance: {
		heading: 'Saldo kredit',
		value: '0',
		meta: 'Belum masuk',
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