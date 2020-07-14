package main

import (
	"fmt"

	"github.com/haboustak/linkctl/internal/networkd"
)

var cmdDisable = &Command{
	Name: "disable",
	Run:  disable,
}

func init() {
}

func disable(self *Command) error {
	args := self.Flags.Args()

	if len(args) != 1 {
		return fmt.Errorf("You must provide the name of the unit to disable")
	}

	netdev, ok := networkd.GetNetDev(args[0])
	if !ok {
		return fmt.Errorf("No netdev with the name %s", args[0])
	}

	if err := netdev.Disable(); err != nil {
		return err
	}

	return networkd.Restart()
}
