// License: GPLv3 Copyright: 2022, Kovid Goyal, <kovid at kovidgoyal.net>

package shm

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"kitty/tools/cli"

	"golang.org/x/sys/unix"
)

var _ = fmt.Print
var ErrPatternHasSeparator = errors.New("The specified pattern has file path separators in it")
var ErrPatternTooLong = errors.New("The specified pattern for the SHM name is too long")

type ErrNotSupported struct {
	err error
}

func (self *ErrNotSupported) Error() string {
	return fmt.Sprintf("POSIX shared memory not supported on this platform: with underlying error: %v", self.err)
}

// prefix_and_suffix splits pattern by the last wildcard "*", if applicable,
// returning prefix as the part before "*" and suffix as the part after "*".
func prefix_and_suffix(pattern string) (prefix, suffix string, err error) {
	for i := 0; i < len(pattern); i++ {
		if os.IsPathSeparator(pattern[i]) {
			return "", "", ErrPatternHasSeparator
		}
	}
	if pos := strings.LastIndexByte(pattern, '*'); pos != -1 {
		prefix, suffix = pattern[:pos], pattern[pos+1:]
	} else {
		prefix = pattern
	}
	return prefix, suffix, nil
}

type MMap interface {
	Close() error
	Unlink() error
	Slice() []byte
	Name() string
	IsFileSystemBacked() bool
	FileSystemName() string
	Stat() (fs.FileInfo, error)
	Flush() error
}

type AccessFlags int

const (
	READ AccessFlags = iota
	WRITE
	COPY
)

func mmap(sz int, access AccessFlags, fd int, off int64) ([]byte, error) {
	flags := unix.MAP_SHARED
	prot := unix.PROT_READ
	switch access {
	case COPY:
		prot |= unix.PROT_WRITE
		flags = unix.MAP_PRIVATE
	case WRITE:
		prot |= unix.PROT_WRITE
	}

	b, err := unix.Mmap(fd, off, sz, prot, flags)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func munmap(s []byte) error {
	return unix.Munmap(s)
}

func CreateTemp(pattern string, size uint64) (MMap, error) {
	return create_temp(pattern, size)
}

func truncate_or_unlink(ans *os.File, size uint64) (err error) {
	for {
		err = unix.Ftruncate(int(ans.Fd()), int64(size))
		if !errors.Is(err, unix.EINTR) {
			break
		}
	}
	if err != nil {
		ans.Close()
		os.Remove(ans.Name())
		return fmt.Errorf("Failed to ftruncate() SHM file %s to size: %d with error: %w", ans.Name(), size, err)
	}
	return
}

const NUM_BYTES_FOR_SIZE = 4

func WriteWithSize(self MMap, b []byte, at int) error {
	if len(self.Slice()) < at+len(b)+NUM_BYTES_FOR_SIZE {
		return io.ErrShortBuffer
	}
	binary.BigEndian.PutUint32(self.Slice()[at:], uint32(len(b)))
	copy(self.Slice()[at+NUM_BYTES_FOR_SIZE:], b)
	return nil
}

func ReadWithSize(self MMap, at int) ([]byte, error) {
	s := self.Slice()[at:]
	if len(s) < NUM_BYTES_FOR_SIZE {
		return nil, io.ErrShortBuffer
	}
	size := int(binary.BigEndian.Uint32(self.Slice()[at : at+NUM_BYTES_FOR_SIZE]))
	s = s[NUM_BYTES_FOR_SIZE:]
	if len(s) < size {
		return nil, io.ErrShortBuffer
	}
	return s[:size], nil
}

func ReadWithSizeAndUnlink(name string, file_callback ...func(fs.FileInfo) error) ([]byte, error) {
	mmap, err := Open(name, 0)
	if err != nil {
		return nil, err
	}
	if len(file_callback) > 0 {
		s, err := mmap.Stat()
		if err != nil {
			return nil, fmt.Errorf("Failed to stat SHM file with error: %w", err)
		}
		for _, f := range file_callback {
			err = f(s)
			if err != nil {
				return nil, err
			}
		}
	}
	defer func() {
		mmap.Close()
		mmap.Unlink()
	}()
	slice, err := ReadWithSize(mmap, 0)
	if err != nil {
		return nil, err
	}
	ans := make([]byte, len(slice))
	copy(ans, slice)
	return ans, nil
}

func test_integration_with_python(args []string) (rc int, err error) {
	switch args[0] {
	default:
		return 1, fmt.Errorf("Unknown test type: %s", args[0])
	case "read":
		data, err := ReadWithSizeAndUnlink(args[1])
		if err != nil {
			return 1, err
		}
		_, err = os.Stdout.Write(data)
		if err != nil {
			return 1, err
		}
	case "write":
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return 1, err
		}
		mmap, err := CreateTemp("shmtest-", uint64(len(data)+NUM_BYTES_FOR_SIZE))
		if err != nil {
			return 1, err
		}
		WriteWithSize(mmap, data, 0)
		mmap.Close()
		fmt.Println(mmap.Name())
	}
	return 0, nil
}

func TestEntryPoint(root *cli.Command) {
	root.AddSubCommand(&cli.Command{
		Name:            "shm",
		OnlyArgsAllowed: true,
		Run: func(cmd *cli.Command, args []string) (rc int, err error) {
			return test_integration_with_python(args)
		},
	})

}
