<script lang="ts">
	import '../app.css';
	import type { Snippet } from 'svelte';
	import { initAuth } from '#lib/auth.svelte';

	let { children }: { children: Snippet } = $props();

	// Fase 10: sesi Supabase dimuat sekali untuk semua halaman.
	initAuth();
</script>

{@render children()}
