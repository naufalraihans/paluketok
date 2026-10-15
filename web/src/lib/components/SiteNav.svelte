<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { LandingContent } from '#lib/mock/types';
	import { auth, logout } from '#lib/auth.svelte';

	let { nav }: { nav: LandingContent['nav'] } = $props();

	// Fase 10 revisi: /login dan /register halaman nyata. Sesi tampil di nav:
	// username + kredit (GET /v1/me), link admin hanya kalau is_admin.
	// Admin ada di subdomain terpisah; di subdomain itu sendiri pakai path relatif.
	const adminUrl = ((): string =>
		typeof location !== 'undefined' && location.hostname.startsWith('admin.')
			? '/admin'
			: 'https://admin.paluketok.web.id/admin')();
</script>

<header
	class="sticky top-0 z-50 bg-white/90 backdrop-blur-md border-b border-slate-100 transition-all"
>
	<div class="max-w-6xl mx-auto px-6 h-20 flex items-center justify-between">
		<!-- Logo Container -->
		<a
			class="flex items-center gap-2.5 font-bold text-slate-900 tracking-tight text-lg"
			href="/"
		>
			<div
				class="w-8 h-8 rounded-lg bg-slate-900 flex items-center justify-center text-white shadow-sm"
			>
				<Icon name="logo" class="w-4 h-4" width={2.5} />
			</div>
			<span>PaluKetok</span>
		</a>
		<!-- Center Navigation Links -->
		<nav class="hidden md:flex items-center gap-8 text-sm font-medium text-slate-600">
			{#each nav.links as link (link.href)}
				<a class="hover:text-slate-900 transition-colors" href={link.href}>{link.label}</a>
			{/each}
			{#if auth.me?.is_admin}
				<a class="hover:text-slate-900 transition-colors" href={adminUrl}>Admin</a>
			{/if}
		</nav>
		<!-- Right Action CTAs -->
		<div class="flex items-center gap-5">
			{#if auth.user}
				<a
					class="text-sm font-medium text-slate-600 hover:text-slate-900 transition-colors"
					href="/billing"
				>
					{auth.me?.username ?? auth.user.email}
					{#if auth.me}<span class="text-slate-400">· {auth.me.credits} kredit</span>{/if}
				</a>
				<button
					type="button"
					class="text-sm font-medium text-slate-600 hover:text-slate-900 transition-colors"
					onclick={logout}>Keluar</button
				>
				<a
					class="inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm"
					href="/projects">Dashboard</a
				>
			{:else}
				<a
					class="text-sm font-medium text-slate-600 hover:text-slate-900 transition-colors"
					href="/login">Masuk</a
				>
				<a
					class="inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm"
					href="/register">Daftar</a
				>
			{/if}
		</div>
	</div>
</header>
