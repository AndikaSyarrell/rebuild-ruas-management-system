// cmd/hashpw adalah utilitas kecil untuk membuat bcrypt hash secara offline,
// dipakai saat men-seed admin pertama langsung lewat SQL (lihat seed.sql).
//
// Pemakaian:
//
//	go run ./cmd/hashpw "PasswordSaya123!"
package main

import (
	"fmt"
	"os"

	"rms-backend/internal/utils"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run ./cmd/hashpw <password>")
		os.Exit(1)
	}
	hash, err := utils.HashPassword(os.Args[1])
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	fmt.Println(hash)
}
