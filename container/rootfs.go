package container

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// pivotRoot will call pivot_root such that rootfs becomes the new root
// filesystem, and everything else is cleaned up.
func pivotRoot(rootfs string) error {
	// While the documentation may claim otherwise, pivot_root(".", ".") is
	// actually valid. What this results in is / being the new root but
	// /proc/self/cwd being the old root. Since we can play around with the cwd
	// with pivot_root this allows us to pivot without creating directories in
	// the rootfs. Shout-outs to the LXC developers for giving us this idea.

	oldroot, err := unix.Open("/", unix.O_DIRECTORY|unix.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer unix.Close(oldroot)

	newroot, err := unix.Open(rootfs, unix.O_DIRECTORY|unix.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer unix.Close(newroot)

	// Change to the new root so that the pivot_root actually acts on it.
	if err := unix.Fchdir(newroot); err != nil {
		return err
	}

	if err := unix.PivotRoot(".", "."); err != nil {
		return fmt.Errorf("pivot_root %s", err)
	}

	// Currently our "." is oldroot (according to the current kernel code).
	// However, purely for safety, we will fchdir(oldroot) since there isn't
	// really any guarantee from the kernel what /proc/self/cwd will be after a
	// pivot_root(2).

	if err := unix.Fchdir(oldroot); err != nil {
		return err
	}

	// Make oldroot rslave to make sure our unmounts don't propagate to the
	// host (and thus bork the machine). We don't use rprivate because this is
	// known to cause issues due to races where we still have a reference to a
	// mount while a process in the host namespace are trying to operate on
	// something they think has no mounts (devicemapper in particular).
	if err := unix.Mount("", ".", "", unix.MS_SLAVE|unix.MS_REC, ""); err != nil {
		return err
	}
	// Preform the unmount. MNT_DETACH allows us to unmount /proc/self/cwd.
	if err := unix.Unmount(".", unix.MNT_DETACH); err != nil {
		return err
	}

	// Switch back to our shiny new root.
	if err := unix.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / %s", err)
	}
	return nil
}

func chroot() error {
	if err := unix.Chroot("."); err != nil {
		return err
	}
	return unix.Chdir("/")
}

// createIfNotExists creates a file or a directory only if it does not already exist.
func createIfNotExists(path string, isDir bool) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if isDir {
				return os.MkdirAll(path, 0755)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(path, os.O_CREATE, 0755)
			if err != nil {
				return err
			}
			f.Close()
		}
	}
	return nil
}

// readonlyPath will make a path read only.
func readonlyPath(path string) error {
	if err := unix.Mount(path, path, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return unix.Mount(path, path, "", unix.MS_BIND|unix.MS_REMOUNT|unix.MS_RDONLY|unix.MS_REC, "")
}

// // remountReadonly will remount an existing mount point and ensure that it is read-only.
// func remountReadonly(m *configs.Mount) error {
// 	var (
// 		dest  = m.Destination
// 		flags = m.Flags
// 	)
// 	for i := 0; i < 5; i++ {
// 		// There is a special case in the kernel for
// 		// MS_REMOUNT | MS_BIND, which allows us to change only the
// 		// flags even as an unprivileged user (i.e. user namespace)
// 		// assuming we don't drop any security related flags (nodev,
// 		// nosuid, etc.). So, let's use that case so that we can do
// 		// this re-mount without failing in a userns.
// 		flags |= unix.MS_REMOUNT | unix.MS_BIND | unix.MS_RDONLY
// 		if err := unix.Mount("", dest, "", uintptr(flags), ""); err != nil {
// 			switch err {
// 			case unix.EBUSY:
// 				time.Sleep(100 * time.Millisecond)
// 				continue
// 			default:
// 				return err
// 			}
// 		}
// 		return nil
// 	}
// 	return fmt.Errorf("unable to mount %s as readonly max retries reached", dest)
// }

// maskPath masks the top of the specified path inside a container to avoid
// security issues from processes reading information from non-namespace aware
// mounts ( proc/kcore ).
// For files, maskPath bind mounts /dev/null over the top of the specified path.
// For directories, maskPath mounts read-only tmpfs over the top of the specified path.
func maskPath(path string, mountLabel string) error {
	if err := unix.Mount("/dev/null", path, "", unix.MS_BIND, ""); err != nil && !os.IsNotExist(err) {
		if err == unix.ENOTDIR {
			return unix.Mount("tmpfs", path, "tmpfs", unix.MS_RDONLY, "")
		}
		return err
	}
	return nil
}
