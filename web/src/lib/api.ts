import { supabase } from '#lib/supabase';
import { goto } from '$app/navigation';

/**
 * Klien API /v1 (Fase 10). Fetch + Bearer token Supabase.
 * Base URL: VITE_API_BASE (default http://localhost:8080).
 * Map error (openapi.md bagian Error): 401 -> /login, 402 -> /billing,
 * 409/422/404/503 -> ApiError untuk pesan inline di halaman.
 */

export const API_BASE: string = import.meta.env.VITE_API_BASE ?? 'http://localhost:8080';

/** Error API dengan kode sesuai docs/plan/openapi.md bagian Error. */
export class ApiError extends Error {
	code: string;
	status: number;
	constructor(status: number, code: string, message: string) {
		super(message);
		this.status = status;
		this.code = code;
	}
}

async function token(): Promise<string> {
	const { data } = await supabase.auth.getSession();
	const t = data.session?.access_token;
	if (!t) {
		await goto('/login', { replaceState: true });
		throw new ApiError(401, 'unauthorized', 'Belum masuk');
	}
	return t;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
	for (let attempt = 0; attempt < 2; attempt++) {
		const bearer = await token();
		let res: Response;
		try {
			res = await fetch(`${API_BASE}${path}`, {
				method,
				headers: {
					Authorization: `Bearer ${bearer}`,
					...(body !== undefined ? { 'Content-Type': 'application/json' } : {})
				},
				...(body !== undefined ? { body: JSON.stringify(body) } : {})
			});
		} catch {
			throw new ApiError(0, 'network', 'Server tidak terjangkau. Coba lagi sebentar.');
		}
		if (res.status === 401 && attempt === 0) {
			// Access token mungkin kedaluwarsa: paksa refresh sekali, lalu ulang.
			const { data } = await supabase.auth.refreshSession();
			if (data.session) continue;
			await goto('/login', { replaceState: true });
			throw new ApiError(401, 'unauthorized', 'Sesi berakhir, masuk kembali');
		}
		if (!res.ok) {
			let code = 'internal';
			let message = `HTTP ${res.status}`;
			try {
				const j = (await res.json()) as { error?: { code?: string; message?: string } };
				code = j.error?.code ?? code;
				message = j.error?.message ?? message;
			} catch {
				/* body bukan JSON: pakai default */
			}
			if (res.status === 401) {
				await goto('/login', { replaceState: true });
			} else if (res.status === 402) {
				await goto('/billing');
			}
			throw new ApiError(res.status, code, message);
		}
		return (await res.json()) as T;
	}
	throw new ApiError(401, 'unauthorized', 'Sesi berakhir, masuk kembali'); // tak terjangkau
}

function get<T>(path: string): Promise<T> {
	return request<T>('GET', path);
}
function post<T>(path: string, body?: unknown): Promise<T> {
	return request<T>('POST', path, body ?? {});
}
function patch<T>(path: string, body?: unknown): Promise<T> {
	return request<T>('PATCH', path, body ?? {});
}
function put<T>(path: string, body?: unknown): Promise<T> {
	return request<T>('PUT', path, body ?? {});
}
function del<T>(path: string): Promise<T> {
	return request<T>('DELETE', path);
}

// --- /v1/me ---
export type Me = {
	id: string;
	username?: string;
	full_name?: string;
	avatar_url?: string;
	plan_tier: string;
	is_admin: boolean;
	credits: number;
	created_at: string;
};
export const getMe = () => get<Me>('/v1/me');

// --- /v1/projects ---
export type Stack = { fe?: string; be?: string; db?: string; payment?: string; mobile?: string };
export type Project = {
	id: string;
	owner_id: string;
	name: string;
	brief: string;
	stack: Stack;
	status: 'draft' | 'ready' | 'archived';
	/** Slug URL enak dibaca (/projects/testing-jembut); kosong untuk proyek lama. */
	slug?: string;
	/**
	 * Ringkasan run TERBARU. Ada bila proyek sudah pernah di-generate.
	 * Dikirim server supaya status generate tidak bergantung pada run id di
	 * sessionStorage (proyek dibuka di perangkat lain tetap benar).
	 */
	latest_run?: Run;
	created_at: string;
	updated_at: string;
};
export const listProjects = () => get<{ projects: Project[] }>('/v1/projects');
/**
 * ref boleh UUID atau slug. Halaman memakai slug; UUID tetap diterima supaya
 * tautan lama yang sudah dibagikan tidak mati.
 */
