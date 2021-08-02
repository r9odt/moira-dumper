package main

import (
	"flag"
	"log"
	"os"

	"github.com/JIexa24/moira-dumper/lib"
)

var api string
var action string
var file string
var directory string

// usage is function is a handler for runtime flag -h.
func usage() {
	flag.PrintDefaults()
	os.Exit(1)
}

// parseFlags is function initialize application runtime flags.
func parseFlags() {
	flag.Usage = usage
	flag.StringVar(&api, "api", "http://127.0.0.1:8081/api",
		"API url")
	flag.StringVar(&action, "action", "dump",
		"Action")
	flag.StringVar(&file, "file", "",
		"File being used")
	flag.StringVar(&directory, "directory", "",
		"Directory being used")
	flag.Parse()
}

func main() {
	parseFlags()
	moira := lib.MoiraAPI{
		API: api,
	}
	switch action {
	case "dump":
		moira.DumpToDir(directory)
	case "apply":
		moira.ApplyFile(file)
	default:
		log.Fatalf("Unknown action %s!", action)
	}
}
