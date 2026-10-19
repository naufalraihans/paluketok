<script lang="ts">
	import DashSidebar from '#lib/components/DashSidebar.svelte';
	import DashTopbar from '#lib/components/DashTopbar.svelte';
	import DashHero from '#lib/components/DashHero.svelte';
	import DashMetrics from '#lib/components/DashMetrics.svelte';
	import DashProjectList from '#lib/components/DashProjectList.svelte';
	import DashBanners from '#lib/components/DashBanners.svelte';
	import DashNotice from '#lib/components/DashNotice.svelte';
	import DashStates from '#lib/components/DashStates.svelte';
	import { dashboard } from '#lib/mock/dashboard';
	import { listProjects, getMe, ApiError } from '#lib/api';
	import { auth, refreshMe } from '#lib/auth.svelte';
	import { goto } from '$app/navigation';
	import type { ProjectCard, DashMetric } from '#lib/mock/dashboard-types';

	// Fase 10 langkah 36: data dari API /v1. Kerangka halaman (teks statis,
	// susunan blok) tetap dari mock = mockup beku; hanya kartu proyek, metrik,
	// kredit yang datang dari server.
	let projects = $state<ProjectCard[]>([]);
	let credits = $state('…');
	let credValue = $state<string | null>(null);
	let tier = $state('');
	let loading = $state(true);
	let error = $state<{ title: string; body: string } | null>(null);
	let showCreate = $state(false);
	let newName = $state('');
	let newBrief = $state('');
	let busy = $state(false);
	let formError = $state('');
	// Konfirmasi arsip memakai window.confirm supaya tidak menambah komponen
	// modal hanya untuk satu aksi destruktif.
	let archiving = $state('');

	const metrics = $derived.by<DashMetric[]>(() => {
		const ready = projects.filter((p) => p.status === 'ready').length;
		const draft = projects.filter((p) => p.status === 'draft').length;
		const archived = projects.filter((p) => p.status === 'archived').length;
		return [
			{ value: String(projects.length), label: 'Proyek saya', note: 'Data live' },
			{ value: String(ready), label: 'Siap · ready', note: 'Data live' },
			{ value: String(draft), label: 'Draf · draft', note: 'Data live' },
			{ value: String(archived), label: 'Arsip · archived', note: 'Data live' }
		];
	});

	const content = $derived({
		...dashboard,
		chip: auth.me ? 'Data live' : dashboard.chip,
		credits: credits,
		sidebar: {
			...dashboard.sidebar,
			activeProject: projects.length === 1 ? projects[0].name : dashboard.sidebar.activeProject,
			// Hanya menu yang halamannya BENAR-BENAR ada. Menu tahap (PRD, ERD,
			// dst.) hidup di dalam /projects/{slug}; daftarnya sudah ada di sana
			// dan muncul bertahap mengikuti artefak. Menaruhnya di sini dulu
			// membuat 8 tautan yang semuanya menunjuk /projects — bisa dipencet
			// tapi tidak ke mana-mana.
			items: dashboard.sidebar.items.filter((i) => i.href !== '/projects'),
			// Kartu mock tidak punya id; hanya proyek nyata yang bisa dipilih.
			projects: projects
				.filter((p): p is ProjectCard & { id: string } => Boolean(p.id))
				.map((p) => ({ id: p.id, name: p.name, slug: p.slug })),
			user: {
				name: auth.me?.username ?? auth.me?.full_name ?? 'Pengguna',
				meta: auth.me ? `${auth.me.plan_tier} · ${credits}` : 'memuat…'
			}
		},
		list: { ...dashboard.list, count: `${projects.length} proyek · data live`, cards: projects },
		metrics,
		balance: { ...dashboard.balance, value: credValue ?? '0', meta: `Data live · tier ${tier || 'free'}` }
	});

	async function load() {
		loading = true;
		error = null;
		try {
			const [{ projects: ps }] = await Promise.all([listProjects(), refreshMe()]);
			projects = ps.map((p) => ({
				id: p.id,
				slug: p.slug,
				name: p.name,
				brief: p.brief,
				stack:
					[p.stack?.fe, p.stack?.be, p.stack?.db, p.stack?.payment]
						.filter(Boolean)
						.join(' / ') || 'Stack belum diisi',
				status: p.status,
				meta:
					p.status === 'archived'
						? `Diarsipkan · pemilik ${auth.me?.username ?? 'saya'}`
						: `Dibuat ${new Date(p.created_at).toLocaleDateString('id-ID')} · pemilik ${auth.me?.username ?? 'saya'}`,
				action: p.status === 'archived' ? 'Lihat arsip' : 'Buka proyek'
			}));
			if (auth.me) {
				credits = `${auth.me.credits} kredit`;
				credValue = String(auth.me.credits);
				tier = auth.me.plan_tier;
			}
		} catch (e) {
			if (e instanceof ApiError && e.status === 401) {
				goto('/?login=1');
				return;
			}
			error = {
				title: 'Permintaan tidak berhasil',
				body:
					e instanceof Error
						? e.message
						: 'Terjadi kesalahan atau terlalu banyak permintaan. Coba muat ulang daftar proyek.'
			};
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (auth.user && loading && !error) load();
		else if (!auth.loading && !auth.user && loading) loading = false;
	});

	async function create() {
		formError = '';
		if (!newName || !newBrief || busy) return;
		busy = true;
		try {
			await createProjectSafe();
			showCreate = false;
			newName = '';
			newBrief = '';
			loading = true;
			await load();
		} catch (e) {
			formError = e instanceof Error ? e.message : 'Gagal membuat proyek';
		} finally {
			busy = false;
		}
	}

	async function createProjectSafe() {
		const { createProject } = await import('#lib/api');
		await createProject({ name: newName, brief: newBrief });
	}

	/**
	 * Arsipkan proyek. Server menandai `archived`, tidak menghapus baris, jadi
	 * artefak dan riwayatnya tetap utuh bila proyek dibuka lagi.
	 */
	async function archive(card: ProjectCard) {
		if (!card.id || archiving) return;
		const ok = window.confirm(
			`Arsipkan "${card.name}"?\n\nProyek tidak dihapus permanen. Dokumen dan riwayatnya tetap tersimpan, dan proyek bisa dibuka lagi kapan saja.`
		);
		if (!ok) return;
		archiving = card.id;
		try {
			const { archiveProject } = await import('#lib/api');
			await archiveProject(card.slug ?? card.id);
			await load();
		} catch (e) {
			formError = e instanceof Error ? e.message : 'Gagal mengarsipkan proyek';
		} finally {
			archiving = '';
		}
	}
