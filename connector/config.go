package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func ParseConfig(pathConfig string) ([]uint16, error) {
	dat, err := os.ReadFile(pathConfig)
	if err != nil {
		log.Println("could not read config file")
		return nil, fmt.Errorf("could not read config file")
	}

	l := strings.Split(string(dat), "\n")
	d := make([]uint16, 0)
	for _, v := range l {
		c, err := strconv.Atoi(v)
		if err != nil {
			log.Println("found incorrect entry in config file, the entry:", v)
			return nil, fmt.Errorf("found incorrect entry in config file")
		}
		d = append(d, uint16(c))
	}
	return d, nil
}
