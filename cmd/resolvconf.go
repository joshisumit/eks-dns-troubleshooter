package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

//ResolvConf struct stores /etc/resolv.conf of a pod
type ResolvConf struct {
	SearchPath []string
	Nameserver []string
	Options    []string
	Ndots      int
}

func (rc *ResolvConf) readResolvConf() error {
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
			rc.SearchPath = fields[1:]
			continue
		} else if fields[0] == "nameserver" {
			rc.Nameserver = fields[1:]
			continue
		} else {
			rc.Options = fields[1:]
			for _, opt := range rc.Options {
				tmp := strings.Split(opt, ":")
				if tmp[0] == "ndots" {
					rc.Ndots, err = strconv.Atoi(tmp[1])
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
