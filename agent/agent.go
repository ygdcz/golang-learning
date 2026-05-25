package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	appagent "github.com/ygdcz/golang-learning/agent/agent"
	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
)

func main() {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("load .env failed: %v", err)
	}

	rootAgent, err := appagent.NewRootAgent(ctx, os.Getenv("GOOGLE_API_KEY"), os.Getenv)
	if err != nil {
		log.Fatalf("create root agent failed: %v", err)
	}

	// run agent
	config := &launcher.Config{
		AgentLoader: adkagent.NewSingleLoader(rootAgent),
	}
	l := full.NewLauncher()
	if err = l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run agent failed: %v, cmdline: %v", err, l.CommandLineSyntax())
	}
}
