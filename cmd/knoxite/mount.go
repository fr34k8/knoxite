// +build !openbsd
// +build !windows

/*
 * knoxite
 *     Copyright (c) 2016-2020, Christian Muehlhaeuser <muesli@gmail.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	"github.com/knoxite/knoxite"
)

var (
	mountCmd = &cobra.Command{
		Use:   "mount <snapshot> <target>",
		Short: "mount a snapshot",
		Long:  `The mount command mounts a repository read-only to a given directory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("mount needs to know which snapshot to work on")
			}
			if len(args) < 2 {
				return fmt.Errorf("mount needs to know where to mount the snapshot to")
			}
			return executeMount(args[0], args[1])
		},
	}
)

func init() {
	RootCmd.AddCommand(mountCmd)
}

func executeMount(snapshotID, mountpoint string) error {
	logger.Log(knoxite.Info, "Opening repository")
	repository, err := openRepository(globalOpts.Repo, globalOpts.Password)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Opened repository")

	logger.Log(knoxite.Info, "Finding snapshot "+snapshotID)
	_, snapshot, err := repository.FindSnapshot(snapshotID)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Found snapshot "+snapshot.Description)

	if _, serr := os.Stat(mountpoint); os.IsNotExist(serr) {
		fmt.Printf("Mountpoint %s doesn't exist, creating it\n", mountpoint)

		logger.Log(knoxite.Warning, fmt.Sprintf("Mountpoint %s doesn't exist, creating it", mountpoint))
		err = os.Mkdir(mountpoint, os.ModeDir|0700)
		if err != nil {
			return err
		}
		logger.Log(knoxite.Warning, "Created mountpoint")

	}

	logger.Log(knoxite.Info, "Mounting fuse")
	c, err := fuse.Mount(
		mountpoint,
		fuse.ReadOnly(),
		fuse.FSName("knoxite"),
	)
	if err != nil {
		return err
	}
	logger.Log(knoxite.Info, "Mounted fuse")

	roottree := fs.Tree{}

	fmt.Println("Updating index")

	logger.Log(knoxite.Info, "Updating index")
	updateIndex(&repository, snapshot)
	fmt.Println("Updating index done")

	logger.Log(knoxite.Info, "Adding items to root tree")
	for _, arc := range root.Items {
		roottree.Add(arc.Archive.Path, arc)
	}

	ready := make(chan struct{}, 1)
	done := make(chan struct{})
	ready <- struct{}{}

	errServe := make(chan error)
	go func() {
		logger.Log(knoxite.Info, "Serving file system")
		err = fs.Serve(c, &roottree)
		if err != nil {
			errServe <- err
		}
		logger.Log(knoxite.Info, "Served file system")

		<-c.Ready
		errServe <- c.MountError
	}()

	select {
	case err := <-errServe:
		return err
	case <-done:
		logger.Log(knoxite.Info, "Unmounting fuse")
		err := fuse.Unmount(mountpoint)
		if err != nil {
			fmt.Printf("Error umounting: %s\n", err)
		}
		logger.Log(knoxite.Info, "Unmounted fuse")

		logger.Log(knoxite.Info, "Closing file system")
		err = c.Close()
		if err != nil {
			return err
		}
		logger.Log(knoxite.Info, "Closed file system")
		logger.Log(knoxite.Info, "Mount command finished successfully")

		return nil
	}
}

// Node in our virtual filesystem.
type Node struct {
	Items      map[string]*Node
	Archive    knoxite.Archive
	Repository *knoxite.Repository
	//	sync.RWMutex
}

var (
	root *Node
)

func node(name string, arc knoxite.Archive, repository *knoxite.Repository) *Node {
	l := strings.Split(name, string(filepath.Separator))
	item := root

	logger.Log(knoxite.Debug, "Iterating over filepath")
	for k, s := range l {
		if len(s) == 0 {
			continue
		}
		// fmt.Println("Finding:", s)
		v, ok := item.Items[s]
		if !ok {
			path := filepath.Join(l[:k+1]...)
			logger.Log(knoxite.Debug, fmt.Sprintf("Adding %s to tree", path))
			fmt.Println("Adding to tree:", path)
			if name != path {
				// We stored an absolute path and need to fake the parent
				// dirs for the first item in the archive
				arc = knoxite.Archive{
					Type:    knoxite.Directory,
					GID:     arc.GID,
					ModTime: arc.ModTime,
					Mode:    arc.Mode,
					Path:    path,
				}
			}

			v = &Node{}
			v.Items = make(map[string]*Node)
			v.Archive = arc
			v.Repository = repository
			item.Items[s] = v
		}

		item = v
	}

	return item
}

func updateIndex(repository *knoxite.Repository, snapshot *knoxite.Snapshot) {
	root = &Node{}
	root.Items = make(map[string]*Node)

	logger.Log(knoxite.Debug, "Iterating over archives")
	for _, arc := range snapshot.Archives {
		path := arc.Path
		if path[0] == '/' {
			// This archive contains an absolute path
			// Strip the leading slash for mounting
			path = path[1:]
		}
		logger.Log(knoxite.Debug, fmt.Sprintf("Adding %s to index", path))
		fmt.Println("Adding to index:", path)
		node(path, *arc, repository)
	}
	logger.Log(knoxite.Info, "Update index command finished successfully")
}

// Attr returns this node's filesystem attributes.
func (node *Node) Attr(ctx context.Context, a *fuse.Attr) error {
	// fmt.Println("Attr:", node.Item.Path)
	// a.Inode = node.Inode
	a.Mode = node.Archive.Mode
	a.Size = node.Archive.Size

	switch node.Archive.Type {
	case knoxite.SymLink:
		a.Mode |= os.ModeSymlink
	case knoxite.Directory:
		a.Mode |= os.ModeDir
	}

	return nil
}

// Lookup is used to stat items.
func (node *Node) Lookup(_ context.Context, name string) (fs.Node, error) {
	// fmt.Println("Lookup:", name)
	item, ok := node.Items[name]
	if ok {
		return item, nil
	}

	return nil, fuse.ENOENT
}

// ReadDirAll returns all items directly below this node.
func (node *Node) ReadDirAll(_ context.Context) ([]fuse.Dirent, error) {
	// fmt.Println("ReadDirAll:", node.Item.Path)
	entries := []fuse.Dirent{}

	for k, v := range node.Items {
		ent := fuse.Dirent{Name: k}
		switch v.Archive.Type {
		case knoxite.File:
			ent.Type = fuse.DT_File
		case knoxite.Directory:
			ent.Type = fuse.DT_Dir
		case knoxite.SymLink:
			ent.Type = fuse.DT_Link
		}

		entries = append(entries, ent)
	}

	return entries, nil
}

// Open opens a file.
func (node *Node) Open(_ context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	if !req.Flags.IsReadOnly() {
		logger.Log(knoxite.Info, fmt.Sprintf("Opening file from %s", node.Archive.Path))
		errno := fuse.Errno(syscall.EACCES)
		if errno != 0 {
			return nil, errno
		}
		logger.Log(knoxite.Info, "Opened file")
	}
	resp.Flags |= fuse.OpenKeepCache
	logger.Log(knoxite.Info, "Open node command finished successfully")

	return node, nil
}

// Read reads from a file.
func (node *Node) Read(_ context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	logger.Log(knoxite.Info, fmt.Sprintf("Reading archive %s", node.Archive.Path))
	d, err := knoxite.ReadArchive(*node.Repository, node.Archive, int(req.Offset), req.Size)
	if err != nil {
		if err != io.EOF {
			return err
		}
		resp.Data = nil
	} else {
		logger.Log(knoxite.Info, "Read archive")
		resp.Data = *d
	}
	logger.Log(knoxite.Info, "Read node command finished successfully")

	return nil
}

// Readlink returns the target a symlink is pointing to.
func (node *Node) Readlink(_ context.Context, _ *fuse.ReadlinkRequest) (string, error) {
	logger.Log(knoxite.Info, "Readlink command finished successfully")
	return node.Archive.PointsTo, nil
}

// ReadAll reads an entire archive's content.
/*func (node *Node) ReadAll(_ context.Context) ([]byte, error) {
	d, _, err := knoxite.DecodeArchiveData(*node.Repository, node.Item)
	if err != nil {
		return d, err
	}
	return d, nil
}*/
