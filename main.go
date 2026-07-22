package main

import (
	"flag"
	"fmt"
	"log"
)

const defaultMetadataURL = "http://169.254.169.250/2015-12-19"

var (
	metadataURL = flag.String("metadata-url", defaultMetadataURL, "Metadata API base URL")
	showVersion = flag.Bool("version", false, "Print version and exit")
	version     = "dev"
)

func main() {
	flag.Parse()
	if *showVersion {
		fmt.Printf("metadata-healthcheck %s\n", version)
		return
	}
	if err := Poll(*metadataURL); err != nil {
		log.Fatal(err)
	}
}
