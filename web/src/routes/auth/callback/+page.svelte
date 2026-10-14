<script lang="ts">
	import { goto } from '$app/navigation';
	import { supabase } from '#lib/supabase';

	// Tukar code PKCE jadi sesi (magic link redirect), lalu masuk aplikasi.
	$effect(() => {
		const url = new URL(location.href);
		const code = url.searchParams.get('code');
		if (code) {
			supabase.auth.exchangeCodeForSession(code).then(() => goto('/projects', { replaceState: true }));
		} else {
			goto('/projects', { replaceState: true });
		}
	});
</script>

<div class="min-h-screen grid place-items-center bg-[#fafbfc] text-[#0f172a]">
	<p class="text-sm text-[#475569]">Memeriksa tautan masuk…</p>
</div>
