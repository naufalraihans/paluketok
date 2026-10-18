<script lang="ts">
	import SummarySidebar from '#lib/components/SummarySidebar.svelte';
	import SummaryTopbar from '#lib/components/SummaryTopbar.svelte';
	import SummaryHeader from '#lib/components/SummaryHeader.svelte';
	import SummaryMetrics from '#lib/components/SummaryMetrics.svelte';
	import SummaryThread from '#lib/components/SummaryThread.svelte';
	import SummaryBottom from '#lib/components/SummaryBottom.svelte';
	import ProjectChat from '#lib/components/ProjectChat.svelte';
	import { summary } from '#lib/mock/summary';
	import { getProject, getRun, getProjectStages, listRunArtifacts, listProjectArtifacts, createRun, regenerateArtifact, ApiError } from '#lib/api';
	import { STAGES, DEFAULT_STAGES, STAGE_LABEL, stageLabel, defaultPlanStages, type PlanStage } from '#lib/stages';
	import { auth } from '#lib/auth.svelte';
	import { goto, replaceState } from '$app/navigation';
	import type { PageProps } from './$types';
	import type { SummaryContent } from '#lib/mock/summary-types';
	import type { Artifact } from '#lib/api';

	// Fase 10 langkah 36: data live /v1/projects/{id} + run + artifacts.
	// 6 tile tahap (PRD -> structure) digambar dari artifacts; kerangka halaman
	// tetap mock = mockup beku.
	let { params }: PageProps = $props();

	let loading = $state(true);
	let errorMsg = $state('');
	let name = $state('');
	// UUID proyek (bukan bagian URL) — kunci stabil untuk cache run.
	let projectId = $state('');
	let brief = $state('');
	let status = $state('');
	let createdAt = $state('');
	let runId = $state('');
	let runStatus = $state('');
	// Tahap yang diusulkan chat; hanya untuk sorotan, tidak mengubah alur run.
	let usulTahap = $state('');
	let progress = $state(0);
	let total = $state(6);
	let currentStage = $state('');
	let tiles = $state<{ label: string; ok: boolean; note: string; kind: string; artifactId?: string }[]>([]);
	// Artefak terbaru per kind untuk proyek ini: sumber sidebar progresif.
	let projectArtifacts = $state<Artifact[]>([]);
	// Artefak run terakhir, LENGKAP dengan `content`. Daftar per-proyek sengaja
	// tidak mengirim isi dokumen (payload sidebar), sedangkan angka metrik dan
	// seksi proteksi harus dihitung dari isinya — jadi diambil dari jalur run
	// yang memang sudah dipanggil saat run selesai.
	let runArtifacts = $state<Artifact[]>([]);
	// Skema tahap milik run. Kosong selama belum ada run: saat itu dipakai
	// susunan awal enam tahap sebagai pratinjau (lib/stages.ts).
	let stages = $state<PlanStage[]>([]);
	// Slug URL: menu sidebar menunjuk /projects/{slug}/{key}.
	let slug = $state('');
	let busy = $state(false);
	let regenBusy = $state('');
	let regenMsg = $state('');
	let pollTimer: ReturnType<typeof setTimeout> | null = null;

	/**
	 * Urutan tahap yang dipakai halaman ini: skema run bila ada, kalau tidak
	 * susunan awal. Semua daftar (tile, benang merah, aktivitas) mengikuti ini,
	 * jadi proyek 4 tahap menampilkan 4 kartu tanpa perubahan lain.
	 */
	const stagesEff = $derived(stages.length > 0 ? stages : DEFAULT_STAGES.map((key, i) => ({ key, label: STAGE_LABEL[key], position: i })));
	const stageKeys = $derived(stagesEff.map((s) => s.key));
	/** Label tampilan satu tahap; skema run menang atas peta bawaan. */
	const lbl = (key: string) => stageLabel(key, stagesEff.find((s) => s.key === key));

	async function loadAll() {
		loading = true;
		errorMsg = '';
		try {
			const p = await getProject(params.id);
			// URL ramah baca: bila dibuka lewat UUID (tautan lama) dan proyeknya
			// sudah punya slug, ganti isi address bar ke slug. replaceState
			// dipakai (bukan goto) supaya tidak memicu muat ulang data.
			if (p.slug && p.slug !== params.id) {
				replaceState(`/projects/${p.slug}`, {});
			}
			name = p.name;
			// UUID proyek dipakai sebagai kunci cache run: URL boleh berubah
			// dari UUID ke slug tanpa kehilangan riwayat run di sesi ini.
			projectId = p.id;
			slug = p.slug ?? '';
			brief = p.brief;
			status = p.status;
			createdAt = new Date(p.created_at).toLocaleDateString('id-ID');
			tiles = stageKeys.map((s) => ({ label: lbl(s), ok: false, note: 'Belum ada artefak', kind: s }));
			// Artefak per PROYEK (bukan per run): sidebar progresif dan tile tahap
			// tetap benar walau dibuka di browser/perangkat lain, karena tidak
			// bergantung pada run id yang cuma tersimpan di sessionStorage.
			try {
				const { artifacts } = await listProjectArtifacts(params.id);
				projectArtifacts = artifacts;
				// Tile ikut artefak per proyek, jadi tetap benar di perangkat
				// baru (sessionStorage kosong) selama run-nya sudah selesai.
				if (artifacts.length > 0) {
					tiles = stageKeys.map((s) => {
						const a = artifacts.find((x) => x.kind === s);
						return {
							label: lbl(s),
							ok: Boolean(a),
							note: a ? `v${a.version} · siap` : 'Belum ada artefak',
							kind: s,
							artifactId: a?.id
						};
					});
				}
			} catch {
				projectArtifacts = []; // sidebar tetap tampil, hanya tanpa menu tahap
			}
			// Ringkasan run terbaru ikut di respons proyek, jadi status generate
			// tidak lagi bergantung pada sessionStorage (proyek dibuka di
			// perangkat lain tetap benar). sessionStorage cuma dipakai untuk
			// polling run yang sedang jalan di tab ini.
			if (p.latest_run) {
				runId = p.latest_run.id;
				await loadRun(p.latest_run.id);
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
		// Skema tahap dari API mengalahkan susunan bawaan: jumlah tahapnya bisa
		// berapa saja sesuai usulan AI judge yang disetujui pengguna.
		stages = run.stages ?? [];
		if (run.status === 'done' || run.status === 'failed') {
			const { artifacts } = await listRunArtifacts(rid);
			runArtifacts = artifacts;
			tiles = stageKeys.map((s) => {
				const a = artifacts.find((x) => x.kind === s);
				return {
					label: lbl(s),
					ok: Boolean(a),
					note: a ? `v${a.version} · siap` : 'Belum ada artefak',
					kind: s,
					artifactId: a?.id
				};
			});
		} else {
			tiles = stageKeys.map((s, i) => ({
				label: lbl(s),
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

	/**
	 * Sorot tile tahap yang diusulkan model di chat.
	 *
	 * Kalau tile-nya belum ada di DOM (tahap itu belum jalan), tidak ada yang
	 * bisa disorot — nilai tetap disimpan supaya sidebar bisa memakainya, dan
	 * pengguna tidak diberi umpan balik palsu.
	 */
	function sorotTahap(stage: string): void {
		usulTahap = stage;
		const el = document.getElementById(`tahap-${stage}`);
		if (!el) return;
		el.scrollIntoView({ behavior: 'smooth', block: 'center' });
		el.classList.add('ring-2', 'ring-[#2383e2]');
		setTimeout(() => el.classList.remove('ring-2', 'ring-[#2383e2]'), 2200);
	}

	/**
	 * Metrik yang dihitung dari artefak, bukan angka karangan.
	 *
	 * Yang bisa dihitung jujur dari isi dokumen: jumlah dokumen, jumlah tabel
	 * DBML, jumlah operasi OpenAPI, jumlah seksi dokumen proteksi. Kalau
	 * dokumennya belum ada, nilainya "—": halaman proyek lebih baik mengaku
	 * kosong daripada menampilkan angka contoh yang tampak seperti hasil
	 * analisis.
	 */
	function hitungMetrik(arts: Artifact[]) {
		const by = (k: string) => arts.find((a) => a.kind === k);
		const cnt = (re: RegExp, s?: string) => (s ? (s.match(re) ?? []).length : 0);

		const erd = by('erd');
		const openapi = by('openapi');
		const security = by('security');
		const tabel = cnt(/^\s*Table\s+\w+/gm, erd?.content);
		const op = cnt(/^\s{6,}(get|post|put|patch|delete):/gm, openapi?.content);
		const seksi = cnt(/^##\s+\S/gm, security?.content);

		return [
			{
				value: arts.length ? String(arts.length) : '—',
				label: 'Dokumen',
				note: arts.length ? `${arts.length}/${stageKeys.length} tahap tersedia` : 'Belum ada dokumen',
				icon: 'document'
			},
			{
				value: erd ? String(tabel) : '—',
				label: 'Entitas data',
				note: erd ? `${tabel} tabel di ERD v${erd.version}` : 'Belum ada ERD',
				icon: 'db-simple'
			},
			{
				value: openapi ? String(op) : '—',
				label: 'Endpoint',
				note: openapi ? `operasi di OpenAPI v${openapi.version}` : 'Belum ada OpenAPI',
				icon: 'code'
			},
			{
				value: security ? String(seksi) : '—',
				label: 'Bagian proteksi',
				note: security ? `seksi di Proteksi v${security.version}` : 'Belum ada dokumen proteksi',
				icon: 'shield'
			}
		];
	}

	/** Simpul benang merah = artefak terbaru per tahap, urut alur planner. */
	function hitungThread(arts: Artifact[]) {
		const TAG: Record<string, string> = {
			prd: 'PRD',
			erd: 'ENTITAS',
			openapi: 'KONTRAK',
			roles: 'ROLE',
			security: 'PROTEKSI',
			structure: 'STRUKTUR'
		};
		const ICON: Record<string, string> = {
			prd: 'document',
			erd: 'db-simple',
			openapi: 'code',
			roles: 'users',
			security: 'shield',
			structure: 'layers-row'
		};
		const nodes = stageKeys.filter((s) => arts.some((a) => a.kind === s)).map((s) => {
			const a = arts.find((x) => x.kind === s)!;
			return {
				tag: `${TAG[s] ?? s.toUpperCase()} · v${a.version}`,
				title: lbl(s),
				note: `${new Date(a.created_at).toLocaleDateString('id-ID')} · ${a.status}`,
				icon: ICON[s],
				artifactId: a.id
			};
		});
		return {
			nodes,
			connected: nodes.length ? `${nodes.length} dari ${stageKeys.length} dokumen sudah jadi` : '',
			pending: nodes.length ? 'Dokumen lain menyusul setelah generate' : 'Belum ada dokumen'
		};
	}

	/** Item proteksi = seksi dokumen security, apa adanya dari isinya. */
	function hitungProteksi(arts: Artifact[]) {
		const a = arts.find((x) => x.kind === 'security');
		if (!a) return { body: 'Belum ada dokumen proteksi untuk proyek ini.', items: [] };
		const items = (a.content.match(/^##\s+(.+)$/gm) ?? []).map((h) => ({
			label: h.replace(/^##\s+/, '').trim(),
			status: 'Tertulis di dokumen',
			ok: true
		}));
		return {
			body: `Ringkasan seksi dari Proteksi v${a.version}. Bukan audit keamanan.`,
			items
		};
	}

	/** Aktivitas = versi artefak (nyata, bertanggal), bukan log karangan. */
	function hitungAktivitas(arts: Artifact[]) {
		return [...arts]
			.sort((x, y) => y.created_at.localeCompare(x.created_at))
			.map((a) => ({
				time: new Date(a.created_at).toLocaleDateString('id-ID'),
				text: `${lbl(a.kind)} v${a.version} · ${a.status}`
			}));
	}

	// Layar konfirmasi: usulan skema tahap dari judge, boleh disunting sebelum
	// run dibuat. `null` berarti layar tertutup.
	let confirmOpen = $state(false);
	let confirmBusy = $state(false);
	let confirmNote = $state('');
	let confirmErr = $state('');
	let draft = $state<PlanStage[]>([]);

	async function openConfirm() {
		if (busy) return;
		confirmOpen = true;
		confirmBusy = true;
		confirmErr = '';
		confirmNote = '';
		try {
			const res = await getProjectStages(params.id);
			// Server sudah mengembalikan bentuk sah; salin supaya suntingan di
			// layar tidak menyentuh objek respons.
			draft = res.stages.map((s) => ({ ...s, depends_on: [...(s.depends_on ?? [])] }));
			confirmNote = res.note ?? '';
		} catch (e) {
			if (e instanceof ApiError && e.status === 401) {
				goto('/?login=1');
				return;
			}
			// Gagal menyusun usulan bukan jalan buntu: susunan awal tetap bisa
			// dipakai, dan itu memang yang dilakukan server tanpa body.
			confirmErr = e instanceof Error ? e.message : 'Gagal menyusun usulan tahap';
			draft = defaultPlanStages();
		} finally {
			confirmBusy = false;
		}
	}

	function closeConfirm() {
		if (busy) return;
		confirmOpen = false;
	}

	function moveStage(i: number, delta: number) {
		const j = i + delta;
		if (j < 0 || j >= draft.length) return;
		const next = [...draft];
		[next[i], next[j]] = [next[j], next[i]];
		draft = next.map((s, k) => ({ ...s, position: k }));
	}

	function removeStage(i: number) {
		// Minimal dua tahap: sisanya bukan lagi perancangan bertahap.
		if (draft.length <= 2) return;
		draft = draft.filter((_, k) => k !== i).map((s, k) => ({ ...s, position: k }));
	}

	async function startRun(skipConfirm = false) {
		if (busy) return;
		// Tekan "Mulai" tanpa membuka layar konfirmasi = tetap lewat layar itu,
		// supaya user selalu melihat tahap apa yang akan dibuat.
		if (!confirmOpen && !skipConfirm) {
			await openConfirm();
			return;
		}
		busy = true;
		errorMsg = '';
		try {
			const run = await createRun(params.id, confirmOpen ? draft : undefined);
			confirmOpen = false;
			runId = run.id;
			sessionStorage.setItem(`pk-run-${projectId || params.id}`, run.id);
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
			regenMsg = `${lbl(kind)} diperbarui.`;
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

	/**
	 * Semua blok di bawah kartu proyek dihitung dari artefak NYATA. Mockup beku
	 * (design/web/ringkasanProyek.html) dulu menaruh angka contoh, nama
	 * kolaborator karangan, dan log aktivitas palsu di sini; di halaman proyek
	 * pengguna, angka yang tampak seperti hasil analisis tapi bukan jauh lebih
	 * buruk daripada kolom kosong.
	 */
	const content = $derived.by<SummaryContent>(() => {
		// Metadata (versi, status, tanggal) dari daftar per-proyek; isi dokumen
		// untuk menghitung angka dari artefak run (daftar per-proyek tidak
		// mengirim `content`, lihat handler ListProjectArtifacts).
		const arts = projectArtifacts;
		const full = runArtifacts.length > 0 ? runArtifacts : arts;
		return {
			...summary,
			meta: { ...summary.meta, title: `${name || 'Proyek'} · Ringkasan blueprint` },
			breadcrumb: { ...summary.breadcrumb, project: name || '' },
			pageHeader: {
				...summary.pageHeader,
				eyebrow: `${name || 'Proyek'} / Ringkasan`,
				title: 'Rancangan yang saling terhubung.',
				subtitle: brief || 'Blueprint proyek ini, tahap demi tahap.'
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
				description: brief || 'Brief masih kosong.',
				// Pemilik = pengguna sendiri: belum ada kolaborator di model data.
				facts: createdAt ? [`Dibuat ${createdAt}`, `Pemilik: ${auth.me?.username ?? 'saya'}`] : [],
				readiness: {
					...summary.overview.readiness,
					percent: total > 0 ? Math.round((progress / total) * 100) : 0,
					note: runStatus ? `${progress}/${total} tahap selesai` : 'Belum ada generate'
				}
			},
			metrics: hitungMetrik(full),
			thread: { ...summary.thread, ...hitungThread(arts) },
			security: { ...summary.security, ...hitungProteksi(full) },
			review: {
				...summary.review,
				counter: `${arts.length}/${stageKeys.length} dokumen`,
				activities: hitungAktivitas(arts)
			}
		};
	});
</script>

<svelte:head>
	<title>{content.meta.title}</title>
</svelte:head>
<div class={content.meta.bodyClass} id="atas">
	<SummarySidebar
		projectName={name}
		ready={progress}
		total={total || stagesEff.length}
		artifacts={projectArtifacts}
		{slug}
		stages={stagesEff}
	/>

	<div class="flex-1 flex flex-col h-full overflow-hidden bg-[#F8F9FA]" data-purpose="main-layout">
		<SummaryTopbar breadcrumb={content.breadcrumb} />

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
				<SummaryHeader overview={content.overview} pageHeader={content.pageHeader} />
				{#if !runId || runStatus === 'failed'}
					<div class="rounded-xl border border-slate-200 bg-white p-6 flex items-center justify-between">
						<p class="text-sm text-[#475569]">
							{runStatus === 'failed'
								? 'Generate terakhir gagal. Kredit otomatis dikembalikan; coba lagi.'
								: `Belum ada generate. Mulai untuk menyusun ${stagesEff.length} dokumen planning.`}
						</p>
						<button
							type="button"
							disabled={busy}
							class="inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm disabled:opacity-50"
							onclick={() => startRun()}
							>{busy ? 'Memulai…' : 'Mulai generate (1 kredit)'}</button
						>
					</div>
					{#if confirmOpen}
						<!-- Skema tahap disusun judge dari brief; user melihat dan
						     menyuntingnya SEBELUM kredit terpotong. -->
						<div class="mt-4 rounded-xl border border-slate-200 bg-white p-5">
							<div class="flex items-start justify-between gap-4">
								<div>
									<p class="text-sm font-semibold">Tahap yang akan disusun</p>
									<p class="text-xs text-slate-500 mt-1">
										Usulan AI dari brief proyek. Urutan bisa diubah, tahap bisa dibuang.
										{confirmBusy ? 'Menyusun…' : ''}
									</p>
								</div>
								<button
									type="button"
									class="text-xs text-slate-500 hover:text-slate-800 transition-all"
									onclick={closeConfirm}>Tutup</button
								>
							</div>

							{#if confirmErr}
								<p class="mt-3 text-xs text-amber-700">
									{confirmErr} — dipakai susunan awal {draft.length} tahap.
								</p>
							{/if}
							{#if confirmNote}
								<p class="mt-3 text-xs text-amber-700">{confirmNote}</p>
							{/if}

							<ol class="mt-4 space-y-2">
								{#each draft as s, i (s.key + '-' + i)}
									<li class="flex items-center gap-3 rounded-lg border border-slate-200 p-3">
										<span class="text-xs text-slate-400 w-5">{i + 1}</span>
										<div class="min-w-0 flex-1">
											<p class="text-sm font-medium truncate">{s.label}</p>
											<p class="text-xs text-slate-500 truncate">
												{s.output_format}
												{#if s.intent}· {s.intent}{/if}
											</p>
										</div>
										<div class="flex items-center gap-1">
											<button
												type="button"
												disabled={i === 0}
												class="px-2 py-1 text-xs rounded border border-slate-200 hover:bg-slate-50 transition-all disabled:opacity-40"
												onclick={() => moveStage(i, -1)}>↑</button
											>
											<button
												type="button"
												disabled={i === draft.length - 1}
												class="px-2 py-1 text-xs rounded border border-slate-200 hover:bg-slate-50 transition-all disabled:opacity-40"
												onclick={() => moveStage(i, 1)}>↓</button
											>
											<button
												type="button"
												disabled={draft.length <= 2}
												class="px-2 py-1 text-xs rounded border border-slate-200 text-rose-600 hover:bg-rose-50 transition-all disabled:opacity-40"
												onclick={() => removeStage(i)}>Buang</button
											>
										</div>
									</li>
								{/each}
							</ol>

							<div class="mt-4 flex items-center gap-3">
								<button
									type="button"
									disabled={busy || confirmBusy || draft.length < 2}
									class="inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all disabled:opacity-50"
									onclick={() => startRun(true)}
									>{busy ? 'Memulai…' : `Mulai ${draft.length} tahap (1 kredit)`}</button
								>
								<button
									type="button"
									disabled={busy || confirmBusy}
									class="text-xs text-slate-500 hover:text-slate-800 transition-all"
									onclick={openConfirm}>Susun ulang</button
								>
							</div>
						</div>
					{/if}
				{/if}
				{#if runId}
					<div class="rounded-xl border border-slate-200 bg-white p-6" id="tahap">
						<p class="text-sm font-semibold mb-3">
							{stagesEff.length} tahap planning
							<span class="ml-2 text-xs font-normal text-slate-500">
								{#if runStatus === 'running' || runStatus === 'queued'}
									{currentStage ? `sedang: ${currentStage}` : 'antre…'} · {progress}/{total}
								{:else if runStatus === 'done'}
									selesai
								{/if}
							</span>
						</p>
						<!-- Kolom tetap: Tailwind tidak menghasilkan kelas dari nilai
						     runtime, jadi jumlah kolom TIDAK boleh disisipkan di sini. -->
						<div class="grid grid-cols-2 md:grid-cols-6 gap-3">
							{#each tiles as t (t.label)}
								<div
									class="rounded-lg border p-3 transition-all {t.ok
										? 'border-emerald-200 bg-emerald-50'
										: 'border-slate-200 bg-slate-50'}"
									id="tahap-{t.kind}"
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
				<!-- Chat konteks proyek: hanya perancangan, gratis. Server menyimpan
				     percakapan, dan usul tahap dari model dipakai untuk menyorot
				     menu sidebar. -->
				<ProjectChat projectId={projectId || params.id} onsuggest={sorotTahap} />
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
