/**
 * Tipe data halaman /projects/[id] (design/web/ringkasanProyek.html).
 * Struktur mengikuti blok ber-`data-purpose` di mockup.
 */

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
	/** Breadcrumb "Proyek / Nama Proyek". */
	breadcrumb: {
		root: string;
		project: string;
		chip: string;
	};
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
	/**
	 * 4 metrik ringkas. Blok ini SELALU empat kartu (layout mockup); metrik yang
	 * tidak bisa dihitung dari artefak tampil sebagai "—" dengan label kosong,
	 * bukan angka karangan.
	 */
	metrics: {
		value: string;
		label: string;
		note: string;
		icon: string;
		/** Tahap artefak yang jadi sumber angka ini (untuk membuka dokumennya). */
		kind?: string;
	}[];
	/** Dokumen terbaru per tahap; alur mockup dipertahankan, isinya nyata. */
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
			/** Artefak yang diwakili simpul ini. */
			artifactId?: string;
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
	/** Aktivitas nyata dari artefak (versi + waktu), bukan feed karangan. */
	review: {
		title: string;
		counter: string;
		feedTitle: string;
		activities: SummaryActivity[];
		feedLink: string;
	};
	footerLeft: string;
	footerRight: string;
}
