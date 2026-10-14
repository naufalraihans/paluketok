<script lang="ts">
	import SummarySidebar from '#lib/components/SummarySidebar.svelte';
	import SummaryTopbar from '#lib/components/SummaryTopbar.svelte';
	import SummaryHeader from '#lib/components/SummaryHeader.svelte';
	import SummaryMetrics from '#lib/components/SummaryMetrics.svelte';
	import SummaryThread from '#lib/components/SummaryThread.svelte';
	import SummaryBottom from '#lib/components/SummaryBottom.svelte';
	import { summary } from '#lib/mock/summary';
	import { getProject, getRun, listRunArtifacts, createRun, regenerateArtifact, ApiError } from '#lib/api';
	import { auth } from '#lib/auth.svelte';
	import { goto } from '$app/navigation';
	import type { PageProps } from './$types';
	import type { SummaryContent } from '#lib/mock/summary-types';

	// Fase 10 langkah 36: data live /v1/projects/{id} + run + artifacts.
	// 6 tile tahap (PRD -> structure) digambar dari artifacts; kerangka halaman
	// tetap mock = mockup beku.
	let { params }: PageProps = $props();

	let loading = $state(true);
	let errorMsg = $state('');
	let name = $state('');
	let brief = $state('');
	let status = $state('');
	let createdAt = $state('');
	let runId = $state('');
	let runStatus = $state('');
	let progress = $state(0);
	let total = $state(6);
	let currentStage = $state('');
	let tiles = $state<{ label: string; ok: boolean; note: string; kind: string; artifactId?: string }[]>([]);
	let busy = $state(false);
	let regenBusy = $state('');
	let regenMsg = $state('');
	let pollTimer: ReturnType<typeof setTimeout> | null = null;

	const STAGES = ['prd', 'erd', 'openapi', 'roles', 'security', 'structure'] as const;
	const STAGE_LABEL: Record<string, string> = {
		prd: 'PRD',
		erd: 'ERD',
		openapi: 'OpenAPI',
		roles: 'Role access',
		security: 'Proteksi',
		structure: 'Struktur'
	};

	async function loadAll() {
		loading = true;
		errorMsg = '';
		try {
			const p = await getProject(params.id);
			name = p.name;
			brief = p.brief;
			status = p.status;
			createdAt = new Date(p.created_at).toLocaleDateString('id-ID');
			tiles = STAGES.map((s) => ({ label: STAGE_LABEL[s], ok: false, note: 'Belum ada artefak', kind: s }));
			// Ringkasan run terakhir: ambil via run terakhir yang tercatat di detail
			// (API tak sediakan list-run; pakai cache sessionStorage singkat).
			const lastRunId = sessionStorage.getItem(`pk-run-${params.id}`);
			if (lastRunId) {
				runId = lastRunId;
				await loadRun(lastRunId);
			}
		} catch (e) {
			if (e instanceof ApiError && e.status === 401) {
				goto('/?login=1');
				return;
			}
			errorMsg = e instanceof Error ? e.message : 'Gagal memuat proyek';
		} finally {
			loading = false;
		}
	}

	async function loadRun(rid: string) {
		const run = await getRun(rid);
		runStatus = run.status;
		progress = run.progress;
		total = run.total;
		currentStage = run.current_stage ?? '';
		if (run.status === 'done' || run.status === 'failed') {
			const { artifacts } = await listRunArtifacts(rid);
			tiles = STAGES.map((s) => {
				const a = artifacts.find((x) => x.kind === s);
				return {
					label: STAGE_LABEL[s],
					ok: Boolean(a),
					note: a ? `v${a.version} · siap` : 'Belum ada artefak',
					kind: s,
					artifactId: a?.id
				};
			});
		} else {
			tiles = STAGES.map((s, i) => ({
				label: STAGE_LABEL[s],
				ok: i < progress,
				note: run.current_stage === s ? 'Sedang berjalan…' : i < progress ? 'Selesai' : 'Menunggu',
				kind: s
			}));
			schedulePoll(rid);
		}
	}

	function schedulePoll(rid: string) {
		if (pollTimer) clearTimeout(pollTimer);
		pollTimer = setTimeout(async () => {
			try {
				await loadRun(rid);
			} catch {
				schedulePoll(rid);
			}
		}, 3000);
	}

	async function startRun() {
		if (busy) return;
		busy = true;
		errorMsg = '';
		try {
			const run = await createRun(params.id);
			runId = run.id;
			sessionStorage.setItem(`pk-run-${params.id}`, run.id);
			runStatus = run.status;
			await loadRun(run.id);
		} catch (e) {
			if (e instanceof ApiError && e.status === 401) {
				goto('/?login=1');
				return;
			}
			errorMsg = e instanceof Error ? e.message : 'Gagal memulai generate';
		} finally {
			busy = false;
		}
	}

	async function regenerate(kind: string, artifactId?: string) {
		if (!artifactId || regenBusy) return;
		regenBusy = kind;
		regenMsg = '';
		try {
			await regenerateArtifact(artifactId);
			if (runId) await loadRun(runId);
			regenMsg = `${STAGE_LABEL[kind]} diperbarui.`;
		} catch (e) {
			if (e instanceof ApiError && e.status === 401) {
				goto('/?login=1');
				return;
			}
			regenMsg = e instanceof Error ? e.message : 'Gagal regenerate';
		} finally {
			regenBusy = '';
		}
	}

	$effect(() => {
		loadAll();
		return () => {
			if (pollTimer) clearTimeout(pollTimer);
		};
	});

	const content = $derived.by<SummaryContent>(() => ({
		...summary,
		breadcrumb: { ...summary.breadcrumb, project: name || summary.breadcrumb.project },
		pageHeader: {
			...summary.pageHeader,
			eyebrow: `${name || 'Proyek'} / Ringkasan`
		},
		overview: {
			...summary.overview,
			monogram: (name || 'P.').slice(0, 2) + '.',
			name: name || '…',
			status:
				runStatus === 'running'
					? 'Generate berjalan'
					: runStatus === 'done'
						? 'Siap · ready'
						: status === 'draft'
							? 'Draf'
							: status || '…',
			description: brief || summary.overview.description,
			facts: [`Dibuat ${createdAt}`, `Pemilik: ${auth.me?.username ?? 'saya'}`],
			readiness: {
				...summary.overview.readiness,
				percent: total > 0 ? Math.round((progress / total) * 100) : 0,
				note: runStatus ? `${progress}/${total} tahap selesai` : summary.overview.readiness.note
			}
		},
		metrics: summary.metrics
	}));
