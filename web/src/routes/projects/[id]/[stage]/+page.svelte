<script lang="ts">
	import SummarySidebar from '#lib/components/SummarySidebar.svelte';
	import SummaryTopbar from '#lib/components/SummaryTopbar.svelte';
	import { getProject, getArtifact, listProjectArtifacts, getRun, ApiError } from '#lib/api';
	import { mdToHtml } from '#lib/markdown';
	import { stageLabel, type PlanStage } from '#lib/stages';
	import { goto } from '$app/navigation';
	import type { PageProps } from './$types';
	import type { Artifact, Project } from '#lib/api';

	// Halaman dokumen /projects/[id]/[stage].
	//
	// Sebelum ini tombol sidebar memakai anchor "#tahap-{key}" sehingga kliknya
	// cuma menggeser scroll di halaman chat: halaman dokumennya belum pernah ada.
	// Halaman ini membaca artefak dari jalur PROYEK (bukan run id yang cuma
	// tersimpan di sessionStorage), jadi tautan yang dibagikan tetap benar.
	let { params }: PageProps = $props();

	let loading = $state(true);
	let errorMsg = $state('');
	let notFound = $state(false);
	let project = $state<Project | null>(null);
	let artifacts = $state<Artifact[]>([]);
	let stages = $state<PlanStage[]>([]);
	let artifact = $state<Artifact | null>(null);
	let copied = $state(false);
	let copyErr = $state('');

	const label = $derived(
		stageLabel(
			params.stage,
			stages.find((s) => s.key === params.stage)
		)
	);
	const spec = $derived(stages.find((s) => s.key === params.stage));
	// DBML dan JSON ditampilkan apa adanya di <pre>: markdown tidak cocok untuk
	// keduanya, dan menampilkan teks mentah lebih jujur daripada merender salah.
	const format = $derived(spec?.output_format ?? 'markdown');
	const asText = $derived(format === 'dbml' || format === 'json');
	const html = $derived(!asText && artifact ? mdToHtml(artifact.content) : '');

	async function loadAll() {
		loading = true;
		errorMsg = '';
		notFound = false;
		try {
			const p = await getProject(params.id);
			project = p;
			try {
				const { artifacts: list } = await listProjectArtifacts(params.id);
				artifacts = list;
				artifact = list.find((a) => a.kind === params.stage) ?? null;
			} catch {
				artifacts = [];
				artifact = null;
			}
			// Daftar artefak per-proyek tidak mengirim `content`, jadi isi dokumen
			// diambil per artefak. Jalur ini dipakai supaya payload sidebar tetap
			// ringan.
			if (artifact && !artifact.content) {
				try {
					artifact = await getArtifact(artifact.id);
				} catch {
					/* biarkan kosong; halaman menampilkan pesan belum disusun */
				}
			}
			// Label dari skema run terbaru (Fase C). Proyek sudah mengirim
			// `latest_run`, jadi tidak perlu endpoint daftar run.
			stages = await loadStages(p.latest_run?.id);
		} catch (e) {
			if (e instanceof ApiError && e.status === 401) {
				goto('/?login=1');
				return;
			}
			if (e instanceof ApiError && e.status === 404) {
				notFound = true;
			} else {
				errorMsg = e instanceof Error ? e.message : 'Gagal memuat dokumen';
			}
		} finally {
			loading = false;
		}
	}

	/**
	 * Skema tahap milik satu run. Run lama tanpa baris plan_stages mengembalikan
	 * daftar kosong, dan halaman jatuh ke label bawaan.
	 */
	async function loadStages(runId?: string): Promise<PlanStage[]> {
		if (!runId) return [];
		try {
			const run = await getRun(runId);
			return run.stages ?? [];
		} catch {
			return [];
		}
	}

	async function copyContent() {
		if (!artifact) return;
		copyErr = '';
		try {
			await navigator.clipboard.writeText(artifact.content);
			copied = true;
			setTimeout(() => (copied = false), 1800);
		} catch {
			copyErr = 'Gagal menyalin. Pilih teksnya lalu salin manual.';
		}
	}

	$effect(() => {
		loadAll();
	});
