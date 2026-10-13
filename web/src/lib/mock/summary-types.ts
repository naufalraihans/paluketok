/**
 * Tipe data halaman /projects/[id] (design/web/ringkasanProyek.html).
 * Struktur mengikuti blok ber-`data-purpose` di mockup.
 */

export interface SummaryNavItem {
	label: string;
	icon: string;
	/** Angka kecil di kanan (mis. hitungan entitas). Tidak ada di mockup = undefined. */
	count?: string;
	active?: boolean;
}

export interface SummaryTask {
	label: string;
	tag: string;
	assignee: string;
}

export interface SummaryActivity {
	/** "09.32" atau "Kemarin". */
	time: string;
	text: string;
}

export interface SummaryContent {
	meta: {
		title: string;
		bodyClass: string;
	};
	/** Breadcrumb "Proyek / Ruang Ramu [MVP · v0.8]". */
	breadcrumb: {
		root: string;
		project: string;
		chip: string;
	};
	savedAt: string;
	collaborators: { initials: string; class: string }[];
	shareLabel: string;
	exportLabel: string;
	pageHeader: {
		eyebrow: string;
		title: string;
		subtitle: string;
	};
	/** Kartu metadata proyek + progress kesiapan. */
	overview: {
		monogram: string;
		name: string;
		status: string;
		description: string;
		facts: string[];
		readiness: {
			label: string;
			percent: number;
			note: string;
		};
	};
	/** 4 metrik ringkas. */
	metrics: {
		value: string;
		label: string;
		note: string;
		icon: string;
	}[];
	/** Alur keterkaitan 4 tahap: PRD -> Halaman -> Entitas -> Proteksi. */
	thread: {
		title: string;
		link: string;
		subtitle: string;
		nodes: {
			tag: string;
			title: string;
			note: string;
			icon: string;
			mono?: boolean;
		}[];
		connected: string;
		pending: string;
	};
	/** Proteksi; tiap item ada ikon status: check (hijau) / dot (merah putus-putus). */
	security: {
		title: string;
		body: string;
		items: { label: string; status: string; ok: boolean }[];
		link: string;
	};
	/** Tugas sebelum review + catatan aktivitas. */
	review: {
		title: string;
		counter: string;
		tasks: SummaryTask[];
		feedTitle: string;
		activities: SummaryActivity[];
		feedLink: string;
	};
	footerLeft: string;
	footerRight: string;
}
