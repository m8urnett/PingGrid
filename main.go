package main

import (
	"context"
	"os"

	"github.com/m8urnett/PingGrid/internal/app"
	"github.com/m8urnett/toolkit/errors"
)

const version = "1.04.000"

func main() {
	if err := app.Execute(context.Background(), os.Args, version); err != nil {
		errors.Fatal(err)
	}
}