</script>

<svelte:head>
	<title>{content.meta.title}</title>
</svelte:head>
<div class={content.meta.bodyClass}>
	<SummarySidebar />

	<div class="flex-1 flex flex-col h-full overflow-hidden bg-[#F8F9FA]" data-purpose="main-layout">
		<SummaryTopbar
			breadcrumb={content.breadcrumb}
			savedAt={content.savedAt}
			collaborators={content.collaborators}
			shareLabel={content.shareLabel}
		/>

		<main class="flex-1 overflow-y-auto px-8 py-6 space-y-6" data-purpose="dashboard-scrollable-content">
			{#if loading}
				<div class="rounded-xl border border-blue-200 bg-blue-50 p-6 text-sm text-blue-700">
					Memuat proyek…
				</div>
			{:else if errorMsg}
				<div class="rounded-xl border border-rose-200 bg-rose-50 p-6 space-y-2">
					<p class="text-sm font-semibold text-rose-700">Permintaan tidak berhasil</p>
					<p class="text-sm text-rose-600">{errorMsg}</p>
					<button
						type="button"
						class="text-sm font-medium text-rose-700 underline underline-offset-2 hover:text-rose-900 transition-colors"
						onclick={loadAll}>Coba lagi</button
					>
				</div>
			{:else}
				<SummaryHeader
					overview={content.overview}
					pageHeader={content.pageHeader}
					exportLabel={content.exportLabel}
				/>
				{#if !runId || runStatus === 'failed'}
					<div class="rounded-xl border border-slate-200 bg-white p-6 flex items-center justify-between">
						<p class="text-sm text-[#475569]">
							{runStatus === 'failed'
								? 'Generate terakhir gagal. Kredit otomatis dikembalikan; coba lagi.'
								: 'Belum ada generate. Mulai untuk menyusun enam dokumen planning.'}
						</p>
						<button
							type="button"
							disabled={busy}
							class="inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm disabled:opacity-50"
							onclick={startRun}
							>{busy ? 'Memulai…' : 'Mulai generate (1 kredit)'}</button
						>
					</div>
				{/if}
				{#if runId}
					<div class="rounded-xl border border-slate-200 bg-white p-6">
						<p class="text-sm font-semibold mb-3">
							Enam tahap planning
							<span class="ml-2 text-xs font-normal text-slate-500">
								{#if runStatus === 'running' || runStatus === 'queued'}
									{currentStage ? `sedang: ${currentStage}` : 'antre…'} · {progress}/{total}
								{:else if runStatus === 'done'}
									selesai
								{/if}
							</span>
						</p>
						<div class="grid grid-cols-2 md:grid-cols-6 gap-3">
							{#each tiles as t (t.label)}
								<div
									class="rounded-lg border p-3 transition-all {t.ok
										? 'border-emerald-200 bg-emerald-50'
										: 'border-slate-200 bg-slate-50'}"
								>
									<p class="text-xs font-semibold {t.ok ? 'text-emerald-700' : 'text-slate-500'}">
										{t.label}
									</p>
									<p class="text-xs mt-1 {t.ok ? 'text-emerald-700' : 'text-slate-500'}">
									{t.note}
								</p>
								{#if t.ok && t.artifactId && runStatus === 'done'}
									<button
										type="button"
										class="mt-2 text-[11px] font-medium text-[#2383e2] hover:text-[#1a6ec0] transition-colors disabled:opacity-50"
										disabled={regenBusy === t.kind}
										onclick={() => regenerate(t.kind, t.artifactId)}
									>
										{regenBusy === t.kind ? 'Memproses…' : 'Regenerate'}
									</button>
								{/if}
								</div>
							{/each}
						</div>
					</div>
				{/if}
				<SummaryMetrics metrics={content.metrics} />
				<SummaryThread thread={content.thread} security={content.security} />
				<SummaryBottom
					review={content.review}
					footerLeft={content.footerLeft}
					footerRight={content.footerRight}
				/>
			{/if}
		</main>
	</div>
</div>
