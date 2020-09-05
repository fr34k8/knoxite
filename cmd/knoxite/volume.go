/*
 * knoxite
 *     Copyright (c) 2016-2020, Christian Muehlhaeuser <muesli@gmail.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"fmt"

	shutdown "github.com/klauspost/shutdown2"
	"github.com/muesli/gotable"
	"github.com/spf13/cobra"

	"github.com/knoxite/knoxite"
)

// VolumeInitOptions holds all the options that can be set for the 'volume init' command.
type VolumeInitOptions struct {
	Description string
}

var (
	volumeInitOpts = VolumeInitOptions{}

	volumeCmd = &cobra.Command{
		Use:   "volume",
		Short: "manage volumes",
		Long:  `The volume command manages volumes`,
		RunE:  nil,
	}
	volumeInitCmd = &cobra.Command{
		Use:   "init <name>",
		Short: "initialize a new volume",
		Long:  `The init command initializes a new volume`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("init needs a name for the new volume")
			}

			logger.Log(knoxite.Info, fmt.Sprintf("Initializing volume %s", volumeInitOpts.Description))
			err := executeVolumeInit(args[0], volumeInitOpts.Description)
			if err != nil {
				return err
			}
			logger.Log(knoxite.Info, "Initialized volume")

			return nil
		},
	}
	volumeListCmd = &cobra.Command{
		Use:   "list",
		Short: "list all volumes inside a repository",
		Long:  `The list command lists all volumes stored in a repository`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeVolumeList()
		},
	}
)

func init() {
	volumeInitCmd.Flags().StringVarP(&volumeInitOpts.Description, "desc", "d", "", "a description or comment for this volume")

	volumeCmd.AddCommand(volumeInitCmd)
	volumeCmd.AddCommand(volumeListCmd)
	RootCmd.AddCommand(volumeCmd)
}

func executeVolumeInit(name, description string) error {
	// we don't want these next calls to be interrupted
	logger.Log(knoxite.Info, "Acquiring shutdown lock")
	lock := shutdown.Lock()
	if lock == nil {
		return nil
	}
	logger.Log(knoxite.Info, "Acquired and locked shutdown lock")

	defer lock()
	defer logger.Log(knoxite.Info, "Shutdown lock released")

	logger.Log(knoxite.Info, "Opening repository")
	repository, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, fmt.Sprintf("Creating volume %s", description))
	vol, verr := knoxite.NewVolume(name, description)
	if verr != nil {
		return verr
	}
	logger.Log(knoxite.Info, fmt.Sprintf("Created volume %s", vol.ID))

	logger.Log(knoxite.Info, fmt.Sprintf("Adding volume %s to repository", vol.ID))
	verr = repository.AddVolume(vol)
	if verr != nil {
		return fmt.Errorf("Creating volume %s failed: %v", name, verr)
	}
	logger.Log(knoxite.Info, "Added volume to repository")

	annotation := "Name: " + vol.Name
	if len(vol.Description) > 0 {
		annotation += ", Description: " + vol.Description
	}
	fmt.Printf("Volume %s (%s) created\n", vol.ID, annotation)

	logger.Log(knoxite.Info, "Saving repository")
	err = repository.Save()
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Saved repository")
	logger.Log(knoxite.Info, "Volume init command finished successfully")

	return nil
}

func executeVolumeList() error {
	logger.Log(knoxite.Info, "Opening repository")
	repository, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, "Initialising new gotable for output")
	tab := gotable.NewTable([]string{"ID", "Name", "Description"},
		[]int64{-8, -32, -48}, "No volumes found. This repository is empty.")

	logger.Log(knoxite.Info, "Iterating over volumes to print details")
	for _, volume := range repository.Volumes {
		tab.AppendRow([]interface{}{volume.ID, volume.Name, volume.Description})
	}

	logger.Log(knoxite.Info, "Printing volume list output")
	err = tab.Print()
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Printed volume list output")
	logger.Log(knoxite.Info, "Volume list command finished successfully")

	return nil
}
