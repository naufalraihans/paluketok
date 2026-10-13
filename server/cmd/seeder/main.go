// Seeder bootstrap admin (workflow.md 26): set profiles.is_admin=true untuk
// satu email. Aman dipanggil ulang (update idempoten).
//
//	go run ./cmd/seeder admin@example.com
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"

	"paluketok/server/pkg/config"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "pemakaian: seeder <email>")
		os.Exit(2)
	}
	email := os.Args[1]
	cfg, err := config.Load(".env")
	if err != nil {
		fatal(err)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal(err)
	}
	defer conn.Close(ctx)

	tag, err := conn.Exec(ctx,
		`update profiles set is_admin = true where user_id = (select id from auth.users where email = $1)`, email)
	if err != nil {
		fatal(err)
	}
	if tag.RowsAffected() == 0 {
		fmt.Fprintf(os.Stderr, "user %s tidak ditemukan\n", email)
		os.Exit(1)
	}
	fmt.Printf("is_admin=true untuk %s (%d baris)\n", email, tag.RowsAffected())
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "seeder:", err)
	os.Exit(1)
}
