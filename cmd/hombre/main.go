package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"bits.chrsm.org/hombre"
)

func main() {
	c, err := loadConfig("config.json")
	if err != nil {
		log.Println("Could not load config.json properly:", err)
		return
	}

	bot := hombre.New(c.Token, hombre.OptionLuaPath(c.Lua.Path))
	for _, v := range c.Lua.Scripts {
		bot.AddScript(hombre.Script{
			Name:     v.Name,
			Commands: v.Commands,
		})
	}

	for _, v := range c.Lua.Services {
		bot.AddService(hombre.Script{
			Name:     v.Name,
			Commands: v.Commands,
		})
	}

	log.Printf("starting hombre..")
	go bot.Listen()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	log.Printf("waiting for ctrl-c to quit :-)")
	<-quit
	bot.Close()
	log.Println("See ya!")
}
