package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/chrsm/hombre"
)

func main() {
	c, err := hombre.LoadConfig("config.json")
	if err != nil {
		fmt.Println("Could not load config.json properly:", err)
		return
	}

	bot := hombre.New(c)
	go bot.Listen()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	bot.Close()
	fmt.Println("See ya!")
}
