-- 0002_grants_and_guard.sql - PaluKetok
-- Perbaikan 0001 setelah uji RLS:
--  1) REVOKE ALL dari anon/authenticated membuang semua grant, termasuk yang dibutuhkan policy.
--     Grant dipasang ulang hanya untuk tabel yang memang punya policy (security.md bagian 3).
--  2) guard_profiles_update memblokir juga service role/postgres (server), padahal server harus
--     boleh mengubah credits/is_admin. Guard sekarang hanya berlaku untuk anon/authenticated.
-- Sumber: plan/security.md bagian 3 baris 52-55.

begin;

-- ---------- 1) guard: hanya klien (anon/authenticated) yang dibatasi ----------
create or replace function public.guard_profiles_update()
returns trigger language plpgsql as $$
begin
  if current_user not in ('anon', 'authenticated') then
    return new; -- server (postgres/service_role) boleh ubah credits, is_admin, dll.
  end if;
  if new.is_admin      is distinct from old.is_admin
  or new.credits       is distinct from old.credits
  or new.plan_tier     is distinct from old.plan_tier
  or new.token_version is distinct from old.token_version
  or new.id            is distinct from old.id
  or new.created_at    is distinct from old.created_at then
    raise exception 'profiles: kolom ini hanya bisa diubah lewat server';
  end if;
  return new;
end $$;

-- ---------- 2) grant sesuai policy ----------
-- profiles: baca sendiri; ubah hanya kolom yang boleh (grant kolom, bukan tabel)
grant select on public.profiles to authenticated;
grant update (username, full_name, avatar_url) on public.profiles to authenticated;

-- tabel dengan policy baca/tulis milik sendiri
grant select, insert, update, delete on public.projects, public.plan_runs,
  public.artifacts, public.decisions, public.artifact_versions to authenticated;

-- orders: baca sendiri saja (insert lewat server)
grant select on public.orders to authenticated;

-- credits_ledger, ai_settings: tidak ada grant ke anon/authenticated (sengaja)

commit;
