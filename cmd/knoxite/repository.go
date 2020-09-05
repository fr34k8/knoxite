/*
 * knoxite
 *     Copyright (c) 2016-2020, Christian Muehlhaeuser <muesli@gmail.com>
 *     Copyright (c) 2020,      Nicolas Martin <penguwin@penguwin.eu>
 *
 *   For license see LICENSE
 */

package main

import (
	"encoding/json"
	"fmt"

	shutdown "github.com/klauspost/shutdown2"
	"github.com/muesli/gotable"
	"github.com/spf13/cobra"

	"github.com/knoxite/knoxite"
	"github.com/knoxite/knoxite/cmd/knoxite/utils"
)

var (
	repoCmd = &cobra.Command{
		Use:   "repo",
		Short: "manage repository",
		Long:  `The repo command manages repositories`,
		RunE:  nil,
	}
	repoInitCmd = &cobra.Command{
		Use:   "init",
		Short: "initialize a new repository",
		Long:  `The init command initializes a new repository`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRepoInit()
		},
	}
	repoChangePasswordCmd = &cobra.Command{
		Use:   "passwd",
		Short: "changes the password of a repository",
		Long:  `The passwd command changes the password of a repository`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRepoChangePassword()
		},
	}
	repoCatCmd = &cobra.Command{
		Use:   "cat",
		Short: "display repository information as JSON",
		Long:  `The cat command displays the internal repository information as JSON`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRepoCat()
		},
	}
	repoInfoCmd = &cobra.Command{
		Use:   "info",
		Short: "display repository information",
		Long:  `The info command displays the repository status & information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRepoInfo()
		},
	}
	repoAddCmd = &cobra.Command{
		Use:   "add <url>",
		Short: "add another storage backend to a repository",
		Long:  `The add command adds another storage backend to a repository`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("add needs a URL to be added")
			}
			return executeRepoAdd(args[0])
		},
	}
	repoPackCmd = &cobra.Command{
		Use:   "pack",
		Short: "pack repository and release redundant data",
		Long:  `The pack command deletes all unused data chunks from storage`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRepoPack()
		},
	}
)

func init() {
	repoCmd.AddCommand(repoInitCmd)
	repoCmd.AddCommand(repoChangePasswordCmd)
	repoCmd.AddCommand(repoCatCmd)
	repoCmd.AddCommand(repoInfoCmd)
	repoCmd.AddCommand(repoAddCmd)
	repoCmd.AddCommand(repoPackCmd)
	RootCmd.AddCommand(repoCmd)
}

func executeRepoInit() error {
	// we don't want these next calls to be interrupted
	logger.Log(knoxite.Info, "Acquiring shutdown lock")
	lock := shutdown.Lock()
	if lock == nil {
		return nil
	}
	logger.Log(knoxite.Info, "Acquired and locked shutdown lock")

	defer lock()
	defer logger.Log(knoxite.Info, "Shutdown lock released")

	logger.Log(knoxite.Info, "Creating new repository")
	r, err := newRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return fmt.Errorf("Creating repository at %s failed: %v", globalOpts.Repo, err)
	}
	logger.Log(knoxite.Info, "Created repository")

	fmt.Printf("Created new repository at %s\n", (*r.BackendManager().Backends[0]).Location())
	logger.Log(knoxite.Info, "Repo init command finished successfully")

	return nil
}

func executeRepoChangePassword() error {
	logger.Log(knoxite.Info, "Opening repository")
	r, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, "Expecting user input for password, twice")
	password, err := utils.ReadPasswordTwice("Enter new password:", "Confirm password:")
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "User input read")

	logger.Log(knoxite.Info, "Changing password of repository")
	err = r.ChangePassword(password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Password changed")
	fmt.Printf("Changed password successfully\n")
	logger.Log(knoxite.Info, "Change password command finished successfully")

	return nil
}

func executeRepoAdd(url string) error {
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
	r, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, "Find backend from url")
	backend, err := knoxite.BackendFromURL(url)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, fmt.Sprintf("Found backend with protocols: %s", backend.Protocols()))

	logger.Log(knoxite.Info, "Initialising repository")
	err = backend.InitRepository()
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Initialised repository")

	logger.Log(knoxite.Debug, "Adding backend to backend manager")
	r.BackendManager().AddBackend(&backend)

	logger.Log(knoxite.Info, "Saving repository")
	err = r.Save()
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Saved repository")
	fmt.Printf("Added %s to repository\n", backend.Location())
	logger.Log(knoxite.Info, "Repo add command finished successfully")

	return nil
}

func executeRepoCat() error {
	logger.Log(knoxite.Info, "Opening repository")
	r, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, "Marshalling repo json")
	json, err := json.MarshalIndent(r, "", "    ")
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Marshalled repo json")

	fmt.Printf("%s\n", json)
	logger.Log(knoxite.Info, "Repo cat command finished successfully")

	return nil
}

func executeRepoPack() error {
	logger.Log(knoxite.Info, "Opening repository")
	r, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, "Opening chunk index")
	index, err := knoxite.OpenChunkIndex(&r)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened chunk index")

	logger.Log(knoxite.Info, "Packing chunk index")
	freedSize, err := index.Pack(&r)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Packed chunk index")

	logger.Log(knoxite.Info, "Saving repository")
	err = index.Save(&r)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Saved repository")

	fmt.Printf("Freed storage space: %s\n", knoxite.SizeToString(freedSize))
	logger.Log(knoxite.Info, "Repo pack command finished successfully")

	return nil
}

func executeRepoInfo() error {
	logger.Log(knoxite.Info, "Opening repository")
	r, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, "Initialising new gotable for output")
	tab := gotable.NewTable([]string{"Storage URL", "Available Space"},
		[]int64{-48, 15},
		"No backends found.")

	logger.Log(knoxite.Debug, "Iterating over backends to print output")
	for _, be := range r.BackendManager().Backends {
		space, _ := (*be).AvailableSpace()
		tab.AppendRow([]interface{}{
			(*be).Location(),
			knoxite.SizeToString(space)})
	}

	logger.Log(knoxite.Info, "Printing output")
	err = tab.Print()
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Repo info command finished successfully")

	return nil
}

func openRepository(path, password string) (knoxite.Repository, error) {
	logger.Log(knoxite.Info, "Checking if password provided via -p command")
	if password == "" {
		var err error

		logger.Log(knoxite.Info, "Reading password")
		password, err = utils.ReadPassword("Enter password:")
		if err != nil {
			return knoxite.Repository{}, err
		}
		logger.Log(knoxite.Info, "Read password")

	}

	if rep, ok := cfg.Repositories[path]; ok {
		logger.Log(knoxite.Info, "Opening repository with path supplied by config")
		r, err := knoxite.OpenRepository(rep.Url, password)
		if err != nil {
			return knoxite.Repository{}, err
		}
		logger.Log(knoxite.Info, "Opened repository")

		return r, nil
	}

	logger.Log(knoxite.Info, "No path supplied via config. Opening repository with path.")
	r, err := knoxite.OpenRepository(path, password)
	if err != nil {
		return knoxite.Repository{}, err
	}
	logger.Log(knoxite.Info, "Opened repository")
	logger.Log(knoxite.Info, "Open repository finished successfully")

	return r, nil
}

func newRepository(path, password string) (knoxite.Repository, error) {
	logger.Log(knoxite.Info, "Checking if password provided via -p command")
	if password == "" {
		var err error

		logger.Log(knoxite.Info, "Reading password")
		password, err = utils.ReadPasswordTwice("Enter a password to encrypt this repository with:", "Confirm password:")
		if err != nil {
			return knoxite.Repository{}, err
		}
		logger.Log(knoxite.Info, "Read password")
	}

	logger.Log(knoxite.Info, "Creating new repository")
	r, err := knoxite.NewRepository(path, password)
	if err != nil {
		return knoxite.Repository{}, err
	}
	logger.Log(knoxite.Info, "Created repository")
	logger.Log(knoxite.Info, "New repository finished successfully")
	return r, nil
}
