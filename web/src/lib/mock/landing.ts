import type { LandingContent } from './types';

/**
 * Isi landing page, dipakai Fase 9 langkah 35 (belum memanggil API).
 *
 * Asal teks: seluruhnya dari design/web/landingPage.html (template beku).
 * Bagian yang di mockup menampilkan proyek contoh "Ruang Ramu" (website
 * reservasi workshop) tetap apa adanya: itu memang contoh ilustratif yang
 * sengaja ditampilkan mockup, bukan data PaluKetok sendiri. Mockup menandainya
 * eksplisit "CONTOH PROYEK ILUSTRATIF / BUKAN DATA PELANGGAN".
 *
 * Fase 36 akan mengganti nilai di sini dengan respons API; bentuknya sudah
 * dipisah per blok supaya penggantian itu tidak menyentuh komponen.
 */
export const landing: LandingContent = {
	meta: {
		title: 'PaluKetok — Rancang dulu. Bangun dengan benang merah.',
		description:
			'Satukan kebutuhan, struktur halaman, prototipe, data, dan proteksi. Supaya setiap keputusan punya alasan—sebelum menjadi kode.'
	},

	nav: {
		links: [
			{ label: 'Produk', href: '#produk' },
			{ label: 'Cara kerja', href: '#cara-kerja' },
			{ label: 'Untuk siapa', href: '#untuk-siapa' }
		],
		signIn: { label: 'Masuk', href: '/login' },
		primaryCta: { label: 'Mulai merancang', href: '/register' }
	},

	hero: {
		titleLead: 'Rancang dulu.',
		titleAccent: 'Bangun dengan ·benang merah·',
		subtitle:
			'Satukan kebutuhan, struktur halaman, prototipe, data, dan proteksi. Supaya setiap keputusan punya alasan—sebelum menjadi kode.',
		ctaPrimary: { label: 'Mulai rancanganmu', href: '/register' },
		ctaSecondary: { label: 'Jelajahi contoh proyek', href: '/projects' },
		window: {
			url: 'paluketok.id / ruang-ramu',
			status: 'Dalam perencanaan · v0.8',
			sidebarTitle: 'Ruang Ramu',
			sidebar: [
				{ label: 'Dokumen PRD', active: false, icon: 'document' },
				{ label: 'Sitemap', active: true, icon: 'sitemap' },
				{ label: 'Prototipe', active: false, icon: 'cursor' },
				{ label: 'Model Data ERD', active: false, icon: 'database' },
				{ label: 'Rencana Proteksi', active: false, icon: 'shield' }
			],
			canvasTitle: 'Ruang Ramu - Peta halaman',
			canvasSubtitle: 'Visualisasi arsitektur navigasi & hierarki akses',
			canvasBadge: '12 halaman · 3 area akses',
			rootName: 'Ruang Ramu',
			rootMeta: 'ruangramu.id',
			groups: [
				{
					title: 'Area publik',
					meta: 'Tanpa autentikasi',
					highlightIndex: 0,
					nodes: ['Detail workshop', 'Beranda', 'Eksplorasi workshop']
				},
				{
					title: 'Area peserta',
					meta: 'Role: participant',
					nodes: ['Reservasi', 'Pembayaran', 'Reservasi saya']
				},
				{
					title: 'Area admin',
					meta: 'Role: admin',
					nodes: ['Dashboard', 'Workshop', 'Jadwal & kapasitas']
				}
			],
			footerStrip: 'GARIS IDE KE BLUEPRINT',
			tabs: [
				{ label: '# Dokumen PRD', active: false },
				{ label: '# Sitemap', active: true },
				{ label: '# Rencana prototipe', active: false },
				{ label: '# Model data ERD', active: false },
				{ label: '# Rencana proteksi', active: false }
			]
		}
	},

	problem: {
		headingLine1: 'Yang sulit bukan memulai.',
		headingLine2: 'Yang sulit: tetap searah.',
		body: 'Brief ada di dokumen. Halaman ada di papan desain. Data dibahas di chat. Saat satu hal berubah, konteksnya mudah tertinggal. PaluKetok memberi semuanya satu tempat untuk bertemu.',
		quote1: '“Ini dibuat untuk kebutuhan yang mana?”',
		quote2: '“Ini alasannya. Ini kaitannya.”'
	},

	feature: {
		heading: 'Peta sebelum perjalanan.',
		body: 'Lihat bagaimana Ruang Ramu merencanakan website reservasi workshop: dari area akses sampai alur pesan tempat.',
		disclaimer: 'CONTOH PROYEK ILUSTRATIF · BUKAN DATA PELANGGAN',
		toolbar: {
			mark: 'Rr.',
			name: 'Ruang Ramu',
			section: 'Struktur & prototipe',
			status: 'Dalam perencanaan · v0.8'
		},
		tabs: [
			{ label: 'Sitemap', active: true },
			{ label: 'Alur pengguna', active: false },
			{ label: 'Prototipe', active: false }
		],
		meta: '12 halaman · 3 area akses',
		rootName: 'Ruang Ramu',
		groups: [
			{
				title: 'Area publik',
				meta: 'Tanpa autentikasi',
				highlightIndex: 2,
				nodes: ['Beranda', 'Eksplorasi workshop', 'Detail workshop']
			},
			{
				title: 'Area peserta',
				meta: 'Role: participant',
				nodes: ['Reservasi', 'Pembayaran', 'Reservasi saya']
			},
			{
				title: 'Area admin',
				meta: 'Role: admin',
				nodes: ['Dashboard', 'Workshop', 'Jadwal & kapasitas']
			}
		],
		flowTitle: 'Alur prototipe: pesan tempat',
		flowNote: 'Rencana tampilan & state · bukan prototipe aktif',
		flowSteps: [
			{ title: 'Detail workshop', state: 'Sesi tersedia', code: 'P-03 → F-04', active: false },
			{ title: 'Isi reservasi', state: 'Jadwal dipilih', code: 'P-04 → F-04', active: true },
			{ title: 'Konfirmasi', state: 'Reservasi pending', code: 'P-05 → F-06', active: false }
		],
		detail: {
			ref: 'DETAIL HALAMAN',
			code: 'P-03',
			title: 'Detail workshop',
			badge: 'Siap prototipe',
			body: 'Membantu peserta memahami workshop dan memilih sesi yang tersedia.',
			traceLabel: 'BENANG MERAH',
			trace: [
				{ tone: 'brand', label: 'F-04:', text: 'Reservasi workshop ↗' },
				{ tone: 'emerald', label: 'Data:', text: 'workshops, sessions' },
				{ tone: 'amber', label: 'S-02:', text: 'Sanitasi konten publik' }
			],
			footerNote: 'Konteks tersinkronisasi otomatis dengan PRD dan skema basis data.'
		},
		footerStrip:
			'Bukan hanya daftar halaman. Setiap halaman membawa tujuan, kebutuhan, dan batas aksesnya.',
		footerCta: 'Jelajahi blueprint Ruang Ramu'
	},

	traceability: {
		headingLine1: 'Satu kebutuhan.',
		headingLine2: 'Telusuri sampai akarnya.',
		body: 'Bukan lima file yang kebetulan satu folder. Ini hubungan yang bisa ditelusuri, dari niat produk sampai keputusan teknis.',
		nodes: [
			{
				ref: 'PRD / F-04',
				question: 'Apa yang perlu dibangun?',
				answer: 'Reservasi workshop',
				detail: 'Peserta memilih sesi dan jumlah tempat.',
				icon: 'document'
			},
			{
				ref: 'HALAMAN / P-04',
				question: 'Di mana kebutuhan terjadi?',
				answer: 'Form reservasi',
				detail: 'Jadwal, kapasitas, dan konfirmasi pesanan.',
				icon: 'sitemap'
			},
			{
				ref: 'DATA / E-04',
				question: 'Data apa yang dibutuhkan?',
				answer: 'bookings',
				detail: 'user_id, session_id, quantity, status.',
				mono: true,
				icon: 'database'
			},
			{
				ref: 'PROTEKSI / S-03',
				question: 'Siapa boleh melakukan apa?',
				answer: 'Akses berbasis peran',
				detail: 'Peserta hanya mengakses reservasi miliknya.',
				icon: 'shield'
			}
		],
		callout: {
			label: 'MISALNYA, KEBUTUHAN BERUBAH...',
			quote: '“Batasi tempat per sesi.”',
			body: 'Tinjau kriteria F-07, state ‘sesi penuh’ pada halaman reservasi, dan aturan quantity pada bookings. Jejak keputusan tetap terlihat saat rancangan berkembang.',
			badge: 'Konteks ikut bergerak.'
		}
	},

	process: {
		heading: 'Dari "bagaimana kalau" ke "ini rencananya".',
		body: 'Empat langkah teratur untuk menyelaraskan ide sampai siap eksekusi.',
		steps: [
			{
				no: '01',
				title: 'Letakkan niatnya.',
				body: 'Tentukan masalah, pengguna, tujuan, dan batas MVP. Tidak harus langsung sempurna.',
				outcome: 'brief & kebutuhan awal'
			},
			{
				no: '02',
				title: 'Bentangkan petanya.',
				body: 'Turunkan kebutuhan menjadi halaman, alur prototipe, entitas, dan rencana kontrol.',
				outcome: 'blueprint lintas modul'
			},
			{
				no: '03',
				title: 'Sambungkan & tinjau.',
				body: 'Telusuri keterkaitan. Bahas asumsi dan tandai keputusan yang masih terbuka.',
				outcome: 'rancangan untuk review'
			},
			{
				no: '04',
				title: 'Bangun dari acuan.',
				body: 'Bawa sitemap, ERD, dan PRD ke proses implementasi dengan konteks yang utuh.',
				outcome: 'acuan untuk handoff'
			}
		],
		footnote:
			'Alurnya tidak kaku. Kembali, revisi, dan sambungkan lagi saat kamu menemukan hal baru.'
	},

	personas: {
		heading: 'Berbeda peran. Satu bahasa rancangan.',
		body: 'Untuk proyek mandiri maupun diskusi lintas fungsi.',
		items: [
			{
				kicker: 'FOUNDER & PRODUCT',
				title: 'Tajamkan MVP-mu.',
				body: 'Pisahkan kebutuhan inti dari ide nanti. Bawa lingkup yang jelas ke percakapan dengan tim teknis.',
				scenarioTitle: 'Platform reservasi workshop',
				scenarioBody: 'Katalog, sesi, dan reservasi masuk MVP. Program loyalitas menunggu.',
				icon: 'bulb'
			},
			{
				kicker: 'DESIGNER & STUDIO',
				title: 'Bawa alasan ke desain.',
				body: 'Ubah brief menjadi struktur dan rencana layar yang bisa dibahas bersama klien, bukan sekadar tampilan.',
				scenarioTitle: 'Website layanan konsultasi',
				scenarioBody: 'Petakan penemuan layanan + pilih konsultan + jadwalkan sesi.',
				icon: 'puzzle'
			},
			{
				kicker: 'DEVELOPER & ENGINEERING',
				title: 'Bangun dengan konteks.',
				body: 'Pahami asal sebuah field, kebutuhan halaman, dan aturan akses sebelum memutuskan implementasinya.',
				scenarioTitle: 'Portal operasional internal',
				scenarioBody: 'Hubungan peran staf, data pengajuan, dan alur persetujuan.',
				icon: 'code'
			}
		]
	},

	security: {
		heading: 'Keamanan bukan catatan di akhir.',
		body: 'Saat merancang data dan halaman, rancang juga batas aksesnya. Jadikan risiko bagian dari keputusan produk, bukan pekerjaan yang terlupakan.',
		points: [
			{ title: 'Siapa yang boleh mengakses?', body: 'Peran, izin, dan kepemilikan data.' },
			{ title: 'Data apa perlu dijaga?', body: 'Field sensitif, validasi, dan retensi.' },
			{ title: 'Apa yang belum diputuskan?', body: 'Kontrol terbuka, pemilik, dan tindak lanjut.' }
		],
		disclaimer:
			'PaluKetok membantu merencanakan kontrol. Bukan audit keamanan, pemindai kerentanan, atau jaminan bahwa implementasi sudah aman.',
		panel: {
			title: 'Data punya batas akses.',
			ref: 'Ruang Ramu / Ilustrasi',
			tables: [
				{
					name: 'users',
					ref: 'E-01',
					columns: [
						{ name: 'PK id', key: true, type: 'uuid' },
						{ name: 'email', type: '', sensitive: true },
						{ name: 'role', type: 'text' }
					]
				},
				{
					name: 'bookings',
					ref: 'E-04',
					columns: [
						{ name: 'PK id', key: true, type: 'uuid' },
						{ name: 'FK user_id', key: true, type: 'uuid' },
						{ name: 'quantity', type: 'int' },
						{ name: 'status', type: 'text' }
					]
				}
			],
			policy: {
				code: 'S-03 · Akses berbasis peran',
				lines: [
					{ actor: 'Peserta:', text: 'Hanya membaca dan mengelola reservasi miliknya.' },
					{ actor: 'Admin:', text: 'Mengelola reservasi sesuai izin operasional.' }
				]
			},
			pending: {
				code: 'S-09 · Kebijakan retensi data peserta',
				title: 'Batas penyimpanan email & nomor kontak',
				badge: 'Perlu keputusan'
			}
		}
	},

	finalCta: {
		heading: 'Rancang dari benang merah.',
		body: 'Mulai dari kebutuhan, turunkan ke struktur, prototipe, data, dan proteksi. Jangan biarkan konteksnya terpisah.',
		primary: { label: 'Mulai rancanganmu', href: '/register' },
		secondaryLabel: 'Buka contoh proyek Ruang Ramu'
	},

	footer: {
		tagline:
			'Ruang kerja untuk rancangan website, struktur, prototipe, data, dan proteksi.',
		columns: [
			{
				title: 'Produk',
				links: [
					{ label: 'Dokumen PRD', href: '/projects' },
					{ label: 'Sitemap', href: '/projects' },
					{ label: 'Prototipe', href: '/projects' },
					{ label: 'ERD', href: '/projects' },
					{ label: 'Proteksi', href: '/projects' }
				]
			},
			{
				title: 'Sumber',
				links: [
					{ label: 'Cara kerja', href: '#cara-kerja' },
					{ label: 'Untuk siapa', href: '#untuk-siapa' },
					{ label: 'Contoh proyek', href: '/projects' }
				]
			},
			{
				title: 'Kontak',
				links: [
					{ label: 'hello@paluketok.id', href: 'mailto:hello@paluketok.id' },
					{ label: 'Dokumentasi & sumber', href: '/projects' }
				]
			}
		],
		copyright: '© 2026 PaluKetok. Rancangan statis editable.',
		legal: [
			{ label: 'Privacy', href: '#' },
			{ label: 'Terms', href: '#' }
		]
	}
};
