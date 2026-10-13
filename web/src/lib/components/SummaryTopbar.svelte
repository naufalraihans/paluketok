<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { SummaryContent } from '#lib/mock/summary-types';

	let {
		breadcrumb,
		savedAt,
		collaborators,
		shareLabel
	}: Pick<SummaryContent, 'breadcrumb' | 'savedAt' | 'collaborators' | 'shareLabel'> = $props();
</script>

<header
	class="h-14 border-b border-[#E2E8F0] bg-white flex items-center justify-between px-8 shrink-0"
	data-purpose="top-navigation-bar"
>
	<div class="flex items-center gap-2 text-xs">
		<div class="flex items-center gap-1.5 text-[#64748B]">
			<Icon name="folder" class="w-4 h-4" />
			<span class="hover:underline cursor-pointer">{breadcrumb.root}</span>
		</div>
		<span class="text-[#CBD5E1]">/</span>
		<span class="font-bold text-[#0F172A]">{breadcrumb.project}</span>
		<span
			class="ml-1 text-[11px] font-semibold bg-[#F1F5F9] text-[#475569] px-2 py-0.5 rounded-full border border-[#E2E8F0]"
			>{breadcrumb.chip}</span
		>
	</div>

	<div class="flex items-center gap-4">
		<span class="text-xs text-[#64748B]">{savedAt}</span>
		<div class="flex -space-x-1.5">
			{#each collaborators as c (c.initials)}
				<div
					class="w-6 h-6 rounded-full {c.class} border-2 border-white flex items-center justify-center text-[10px] font-semibold"
				>
					{c.initials}
				</div>
			{/each}
		</div>
		<button
			class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-[#E2E8F0] bg-white text-xs font-semibold text-[#0F172A] hover:bg-slate-50 shadow-2xs"
			type="button"
		>
			<Icon name="share" class="w-3.5 h-3.5 text-[#64748B]" />
			{shareLabel}
		</button>
	</div>
</header>
