package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/marcboeker/supertar/archive"
	"github.com/marcboeker/supertar/config"
	"github.com/marcboeker/supertar/item"
	"github.com/marcboeker/supertar/server"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

func init() {
	RootCmd.AddCommand(createCmd)
	RootCmd.AddCommand(listCmd)
	RootCmd.AddCommand(extractCmd)
	RootCmd.AddCommand(addCmd)
	RootCmd.AddCommand(deleteCmd)
	RootCmd.AddCommand(moveCmd)
	RootCmd.AddCommand(compactCmd)
	RootCmd.AddCommand(serveCmd)
	RootCmd.AddCommand(updatePwdCmd)

	RootCmd.PersistentFlags().StringVarP(&archiveFile, "file", "f", "", "archive file (*.star)")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	createCmd.PersistentFlags().BoolVarP(&useCompression, "compression", "c", false, "enable compression")
	createCmd.PersistentFlags().IntVarP(&chunkSize, "chunk-size", "", defaultChunkSize, "Chunk size in bytes")
}

const (
	defaultChunkSize = 1024 * 1024 * 4
	minChunkSize     = 1024 * 64
)

var (
	arch           *archive.Archive
	archiveFile    string
	useCompression bool
	verbose        bool
	chunkSize      int
)

// RootCmd is the main command that is always executed.
var RootCmd = &cobra.Command{
	Use: "",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() == "help" {
			return
		}

		if len(archiveFile) == 0 {
			exitWithErr(errNoArchiveFile)
		}
		archiveFile = fixArchivePath(archiveFile)

		if cmd.Name() == "create" {
			if archiveExists(archiveFile) {
				exitWithErr(errArchiveExists)
			}
		} else {
			if !archiveExists(archiveFile) {
				exitWithErr(errArchiveDoesNotExist)
			}
		}

		if chunkSize < minChunkSize {
			exitWithErr(errInvalidChunkSize)
		}

		envPwd := os.Getenv("PASSWORD")
		password := []byte(envPwd)
		if len(password) == 0 {
			password = readPassword("Password")

			if cmd.Name() == "create" {
				pwdRepeat := readPassword("Repeat password")
				if !bytes.Equal(password, pwdRepeat) {
					exitWithErr(errPWDoNotMatch)
				}
			}
		}

		config := config.Config{
			Path:        archiveFile,
			Password:    password,
			Compression: useCompression,
			ChunkSize:   chunkSize,
		}

		var err error
		arch, err = archive.NewArchive(&config)
		if err != nil {
			exitWithErr(err)
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if arch != nil {
			arch.Close()
		}
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an archive from the given files",
	Example: `create -cf foo_compressed.star /home/bar
create -f foo_uncompressed.star /home/bar/baz.txt
create -cf foo_uncompressed.star --chunk-size 4 /home/bar/baz.txt`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()

		path := args[0]
		if !strings.HasPrefix(path, "/") {
			path = filepath.Join(cwd, path)
		}

		var ch chan string
		if verbose {
			ch = make(chan string)
			go func() {
				for {
					select {
					case p := <-ch:
						fmt.Printf("+ %s\n", p)
					}
				}
			}()
		}

		basePath := filepath.Dir(path)
		arch.AddRecursive(basePath, path, ch)
	},
}

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all items in the archive",
	Example: "list -f foo.star <filter expression>\nlist -f foo.star *.txt\nlist -f foo.star tmp*",
	Run: func(cmd *cobra.Command, args []string) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		ch := make(chan *item.Item)
		go func() {
			for {
				i, more := <-ch
				if more {
					fmt.Println(i.Header.ToString())
				} else {
					wg.Done()
					return
				}
			}
		}()

		pattern := ""
		if len(args) > 0 {
			pattern = args[0]
		}
		if err := arch.List(ch, pattern); err != nil {
			exitWithErr(err)
		}

		wg.Wait()
	},
}

var extractCmd = &cobra.Command{
	Use:     "extract",
	Short:   "Extract an archive to a given location",
	Example: "extract -f foo.star /home/bar",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		ch := make(chan *item.Item)
		go func() {
			for {
				i, more := <-ch
				if more {
					if verbose {
						fmt.Println(i.Header.ToString())
					}
				} else {
					wg.Done()
					return
				}
			}
		}()

		cwd, _ := os.Getwd()

		path := args[0]
		if !strings.HasPrefix(path, "/") {
			path = filepath.Join(cwd, path)
		}

		if err := arch.Extract(ch, path); err != nil {
			exitWithErr(err)
		}

		wg.Wait()
	},
}

