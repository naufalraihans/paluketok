<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { SummaryContent } from '#lib/mock/summary-types';

	let { thread, security }: Pick<SummaryContent, 'thread' | 'security'> = $props();
</script>

<section
	class="grid grid-cols-12 gap-5"
	data-purpose="traceability-and-protection-section"
>
	<div
		class="col-span-8 bg-white border border-[#E2E8F0] rounded-2xl p-5 shadow-xs flex flex-col justify-between"
		data-purpose="design-thread-card"
	>
		<div>
			<div class="flex items-center justify-between">
				<h3 class="font-bold text-sm text-[#0F172A]">{thread.title}</h3>
				<!-- Halaman keterkaitan belum ada; label, bukan link mati. -->
				<span class="text-xs font-semibold text-[#2563EB] inline-flex items-center gap-1">
					{thread.link}
				</span>
			</div>
			<p class="text-xs text-[#64748B] mt-0.5">{thread.subtitle}</p>

			<div class="grid grid-cols-4 gap-3 mt-6 relative" data-purpose="traceability-flow">
				{#each thread.nodes as node (node.tag)}
					<div class="border border-[#E2E8F0] rounded-xl p-3 bg-white hover:border-slate-300 transition-colors">
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
							class="text-[11px] text-[#64748B] mt-0.5 truncate {node.mono ? 'font-mono text-[10px]' : ''}"
						>
							{node.note}
						</p>
					</div>
				{/each}
			</div>
		</div>

		<div class="flex items-center justify-between pt-5 mt-5 border-t border-[#F1F5F9]">
			<span
				class="inline-flex items-center px-2.5 py-1 rounded-full text-[11px] font-semibold bg-[#ECFDF5] text-[#059669] border border-[#A7F3D0]"
			>
				{thread.connected}
			</span>
			<span class="text-[11px] text-[#64748B]">{thread.pending}</span>
		</div>
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

			<div class="space-y-3 mt-4" data-purpose="security-status-items">
				{#each security.items as item (item.label)}
					<div class="flex items-center justify-between text-xs">
						<div class="flex items-center gap-2">
							{#if item.ok}
								<span
									class="w-4 h-4 rounded-full border border-[#10B981] flex items-center justify-center text-[#10B981]"
								>
									<Icon name="check" class="w-2.5 h-2.5" width={3} />
								</span>
							{:else}
								<span
									class="w-4 h-4 rounded-full border border-dashed border-[#EF4444] flex items-center justify-center text-[#EF4444]"
								>
									<span class="w-1.5 h-1.5 rounded-full bg-[#EF4444]"></span>
								</span>
							{/if}
							<span class="text-[#0F172A] font-medium">{item.label}</span>
						</div>
						<span class="text-[11px] font-medium {item.ok ? 'text-[#10B981]' : 'text-[#EF4444]'}"
							>{item.status}</span
						>
					</div>
				{/each}
			</div>
		</div>

		<div class="pt-4 border-t border-[#F1F5F9] mt-4">
			<!-- Halaman proteksi per-tahap belum ada; tampil sebagai label, bukan link mati. -->
			<span class="text-xs font-semibold text-[#2563EB] inline-flex items-center gap-1">
				{security.link}
			</span>
		</div>
	</div>
</section>
