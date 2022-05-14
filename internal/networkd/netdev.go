package networkd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type NetDev struct {
	Name              string
	Kind              string
	Description       string
	Unit              *Unit
	Status            LinkStatus
	RenameUnit        *Unit
	RenameNetworkUnit *Unit
	Interface         *Interface
	Network           *Network
	ParentNetwork     *Network
}

type LinkStatus string
type LinkType string

const (
	LinkEnabled     = "enabled"
	LinkDisabled    = "disabled"
	LinkUserDefined = "user-defined"
)

const (
	EnabledLink   LinkType = "enabled"
	AvailableLink LinkType = "available"
)

var netdevs map[string]*NetDev

func (self *NetDev) Enable() error {
	switch self.Status {
	case LinkEnabled:
		return fmt.Errorf("The link %s is already enabled", self.Name)
	case LinkUserDefined:
		return fmt.Errorf("The link %s is user-defined and cannot be enabled", self.Name)
	}

	self.Status = LinkEnabled

	if err := self.ParentNetwork.UpdateNetDev(self); err != nil {
		return err
	}

	linkedName := fmt.Sprintf(
		"/etc/systemd/network/%s",
		self.Unit.Name)
	if err := os.Symlink(self.Unit.Path, linkedName); err != nil {
		return fmt.Errorf("Failed to create unit symlink %s", linkedName)
	}

	return nil
}

func (self *NetDev) Disable() error {
	switch self.Status {
	case LinkDisabled:
		return fmt.Errorf("The link %s is already disabled", self.Name)
	case LinkUserDefined:
		return fmt.Errorf("The link %s is user-defined and cannot be disabled", self.Name)
	}

	self.Status = LinkDisabled

	if err := self.ParentNetwork.UpdateNetDev(self); err != nil {
		return err
	}

	err := os.Remove(self.Unit.Path)
	if err != nil {
		return err
	}

	return self.Interface.Delete()
}

func (self *NetDev) Rename(newName string) error {
	if self.RenameUnit == nil {
		unit, err := self.Unit.NewDropin("name")
		if err != nil {
			return fmt.Errorf("Failed to create dropin for unit %s: %w", self.Unit.Path, err)
		}
		self.RenameUnit = unit
	}

	if self.Network != nil && self.Network.Unit != nil && self.RenameNetworkUnit == nil {
		// We only need to extend the network config if it matches this netdev by name
		if self.Network.Unit.ContainsValue("Match", "Name", self.Name) {
			unit, err := self.Network.Unit.NewDropin("name")
			if err != nil {
				return fmt.Errorf("Failed to create dropin for unit %s: %w", self.Unit.Path, err)
			}
			self.RenameNetworkUnit = unit
		}
	}

	if err := self.RenameUnit.Set("NetDev", "Name", newName); err != nil {
		return fmt.Errorf("Unable to save dropin unit %s: %w",
			self.RenameUnit.Path, err)
	}

	if self.RenameNetworkUnit != nil {
		if err := self.RenameNetworkUnit.Replace("Match", "Name", self.Name, newName); err != nil {
			return fmt.Errorf("Unable to save dropin unit %s: %w",
				self.RenameUnit.Path, err)
		}
	}

	self.Interface.Delete()

	if err := self.Reload(); err != nil {
		return err
	}

	if err := self.ParentNetwork.UpdateNetDev(self); err != nil {
		return err
	}

	return nil
}

func (self *NetDev) ResetName() error {
	if self.RenameUnit != nil {
		if err := self.RenameUnit.Remove("NetDev", "Name"); err != nil {
			return err
		}
	}

	if self.RenameNetworkUnit != nil {
		if err := self.RenameNetworkUnit.Exclude("Match", "Name", self.Name); err != nil {
			return err
		}
	}

	self.Interface.Delete()

	if err := self.Reload(); err != nil {
		return err
	}

	if err := self.ParentNetwork.UpdateNetDev(self); err != nil {
		return err
	}

	return nil
}

func (self *NetDev) Reload() error {
	if err := self.loadUnit(); err != nil {
		return err
	}

	if err := self.findNetworkDropin(); err != nil {
		return err
	}

	return nil
}