export const getProject = (ref: string) => get<Project>(`/v1/projects/${ref}`);
export const createProject = (input: { name: string; brief: string; stack?: Stack }) =>
	post<Project>('/v1/projects', { stack: {}, ...input });
/**
 * Arsipkan proyek. Server TIDAK menghapus barisnya: `status` jadi `archived`
 * (docs/plan/erd.md "Aturan status"), jadi artefak dan riwayatnya tetap ada.
 */
export const archiveProject = (ref: string) =>
	del<{ status: string }>(`/v1/projects/${ref}`);

// --- runs & artifacts ---
/**
 * Skema satu tahap planning (tabel `plan_stages`). Dikirim server di `run.stages`
 * supaya UI tidak menebak jumlah tahap dari daftar tetap seperti dulu.
 */
export type PlanStage = {
	key: string;
	label: string;
	position: number;
	intent?: string;
	output_format?: string;
	depends_on?: string[];
};

export type Run = {
	id: string;
	project_id: string;
	user_id?: string;
	status: 'queued' | 'running' | 'done' | 'failed';
	progress: number;
	total: number;
	current_stage?: string;
	escalations: number;
	/** Skema tahap run ini. Kosong untuk run yang dibuat sebelum fitur ini. */
	stages?: PlanStage[];
	credits_charged?: number;
};
export const createRun = (projectId: string, stages?: PlanStage[]) =>
	// Skema yang sudah dikonfirmasi ikut dikirim supaya server tidak menyusunnya
	// dua kali (sekali saat pratinjau, sekali saat mulai).
	post<Run>(`/v1/projects/${projectId}/runs`, stages && stages.length ? { stages } : undefined);
export const getRun = (runId: string) => get<Run>(`/v1/runs/${runId}`);
/**
 * Usulan skema tahap untuk layar konfirmasi. Tidak membuat run dan tidak
 * memotong kredit.
 */
export const getProjectStages = (projectId: string) =>
	get<{ stages: PlanStage[]; note?: string }>(`/v1/projects/${projectId}/stages`);
/** Skema tahap sebuah run. Run lama tanpa skema dibalas susunan awal server. */
export const getRunStages = (runId: string) =>
	get<{ stages: PlanStage[] }>(`/v1/runs/${runId}/stages`);

export type Artifact = {
	id: string;
	run_id: string;
	kind: 'prd' | 'erd' | 'openapi' | 'roles' | 'security' | 'structure';
	version: number;
	status: string;
	content: string;
	created_at: string;
};
export const listRunArtifacts = (runId: string) =>
	get<{ artifacts: Artifact[] }>(`/v1/runs/${runId}/artifacts`);
/**
 * Artefak terbaru per kind untuk satu PROYEK (bukan per run).
 * Dipakai sidebar progresif: sidebar tahu menu tahap mana yang harus muncul
 * tanpa bergantung pada run id di sessionStorage, sehingga proyek lama tetap
 * menampilkan menunya di browser/perangkat lain.
 */
export const listProjectArtifacts = (projectId: string) =>
	get<{ artifacts: Artifact[] }>(`/v1/projects/${projectId}/artifacts`);

// --- chat konteks proyek (perancangan saja, gratis) ---

/** Satu pesan percakapan. `suggested_stage` hanya ada di balasan model. */
export type ProjectMessage = {
	id: string;
	project_id: string;
	role: 'user' | 'assistant';
	content: string;
	suggested_stage?: string;
	model?: string;
	usage?: { input_tokens: number; output_tokens: number };
	created_at: string;
};

export const sendProjectMessage = (projectId: string, content: string) =>
	post<ProjectMessage>(`/v1/projects/${projectId}/messages`, { content });

export const listProjectMessages = (projectId: string) =>
	get<{ messages: ProjectMessage[] }>(`/v1/projects/${projectId}/messages`);
