// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package install contains a CLI for copying the executable to a destination.
package install

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/cloudzero/cloudzero-agent/pkg/config"
)

const (
	configFileDesc = "input " + config.FlagDescConfFile
)

var configAlias = []string{"f"}

func NewCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "install",
		Usage:   "install executable",
		Aliases: []string{"i"},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "destination", Usage: "destination", Required: true},
		},
		Action: func(c *cli.Context) error {
			return installExecutable(c.String("destination"))
		},
	}
	return cmd
}

func installExecutable(destination string) error {
	fmt.Printf("Installing executable from %s to %s\n", os.Args[0], destination)

	source := os.Args[0]
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destinationDirectory := filepath.Dir(destination)
	if _, err := os.Stat(destinationDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(destinationDirectory, 0755); err != nil {
			return err
		}
	}

	destinationFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	os.Chmod(destination, sourceInfo.Mode())

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}
