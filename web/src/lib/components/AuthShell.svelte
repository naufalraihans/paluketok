<script lang="ts">
	import SiteNav from '#lib/components/SiteNav.svelte';
	import SiteFooter from '#lib/components/SiteFooter.svelte';
	import { landing } from '#lib/mock/landing';

	// Shell halaman auth (/login, /register, /reset): turunan template beku.
	// Kelas kartu, tombol, dan input disalin dari pola mockup (dashboard kartu,
	// SiteNav tombol utama). Tanpa gaya baru.
	let { title, children }: { title: string; children: import('svelte').Snippet } = $props();
</script>

<svelte:head>
	<title>{title} · PaluKetok</title>
</svelte:head>
<div class="min-h-screen flex flex-col bg-white text-[#0f172a] font-sans antialiased">
	<SiteNav nav={landing.nav} />
	<main class="flex-1 grid place-items-center px-6 py-16">
		<div class="w-full max-w-sm rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
			{@render children()}
		</div>
	</main>
	<SiteFooter footer={landing.footer} />
</div>
