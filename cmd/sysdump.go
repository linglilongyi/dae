/*
 * SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) 2022-2024, daeuniverse Organization <dae@v2raya.org>
 */

package cmd

import (
	"bytes"
	"io/ioutil"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
    
	"github.com/vishvananda/netlink"
	"github.com/spf13/cobra"
	"github.com/mholt/archiver/v3"
	"github.com/shirou/gopsutil/v4/net"
)

var (
	sysdumpCmd = &cobra.Command{
		Use:   "sysdump",
		Short: "Dumps system network information",
		Run: func(cmd *cobra.Command, args []string) {
			dumpNetworkInfo()
		},
	}
)

func dumpNetworkInfo() {
	tempDir, err := ioutil.TempDir("", "sysdump")
	if err != nil {
		fmt.Printf("Failed to create temp directory: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir)

	dumpRouting(tempDir)
	dumpNetInterfaces(tempDir)
	dumpSysctl(tempDir)
	dumpNetfilter(tempDir)
	dumpIPTables(tempDir)

	tarFile := "sysdump.tar.gz"
	if err := archiver.Archive([]string{tempDir}, tarFile); err != nil {
		fmt.Printf("Failed to create tar archive: %v\n", err)
		return
	}

	fmt.Printf("System network information collected and saved to %s\n", tarFile)
}

func dumpRouting(outputDir string) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		fmt.Printf("Failed to get routing table: %v\n", err)
		return
	}

	var buffer bytes.Buffer
	buffer.WriteString("Routing Table:\n")
	for _, route := range routes {
		buffer.WriteString(fmt.Sprintf("Iface: %s, Dst: %s, Gw: %s, Flags: %d, Table: %d, Proto: %d, Scope: %d, Type: %d, TOS: %d\n",
			route.LinkIndex, route.Dst, route.Gw, route.Flags, route.Table, route.Protocol, route.Scope, route.Type, route.Tos))
	}

	ioutil.WriteFile(filepath.Join(outputDir, "routing.txt"), buffer.Bytes(), 0644)
}

func dumpNetInterfaces(outputDir string) {
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("Failed to get network interfaces: %v\n", err)
		return
	}

	var buffer bytes.Buffer
	buffer.WriteString("Network Interfaces:\n")
	for _, iface := range interfaces {
		buffer.WriteString(fmt.Sprintf("Name: %s, MTU: %d, HardwareAddr: %s, Flags: %v\n",
			iface.Name, iface.MTU, iface.HardwareAddr, iface.Flags))
		for _, addr := range iface.Addrs {
			buffer.WriteString(fmt.Sprintf("  Address: %s\n", addr.Addr))
		}
	}

	ioutil.WriteFile(filepath.Join(outputDir, "interfaces.txt"), buffer.Bytes(), 0644)
}


func dumpSysctl(outputDir string) {
	sysctlPath := "/proc/sys/net"
	var buffer bytes.Buffer

	err := filepath.Walk(sysctlPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			value, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			relativePath := strings.TrimPrefix(path, sysctlPath+"/")
			buffer.WriteString(fmt.Sprintf("%-60s = %s\n", relativePath, string(value)))
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Failed to get sysctl settings: %v\n", err)
		return
	}

	ioutil.WriteFile(filepath.Join(outputDir, "sysctl.txt"), buffer.Bytes(), 0644)
}

func dumpNetfilter(outputDir string) {
	cmd := exec.Command("nft", "list", "ruleset")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to get nftables ruleset: %v\n", err)
		return
	}

	ioutil.WriteFile(filepath.Join(outputDir, "nftables.txt"), output, 0644)
}

func dumpIPTables(outputDir string) {
	cmd := exec.Command("iptables-save", "-t", "nat")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to get iptables NAT table: %v\n", err)
		return
	}

	ioutil.WriteFile(filepath.Join(outputDir, "iptables_nat.txt"), output, 0644)
}

func init() {
	rootCmd.AddCommand(sysdumpCmd)
}
