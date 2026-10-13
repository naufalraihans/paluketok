<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { SummaryContent } from '#lib/mock/summary-types';

	let { metrics }: Pick<SummaryContent, 'metrics'> = $props();
</script>

<section class="grid grid-cols-4 gap-4" data-purpose="metrics-summary">
	{#each metrics as m (m.label)}
		<div class="bg-white border border-[#E2E8F0] rounded-2xl p-4 shadow-xs relative">
			<div class="absolute top-4 right-4 text-[#94A3B8]">
				<Icon name={m.icon} class="w-4 h-4" />
			</div>
			<span class="text-2xl font-bold text-[#0F172A] block tracking-tight">{m.value}</span>
			<p class="text-xs font-semibold text-[#0F172A] mt-1">{m.label}</p>
			<p class="text-[11px] text-[#64748B] mt-0.5">{m.note}</p>
		</div>
	{/each}
</section>
