// Command migrate applies or rolls back database migrations.
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate down
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Girolamone/kiosk/apps/api/internal/config"
	"github.com/Girolamone/kiosk/apps/api/internal/db"
)

func main() {
	flag.Parse()

	direction := flag.Arg(0)
	if direction == "" {
		direction = "up"
	}

	config.LoadDotEnv()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		exit("DATABASE_URL is not set")
	}

	var err error
	switch direction {
	case "up":
		err = db.Up(databaseURL)
	case "down":
		err = db.Down(databaseURL)
	default:
		exit(fmt.Sprintf("unknown direction %q: want \"up\" or \"down\"", direction))
	}
	if err != nil {
		exit(err.Error())
	}

	fmt.Printf("migrations %s: ok\n", direction)
}

func exit(msg string) {
	fmt.Fprintln(os.Stderr, "migrate:", msg)
	os.Exit(1)
}
