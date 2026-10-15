<script lang="ts">
	import { goto } from '$app/navigation';
	import { auth, refreshMe } from '#lib/auth.svelte';
	import DashSidebar from '#lib/components/DashSidebar.svelte';
	import DashTopbar from '#lib/components/DashTopbar.svelte';
	import AdminProjects from '#lib/components/admin/AdminProjects.svelte';
	import AdminRuns from '#lib/components/admin/AdminRuns.svelte';
	import AdminUsers from '#lib/components/admin/AdminUsers.svelte';
	import AdminDecisions from '#lib/components/admin/AdminDecisions.svelte';
	import AdminAI from '#lib/components/admin/AdminAI.svelte';

	// Konsol admin. Layout & komponen sidebar/topbar sama persis dengan /projects
	// (satu desain), isi & menu berbeda: 5 tab operasional, bukan proyek user.
	type Tab = 'projects' | 'runs' | 'users' | 'decisions' | 'ai';
	const TABS: { id: Tab; label: string }[] = [
		{ id: 'projects', label: 'Projects' },
		{ id: 'runs', label: 'Runs' },
		{ id: 'users', label: 'Users' },
		{ id: 'decisions', label: 'Keputusan' },
		{ id: 'ai', label: 'AI Settings' }
	];
	let tab = $state<Tab>('projects');

	$effect(() => {
		if (!auth.loading && !auth.user) goto('/login', { replaceState: true });
	});

	// Guard: /v1/me is_admin. Bukan admin -> /projects.
	let checked = $state(false);
	$effect(() => {
		if (auth.loading || !auth.user) return;
		refreshMe().then(() => {
			checked = true;
			if (!auth.me?.is_admin) goto('/projects', { replaceState: true });
		});
	});

	const credits = $derived(auth.me ? `${auth.me.credits} kredit` : '0 kredit');

	// Data sidebar memakai kontrak DashboardContent yang sama dengan /projects,
	// hanya item menunya yang berbeda (tiap tab = satu entri).
	const sidebar = $derived({
		brand: 'PaluKetok',
		studio: 'Konsol Admin',
		activeLabel: 'Mode',
		activeProject: 'Operasional',
		sectionLabel: 'Menu Admin',
		items: TABS.map((t, i) => ({
			label: t.label,
			icon: ['folder', 'refresh', 'users', 'bolt', 'sliders'][i],
			href: '#',
			active: tab === t.id
		})),
		user: {
			name: auth.me?.username ?? auth.me?.full_name ?? 'Admin',
			meta: auth.me ? `${auth.me.plan_tier} · ${credits}` : 'memuat…'
		}
	});

	function pick(label: string): void {
		const found = TABS.find((t) => t.label === label);
		if (found) tab = found.id;
	}
</script>

<svelte:head><title>Admin · PaluKetok</title></svelte:head>
<div class="bg-[#fafbfc] text-[#0f172a] font-sans antialiased min-h-screen flex">
	<!-- Menu sidebar admin memilih tab (DashSidebar onNavigate). -->
	<DashSidebar {sidebar} onNavigate={pick} />

	<div class="flex-1 flex flex-col min-w-0">
		<DashTopbar
			breadcrumb={{ root: 'Admin', current: TABS.find((t) => t.id === tab)?.label ?? '' }}
			chip="Data live"
			{credits}
			topupLabel="Top-up"
		/>

		<main class="flex-1 p-6 lg:p-8 space-y-6 max-w-[1360px] mx-auto w-full">
			{#if !checked}
				<div class="rounded-xl border border-blue-200 bg-blue-50 p-6 text-sm text-blue-700">
					Memeriksa hak akses…
				</div>
			{:else if !auth.me?.is_admin}
				<div class="rounded-xl border border-rose-200 bg-rose-50 p-6 space-y-2">
					<p class="text-sm font-semibold text-rose-700">Akses ditolak</p>
					<p class="text-sm text-rose-600">Akun ini bukan admin.</p>
				</div>
			{:else}
				<nav class="flex gap-1 border-b border-slate-200 overflow-x-auto" aria-label="Tab admin">
					{#each TABS as t (t.id)}
						<button
							type="button"
							class="px-4 py-2.5 text-xs font-semibold whitespace-nowrap border-b-2 -mb-px transition-colors {tab === t.id
								? 'border-slate-900 text-slate-900'
								: 'border-transparent text-slate-500 hover:text-slate-900'}"
							onclick={() => (tab = t.id)}
							aria-current={tab === t.id ? 'page' : undefined}>{t.label}</button
						>
					{/each}
				</nav>

				{#if tab === 'projects'}<AdminProjects />{:else if tab === 'runs'}<AdminRuns />{:else if tab === 'users'}<AdminUsers />{:else if tab === 'decisions'}<AdminDecisions />{:else}<AdminAI />{/if}
			{/if}
		</main>
	</div>
</div>