func (self *NetDev) parseUnitName() (string, error) {
	nameParts := strings.SplitN(self.Unit.Name, "-", 2)
	if len(nameParts) != 2 {
		return "", fmt.Errorf(
			"link %s does not have a conforming unit name (%s)",
			self.Name, self.Unit.Name)
	}

	networkParts := strings.SplitN(nameParts[1], ".", 2)
	if len(networkParts) != 2 {
		return "", fmt.Errorf(
			"link %s does not have a conforming unit name (%s)",
			self.Name, self.Unit.Name)
	}
	intfName := networkParts[0]

	return intfName, nil
}

func (self *NetDev) loadUnit() error {
	self.applyConfig(self.Unit)

	if err := self.applyDropinConfigs(); err != nil {
		return err
	}

	return nil
}

func (self *NetDev) applyConfig(unit *Unit) {
	section := unit.File.Section("NetDev")

	var keys = []string{"Name", "Kind", "Description"}

	for _, key := range keys {
		if section.HasKey(key) {
			value := section.Key(key).String()
			switch key {
			case "Name":
				self.Name = value
				self.Interface = NewInterface(self.Name)
			case "Kind":
				self.Kind = value
			case "Description":
				self.Description = value
			}
		}
	}
}

func (self *NetDev) applyDropinConfigs() error {
	if self.Unit == nil {
		return nil
	}

	for unit := range self.Unit.DropinUnits() {
		origName := self.Name
		self.applyConfig(unit)

		if self.Name != origName {
			self.RenameUnit = unit
		}
	}

	return nil
}

func (self *NetDev) findNetworkDropin() error {
	if self.Network == nil || self.Network.Unit == nil {
		return nil
	}

	for unit := range self.Network.Unit.DropinUnits() {
		match := unit.Get("Match", "Name")
		for _, name := range strings.Split(match, " ") {
			if name == self.Name {
				self.RenameNetworkUnit = unit
				break
			}
		}
	}

	return nil
}

func NewNetDev(path string, linkType LinkType) (*NetDev, error) {
	var netdev NetDev

	unit, err := NewUnit(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to load unit %s: %w", path, err)
	}
	netdev.Unit = unit

	if err := netdev.loadUnit(); err != nil {
		return nil, err
	}

	intfName, _ := netdev.parseUnitName()
	netdev.Network = NetworkFromIntf(netdev.Name)
	netdev.ParentNetwork = NetworkFromIntf(intfName)

	if err := netdev.findNetworkDropin(); err != nil {
		return nil, err
	}

	netdev.Status = LinkDisabled
	if linkType == EnabledLink {
		fileInfo, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}

		if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
			netdev.Status = LinkEnabled
		} else {
			netdev.Status = LinkUserDefined
		}
	}

	return &netdev, nil
}

func loadNetDevs() {
	linkTypes := [2]LinkType{EnabledLink, AvailableLink}
	configPaths := map[LinkType][]string{
		EnabledLink:   []string{"/etc/systemd/network/*.netdev"},
		AvailableLink: []string{"/etc/linkctl/user/*.netdev", "/etc/linkctl/system/*.netdev", "/etc/systemd/network/netdev.available/*.netdev"},
	}

	if netdevs != nil {
		return
	}

	netdevs = make(map[string]*NetDev)
	for _, linkType := range linkTypes {
		for _, path := range configPaths[linkType] {
			files, err := filepath.Glob(path)
			if err != nil {
				panic(err)
			}

			for _, file := range files {
				netdev, err := NewNetDev(file, linkType)
				if err != nil {
					continue
				}
				_, exist := netdevs[netdev.Name]
				if !exist {
					netdevs[netdev.Name] = netdev
				}
			}
		}
	}
}

func ListNetDev(listAll bool) []*NetDev {
	loadNetDevs()

	var keys []string
	var values []*NetDev
	for k, v := range netdevs {
		if listAll || v.Status != LinkDisabled {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		values = append(values, netdevs[k])
	}
	return values
}

func GetNetDev(linkName string) (*NetDev, bool) {
	loadNetDevs()

	link, ok := netdevs[linkName]
	return link, ok
}
