<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { DashboardContent } from '#lib/mock/dashboard-types';

	let {
		breadcrumb,
		chip,
		credits,
		topupLabel
	}: {
		breadcrumb: DashboardContent['breadcrumb'];
		chip: string;
		credits: string;
		topupLabel: string;
	} = $props();
</script>

<header
	class="h-14 border-b border-slate-200 bg-white flex items-center justify-between px-6 sticky top-0 z-20"
	data-purpose="top-navigation"
>
	<nav aria-label="Breadcrumb" class="flex items-center gap-2 text-xs font-medium text-slate-500">
		<Icon name="folder" class="w-4 h-4 text-slate-400" />
		<span>{breadcrumb.root}</span>
		<span class="text-slate-300">/</span>
		<span class="text-slate-900 font-semibold">{breadcrumb.current}</span>
	</nav>

	<div class="flex items-center gap-3">
		<span class="text-2xs font-medium bg-slate-100 text-slate-500 px-2 py-0.5 rounded border border-slate-200"
			>{chip}</span
		>
		<span class="text-xs font-semibold text-slate-700 bg-slate-50 px-2.5 py-1 rounded border border-slate-200"
			>{credits}</span
		>
		<button
			class="text-xs font-semibold px-3 py-1 rounded border border-slate-300 hover:bg-slate-50 transition-colors text-slate-800 shadow-sm"
			type="button"
		>
			{topupLabel}
		</button>
	</div>
</header>