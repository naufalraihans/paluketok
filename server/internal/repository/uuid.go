package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// beginner adalah pool: bisa membuka transaksi.
type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// isUnique true bila err adalah pelanggaran unique constraint (23505).
func isUnique(err error) bool {
	var pgerr *pgconn.PgError
	return errors.As(err, &pgerr) && pgerr.Code == "23505"
}

// isCheck true bila err adalah pelanggaran CHECK constraint (23514).
func isCheck(err error) bool {
	var pgerr *pgconn.PgError
	return errors.As(err, &pgerr) && pgerr.Code == "23514"
}

// newUUID membuat UUID v4-shaped tanpa menarik google/uuid. Untuk id baris
// baru di ledger (yang id-nya tidak dipakai logika bisnis).
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	dst := make([]byte, 36)
	hex.Encode(dst, b[:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], b[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], b[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], b[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], b[10:16])
	return string(dst)
}
