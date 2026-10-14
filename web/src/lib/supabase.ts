import { createClient } from '@supabase/supabase-js';

/**
 * Klien Supabase Auth (Fase 10, workflow.md 36).
 * URL + anon key dari env publik Vite; anon key bukan rahasia (diproteksi RLS
 * di sisi Supabase, bukan di sini).
 */
export const supabase = createClient(
	import.meta.env.VITE_SUPABASE_URL as string,
	import.meta.env.VITE_SUPABASE_ANON_KEY as string,
	{ auth: { persistSession: true, autoRefreshToken: true } }
);
