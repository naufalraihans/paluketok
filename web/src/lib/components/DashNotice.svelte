<script lang="ts">
	import type { DashboardContent } from '#lib/mock/dashboard-types';

	let { notice }: { notice: string } = $props();
</script>

<section
	class="bg-blue-50/60 border border-blue-100 rounded-xl px-4 py-3 flex items-start sm:items-center gap-3 text-xs text-blue-700"
	data-purpose="notice-banner"
>
	<div
		class="w-4 h-4 rounded-full bg-blue-600 text-white flex items-center justify-center flex-shrink-0 text-3xs font-bold mt-0.5 sm:mt-0"
	>
		i
	</div>
	<p class="leading-normal">{notice}</p>
</section>