</script>

<svelte:head>
	<title>{content.meta.title}</title>
</svelte:head>
<!-- Kelas <body> mockup dipasang di pembungkus: <svelte:body> Svelte 5 hanya
     menerima event handler, bukan atribut class. -->
<div class="bg-[#fafbfc] text-[#0f172a] font-sans antialiased min-h-screen flex">
	<DashSidebar sidebar={content.sidebar} />

	<div class="flex-1 flex flex-col min-w-0">
		<DashTopbar
			breadcrumb={content.breadcrumb}
			chip={content.chip}
			credits={content.credits}
			topupLabel={content.topupLabel}
		/>

		<main class="flex-1 p-6 lg:p-8 space-y-8 max-w-[1360px] mx-auto w-full">
			{#if !auth.user && !auth.loading}
				<div class="rounded-xl border border-slate-200 bg-white p-8 text-center space-y-3">
					<p class="text-sm text-[#475569]">
						Masuk dulu untuk melihat daftar proyek milikmu.
					</p>
					<a
						href="/?login=1"
						class="inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm"
						>Masuk</a
					>
				</div>
			{:else if error}
				<div class="rounded-xl border border-rose-200 bg-rose-50 p-6 space-y-2">
					<p class="text-sm font-semibold text-rose-700">{error.title}</p>
					<p class="text-sm text-rose-600">{error.body}</p>
					<button
						type="button"
						class="text-sm font-medium text-rose-700 underline underline-offset-2 hover:text-rose-900 transition-colors"
						onclick={load}>Muat ulang daftar</button
					>
				</div>
			{:else if loading}
				<div class="rounded-xl border border-blue-200 bg-blue-50 p-6 text-sm text-blue-700">
					Memuat daftar proyek…
				</div>
			{:else if projects.length === 0}
				<div class="rounded-xl border border-slate-200 bg-white p-8 text-center space-y-3">
					<p class="text-sm font-semibold text-[#0f172a]">Belum ada proyek</p>
					<p class="text-sm text-[#475569]">
						0 proyek. Mulai dengan nama dan brief kebutuhan software.
					</p>
					<button
						type="button"
						class="inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm"
						onclick={() => (showCreate = true)}
						>{content.createLabel}</button
					>
				</div>
			{:else}
				<DashHero hero={content.hero} createLabel={content.createLabel} />
				<DashMetrics metrics={content.metrics} />
				<DashProjectList list={content.list} onArchive={archive} />
			{/if}

			{#if showCreate}
				<div class="rounded-xl border border-slate-200 bg-white p-6 space-y-4">
					<p class="text-sm font-semibold">Buat proyek baru</p>
					<input
						class="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-slate-300 transition-all"
						placeholder="Nama proyek"
						bind:value={newName}
					/>
					<textarea
						class="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-slate-300 transition-all"
						rows="3"
						placeholder="Brief kebutuhan software"
						bind:value={newBrief}
					></textarea>
					{#if formError}<p class="text-sm text-rose-600">{formError}</p>{/if}
					<div class="flex gap-2">
						<button
							type="button"
							disabled={busy}
							class="inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm disabled:opacity-50"
							onclick={create}>{busy ? 'Menyimpan…' : 'Simpan proyek'}</button
						>
						<button
							type="button"
							class="text-sm font-medium text-slate-600 hover:text-slate-900 transition-colors"
							onclick={() => (showCreate = false)}>Batal</button
						>
					</div>
				</div>
			{/if}

			<DashBanners
				pipeline={content.pipeline}
				balance={content.balance}
				onCreate={() => (showCreate = true)}
			/>
			{#if !auth.me}
				<DashNotice notice={content.notice} />
			{/if}
		</main>
	</div>
</div>
