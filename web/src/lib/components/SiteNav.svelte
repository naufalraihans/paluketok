<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { LandingContent } from '#lib/mock/types';

	let { nav }: { nav: LandingContent['nav'] } = $props();
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
		</nav>
		<!-- Right Action CTAs -->
		<div class="flex items-center gap-5">
			<a
				class="text-sm font-medium text-slate-600 hover:text-slate-900 transition-colors"
				href={nav.signIn.href}>{nav.signIn.label}</a
			>
			<a
				class="inline-flex items-center justify-center px-4 py-2 rounded-lg text-sm font-medium bg-slate-950 text-white hover:bg-slate-800 transition-all shadow-sm"
				href={nav.primaryCta.href}
			>
				{nav.primaryCta.label}
			</a>
		</div>
	</div>
</header>
