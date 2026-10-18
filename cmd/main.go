package main

import (
	"log"
)

func main() {
	application, err := New()
	if err != nil {
		log.Fatalf("startup failed: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("runtime error: %v", err)
	}
}
