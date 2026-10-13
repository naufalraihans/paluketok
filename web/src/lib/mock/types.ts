/**
 * Tipe data landing page PaluKetok.
 * Struktur mengikuti blok di design/web/landingPage.html supaya komponen bisa
 * dirender apa adanya tanpa mengubah susunan, ukuran, atau kelas utility mockup.
 */

/** Navigasi header: tautan tengah dan aksi kanan. */
export type NavLink = {
	label: string;
	href: string;
};

/** Baris sidebar "Ruang Ramu" di jendela aplikasi (blok HeroSection). */
export type SidebarItem = {
	label: string;
	/** true = item aktif (latar biru, tebal) seperti "Sitemap" di mockup. */
	active: boolean;
	/** Nama ikon garis 24x24; nilainya kunci di lib/components/Icon.svelte. */
	icon: string;
};

/** Node hierarki akses: satu kolom di peta halaman. */
export type AccessGroup = {
	title: string;
	/** Sub-label monospace, mis. "Role: participant" atau "Tanpa autentikasi". */
	meta: string;
	/** Satu node disorot; indeks mengacu ke daftar nodes. */
	highlightIndex?: number;
	nodes: string[];
};

/** Kartu pada bagian telusur benang merah (blok DarkTraceabilitySection). */
export type TraceNode = {
	/** Label atas, monospace, mis. "PRD / F-04". */
	ref: string;
	/** Pertanyaan pemandu, mis. "Apa yang perlu dibangun?". */
	question: string;
	/** Jawaban tebal. */
	answer: string;
	/** Keterangan; dirender monospace bila mono true. */
	detail: string;
	mono?: boolean;
	icon: string;
};

/** Langkah proses (blok ProcessStepsSection). */
export type ProcessStep = {
	/** Nomor dua digit: "01".."04". */
	no: string;
	title: string;
	body: string;
	/** Teks setelah "Hasil /". */
	outcome: string;
};

/** Kartu persona (blok PersonaRolesSection). */
export type Persona = {
	/** Label kategori monospace huruf kapital. */
	kicker: string;
	title: string;
	body: string;
	/** Judul skenario contoh. */
	scenarioTitle: string;
	scenarioBody: string;
	icon: string;
};

/** Kolom tabel mini di diagram ERD (blok SecurityAndDataSection). */
export type ErdColumn = {
	name: string;
	/** Kunci: "PK id" / "FK user_id" dirender biru tebal. */
	key?: boolean;
	/** Tipe kolom, mis. "uuid". */
	type: string;
	/** true = diberi penanda "sensitif". */
	sensitive?: boolean;
};

/** Tabel pada diagram mini ERD. */
export type ErdTable = {
	name: string;
	ref: string;
	columns: ErdColumn[];
};

/** Poin checklist di bagian keamanan. */
export type ChecklistPoint = {
	title: string;
	body: string;
};

/** Kolom tautan di footer. */
export type FooterColumn = {
	title: string;
	links: NavLink[];
};

/** Satu "garis" benang merah di kartu telusur (dot + teks). */
export type TraceLine = {
	/** Kunci gaya dot: brand, emerald, amber. */
	tone: 'brand' | 'emerald' | 'amber';
	/** Label tebal sebelum titik dua, mis. "F-04:". */
	label: string;
	text: string;
};

/** Satu langkah pada strip alur prototipe. */
export type FlowStep = {
	title: string;
	state: string;
	/** Kode lintas dokumen, mis. "P-03 → F-04". */
	code: string;
	/** true = kartu disorot biru (langkah aktif di mockup). */
	active: boolean;
};

/** Isi lengkap landing page. */
export type LandingContent = {
	meta: {
		title: string;
		description: string;
	};
	nav: {
		links: NavLink[];
		signIn: NavLink;
		primaryCta: NavLink;
	};
	hero: {
		titleLead: string;
		titleAccent: string;
		subtitle: string;
		ctaPrimary: NavLink;
		ctaSecondary: NavLink;
		window: {
			url: string;
			status: string;
			sidebarTitle: string;
			sidebar: SidebarItem[];
			canvasTitle: string;
			canvasSubtitle: string;
			canvasBadge: string;
			rootName: string;
			rootMeta: string;
			groups: AccessGroup[];
			footerStrip: string;
			tabs: { label: string; active: boolean }[];
		};
	};
	problem: {
		headingLine1: string;
		headingLine2: string;
		body: string;
		quote1: string;
		quote2: string;
	};
	feature: {
		heading: string;
		body: string;
		disclaimer: string;
		toolbar: {
			mark: string;
			name: string;
			section: string;
			status: string;
		};
		tabs: { label: string; active: boolean }[];
		meta: string;
		rootName: string;
		groups: AccessGroup[];
		flowTitle: string;
		flowNote: string;
		flowSteps: FlowStep[];
		detail: {
			ref: string;
			code: string;
			title: string;
			badge: string;
			body: string;
			traceLabel: string;
			trace: TraceLine[];
			footerNote: string;
		};
		footerStrip: string;
		footerCta: string;
	};
	traceability: {
		headingLine1: string;
		headingLine2: string;
		body: string;
		nodes: TraceNode[];
		callout: {
			label: string;
			quote: string;
			body: string;
			badge: string;
		};
	};
	process: {
		heading: string;
		body: string;
		steps: ProcessStep[];
		footnote: string;
	};
	personas: {
		heading: string;
		body: string;
		items: Persona[];
	};
	security: {
		heading: string;
		body: string;
		points: ChecklistPoint[];
		disclaimer: string;
		panel: {
			title: string;
			ref: string;
			tables: ErdTable[];
			policy: {
				code: string;
				lines: { actor: string; text: string }[];
			};
			pending: {
				code: string;
				title: string;
				badge: string;
			};
		};
	};
	finalCta: {
		heading: string;
		body: string;
		primary: NavLink;
		secondaryLabel: string;
	};
	footer: {
		tagline: string;
		columns: FooterColumn[];
		copyright: string;
		legal: NavLink[];
	};
};
