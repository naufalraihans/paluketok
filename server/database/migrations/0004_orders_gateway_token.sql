-- 0004: kolom jejak gateway pada orders.
--
-- Midtrans memakai invoice_number kita sebagai "order_id" transaksi, jadi tidak
-- ada id transaksi tambahan yang perlu disimpan untuk verifikasi webhook.
-- gateway_token (Snap token) disimpan hanya sebagai jejak audit supaya transaksi
-- bisa dicari di dashboard Midtrans.
--
-- Idempoten: aman dijalankan ulang.

alter table public.orders
  add column if not exists gateway_token text;

comment on column public.orders.gateway_token is
  'Snap token Midtrans (jejak audit). Bukan kunci transaksi; order_id = invoice_number.';

-- Sisa kolom dari percobaan gateway sebelumnya: dibuang bila ada supaya skema
-- tidak menyimpan jejak yang menyesatkan.
alter table public.orders
  drop column if exists mayar_transaction_id;
