<script lang="ts">
	import SummarySidebar from '#lib/components/SummarySidebar.svelte';
	import SummaryTopbar from '#lib/components/SummaryTopbar.svelte';
	import SummaryHeader from '#lib/components/SummaryHeader.svelte';
	import SummaryMetrics from '#lib/components/SummaryMetrics.svelte';
	import SummaryThread from '#lib/components/SummaryThread.svelte';
	import SummaryBottom from '#lib/components/SummaryBottom.svelte';
	import { summary } from '#lib/mock/summary';
	import type { PageProps } from './$types';

	// Fase 9 langkah 34: /projects/[id] = design/web/ringkasanProyek.html.
	// App shell h-screen overflow-hidden; scroll di dalam main (mockup body:24).
	let { params }: PageProps = $props();
</script>

<svelte:head>
	<title>{summary.meta.title}</title>
</svelte:head>

<div class={summary.meta.bodyClass}>
	<SummarySidebar />

	<div class="flex-1 flex flex-col h-full overflow-hidden bg-[#F8F9FA]" data-purpose="main-layout">
		<SummaryTopbar
			breadcrumb={summary.breadcrumb}
			savedAt={summary.savedAt}
			collaborators={summary.collaborators}
			shareLabel={summary.shareLabel}
		/>

		<main class="flex-1 overflow-y-auto px-8 py-6 space-y-6" data-purpose="dashboard-scrollable-content">
			<SummaryHeader
				overview={summary.overview}
				pageHeader={summary.pageHeader}
				exportLabel={summary.exportLabel}
			/>
			<SummaryMetrics metrics={summary.metrics} />
			<SummaryThread thread={summary.thread} security={summary.security} />
			<SummaryBottom
				review={summary.review}
				footerLeft={summary.footerLeft}
				footerRight={summary.footerRight}
			/>
		</main>
	</div>
</div>
