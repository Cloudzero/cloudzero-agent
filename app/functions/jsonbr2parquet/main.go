// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/cloudzero/cloudzero-agent/app/store"
	"github.com/rs/zerolog/log"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s <input file> <output file>", os.Args[0])
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	input, err := store.NewMetricFile(inputFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create input file")
	}

	parquetData, err := io.ReadAll(input)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read input file")
	}

	output, err := os.Create(outputFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create output file")
	}

	_, err = output.Write(parquetData)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to write output file")
	}

	output.Close()
}
