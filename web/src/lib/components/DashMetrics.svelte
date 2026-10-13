<script lang="ts">
	import type { DashboardContent } from '#lib/mock/dashboard-types';

	let { metrics }: { metrics: DashboardContent['metrics'] } = $props();
</script>

<section class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4" data-purpose="stat-metrics">
	{#each metrics as m (m.label)}
		<div class="bg-white border border-slate-200 rounded-xl p-5 shadow-xs">
			<div class="text-3xl font-bold text-slate-900 leading-none mb-3">{m.value}</div>
			<div class="text-xs font-bold text-slate-800">{m.label}</div>
			<div class="text-2xs text-slate-400 mt-0.5">{m.note}</div>
		</div>
	{/each}
</section>