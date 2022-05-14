package networkd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

type Unit struct {
	Path string
	Name string
	File *ini.File
}

func (self *Unit) Delete() error {
	err := os.Remove(self.Path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	unitDir := filepath.Dir(self.Path)
	if dirIsEmpty(unitDir) {
		os.Remove(unitDir)
	}
	return nil
}

func (self *Unit) DeleteIfEmpty() error {
	if self.IsEmpty() {
		return self.Delete()
	}

	return nil
}

func (self *Unit) IsEmpty() bool {
	for _, section := range self.File.Sections() {
		if !SectionEmpty(section) {
			return false
		}
	}

	return true
}

func (self *Unit) Save() error {
	err := os.MkdirAll(filepath.Dir(self.Path), os.ModePerm)
	if err != nil {
		return err
	}

	return self.File.SaveTo(self.Path)
}

func SectionEmpty(section *ini.Section) bool {
	if len(section.Keys()) > 0 {
		return false
	}

	for _, child := range section.ChildSections() {
		if !SectionEmpty(child) {
			return false
		}
	}

	return true
}

func dirIsEmpty(path string) bool {
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) == 0
}

func NewUnit(path string) (*Unit, error) {
	unit := Unit{
		Path: path,
		Name: filepath.Base(path),
	}

	iniFile, err := ini.Load(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		iniFile = ini.Empty()
	}
	unit.File = iniFile

	return &unit, nil
}

func (self *Unit) NewDropin(name string) (*Unit, error) {
	dropinPath := fmt.Sprintf(
		"/etc/systemd/network/%s.d/%s.conf",
		self.Name,
		name)

	return NewUnit(dropinPath)
}

func (self *Unit) Dropins() []string {
	dropinPath := fmt.Sprintf("/etc/systemd/network/%s.d/*.conf", self.Name)
	dropins, err := filepath.Glob(dropinPath)
	if err != nil {
		panic(fmt.Errorf("Failed to list units at %s: %w", dropinPath, err))
	}

	return dropins
}

func (self *Unit) DropinUnits() chan *Unit {
	r := make(chan *Unit, 5)

	go func() {
		for _, dropin := range self.Dropins() {
			unit, err := NewUnit(dropin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARN: Failed to parse networkd unit %s: %v\n", dropin, err)
				continue
			}
			r <- unit
		}
		close(r)
	}()

	return r
}

func (self *Unit) ContainsValue(section string, key string, value string) bool {
	for _, v := range self.GetValues(section, key) {
		if v == value {
			return true
		}
	}

	return false
}

func (self *Unit) Set(section string, key string, value string) error {
	if value == "" {
		return self.Remove(section, key)
	}

	s := self.File.Section(section)
	s.NewKey(key, value)
	return self.Save()
}

func (self *Unit) Get(section string, key string) string {
	s := self.File.Section(section)
	if s == nil || !s.HasKey(key) {
		return ""
	}

	return s.Key(key).String()
}

func (self *Unit) GetValues(section string, key string) []string {
	if value := self.Get(section, key); value != "" {
		return strings.Split(self.Get(section, key), " ")
	}
	return []string{}
}

func (self *Unit) SetValues(section string, key string, values []string) error {
	return self.Set(section, key, strings.Join(values, " "))
}

func (self *Unit) Remove(section string, key string) error {
	s := self.File.Section(section)
	if s == nil {
		return nil
	}

	s.DeleteKey(key)

	// Delete the file if no more config exists
	if self.IsEmpty() {
		if err := self.Delete(); err != nil {
			return err
		}
	} else if err := self.Save(); err != nil {
		return err
	}

	return nil
}

func (self *Unit) Replace(section string, key string, old string, value string) error {
	values := self.GetValues(section, key)
	fmt.Printf("Setting value from %v\n", values)
	update := make([]string, len(values)+1)
	kept := 0
	for _, v := range values {
		if v != old && v != value {
			update[kept] = v
			kept += 1
		}
	}
	fmt.Printf("Setting value after %v with |%s|\n", update, value)
	update[kept] = value
	fmt.Printf("Setting value to %v\n", update)
	return self.SetValues(section, key, update)
}

func (self *Unit) Include(section string, key string, value string) error {
	values := self.GetValues(section, key)
	for _, v := range values {
		if v == value {
			return nil
		}
	}
	values = append(values, value)
	return self.SetValues(section, key, values)
}

func (self *Unit) Exclude(section string, key string, value string) error {
	values := self.GetValues(section, key)
	update := make([]string, len(values))
	kept := 0
	for _, v := range values {
		if v != value {
			update[kept] = v
			kept += 1
		}
	}

	return self.SetValues(section, key, update)
}
