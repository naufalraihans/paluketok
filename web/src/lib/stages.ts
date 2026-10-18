/**
 * Satu sumber label tahap planning untuk seluruh UI.
 *
 * Dipakai sebagai nilai AWAL saja: begitu sebuah run punya skema tahap sendiri
 * (tabel `plan_stages`, disusun AI judge), label dari skema itulah yang dipakai.
 * Peta di bawah ini yang dipakai selama belum ada run, dan jadi pratinjau
 * susunan awal enam tahap.
 */

/** Urutan awal enam tahap, sama dengan entity.Stages di server. */
export const STAGES = ['prd', 'erd', 'openapi', 'roles', 'security', 'structure'] as const;

/**
 * Alias bernama jelas untuk STAGES: dipakai saat sebuah komponen juga mengimpor
 * skema tahap dari run, supaya "tahap mana" tidak tertukar dengan "susunan awal".
 */
export const DEFAULT_STAGES = STAGES;

/** Label ringkas per tahap (dipakai tile, benang merah, dan judul halaman). */
export const STAGE_LABEL: Record<string, string> = {
	prd: 'PRD',
	erd: 'ERD',
	openapi: 'OpenAPI',
	roles: 'Role access',
	security: 'Proteksi',
	structure: 'Struktur'
};

/** Label panjang untuk menu sidebar; jatuh ke STAGE_LABEL bila tidak ada. */
export const STAGE_MENU_LABEL: Record<string, string> = {
	prd: 'Dokumen · PRD',
	erd: 'Model data · ERD',
	openapi: 'Kontrak OpenAPI',
	roles: 'Role access',
	security: 'Proteksi',
	structure: 'Struktur & prototipe'
};

/** Ikon menu sidebar per tahap. */
export const STAGE_ICON: Record<string, string> = {
	prd: 'document',
	erd: 'db-simple',
	openapi: 'code',
	roles: 'users',
	security: 'shield',
	structure: 'layers-row'
};

/**
 * Satu baris skema tahap. Tipenya berasal dari `api.ts` supaya hanya ada satu
 * definisi untuk `run.stages`; diimpor dan diekspor ulang di sini karena seluruh
 * UI mengambilnya bersama label dan ikon.
 */
import type { PlanStage } from '#lib/api';
export type { PlanStage };

/**
 * Label tampilan untuk satu tahap. Skema run menang; peta bawaan jadi cadangan
 * supaya tahap karangan tetap punya label yang enak dibaca.
 */
export function stageLabel(key: string, spec?: PlanStage): string {
	if (spec?.label) return spec.label;
	return STAGE_LABEL[key] ?? key;
}

/** Susunan awal sebagai pratinjau skema saat run belum ada. */
export function defaultPlanStages(): PlanStage[] {
	return STAGES.map((key, i) => ({
		key,
		label: STAGE_LABEL[key],
		position: i,
		output_format: key === 'erd' ? 'dbml' : key === 'openapi' ? 'json' : 'markdown'
	}));
}
