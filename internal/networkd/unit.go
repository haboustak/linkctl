package networkd

import (
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"

    "gopkg.in/ini.v1"
)

type Unit struct {
    Path    string
    Name    string
    File    *ini.File
}

func (self *Unit) Delete() error {
    err := os.Remove(self.Path)
    if err != nil && !os.IsNotExist(err) {
        fmt.Println(err)
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
    unit := Unit {
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
