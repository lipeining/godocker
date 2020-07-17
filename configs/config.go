package configs

type Rlimit struct {
	Type int    `json:"type"`
	Hard uint64 `json:"hard"`
	Soft uint64 `json:"soft"`
}

// IDMap represents UID/GID Mappings for User Namespaces.
type IDMap struct {
	ContainerID int `json:"container_id"`
	HostID      int `json:"host_id"`
	Size        int `json:"size"`
}

// Config defines configuration options for executing a process inside a contained environment.
type Config struct {
	// NoPivotRoot will use MS_MOVE and a chroot to jail the process into the container's rootfs
	// This is a common option when the container is running in ramdisk
	NoPivotRoot bool `json:"no_pivot_root"`

	// ParentDeathSignal specifies the signal that is sent to the container's process in the case
	// that the parent process dies.
	ParentDeathSignal int `json:"parent_death_signal"`

	// Path to a directory containing the container's root filesystem.
	Rootfs string `json:"rootfs"`

	// Readonlyfs will remount the container's rootfs as readonly where only externally mounted
	// bind mounts are writtable.
	Readonlyfs bool `json:"readonlyfs"`

	// Specifies the mount propagation flags to be applied to /.
	RootPropagation int `json:"rootPropagation"`

	// Mounts specify additional source and destination paths that will be mounted inside the container's
	// rootfs and mount namespace if specified
	Mounts []*Mount `json:"mounts"`

	MountLabel string `json:"mount_label"`

	// Hostname optionally sets the container's hostname if provided
	Hostname string `json:"hostname"`

	// Namespaces specifies the container's namespaces that it should setup when cloning the init process
	// If a namespace is not provided that namespace is shared from the container's parent process
	Namespaces Namespaces `json:"namespaces"`

	// Networks specifies the container's network setup to be created
	Networks []*Network `json:"networks"`

	// Routes can be specified to create entries in the route table as the container is started
	Routes []*Route `json:"routes"`

	// Cgroups specifies specific cgroup settings for the various subsystems that the container is
	// placed into to limit the resources the container has available
	Cgroups *Cgroup `json:"cgroups"`

	// ProcessLabel specifies the label to apply to the process running in the container.  It is
	// commonly used by selinux
	ProcessLabel string `json:"process_label,omitempty"`

	// Rlimits specifies the resource limits, such as max open files, to set in the container
	// If Rlimits are not set, the container will inherit rlimits from the parent process
	Rlimits []Rlimit `json:"rlimits,omitempty"`

	// UidMappings is an array of User ID mappings for User Namespaces
	UidMappings []IDMap `json:"uid_mappings"`

	// GidMappings is an array of Group ID mappings for User Namespaces
	GidMappings []IDMap `json:"gid_mappings"`

	// MaskPaths specifies paths within the container's rootfs to mask over with a bind
	// mount pointing to /dev/null as to prevent reads of the file.
	MaskPaths []string `json:"mask_paths"`

	// ReadonlyPaths specifies paths within the container's rootfs to remount as read-only
	// so that these files prevent any writes.
	ReadonlyPaths []string `json:"readonly_paths"`

	// Sysctl is a map of properties and their values. It is the equivalent of using
	// sysctl -w my.property.name value in Linux.
	Sysctl map[string]string `json:"sysctl"`

	// NoNewPrivileges controls whether processes in the container can gain additional privileges.
	NoNewPrivileges bool `json:"no_new_privileges,omitempty"`

	// Version is the version of opencontainer specification that is supported.
	Version string `json:"version"`

	// Labels are user defined metadata that is stored in the config and populated on the state
	Labels []string `json:"labels"`

	// NoNewKeyring will not allocated a new session keyring for the container.  It will use the
	// callers keyring in this case.
	NoNewKeyring bool `json:"no_new_keyring"`

	// RootlessEUID is set when the runc was launched with non-zero EUID.
	// Note that RootlessEUID is set to false when launched with EUID=0 in userns.
	// When RootlessEUID is set, runc creates a new userns for the container.
	// (config.json needs to contain userns settings)
	RootlessEUID bool `json:"rootless_euid,omitempty"`

	// RootlessCgroups is set when unlikely to have the full access to cgroups.
	// When RootlessCgroups is set, cgroups errors are ignored.
	RootlessCgroups bool `json:"rootless_cgroups,omitempty"`
}
