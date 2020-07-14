package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/haboustak/linkctl/internal/networkd"
)

var (
	showAll bool
)

var cmdList = &Command{
	Name: "list",
	Run:  list,
	Usage: `Usage:
    linkctl [-h] list [-a]

Show systemd-networkd netdev links

Options:
    -a      show all links
    -h      show this help
`,
}

func init() {
	cmdList.Flags.BoolVar(&showAll, "a", false, "show all links")
}

func ansiColorStatus(status networkd.LinkStatus) string {
	if !IsATTY {
		return string(status)
	}

	switch status {
	case networkd.LinkEnabled:
		return fmt.Sprintf("\x1B[0;1;32m%s\x1B[0m", status)
	case networkd.LinkUserDefined:
		return fmt.Sprintf("\x1B[0;1;38;5;185m%s\x1B[0m", status)
	default:
		return string(status)
	}
}
func list(self *Command) error {
	links := networkd.ListNetDev(showAll)
	if links == nil {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 15, 3, 1, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "NAME\tTYPE\tSTATUS")
	for _, link := range links {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			link.Name, link.Kind,
			ansiColorStatus(link.Status))
	}
	w.Flush()
	return nil
}
