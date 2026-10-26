package main

import (
	log "github.com/sirupsen/logrus"

	"api/internal/controller"
	"api/internal/repository"
	"api/internal/router"
	"api/internal/service/command"
	"api/internal/service/query"
)

func main() {
	store := repository.NewMemoryStore()

	commands := command.NewHandler(store)
	queries := query.NewHandler(store)

	wallet := controller.NewWalletController(commands, queries)
	r := router.New(wallet)

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
