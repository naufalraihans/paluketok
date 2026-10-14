<script lang="ts">
	import SiteNav from '#lib/components/SiteNav.svelte';
	import SiteFooter from '#lib/components/SiteFooter.svelte';
	import { landing } from '#lib/mock/landing';
	import { auth, refreshMe } from '#lib/auth.svelte';
	import { goto } from '$app/navigation';
	import AdminProjects from '#lib/components/admin/AdminProjects.svelte';
	import AdminRuns from '#lib/components/admin/AdminRuns.svelte';
	import AdminUsers from '#lib/components/admin/AdminUsers.svelte';
	import AdminDecisions from '#lib/components/admin/AdminDecisions.svelte';
	import AdminAI from '#lib/components/admin/AdminAI.svelte';

	// Fase 10 Tahap B: /admin SATU halaman dengan tab (path, bukan subdomain).
	// Deviasi planner.md:136 dicatat di plan/changelog.md.
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
</script>

<svelte:head><title>Admin · PaluKetok</title></svelte:head>
<div class="min-h-screen flex flex-col bg-white text-[#0f172a] font-sans antialiased">
	<SiteNav nav={landing.nav} />
	<main class="flex-1 w-full max-w-6xl mx-auto px-6 py-8 space-y-6">
		<header class="flex items-end justify-between gap-4 flex-wrap">
			<div>
				<h1 class="text-xl font-bold tracking-tight">Admin</h1>
				<p class="text-sm text-[#475569] mt-0.5">Konsol operasional. Satu halaman, lima tab.</p>
			</div>
			<span class="text-2xs font-medium bg-slate-100 text-slate-500 px-2 py-0.5 rounded border border-slate-200"
				>{auth.me?.plan_tier ?? 'free'} · {auth.me?.credits ?? 0} kredit</span
			>
		</header>

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

		{#if !checked}
			<p class="text-sm text-[#475569]">Memeriksa hak akses...</p>
		{:else if auth.me?.is_admin}
			{#if tab === 'projects'}<AdminProjects />{:else if tab === 'runs'}<AdminRuns />{:else if tab === 'users'}<AdminUsers />{:else if tab === 'decisions'}<AdminDecisions />{:else}<AdminAI />{/if}
		{/if}
	</main>
	<SiteFooter footer={landing.footer} />
</div>