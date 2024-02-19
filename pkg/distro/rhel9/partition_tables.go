package rhel9

import (
	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/arch"
	"github.com/osbuild/images/pkg/disk"
)

func defaultBasePartitionTables(t *imageType) (disk.PartitionTable, bool) {
	var bootSize uint64
	switch {
	case common.VersionLessThan(t.arch.distro.osVersion, "9.3") && t.arch.distro.isRHEL():
		// RHEL <= 9.2 had only 500 MiB /boot
		bootSize = 500 * common.MebiByte
	case common.VersionLessThan(t.arch.distro.osVersion, "9.4") && t.arch.distro.isRHEL():
		// RHEL 9.3 had 600 MiB /boot, see RHEL-7999
		bootSize = 600 * common.MebiByte
	default:
		// RHEL >= 9.4 needs to have even a bigger /boot, see COMPOSER-2155
		bootSize = 1 * common.GibiByte
	}

	switch t.platform.GetArch() {
	case arch.ARCH_X86_64:
		return disk.PartitionTable{
			UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
			Type: "gpt",
			Partitions: []disk.Partition{
				{
					Size:     1 * common.MebiByte,
					Bootable: true,
					Type:     disk.BIOSBootPartitionGUID,
					UUID:     disk.BIOSBootPartitionUUID,
				},
				{
					Size: 200 * common.MebiByte,
					Type: disk.EFISystemPartitionGUID,
					UUID: disk.EFISystemPartitionUUID,
					Payload: &disk.Filesystem{
						Type:         "vfat",
						UUID:         disk.EFIFilesystemUUID,
						Mountpoint:   "/boot/efi",
						Label:        "EFI-SYSTEM",
						FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
						FSTabFreq:    0,
						FSTabPassNo:  2,
					},
				},
				{
					Size: bootSize,
					Type: disk.XBootLDRPartitionGUID,
					UUID: disk.FilesystemDataUUID,
					Payload: &disk.Filesystem{
						Type:         "xfs",
						Mountpoint:   "/boot",
						Label:        "boot",
						FSTabOptions: "defaults",
						FSTabFreq:    0,
						FSTabPassNo:  0,
					},
				},
				{
					Size: 2 * common.GibiByte,
					Type: disk.FilesystemDataGUID,
					UUID: disk.RootPartitionUUID,
					Payload: &disk.Filesystem{
						Type:         "xfs",
						Label:        "root",
						Mountpoint:   "/",
						FSTabOptions: "defaults",
						FSTabFreq:    0,
						FSTabPassNo:  0,
					},
				},
			},
		}, true
	case arch.ARCH_AARCH64:
		return disk.PartitionTable{
			UUID: "D209C89E-EA5E-4FBD-B161-B461CCE297E0",
			Type: "gpt",
			Partitions: []disk.Partition{
				{
					Size: 200 * common.MebiByte,
					Type: disk.EFISystemPartitionGUID,
					UUID: disk.EFISystemPartitionUUID,
					Payload: &disk.Filesystem{
						Type:         "vfat",
						UUID:         disk.EFIFilesystemUUID,
						Mountpoint:   "/boot/efi",
						Label:        "EFI-SYSTEM",
						FSTabOptions: "defaults,uid=0,gid=0,umask=077,shortname=winnt",
						FSTabFreq:    0,
						FSTabPassNo:  2,
					},
				},
				{
					Size: bootSize,
					Type: disk.XBootLDRPartitionGUID,
					UUID: disk.FilesystemDataUUID,
					Payload: &disk.Filesystem{
						Type:         "xfs",
						Mountpoint:   "/boot",
						Label:        "boot",
						FSTabOptions: "defaults",
						FSTabFreq:    0,
						FSTabPassNo:  0,
					},
				},
				{
					Size: 2 * common.GibiByte,
					Type: disk.FilesystemDataGUID,
					UUID: disk.RootPartitionUUID,
					Payload: &disk.Filesystem{
						Type:         "xfs",
						Label:        "root",
						Mountpoint:   "/",
						FSTabOptions: "defaults",
						FSTabFreq:    0,
						FSTabPassNo:  0,
					},
				},
			},
		}, true
	case arch.ARCH_PPC64LE:
		return disk.PartitionTable{
			UUID: "0x14fc63d2",
			Type: "dos",
			Partitions: []disk.Partition{
				{
					Size:     4 * common.MebiByte,
					Type:     "41",
					Bootable: true,
				},
				{
					Size: bootSize,
					Payload: &disk.Filesystem{
						Type:         "xfs",
						Mountpoint:   "/boot",
						Label:        "boot",
						FSTabOptions: "defaults",
						FSTabFreq:    0,
						FSTabPassNo:  0,
					},
				},
				{
					Size: 2 * common.GibiByte,
					Payload: &disk.Filesystem{
						Type:         "xfs",
						Mountpoint:   "/",
						FSTabOptions: "defaults",
						FSTabFreq:    0,
						FSTabPassNo:  0,
					},
				},
			},
		}, true

	case arch.ARCH_S390X:
		return disk.PartitionTable{
			UUID: "0x14fc63d2",
			Type: "dos",
			Partitions: []disk.Partition{
				{
					Size: bootSize,
					Payload: &disk.Filesystem{
						Type:         "xfs",
						Mountpoint:   "/boot",
						Label:        "boot",
						FSTabOptions: "defaults",
						FSTabFreq:    0,
						FSTabPassNo:  0,
					},
				},
				{
					Size:     2 * common.GibiByte,
					Bootable: true,
					Payload: &disk.Filesystem{
						Type:         "xfs",
						Mountpoint:   "/",
						FSTabOptions: "defaults",
						FSTabFreq:    0,
						FSTabPassNo:  0,
					},
				},
			},
		}, true

	default:
		return disk.PartitionTable{}, false
	}
}
