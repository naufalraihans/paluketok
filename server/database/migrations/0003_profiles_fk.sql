-- 0003_profiles_fk.sql - PaluKetok
-- erd.dbml:172 "Ref: profiles.id - auth.users.id [delete: cascade]" belum terpasang di 0001.
-- Ditemukan saat uji cleanup: profiles yatim tertinggal setelah auth.users dihapus.

begin;

-- bersihkan yatim dulu supaya FK bisa dipasang (hanya data uji; sandbox kosong)
delete from public.profiles p
where not exists (select 1 from auth.users u where u.id = p.id);

do $$
begin
  if not exists (
    select 1 from pg_constraint
    where conname = 'profiles_id_fkey' and conrelid = 'public.profiles'::regclass
  ) then
    alter table public.profiles
      add constraint profiles_id_fkey
      foreign key (id) references auth.users(id) on delete cascade;
  end if;
end $$;

commit;
