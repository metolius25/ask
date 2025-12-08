package main

// Version information - Version is set at build time via ldflags
// Build: go build -ldflags="-s -w -X main.Version=v1.0.0" .
var (
	Version = "dev" // Overwritten at build time with git tag
	AppName = "ask"
)
