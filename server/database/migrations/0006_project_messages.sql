-- ============================================================
-- 0006: pesan chat per proyek (chat konteks proyek)
-- ============================================================
-- Alasan: pengguna butuh bertanya soal proyeknya sendiri dan mendapat balasan
-- yang TAHU brief + dokumen yang sudah jadi. Kontrak beku POST /v1/ai/chat
-- tidak punya project_id, jadi tidak bisa dipakai untuk itu.
--
-- Aturan:
--   * Hanya peran 'user' dan 'assistant'. 'system' ditolak di lapisan API,
--     dan CHECK di sini mencegahnya lolos lewat jalur lain.
--   * Panjang 1..4000 karakter, sama dengan batas kontrak lama.
--   * suggested_stage: usul tahap berikutnya dari model, salah satu dari enam
--     tahap planning atau NULL.
--   * RLS menyala + grant dicabut, pola sama dengan tabel lain di 0001_init.sql.

create table if not exists public.project_messages (
  id              uuid primary key default gen_random_uuid(),
  project_id      uuid not null references public.projects(id) on delete cascade,
  user_id         uuid not null references public.profiles(id) on delete cascade,
  role            text not null check (role in ('user', 'assistant')),
  content         text not null check (char_length(content) between 1 and 4000),
  suggested_stage text check (
                    suggested_stage is null
                    or suggested_stage in ('prd','erd','openapi','roles','security','structure')
                  ),
  model           text,
  tokens_in       int not null default 0,
  tokens_out      int not null default 0,
  created_at      timestamptz not null default now()
);

comment on table public.project_messages is
  'Percakapan per proyek: pengguna bertanya, asisten menjawab dengan konteks brief + dokumen.';

-- Riwayat dibaca per proyek, urut waktu. Index menutup pola itu.
create index if not exists idx_project_messages_project
  on public.project_messages (project_id, created_at);

alter table public.project_messages enable row level security;
revoke all on public.project_messages from anon, authenticated;

drop policy if exists "project_messages_all_own" on public.project_messages;
create policy "project_messages_all_own" on public.project_messages
  for all using (user_id = auth.uid()) with check (user_id = auth.uid());
