<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { DashboardContent } from '#lib/mock/dashboard-types';

	let {
		sidebar,
		onNavigate
	}: {
		sidebar: DashboardContent['sidebar'];
		// Dipakai konsol admin: item menu memilih tab, bukan berpindah halaman.
		onNavigate?: (label: string) => void;
	} = $props();
</script>

<aside
	class="w-[260px] flex-shrink-0 bg-white border-r border-slate-200 min-h-screen flex flex-col justify-between p-4 hidden lg:flex fixed lg:sticky top-0 h-screen overflow-y-auto z-30"
	data-purpose="sidebar-navigation"
>
	<div>
		<div class="flex items-center gap-3 px-1.5 py-1 mb-5" data-purpose="sidebar-brand">
			<div class="w-8 h-8 rounded-lg bg-slate-900 flex items-center justify-center text-white font-bold shadow-sm">
				<Icon name="wrench" class="w-4 h-4" width={2.2} />
			</div>
			<div>
				<h1 class="text-sm font-bold tracking-tight text-slate-900 leading-none">{sidebar.brand}</h1>
				<span class="text-2xs text-slate-400 font-medium">{sidebar.studio}</span>
			</div>
		</div>

		<div class="mb-6">
			<span class="block text-3xs font-bold text-slate-400 tracking-wider uppercase mb-1.5 px-1.5"
				>{sidebar.activeLabel}</span
			>
			<div
				class="flex items-center justify-between px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg cursor-pointer hover:bg-slate-100/80 transition-colors"
				data-purpose="project-switcher"
			>
				<span class="text-xs font-semibold text-slate-800 truncate">{sidebar.activeProject}</span>
				<Icon name="chevron-up-down" class="w-3.5 h-3.5 text-slate-400 flex-shrink-0" />
			</div>
		</div>

		{#if sidebar.projects && sidebar.projects.length > 1}
			<nav class="space-y-1 mb-6" data-purpose="sidebar-projects">
				<span class="block text-3xs font-bold text-slate-400 tracking-wider uppercase mb-2 px-1.5"
					>Proyek</span
				>
				{#each sidebar.projects as p (p.id)}
					<a
						class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs truncate transition-colors {p.name ===
						sidebar.activeProject
							? 'font-semibold bg-slate-100 text-slate-900'
							: 'font-medium text-slate-600 hover:text-slate-900 hover:bg-slate-50'}"
						href="/projects/{p.slug ?? p.id}"
					>
						<Icon name="folder" class="w-4 h-4 text-slate-400 flex-shrink-0" />
						<span class="truncate">{p.name}</span>
					</a>
				{/each}
			</nav>
		{/if}

		<nav class="space-y-1" data-purpose="sidebar-menu">
			<span class="block text-3xs font-bold text-slate-400 tracking-wider uppercase mb-2 px-1.5"
				>{sidebar.sectionLabel}</span
			>
			{#each sidebar.items as item (item.label)}
				{#if item.active}
					<a
						class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs font-semibold bg-blue-50 text-blue-600 border border-blue-100 transition-colors"
						href={item.href}
						onclick={(e) => {
							if (onNavigate) {
								e.preventDefault();
								onNavigate(item.label);
							}
						}}
					>
						<Icon name={item.icon} class="w-4 h-4 text-blue-600 flex-shrink-0" />
						{item.label}
					</a>
				{:else}
					<a
						class="flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs font-medium text-slate-600 hover:text-slate-900 hover:bg-slate-50 transition-colors"
						href={item.href}
						onclick={(e) => {
							if (onNavigate) {
								e.preventDefault();
								onNavigate(item.label);
							}
						}}
					>
						<Icon name={item.icon} class="w-4 h-4 text-slate-400 flex-shrink-0" />
						{item.label}
					</a>
				{/if}
			{/each}
		</nav>
	</div>

	<div class="pt-4 border-t border-slate-100" data-purpose="user-profile-badge">
		<div class="px-2 py-1.5 rounded-lg hover:bg-slate-50 transition-colors">
			<h4 class="text-xs font-bold text-slate-800 leading-tight">{sidebar.user.name}</h4>
			<p class="text-2xs text-slate-400 mt-0.5">{sidebar.user.meta}</p>
		</div>
	</div>
</aside>