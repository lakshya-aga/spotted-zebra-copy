package main

import (
	"database/sql"
	"log"

	"github.com/banachtech/spotted-zebra/api"
	db "github.com/banachtech/spotted-zebra/db/sqlc"
	"github.com/banachtech/spotted-zebra/util"
	_ "github.com/lib/pq"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load configuration:", err)
	}
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	store := db.NewStore(conn)
	server := api.NewServer(store)
	err = server.Start(config.ServerAddress)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}
