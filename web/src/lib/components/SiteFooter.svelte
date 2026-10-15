<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { LandingContent } from '#lib/mock/types';

	let { footer }: { footer: LandingContent['footer'] } = $props();
</script>

<footer class="bg-white border-t border-slate-100 py-16 px-6 text-xs text-slate-500">
	<div class="max-w-6xl mx-auto">
		<div class="grid grid-cols-1 md:grid-cols-12 gap-10 pb-12 border-b border-slate-100">
			<!-- Brand Info -->
			<div class="md:col-span-5 space-y-3">
				<a
					class="flex items-center gap-2 font-bold text-slate-900 tracking-tight text-base"
					href="/"
				>
					<div class="w-7 h-7 rounded-lg bg-slate-900 flex items-center justify-center text-white">
						<Icon name="logo" class="w-3.5 h-3.5" width={2.5} />
					</div>
					<span>PaluKetok</span>
				</a>
				<p class="text-slate-500 max-w-sm leading-relaxed">{footer.tagline}</p>
			</div>
			<!-- Links Columns -->
			<div class="md:col-span-7 grid grid-cols-3 gap-6">
				{#each footer.columns as col (col.title)}
					<div>
						<div class="font-bold text-slate-900 mb-3">{col.title}</div>
						<ul class="space-y-2">
							{#each col.links as link (link.label)}
								<li>
									<a class="hover:text-slate-900 transition-colors" href={link.href}
										>{link.label}</a
									>
								</li>
							{/each}
						</ul>
					</div>
				{/each}
			</div>
		</div>
		<!-- Bottom Bar -->
		<div
			class="pt-8 flex flex-col sm:flex-row items-center justify-between gap-4 text-slate-400"
		>
			<div>{footer.copyright}</div>
			<div class="flex items-center gap-4">
				{#each footer.legal as link, i (link.label)}
					{#if i > 0}<span>·</span>{/if}
					<!-- Halaman Privacy/Terms belum ada; label, bukan link mati. -->
					<span>{link.label}</span>
				{/each}
			</div>
		</div>
	</div>
</footer>