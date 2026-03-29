package cmd

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	alchemy_build "github.com/csautter/dev-alchemy/pkg/build"
)

type vmTableRowBuilder func(vm alchemy_build.VirtualMachineConfig) ([]string, error)

func printVirtualMachineCombinationTable(
	writer io.Writer,
	title string,
	emptyMessage string,
	vms []alchemy_build.VirtualMachineConfig,
	headers []string,
	rowBuilder vmTableRowBuilder,
) error {
	fmt.Fprintf(writer, "%s\n", title)
	if len(vms) == 0 {
		fmt.Fprintf(writer, "%s\n", emptyMessage)
		return nil
	}

	grouped := alchemy_build.GroupVirtualMachineConfigsByVirtualizationEngine(vms)
	for _, engine := range alchemy_build.VirtualizationEnginesForVirtualMachineConfigs(vms) {
		fmt.Fprintf(writer, "\nVirtualization engine: %s\n", alchemy_build.DisplayVirtualizationEngine(engine))
		if err := writeVirtualMachineTable(writer, headers, grouped[engine], rowBuilder); err != nil {
			return err
		}
	}

	return nil
}

func writeVirtualMachineTable(
	writer io.Writer,
	headers []string,
	vms []alchemy_build.VirtualMachineConfig,
	rowBuilder vmTableRowBuilder,
) error {
	tw := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	for _, vm := range vms {
		row, err := rowBuilder(vm)
		if err != nil {
			return err
		}
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	return tw.Flush()
}

func displayVirtualMachineType(vm alchemy_build.VirtualMachineConfig) string {
	if vm.UbuntuType == "" {
		return "-"
	}
	return vm.UbuntuType
}
