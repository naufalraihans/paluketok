<script lang="ts">
	import type { SummaryContent } from '#lib/mock/summary-types';

	// Aktivitas dokumen = versi artefak yang benar-benar ada (nama + waktu dari
	// server). Daftar tugas dan feed aktivitas karangan dari mockup DIBUANG:
	// tidak ada fitur tugas/kolaborator, jadi isinya tidak bisa diverifikasi
	// siapa pun.
	let { review, footerLeft, footerRight }: Pick<SummaryContent, 'review' | 'footerLeft' | 'footerRight'> =
		$props();
</script>

<section class="grid grid-cols-12 gap-5" data-purpose="actions-and-activity-logs">
	<div class="col-span-12" data-purpose="recent-activity-feed">
		<div class="flex items-center justify-between mb-3">
			<h3 class="font-bold text-sm text-[#0F172A]">{review.title}</h3>
			{#if review.counter}
				<span class="text-xs text-[#64748B]">{review.counter}</span>
			{/if}
		</div>
		<div class="bg-white border border-[#E2E8F0] rounded-2xl p-4 shadow-xs">
			{#if review.activities.length > 0}
				<div class="grid grid-cols-2 md:grid-cols-3 gap-x-6 gap-y-2.5 text-xs text-[#64748B]">
					{#each review.activities as a (a.time + a.text)}
						<p class="truncate">
							<span class="font-medium text-[#0F172A]">{a.time}</span> · {a.text}
						</p>
					{/each}
				</div>
			{:else}
				<p class="text-xs text-[#94A3B8]">
					Belum ada dokumen. Aktivitas muncul setelah generate pertama selesai.
				</p>
			{/if}
		</div>
	</div>
</section>

{#if footerLeft || footerRight}
	<footer
		class="pt-6 pb-2 border-t border-[#E2E8F0] flex items-center justify-between text-[11px] text-[#94A3B8] font-medium"
		data-purpose="canvas-footer-metadata"
	>
		<span>{footerLeft}</span>
		<span>{footerRight}</span>
	</footer>
{/if}
