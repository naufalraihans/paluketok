<script lang="ts">
	import type { LandingContent } from '#lib/mock/types';

	let { finalCta }: { finalCta: LandingContent['finalCta'] } = $props();
</script>

<section class="py-24 px-6 text-center bg-gradient-to-b from-white to-slate-50 border-t border-slate-100">
	<div class="max-w-3xl mx-auto">
		<h2 class="text-3xl sm:text-5xl font-extrabold text-slate-950 tracking-tight mb-4">
			{finalCta.heading}
		</h2>
		<p class="text-sm sm:text-base text-slate-600 max-w-xl mx-auto mb-10">
			{finalCta.body}
		</p>
		<div class="flex flex-col sm:flex-row items-center justify-center gap-4">
			<a
				class="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-6 py-3.5 rounded-xl font-medium text-sm text-white bg-brand-500 hover:bg-brand-600 transition-all shadow-sm"
				href={finalCta.primary.href}
			>
				<span>{finalCta.primary.label}</span>
				<span class="text-brand-200">→</span>
			</a>
			<a
				class="w-full sm:w-auto inline-flex items-center justify-center px-6 py-3.5 rounded-xl font-medium text-sm text-slate-700 bg-white border border-slate-200 hover:border-slate-300 hover:bg-slate-50 transition-all shadow-subtle"
				href="#contoh"
			>
				{finalCta.secondaryLabel}
			</a>
		</div>
	</div>
</section>