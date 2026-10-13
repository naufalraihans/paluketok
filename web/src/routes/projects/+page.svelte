<script lang="ts">
	import DashSidebar from '#lib/components/DashSidebar.svelte';
	import DashTopbar from '#lib/components/DashTopbar.svelte';
	import DashHero from '#lib/components/DashHero.svelte';
	import DashMetrics from '#lib/components/DashMetrics.svelte';
	import DashProjectList from '#lib/components/DashProjectList.svelte';
	import DashBanners from '#lib/components/DashBanners.svelte';
	import DashNotice from '#lib/components/DashNotice.svelte';
	import DashStates from '#lib/components/DashStates.svelte';
	import { dashboard } from '#lib/mock/dashboard';

	// Fase 9 langkah 34: /projects = design/web/dasrhboard.html.
	// Susunan blok mengikuti mockup: sidebar, topbar, hero, metrik, daftar
	// proyek, dua banner, banner info, preview state halaman.
</script>

<svelte:head>
	<title>{dashboard.meta.title}</title>
</svelte:head>

<!-- Kelas <body> mockup dipasang di pembungkus: <svelte:body> Svelte 5 hanya
     menerima event handler, bukan atribut class. -->
<div class="bg-[#fafbfc] text-[#0f172a] font-sans antialiased min-h-screen flex">
	<DashSidebar sidebar={dashboard.sidebar} />

	<div class="flex-1 flex flex-col min-w-0">
		<DashTopbar
			breadcrumb={dashboard.breadcrumb}
			chip={dashboard.chip}
			credits={dashboard.credits}
			topupLabel={dashboard.topupLabel}
		/>

		<main class="flex-1 p-6 lg:p-8 space-y-8 max-w-[1360px] mx-auto w-full">
			<DashHero hero={dashboard.hero} createLabel={dashboard.createLabel} />
			<DashMetrics metrics={dashboard.metrics} />
			<DashProjectList list={dashboard.list} />
			<DashBanners pipeline={dashboard.pipeline} balance={dashboard.balance} />
			<DashNotice notice={dashboard.notice} />
			<DashStates states={dashboard.states} />
		</main>
	</div>
</div>