package main

import (
	"fmt"
	"github.com/mdesson/chatcord/bot"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	b, err := bot.New(slog.LevelDebug)
	if err != nil {
		panic(err)
	}
	if err := b.Start(); err != nil {
		panic(err)
	}
	defer b.Stop()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	select {
	case <-sc:
		fmt.Println("\nExiting...")
	}
}
