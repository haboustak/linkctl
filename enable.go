package main

import (
	"fmt"

	"github.com/haboustak/linkctl/internal/networkd"
)

var cmdEnable = &Command{
	Name: "enable",
	Run:  enable,
}

func enable(self *Command) error {
	args := self.Flags.Args()

	if len(args) != 1 {
		return fmt.Errorf("You must provide the name of the link to enable")
	}

	netdev, ok := networkd.GetNetDev(args[0])
	if !ok {
		return fmt.Errorf("No link with the name %s", args[0])
	}

	if err := netdev.Enable(); err != nil {
		return err
	}

	return networkd.Restart()
}
