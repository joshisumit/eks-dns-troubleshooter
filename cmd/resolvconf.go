package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

//resolvconf struct stores /etc/resolv.conf of a pod
type resolvconf struct {
	searchPath []string
	nameserver []string
	options    []string
	ndots      int
}

func (rc *resolvconf) readResolvConf() error {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		log.Errorf("Failed to read /etc/resolv.conf file: %s", err)
		return err
	}
	in := bufio.NewScanner(f)

	var lines int
	log.Infof("Reading /etc/resolv.conf")

	for in.Scan() {
		lines++
		fields := strings.Fields(in.Text())

		if fields[0] == "search" {
			rc.searchPath = fields[1:]
			continue
		} else if fields[0] == "nameserver" {
			rc.nameserver = fields[1:]
			continue
		} else {
			rc.options = fields[1:]
			for _, opt := range rc.options {
				tmp := strings.Split(opt, ":")
				if tmp[0] == "ndots" {
					rc.ndots, err = strconv.Atoi(tmp[1])
					if err != nil {
						return err
					}
				}
			}
		}
	}

	log.Infof("resolvconf struct values: %+v", rc)
	return err
}
