/*
 * knoxite
 *     Copyright (c) 2016-2020, Christian Muehlhaeuser <muesli@gmail.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"fmt"
	"os/user"
	"strconv"
	"time"

	"github.com/muesli/gotable"
	"github.com/spf13/cobra"

	"github.com/knoxite/knoxite"
)

const timeFormat = "2006-01-02 15:04:05"

var (
	lsCmd = &cobra.Command{
		Use:   "ls <snapshot>",
		Short: "list files",
		Long:  `The ls command lists all files stored in a snapshot`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("ls needs a snapshot ID")
			}

			logger.Log(knoxite.Info, "Execute ls command")
			return executeLs(args[0])
		},
	}
)

func init() {
	RootCmd.AddCommand(lsCmd)
}

func executeLs(snapshotID string) error {
	logger.Log(knoxite.Info, "Opening repository")
	repository, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, "Initialising new gotable for output")
	tab := gotable.NewTable([]string{"Perms", "User", "Group", "Size", "ModTime", "Name"},
		[]int64{-10, -8, -5, 12, -19, -48},
		"No files found.")

	logger.Log(knoxite.Info, "Finding snapshot "+snapshotID)
	_, snapshot, ferr := repository.FindSnapshot(snapshotID)
	if ferr != nil {
		return ferr
	}
	logger.Log(knoxite.Info, "Found snapshot "+snapshot.Description)

	logger.Log(knoxite.Info, "Iterating archives to print details")
	for _, archive := range snapshot.Archives {
		username := strconv.FormatInt(int64(archive.UID), 10)

		logger.Log(knoxite.Info, fmt.Sprintf("Looking up OS username with archive's UID %d", archive.UID))
		u, uerr := user.LookupId(username)
		if uerr != nil {
			logger.Log(knoxite.Warning, "Looking up username failed. Using default value.")
		}
		username = u.Username

		groupname := strconv.FormatInt(int64(archive.GID), 10)
		tab.AppendRow([]interface{}{
			archive.Mode,
			username,
			groupname,
			knoxite.SizeToString(archive.Size),
			time.Unix(archive.ModTime, 0).Format(timeFormat),
			archive.Path})
	}

	logger.Log(knoxite.Info, "Printing ls output")
	err = tab.Print()
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Printed ls output")
	logger.Log(knoxite.Info, "ls command finished successfully")

	return nil
}
