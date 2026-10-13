<script lang="ts">
	import type { DashboardContent } from '#lib/mock/dashboard-types';

	let {
		hero,
		createLabel
	}: {
		hero: DashboardContent['hero'];
		createLabel: string;
	} = $props();
</script>

<section
	class="flex flex-col md:flex-row md:items-center md:justify-between gap-4 pb-2"
	data-purpose="dashboard-header"
>
	<div>
		<span class="text-2xs font-bold text-slate-400 tracking-wider uppercase block mb-1.5"
			>{hero.kicker}</span
		>
		<h2 class="text-2xl lg:text-3xl font-extrabold text-slate-900 tracking-tight">{hero.heading}</h2>
		<p class="text-xs lg:text-sm text-slate-500 mt-1">{hero.body}</p>
	</div>
	<button
		class="inline-flex items-center justify-center gap-1.5 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-xs font-semibold rounded-lg shadow-sm transition-colors whitespace-nowrap"
		type="button"
	>
		<span>{createLabel}</span>
	</button>
</section>