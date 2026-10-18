<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { SummaryContent } from '#lib/mock/summary-types';

	// Kartu metadata proyek. Tombol "Ekspor blueprint" dari mockup DIBUANG:
	// belum ada endpoint ekspor, jadi tombolnya cuma janji kosong.
	let { overview, pageHeader }: Pick<SummaryContent, 'overview' | 'pageHeader'> = $props();
</script>

<section class="flex items-start justify-between" data-purpose="page-title-header">
	<div>
		<span class="text-[11px] font-bold text-[#64748B] tracking-wider uppercase block mb-1"
			>{pageHeader.eyebrow}</span
		>
		<h2 class="text-2xl font-bold text-[#0F172A] tracking-tight">{pageHeader.title}</h2>
		<p class="text-xs text-[#64748B] mt-1">{pageHeader.subtitle}</p>
	</div>
</section>

<section
	class="border border-[#E2E8F0] bg-white rounded-2xl p-5 shadow-xs flex items-center justify-between"
	data-purpose="project-metadata-card"
>
	<div class="flex items-center gap-4">
		<div
			class="w-16 h-16 rounded-xl bg-[#F8FAFC] border border-[#E2E8F0] flex items-center justify-center shrink-0"
		>
			<span class="font-serif-logo text-2xl font-bold text-[#0F172A] italic">{overview.monogram}</span>
		</div>
		<div>
			<div class="flex items-center gap-2 mb-1">
				<h3 class="text-base font-bold text-[#0F172A]">{overview.name}</h3>
				<span
					class="text-[11px] font-medium bg-[#F1F5F9] text-[#475569] px-2 py-0.5 rounded-full border border-[#E2E8F0]"
					>{overview.status}</span
				>
			</div>
			<p class="text-xs text-[#64748B]">{overview.description}</p>
			<div class="flex items-center gap-2 text-[11px] text-[#64748B] mt-1.5 font-medium">
				{#each overview.facts as fact, i (fact)}
					{#if i > 0}<span>·</span>{/if}
					<span>{fact}</span>
				{/each}
			</div>
		</div>
	</div>
	<div class="w-56 shrink-0 pl-4">
		<div class="flex justify-between items-baseline mb-1.5">
			<span class="text-[11px] text-[#64748B] font-medium">{overview.readiness.label}</span>
			<span class="text-xs font-bold text-[#0F172A]">{overview.readiness.percent}%</span>
		</div>
		<div class="w-full h-1.5 bg-[#E2E8F0] rounded-full overflow-hidden">
			<div class="h-full bg-[#0F172A] rounded-full" style="width: {overview.readiness.percent}%;"></div>
		</div>
		<p class="text-[11px] text-[#94A3B8] mt-1.5 text-right font-medium">{overview.readiness.note}</p>
	</div>
</section>
