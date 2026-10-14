<script lang="ts">
	import SiteNav from '#lib/components/SiteNav.svelte';
	import SiteFooter from '#lib/components/SiteFooter.svelte';
	import { landing } from '#lib/mock/landing';
</script>

<svelte:head><title>404 · PaluKetok</title></svelte:head>
<div class="min-h-screen flex flex-col bg-white text-[#0f172a] font-sans antialiased">
	<SiteNav nav={landing.nav} />
	<main class="flex-1 max-w-4xl w-full mx-auto px-6 py-24 text-center space-y-4">
		<p class="text-6xl font-bold tracking-tight">404</p>
		<h1 class="text-lg font-semibold">Halaman tidak ditemukan</h1>
		<p class="text-sm text-[#475569] max-w-md mx-auto">
			Alamat yang dibuka tidak ada atau sudah dipindah. Periksa kembali tautan.
		</p>
		<a href="/"
			class="inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm">Kembali ke beranda</a
		>
	</main>
	<SiteFooter footer={landing.footer} />
</div>