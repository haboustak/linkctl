package networkd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Network struct {
	Interface *Interface
	Unit      *Unit
}

var intfNetwork map[string]*Network

func init() {
	intfNetwork = make(map[string]*Network)
}

func (self *Network) UpdateNetDev(netdev *NetDev) error {
	dropinUnit, err := self.DropinForNetDev(netdev)
	if err != nil {
		return err
	}

	switch netdev.Status {
	case LinkDisabled:
		if err := dropinUnit.Delete(); err != nil {
			return fmt.Errorf("Failed to remove parent unit %s: %w",
				dropinUnit.Path, err)
		}
	case LinkEnabled:
		dropinUnit.File.NewSection("Network")
		dropinUnit.File.Section("Network").NewKey("VLAN", netdev.Name)
		if err := dropinUnit.Save(); err != nil {
			return fmt.Errorf("Failed to update parent unit %s: %w",
				dropinUnit.Path, err)
		}
	}

	return nil
}

func (self *Network) DropinForNetDev(netdev *NetDev) (*Unit, error) {
	if self.Interface.NetIf == nil {
		if self.Interface.Name != "" {
			return nil, fmt.Errorf("There is no interface \"%s\" for link %s (%s)",
				self.Interface.Name, netdev.Name, netdev.Unit.Name)
		}
		return nil, fmt.Errorf(
			"Unable to determine the parent interface for link %s (%s)",
			netdev.Name, netdev.Unit.Name)
	}

	if self.Unit == nil {
		return nil, fmt.Errorf("Unable to determine network unit for interface %s",
			self.Interface.Name)
	}

	dropinPath := fmt.Sprintf(
		"/etc/systemd/network/%s.d/%s.conf",
		netdev.ParentNetwork.Unit.Name,
		strings.TrimSuffix(netdev.Unit.Name, ".netdev"))

	dropinUnit, err := NewUnit(dropinPath)
	if err != nil {
		return nil, err
	}

	return dropinUnit, nil
}

func getNetworkUnit(intf *Interface) (*Unit, error) {
	var networkFile string

	stateFile := fmt.Sprintf("/run/systemd/netif/links/%d", intf.NetIf.Index)
	file, err := os.Open(stateFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "NETWORK_FILE") {
			continue
		}
		stateParts := strings.SplitN(line, "=", 2)
		if len(stateParts) != 2 {
			return nil, fmt.Errorf("Unable to parse state file for %d", intf.NetIf.Index)
		}
		networkFile = stateParts[1]
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	unit, err := NewUnit(networkFile)
	return unit, err
}

func NetworkFromIntf(intfName string) *Network {
	if intfName == "" {
		return nil
	}

	if network, ok := intfNetwork[intfName]; ok {
		return network
	}

	var newNetwork Network

	newNetwork.Interface = NewInterface(intfName)
	if newNetwork.Interface.NetIf != nil {
		newNetwork.Unit, _ = getNetworkUnit(newNetwork.Interface)
	}

	intfNetwork[intfName] = &newNetwork
	return intfNetwork[intfName]
}