</script>

<svelte:head>
	<title>{`${project?.name ?? 'Proyek'} · ${label}`}</title>
</svelte:head>

<div class="flex h-full w-full bg-[#F8F9FA]">
	<SummarySidebar
		projectName={project?.name ?? ''}
		total={stages.length}
		ready={artifacts.length}
		{artifacts}
		slug={project?.slug ?? params.id}
		{stages}
		activeKey={params.stage}
	/>

	<div class="flex-1 flex flex-col h-full overflow-hidden">
		<SummaryTopbar
			breadcrumb={{
				root: 'Proyek',
				project: project?.name ?? 'Proyek',
				chip: label
			}}
		/>

		<main class="flex-1 overflow-y-auto px-8 py-6" data-purpose="dokumen-scrollable">
			{#if loading}
				<div class="rounded-xl border border-blue-200 bg-blue-50 p-6 text-sm text-blue-700">
					Memuat dokumen…
				</div>
			{:else if notFound}
				<div class="rounded-xl border border-slate-200 bg-white p-6 space-y-3">
					<p class="text-sm font-semibold text-[#0F172A]">Proyek tidak ditemukan</p>
					<a class="text-sm text-[#2383e2] hover:underline" href="/projects">Kembali ke daftar proyek</a>
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
				<div class="max-w-4xl space-y-4">
					<div class="flex items-start justify-between gap-4">
						<div>
							<p class="text-[11px] font-semibold uppercase tracking-wider text-[#94A3B8]">
								{format}{spec ? ` · ${format}` : ''}
							</p>
							<h1 class="text-xl font-bold text-[#0F172A]">{label}</h1>
							{#if spec?.intent}
								<p class="mt-1 text-sm text-[#475569]">{spec.intent}</p>
							{/if}
						</div>
						<div class="flex items-center gap-3 shrink-0">
							{#if artifact}
								<span class="text-[11px] font-medium text-[#64748B]">
									v{artifact.version} · {artifact.status}
								</span>
								<button
									type="button"
									class="px-3 py-1.5 rounded-lg text-xs font-medium border border-[#E2E8F0] bg-white hover:bg-slate-50 transition-all"
									onclick={copyContent}
								>
									{copied ? 'Tersalin' : 'Salin'}
								</button>
							{/if}
							<a
								class="px-3 py-1.5 rounded-lg text-xs font-medium border border-[#E2E8F0] bg-white hover:bg-slate-50 transition-all"
								href={`/projects/${project?.slug ?? params.id}`}>Kembali</a
							>
						</div>
					</div>

					{#if copyErr}
						<p class="text-xs text-rose-600">{copyErr}</p>
					{/if}

					{#if artifact && asText}
						<pre
							class="rounded-xl border border-[#E2E8F0] bg-white p-5 text-xs text-[#0F172A] overflow-x-auto whitespace-pre-wrap leading-relaxed">{artifact.content}</pre>
					{:else if artifact}
						<div class="md rounded-xl border border-[#E2E8F0] bg-white p-6 max-w-none">
							<!-- mdToHtml membersihkan seluruh tag di luar daftar putih
							     (lib/markdown.ts), jadi hasil marked aman ditempel. -->
							{@html html}
						</div>
					{:else}
						<div class="rounded-xl border border-slate-200 bg-white p-6 space-y-2">
							<p class="text-sm text-[#475569]">Dokumen ini belum disusun.</p>
							<p class="text-xs text-[#94A3B8]">
								Jalankan generate di halaman proyek, lalu buka lagi menu ini.
							</p>
							<a
								class="inline-block text-sm font-medium text-[#2383e2] hover:underline"
								href={`/projects/${project?.slug ?? params.id}`}>Ke halaman proyek</a
							>
						</div>
					{/if}
				</div>
			{/if}
		</main>
	</div>
</div>
