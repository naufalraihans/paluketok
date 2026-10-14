import { supabase } from '#lib/supabase';
import type { Me } from '#lib/api';
import { getMe } from '#lib/api';

/**
 * State auth+profil (Fase 10). Svelte 5 runes; dipakai navbar/topbar (kredit)
 * dan guard halaman. Auth email+password via Supabase (auth_billing.md 1.3):
 * email konfirmasi aktif, sesi baru ada setelah konfirmasi.
 */
export const auth = $state<{ user: null | { email?: string }; me: Me | null; loading: boolean }>({
	user: null,
	me: null,
	loading: true
});

/** Dipanggil sekali di +layout.svelte. */
export function initAuth(): void {
	supabase.auth.getSession().then(async ({ data }) => {
		auth.user = data.session?.user ?? null;
		auth.loading = false;
		if (auth.user) await refreshMe();
	});
	supabase.auth.onAuthStateChange((_event, session) => {
		auth.user = session?.user ?? null;
		if (!auth.user) auth.me = null;
		else refreshMe();
	});
}

export async function refreshMe(): Promise<void> {
	try {
		auth.me = await getMe();
	} catch {
		auth.me = null;
	}
}

/** Registrasi email+password. Email konfirmasi dikirim Supabase. */
export async function register(email: string, password: string): Promise<void> {
	const { error } = await supabase.auth.signUp({
		email,
		password,
		options: { emailRedirectTo: `${location.origin}/auth/callback` }
	});
	if (error) throw error;
}

/** Login email+password. */
export async function login(email: string, password: string): Promise<void> {
	const { error } = await supabase.auth.signInWithPassword({ email, password });
	if (error) throw error;
}

/** Kirim email lupa password; link kembali ke /reset (auth_billing.md 1.3). */
export async function requestReset(email: string): Promise<void> {
	const { error } = await supabase.auth.resetPasswordForEmail(email, {
		redirectTo: `${location.origin}/reset`
	});
	if (error) throw error;
}

/** Simpan password baru dari sesi reset (link reset membawa sesi). */
export async function updatePassword(password: string): Promise<void> {
	const { error } = await supabase.auth.updateUser({ password });
	if (error) throw error;
}

/** Masuk tanpa password: magic link (jalur alternatif). */
export async function loginMagic(email: string): Promise<void> {
	const { error } = await supabase.auth.signInWithOtp({
		email,
		options: { emailRedirectTo: `${location.origin}/auth/callback` }
	});
	if (error) throw error;
}

export async function logout(): Promise<void> {
	await supabase.auth.signOut();
}
