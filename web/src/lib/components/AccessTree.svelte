<script lang="ts">
	import Icon from '#lib/components/Icon.svelte';
	import type { AccessGroup } from '#lib/mock/types';

	/**
	 * Peta halaman: satu kolom per area akses.
	 * `variant="window"` dipakai di jendela hero, `variant="diagram"` di kartu
	 * "Struktur & prototipe" -- keduanya punya ukuran dan kelas berbeda di mockup,
	 * jadi perbedaannya dinyatakan di sini, bukan disatukan.
	 */
	type Props = {
		groups: AccessGroup[];
		variant: 'window' | 'diagram';
	};

	let { groups, variant }: Props = $props();
</script>

{#if variant === 'window'}
	<div class="w-full grid grid-cols-3 gap-3 relative">
		{#each groups as g (g.title)}
			<div class="bg-white border border-slate-200 rounded-xl p-3 shadow-2xs">
				<div class="text-2xs font-bold text-slate-800 mb-0.5">{g.title}</div>
				<div class="text-4xs text-slate-400 font-mono mb-2">{g.meta}</div>
				<div class="space-y-1.5">
					{#each g.nodes as node, i (node)}
						{#if i === g.highlightIndex}
							<div
								class="p-1.5 rounded bg-blue-50/70 border border-blue-200/80 text-3xs font-medium text-brand-700"
							>
								{node}
							</div>
						{:else}
							<div class="p-1.5 rounded bg-slate-50 border border-slate-100 text-3xs text-slate-600">
								{node}
							</div>
						{/if}
					{/each}
				</div>
			</div>
		{/each}
	</div>
{:else}
	<div class="w-full grid grid-cols-3 gap-3">
		{#each groups as g (g.title)}
			<div class="p-3 bg-white rounded-xl border border-slate-200">
				<div class="text-2xs font-bold text-slate-900">{g.title}</div>
				<div class="text-4xs text-slate-400 font-mono mb-2">{g.meta}</div>
				<div class="space-y-1 text-3xs">
					{#each g.nodes as node, i (node)}
						{#if i === g.highlightIndex}
							<div
								class="p-1 bg-blue-50 text-brand-700 font-semibold rounded border border-blue-200"
							>
								{node}
							</div>
						{:else}
							<div class="p-1 bg-slate-50 rounded text-slate-600">{node}</div>
						{/if}
					{/each}
				</div>
			</div>
		{/each}
	</div>
{/if}
