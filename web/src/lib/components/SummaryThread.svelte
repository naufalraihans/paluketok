<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { SummaryContent } from '#lib/mock/summary-types';

	// Benang merah = dokumen terbaru per tahap (0–6 simpul, bukan selalu 4
	// seperti mockup), dan proteksi = seksi dokumen security kalau ada.
	let { thread, security }: Pick<SummaryContent, 'thread' | 'security'> = $props();
</script>

<section class="grid grid-cols-12 gap-5" data-purpose="traceability-and-protection-section">
	<div
		class="col-span-8 bg-white border border-[#E2E8F0] rounded-2xl p-5 shadow-xs flex flex-col justify-between"
		data-purpose="design-thread-card"
	>
		<div>
			<div class="flex items-center justify-between">
				<h3 class="font-bold text-sm text-[#0F172A]">{thread.title}</h3>
				{#if thread.link}
					<span class="text-xs font-semibold text-[#2563EB] inline-flex items-center gap-1">
						{thread.link}
					</span>
				{/if}
			</div>
			<p class="text-xs text-[#64748B] mt-0.5">{thread.subtitle}</p>

			{#if thread.nodes.length > 0}
				<div
					class="grid grid-cols-2 md:grid-cols-{Math.min(thread.nodes.length, 3)} gap-3 mt-6"
					data-purpose="traceability-flow"
				>
					{#each thread.nodes as node (node.tag)}
						<a
							class="border border-[#E2E8F0] rounded-xl p-3 bg-white hover:border-slate-300 transition-colors block"
							href="#tahap-{node.tag.split(' · ')[0].toLowerCase()}"
						>
							<div
								class="w-7 h-7 rounded-lg bg-[#F8FAFC] border border-[#E2E8F0] flex items-center justify-center text-[#64748B] mb-2.5"
							>
								<Icon name={node.icon} class="w-3.5 h-3.5" />
							</div>
							<span class="text-[10px] font-bold text-[#94A3B8] tracking-wider uppercase block"
								>{node.tag}</span
							>
							<p class="text-xs font-bold text-[#0F172A] mt-0.5 truncate {node.mono ? 'font-mono' : ''}"
								>{node.title}</p
							>
							<p
								class="text-[11px] text-[#64748B] mt-0.5 truncate {node.mono
									? 'font-mono text-[10px]'
									: ''}"
							>
								{node.note}
							</p>
						</a>
					{/each}
				</div>
			{:else}
				<p
					class="text-xs text-[#94A3B8] mt-6 border border-dashed border-[#E2E8F0] rounded-xl p-4"
					data-purpose="thread-empty"
				>
					Belum ada dokumen. Jalankan generate untuk menyusun enam tahap planning.
				</p>
			{/if}
		</div>

		{#if thread.connected || thread.pending}
			<div class="flex items-center justify-between pt-5 mt-5 border-t border-[#F1F5F9]">
				{#if thread.connected}
					<span
						class="inline-flex items-center px-2.5 py-1 rounded-full text-[11px] font-semibold bg-[#ECFDF5] text-[#059669] border border-[#A7F3D0]"
					>
						{thread.connected}
					</span>
				{/if}
				{#if thread.pending}
					<span class="text-[11px] text-[#64748B]">{thread.pending}</span>
				{/if}
			</div>
		{/if}
	</div>

	<div
		class="col-span-4 bg-white border border-[#E2E8F0] rounded-2xl p-5 shadow-xs flex flex-col justify-between"
		data-purpose="security-controls-card"
	>
		<div>
			<div class="flex items-center justify-between">
				<h3 class="font-bold text-sm text-[#0F172A]">{security.title}</h3>
				<Icon name="shield" class="w-4 h-4 text-[#94A3B8]" />
			</div>
			<p class="text-xs text-[#64748B] mt-1 leading-relaxed">
				{security.body}
			</p>

			{#if security.items.length > 0}
				<div class="space-y-3 mt-4" data-purpose="security-status-items">
					{#each security.items as item (item.label)}
						<div class="flex items-center justify-between text-xs gap-3">
							<div class="flex items-center gap-2 min-w-0">
								<span
									class="w-4 h-4 shrink-0 rounded-full border border-[#10B981] flex items-center justify-center text-[#10B981]"
								>
									<Icon name="check" class="w-2.5 h-2.5" width={3} />
								</span>
								<span class="text-[#0F172A] font-medium truncate">{item.label}</span>
							</div>
							<span class="text-[11px] font-medium text-[#10B981] shrink-0">{item.status}</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</div>
</section>
