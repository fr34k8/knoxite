/*
 * knoxite
 *     Copyright (c) 2016-2020, Christian Muehlhaeuser <muesli@gmail.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"fmt"

	"github.com/muesli/gotable"
	"github.com/spf13/cobra"

	"github.com/knoxite/knoxite"
)

var (
	snapshotCmd = &cobra.Command{
		Use:   "snapshot",
		Short: "manage snapshots",
		Long:  `The snapshot command manages snapshots`,
		RunE:  nil,
	}
	snapshotListCmd = &cobra.Command{
		Use:   "list <volume>",
		Short: "list all snapshots inside a volume",
		Long:  `The list command lists all snapshots stored in a volume`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("list needs a volume ID to work on")
			}
			return executeSnapshotList(args[0])
		},
	}
	snapshotRemoveCmd = &cobra.Command{
		Use:   "remove <snapshot>",
		Short: "remove a snapshot",
		Long:  `The remove command deletes a snapshot from a volume`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("remove needs a snapshot ID to work on")
			}
			return executeSnapshotRemove(args[0])
		},
	}
)

func init() {
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotRemoveCmd)
	RootCmd.AddCommand(snapshotCmd)
}

func executeSnapshotRemove(snapshotID string) error {
	logger.Log(knoxite.Info, "Opening repository")
	repository, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, "Opening chunk index")
	chunkIndex, err := knoxite.OpenChunkIndex(&repository)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened chunk index")

	logger.Log(knoxite.Info, fmt.Sprintf("Finding snapshot %s", snapshotID))
	volume, snapshot, err := repository.FindSnapshot(snapshotID)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Found snapshot")

	logger.Log(knoxite.Info, fmt.Sprintf("Removing snapshot %s. Description: %s. Date: %s", snapshotID, snapshot.Description, snapshot.Date))
	err = volume.RemoveSnapshot(snapshot.ID)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Removed snapshot")

	logger.Log(knoxite.Info, fmt.Sprintf("Removing snapshot %s from chunk index", snapshotID))
	chunkIndex.RemoveSnapshot(snapshot.ID)

	logger.Log(knoxite.Info, "Saving chunk index for repository")
	err = chunkIndex.Save(&repository)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Saved chunk index")

	logger.Log(knoxite.Info, "Saving repository")
	err = repository.Save()
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Saved repository")

	fmt.Printf("Snapshot %s removed: %s\n", snapshot.ID, snapshot.Stats.String())
	fmt.Println("Do not forget to run 'repo pack' to delete un-referenced chunks and free up storage space!")
	logger.Log(knoxite.Info, "Snapshot remove command finished successfully")

	return nil
}

func executeSnapshotList(volID string) error {
	logger.Log(knoxite.Info, "Opening repository")
	repository, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, fmt.Sprintf("Finding volume %s", volID))
	volume, err := repository.FindVolume(volID)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Found volume")

	logger.Log(knoxite.Info, "Initialising new gotable for output")
	tab := gotable.NewTable([]string{"ID", "Date", "Original Size", "Storage Size", "Description"},
		[]int64{-8, -19, 13, 12, -48}, "No snapshots found. This volume is empty.")
	totalSize := uint64(0)
	totalStorageSize := uint64(0)

	logger.Log(knoxite.Info, "Iterating over snapshots to print details")
	for _, snapshotID := range volume.Snapshots {
		logger.Log(knoxite.Debug, fmt.Sprintf("Loading snapshot %s", snapshotID))
		snapshot, err := volume.LoadSnapshot(snapshotID, &repository)
		if err != nil {
			return err
		}
		logger.Log(knoxite.Debug, "Loaded snapshot")

		logger.Log(knoxite.Debug, "Appending snapshot information to gotable")
		tab.AppendRow([]interface{}{
			snapshot.ID,
			snapshot.Date.Format(timeFormat),
			knoxite.SizeToString(snapshot.Stats.Size),
			knoxite.SizeToString(snapshot.Stats.StorageSize),
			snapshot.Description})
		totalSize += snapshot.Stats.Size
		totalStorageSize += snapshot.Stats.StorageSize
	}

	tab.SetSummary([]interface{}{"", "", knoxite.SizeToString(totalSize), knoxite.SizeToString(totalStorageSize), ""})

	logger.Log(knoxite.Info, "Printing snapshot list output")
	err = tab.Print()
	if err != nil {
		return err
	}

	logger.Log(knoxite.Info, "Snapshot list command finished successfully")
	return nil
}
