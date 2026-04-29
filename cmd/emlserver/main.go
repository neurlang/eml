package main

import (
	"embed"
	"log"
	"net/http"
)

//go:embed emlui/*
var content embed.FS

func main() {
	http.Handle("/", http.FileServerFS(content))
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
