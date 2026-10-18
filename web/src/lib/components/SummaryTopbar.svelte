<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { SummaryContent } from '#lib/mock/summary-types';

	// Topbar ringkasan. Tombol "Bagikan" dan avatar kolaborator dari mockup
	// DIBUANG: belum ada fitur berbagi maupun kolaborator, jadi keduanya cuma
	// tombol mati yang menjanjikan hal yang tidak ada.
	let { breadcrumb }: Pick<SummaryContent, 'breadcrumb'> = $props();
</script>

<header
	class="h-14 border-b border-[#E2E8F0] bg-white flex items-center justify-between px-8 shrink-0"
	data-purpose="top-navigation-bar"
>
	<div class="flex items-center gap-2 text-xs">
		<a class="flex items-center gap-1.5 text-[#64748B] hover:text-[#0F172A] transition-colors" href="/projects">
			<Icon name="folder" class="w-4 h-4" />
			<span>{breadcrumb.root}</span>
		</a>
		<span class="text-[#CBD5E1]">/</span>
		<span class="font-bold text-[#0F172A]">{breadcrumb.project}</span>
		{#if breadcrumb.chip}
			<span
				class="ml-1 text-[11px] font-semibold bg-[#F1F5F9] text-[#475569] px-2 py-0.5 rounded-full border border-[#E2E8F0]"
				>{breadcrumb.chip}</span
			>
		{/if}
	</div>
</header>
