package main

import (
	"log"

	"github.com/mxschmitt/playwright-go"
)

func main() {
	if err := playwright.Install(&playwright.RunOptions{
		DriverDirectory:     "/opt/playwright-driver",
		SkipInstallBrowsers: true,
	}); err != nil {
		log.Fatalf("install Playwright driver: %v", err)
	}
}
