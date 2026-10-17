/**
 * Tipe data halaman /projects (design/web/dasrhboard.html).
 * Struktur mengikuti blok ber-`data-purpose` di mockup, jadi tiap bagian bisa
 * dibandingkan berdampingan dengan mockup tanpa menerjemahkan nama.
 */

/** Satu item menu sidebar. */
export type DashNavItem = {
	label: string;
	icon: string;
	href: string;
	/** true = item aktif, latar biru seperti "Daftar proyek" di mockup. */
	active?: boolean;
};

/** Kartu angka di grid metrik. */
export type DashMetric = {
	value: string;
	label: string;
	/** Baris kecil di bawah label, mis. "Data contoh". */
	note: string;
};

/** Status proyek -> warna badge. */
export type ProjectStatus = 'ready' | 'draft' | 'archived';

/** Kartu proyek pada daftar. */
export type ProjectCard = {
	/** UUID proyek dari API; kartu mock boleh kosong. */
	id?: string;
	/** Slug URL proyek (enak dibaca); UUID dipakai bila kosong. */
	slug?: string;
	name: string;
	brief: string;
	stack: string;
	status: ProjectStatus;
	/** Baris kecil di atas tombol, mis. "6 artefak tersedia · pemilik Naufal". */
	meta: string;
	/** Label tombol; kartu arsip memakai "Lihat arsip". */
	action: string;
};

/** Kartu contoh tampilan alternatif (blok StateHalamanSection). */
export type PageState = {
	/** Label kotak: EMPTY / LOADING / ERROR. */
	badge: string;
	/** Warna label: slate (empty), blue (loading), rose (error). */
	tone: 'slate' | 'blue' | 'rose';
	title: string;
	body: string;
	/** Label tombol; kartu loading memakai teks status, bukan tombol. */
	action: string;
};

/** Isi halaman /projects. */
export type DashboardContent = {
	meta: { title: string };
	breadcrumb: { root: string; current: string };
	notice: string;
	/** Label kecil di topbar, mis. "Data contoh". */
	chip: string;
	credits: string;
	topupLabel: string;
	createLabel: string;
	sidebar: {
		brand: string;
		studio: string;
		activeLabel: string;
		activeProject: string;
		/** Dipakai halaman proyek: daftar proyek untuk sidebar. slug = URL resmi. */
		projects?: { id: string; name: string; slug?: string }[];
		sectionLabel: string;
		items: DashNavItem[];
		user: { name: string; meta: string };
	};
	hero: {
		kicker: string;
		heading: string;
		body: string;
	};
	metrics: DashMetric[];
	list: {
		heading: string;
		/** Mis. "3 proyek · data contoh". */
		count: string;
		cards: ProjectCard[];
	};
	pipeline: {
		heading: string;
		body: string;
		/** Rantai tahap, mis. "prd → erd → openapi → roles → security → structure". */
		chain: string;
		action: string;
	};
	balance: {
		heading: string;
		value: string;
		meta: string;
		note: string;
		action: string;
	};
	states: {
		heading: string;
		note: string;
		items: PageState[];
		/** Keterangan di bawah kartu loading. */
		loadingNote: string;
	};
};