var addCmd = &cobra.Command{
	Use:     "add",
	Short:   "Add files to the archive",
	Example: "add -f foo.star /home/bar/baz.txt\nadd -f foo.star /home/blah",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()

		path := args[0]
		if !strings.HasPrefix(path, "/") {
			path = filepath.Join(cwd, path)
		}

		basePath := path
		stat, err := os.Stat(basePath)
		if err != nil {
			exitWithErr(errInvalidPath)
		}
		if !stat.Mode().IsDir() {
			basePath = filepath.Dir(basePath)
		}

		var ch chan string
		if verbose {
			ch = make(chan string)
			go func() {
				for {
					select {
					case p := <-ch:
						fmt.Printf("+ %s\n", p)
					}
				}
			}()
		}

		arch.AddRecursive(basePath, path, ch)
	},
}

var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete items from the archive",
	Example: "delete -f foo.star <filter expression>\ndelete -f foo.star home/bar/baz.txt\ndelete -f foo.star home/bar/blah*",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		ch := make(chan *item.Item)
		go func() {
			for {
				i, more := <-ch
				if more {
					if verbose {
						fmt.Println(i.Header.ToString())
					}
				} else {
					wg.Done()
					return
				}
			}
		}()

		pattern := args[0]
		if err := arch.Delete(ch, pattern); err != nil {
			exitWithErr(err)
		}

		wg.Wait()
	},
}

var moveCmd = &cobra.Command{
	Use:     "move <source pattern> <target>",
	Short:   "Move item(s) to another path.",
	Long:    "For a single item you have to specify the full target path. For multiple items you need to specify a prefix where all items are moved to.",
	Example: "Single file: move -f foo.star bar/baz.txt bam/baz.txt\nMultiple files: move -f foo.star bar/* bam",
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		ch := make(chan *item.Item)
		go func() {
			for {
				i, more := <-ch
				if more {
					if verbose {
						fmt.Println(i.Header.ToString())
					}
				} else {
					wg.Done()
					return
				}
			}
		}()

		src := args[0]
		target := args[1]
		if err := arch.Move(ch, src, target); err != nil {
			exitWithErr(err)
		}

		wg.Wait()
	},
}

var compactCmd = &cobra.Command{
	Use:     "compact",
	Short:   "Remove deleted items from the archive",
	Example: "compact -f foo.star",
	Run: func(cmd *cobra.Command, args []string) {
		if err := arch.Compact(); err != nil {
			exitWithErr(err)
		}
	},
}

var serveCmd = &cobra.Command{
	Use:     "serve",
	Short:   "Serve serves the archive using the integrated webserver",
	Example: "serve -f foo.star",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Open browser at http://localhost:1337/ to view archive.\nPress CTRL/CMD+C to quit...")
		if _, err := server.Start(arch); err != nil {
			exitWithErr(err)
		}
	},
}

var updatePwdCmd = &cobra.Command{
	Use:     "update-password",
	Short:   "Change the password of the archive",
	Example: "update-password -f foo.star",
	Run: func(cmd *cobra.Command, args []string) {
		newPwd := readPassword("New password")
		pwdRepeat := readPassword("Repeat new password")
		if !bytes.Equal(newPwd, pwdRepeat) {
			exitWithErr(errPWDoNotMatch)
		}
		if err := arch.UpdatePassword(newPwd); err != nil {
			exitWithErr(err)
		}
	},
}

func readPassword(desc string) []byte {
	fmt.Printf("%s: ", desc)
	key, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil
	}
	fmt.Println("")

	return key
}

func archiveExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func fixArchivePath(path string) string {
	if !strings.HasPrefix(archiveFile, "/") {
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, archiveFile)
	}

	return path
}

func exitWithErr(err error) {
	fmt.Printf("Error: %s\n", err.Error())
	os.Exit(1)
}

var (
	errNoArchiveFile       = errors.New("Archive file not specified")
	errArchiveExists       = errors.New("Archive file already exist")
	errArchiveDoesNotExist = errors.New("Archive file does not exist")
	errPWDoNotMatch        = errors.New("Passwords do not match")
	errInvalidChunkSize    = errors.New("Chunk size smaller than 64kb")
	errInvalidPath         = errors.New("Invalid path")
)
