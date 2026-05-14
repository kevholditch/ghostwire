package main

import (
	"context"
	"os"

	"github.com/kevholditch/ghostwire/internal/operatorcli"
)

func main() {
	os.Exit(operatorcli.Run(context.Background(), os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}
