/*
 * knoxite
 *     Copyright (c) 2016-2020, Christian Muehlhaeuser <muesli@gmail.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"fmt"
	"os"

	"github.com/knoxite/knoxite"

	"github.com/spf13/cobra"
)

var (
	catCmd = &cobra.Command{
		Use:   "cat <snapshot> <file>",
		Short: "print file",
		Long:  `The cat command prints a file on the standard output`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("cat needs a snapshot ID and filename")
			}
			return executeCat(args[0], args[1])
		},
	}
)

func init() {
	RootCmd.AddCommand(catCmd)
}

func executeCat(snapshotID string, file string) error {
	logger.Log(Info, "Opening repository")
	repository, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(Info, "Opened repository")

	logger.Log(Info, "Finding snapshot "+snapshotID)
	_, snapshot, ferr := repository.FindSnapshot(snapshotID)
	if ferr != nil {
		return ferr
	}
	logger.Log(Info, "Found snapshot "+snapshot.Description)

	logger.Log(Info, "Reading snapshot "+snapshotID)
	if archive, ok := snapshot.Archives[file]; ok {
		logger.Log(Info, "Found and read archive from location "+archive.Path)

		logger.Log(Info, "Decoding archive data")
		b, _, erra := knoxite.DecodeArchiveData(repository, *archive)
		if erra != nil {
			return erra
		}
		logger.Log(Info, "Decoded archive data")

		logger.Log(Info, "Output file content")
		_, err = os.Stdout.Write(b)

		logger.Log(Info, "Cat command finished successfully.")
		return err
	}

	return fmt.Errorf("%s: No such file or directory", file)
}
