package main

import (
	"flag"
)

type Flags struct {
	ConfigFile string
}

func flags() Flags {
	config := flag.String("config-file", "~/.simplem2m/config.toml", "path to configuration file")
	flag.Parse()
	return Flags{
		ConfigFile: *config,
	}
}