export const getArtifact = (id: string) => get<Artifact>(`/v1/artifacts/${id}`);
export const regenerateArtifact = (id: string) => post<Artifact>(`/v1/artifacts/${id}/regenerate`);

// --- decisions ---
export type Decision = {
	id: string;
	run_id: string;
	stage: string;
	question: string;
	gate: string;
	round: number;
	choice: string;
	confidence: number;
	jev_decision_id?: string;
	created_at: string;
};
export const listDecisions = (runId: string) =>
	get<{ decisions: Decision[] }>(`/v1/decisions?run_id=${encodeURIComponent(runId)}`);

// --- orders (top-up) ---
// Status di BE: pending | settlement | expire (0001_init.sql bagian orders).
export type Order = {
	order_id: string;
	invoice_number: string;
	package_id: string;
	amount_idr: number;
	credits_granted: number;
	status: 'pending' | 'settlement' | 'expire';
	payment_url?: string;
	expired_date?: string;
	created_at: string;
};
export const createOrder = (packageId: string) => post<Order>('/v1/orders', { package_id: packageId });
export const getOrder = (id: string) => get<Order>(`/v1/orders/${id}`);

// --- admin (grup /v1/admin, guard is_admin di BE 403) ---
export const adminListProjects = (limit = 50, offset = 0) =>
	get<{ projects: Project[]; limit: number; offset: number }>(
		`/v1/admin/projects?limit=${limit}&offset=${offset}`
	);
export const adminListRuns = (filters: { status?: string; user_id?: string } = {}) => {
	const q = new URLSearchParams({ limit: '50', offset: '0' });
	if (filters.status) q.set('status', filters.status);
	if (filters.user_id) q.set('user_id', filters.user_id);
	return get<{ runs: Run[] }>(`/v1/admin/runs?${q.toString()}`);
};
export const adminPatchUser = (id: string, planTier: string) =>
	patch<{ id: string; plan_tier: string }>(`/v1/admin/users/${id}`, { plan_tier: planTier });
export const adminGrantCredits = (id: string, credits: number, reason: string) =>
	// BE tak punya kolom "alasan" terpisah; alasan dibawa di idempotency_key
	// (tercatat sebagai sumber di credits_ledger.idempotency_key).
	post<{ id: string; credits_granted: number }>(`/v1/admin/users/${id}/credits`, {
		credits,
		idempotency_key: `admin-${reason.toLowerCase().replace(/[^a-z0-9]+/g, '-').slice(0, 24)}-${Date.now()}`
	});
export const adminListDecisions = (limit = 50, offset = 0) =>
	get<{ decisions: Decision[] }>(`/v1/admin/decisions?limit=${limit}&offset=${offset}`);

export type AISetting = {
	slot: string;
	provider: string;
	base_url: string;
	model: string;
	api_key: string;
	max_input_tokens: number;
	max_output_tokens: number;
	is_active: boolean;
};
// PUT body: api_key HANYA disertakan bila user mengetik nilai baru (masked
// dari GET tak boleh dikirim balik, akan tersimpan sebagai "abcd...wxyz").
// provider, model, dan batas token juga tidak dikirim: server mengisinya dari
// nilai lama atau nilai awal per slot, jadi admin cukup mengisi base_url.
export type AISettingInput = {
	slot: string;
	base_url: string;
	model: string;
	is_active: boolean;
	api_key?: string;
};
/** Uji koneksi ke penyedia tanpa menyimpan apa pun. api_key kosong = pakai tersimpan. */
export type AISettingTestInput = {
	slot: string;
	base_url: string;
	model: string;
	api_key?: string;
};
export const adminListAISettings = () => get<{ slots: AISetting[] }>('/v1/admin/settings/ai');
export const adminSetAISetting = (slot: string, body: AISettingInput) =>
	put<{ slot: string; is_active: boolean }>(`/v1/admin/settings/ai/${encodeURIComponent(slot)}`, body);
export const adminTestAISetting = (body: AISettingTestInput) =>
	post<{ ok: boolean; message: string; latency_ms: number }>('/v1/admin/settings/ai/test', body);
