<script lang="ts">
	import type { LandingContent } from '#lib/mock/types';

	let { problem }: { problem: LandingContent['problem'] } = $props();
</script>

<section class="py-24 px-6 border-t border-slate-100 bg-white" id="cara-kerja">
	<div class="max-w-3xl mx-auto text-center">
		<!-- Section Eyebrow Badge: di mockup elemen ini kosong, dipertahankan apa adanya -->
		<span class="inline-block text-2xs font-bold tracking-widest text-brand-600 uppercase mb-4"
		></span>
		<!-- Main Statement Heading -->
		<h2
			class="text-3xl sm:text-5xl font-extrabold text-slate-950 tracking-tight mb-6"
		>
			{problem.headingLine1}<br />
			{problem.headingLine2}
		</h2>
		<!-- Context Description -->
		<p class="text-base sm:text-lg text-slate-600 max-w-2xl mx-auto mb-10">
			{problem.body}
		</p>
		<!-- Connecting Dialogue Snippet -->
		<div
			class="inline-flex flex-wrap items-center justify-center gap-2 px-5 py-3 rounded-xl bg-slate-50 border border-slate-200/80 shadow-2xs text-xs sm:text-sm"
		>
			<span class="text-slate-500 font-medium">{problem.quote1}</span>
			<span class="text-slate-300">→</span>
			<span class="text-brand-600 font-semibold">{problem.quote2}</span>
		</div>
	</div>
</section>
