<script lang="ts">
	import type { SummaryContent } from '#lib/mock/summary-types';

	let { review, footerLeft, footerRight }: Pick<SummaryContent, 'review' | 'footerLeft' | 'footerRight'> =
		$props();
</script>

<section class="grid grid-cols-12 gap-5" data-purpose="actions-and-activity-logs">
	<div class="col-span-8" data-purpose="pending-review-tasks">
		<div class="flex items-center justify-between mb-3">
			<h3 class="font-bold text-sm text-[#0F172A]">{review.title}</h3>
			<span class="text-xs text-[#64748B]">{review.counter}</span>
		</div>
		<div class="space-y-2">
			{#each review.tasks as task (task.label)}
				<div
					class="bg-white border border-[#E2E8F0] rounded-xl p-3 shadow-xs flex items-center justify-between hover:border-slate-300 transition-colors"
				>
					<div class="flex items-center gap-3">
						<div class="w-4 h-4 rounded-full border-2 border-[#94A3B8] shrink-0"></div>
						<span class="text-xs font-medium text-[#0F172A]">{task.label}</span>
					</div>
					<div class="flex items-center gap-4 text-[11px] text-[#64748B]">
						<span>{task.tag}</span>
						<span class="font-medium text-[#0F172A]">{task.assignee}</span>
					</div>
				</div>
			{/each}
		</div>
	</div>

	<div class="col-span-4" data-purpose="recent-activity-feed">
		<h3 class="font-bold text-sm text-[#0F172A] mb-3">{review.feedTitle}</h3>
		<div class="bg-white border border-[#E2E8F0] rounded-2xl p-4 shadow-xs flex flex-col justify-between h-[156px]">
			<div class="space-y-2.5 text-xs text-[#64748B]">
				{#each review.activities as a (a.time + a.text)}
					<p class="truncate"><span class="font-medium text-[#0F172A]">{a.time}</span> · {a.text}</p>
				{/each}
			</div>
			<div>
				<!-- Halaman aktivitas belum ada; label, bukan link mati. -->
				<span class="text-xs font-semibold text-[#2563EB] inline-flex items-center gap-1">
					{review.feedLink}
				</span>
			</div>
		</div>
	</div>
</section>

<footer
	class="pt-6 pb-2 border-t border-[#E2E8F0] flex items-center justify-between text-[11px] text-[#94A3B8] font-medium"
	data-purpose="canvas-footer-metadata"
>
	<span>{footerLeft}</span>
	<span>{footerRight}</span>
</footer>
