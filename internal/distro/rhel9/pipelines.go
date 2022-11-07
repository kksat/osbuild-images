package rhel9

import (
	"fmt"
	"math/rand"
	"path"
	"strings"

	"github.com/osbuild/osbuild-composer/internal/blueprint"
	"github.com/osbuild/osbuild-composer/internal/common"
	"github.com/osbuild/osbuild-composer/internal/container"
	"github.com/osbuild/osbuild-composer/internal/disk"
	"github.com/osbuild/osbuild-composer/internal/distro"
	"github.com/osbuild/osbuild-composer/internal/osbuild"
	"github.com/osbuild/osbuild-composer/internal/rpmmd"
	"github.com/osbuild/osbuild-composer/internal/users"
)

func prependKernelCmdlineStage(pipeline *osbuild.Pipeline, kernelOptions string, pt *disk.PartitionTable) *osbuild.Pipeline {
	rootFs := pt.FindMountable("/")
	if rootFs == nil {
		panic("root filesystem must be defined for kernel-cmdline stage, this is a programming error")
	}
	rootFsUUID := rootFs.GetFSSpec().UUID
	kernelStage := osbuild.NewKernelCmdlineStage(osbuild.NewKernelCmdlineStageOptions(rootFsUUID, kernelOptions))
	pipeline.Stages = append([]*osbuild.Stage{kernelStage}, pipeline.Stages...)
	return pipeline
}

func tarPipelines(t *imageType, customizations *blueprint.Customizations, options distro.ImageOptions, repos []rpmmd.RepoConfig, packageSetSpecs map[string][]rpmmd.PackageSpec, containers []container.Spec, rng *rand.Rand) ([]osbuild.Pipeline, error) {
	pipelines := make([]osbuild.Pipeline, 0)
	pipelines = append(pipelines, *buildPipeline(repos, packageSetSpecs[buildPkgsKey], t.arch.distro.runner.String()))

	treePipeline, err := osPipeline(t, repos, packageSetSpecs[osPkgsKey], containers, customizations, options, nil)
	if err != nil {
		return nil, err
	}
	pipelines = append(pipelines, *treePipeline)
	tarPipeline := tarArchivePipeline("root-tar", treePipeline.Name, &osbuild.TarStageOptions{Filename: "root.tar.xz"})
	pipelines = append(pipelines, *tarPipeline)
	return pipelines, nil
}

//makeISORootPath return a path that can be used to address files and folders in
//the root of the iso
func makeISORootPath(p string) string {
	fullpath := path.Join("/run/install/repo", p)
	return fmt.Sprintf("file://%s", fullpath)
}

func imageInstallerPipelines(t *imageType, customizations *blueprint.Customizations, options distro.ImageOptions, repos []rpmmd.RepoConfig, packageSetSpecs map[string][]rpmmd.PackageSpec, containers []container.Spec, rng *rand.Rand) ([]osbuild.Pipeline, error) {
	pipelines := make([]osbuild.Pipeline, 0)
	pipelines = append(pipelines, *buildPipeline(repos, packageSetSpecs[buildPkgsKey], t.arch.distro.runner.String()))

	treePipeline, err := osPipeline(t, repos, packageSetSpecs[osPkgsKey], containers, customizations, options, nil)
	if err != nil {
		return nil, err
	}
	pipelines = append(pipelines, *treePipeline)

	var kernelPkg *rpmmd.PackageSpec
	installerPackages := packageSetSpecs[installerPkgsKey]
	for _, pkg := range installerPackages {
		if pkg.Name == "kernel" {
			// Implicit memory alasing doesn't couse any bug in this case
			/* #nosec G601 */
			kernelPkg = &pkg
			break
		}
	}
	if kernelPkg == nil {
		return nil, fmt.Errorf("kernel package not found in installer package set")
	}
	kernelVer := fmt.Sprintf("%s-%s.%s", kernelPkg.Version, kernelPkg.Release, kernelPkg.Arch)

	tarPath := "/liveimg.tar"
	tarPayloadStages := []*osbuild.Stage{osbuild.NewTarStage(&osbuild.TarStageOptions{Filename: tarPath}, treePipeline.Name)}
	kickstartOptions, err := osbuild.NewKickstartStageOptions(kspath, makeISORootPath(tarPath), users.UsersFromBP(customizations.GetUsers()), users.GroupsFromBP(customizations.GetGroups()), "", "", "rhel")
	if err != nil {
		return nil, err
	}
	archName := t.Arch().Name()
	d := t.arch.distro
	pipelines = append(pipelines, *anacondaTreePipeline(repos, installerPackages, kernelVer, archName, d.product, d.osVersion, "BaseOS", true))
	isolabel := fmt.Sprintf(d.isolabelTmpl, archName)
	pipelines = append(pipelines, *bootISOTreePipeline(kernelVer, archName, d.vendor, d.product, d.osVersion, isolabel, kickstartOptions, tarPayloadStages))
	pipelines = append(pipelines, *bootISOPipeline(t.Filename(), d.isolabelTmpl, t.Arch().Name(), t.Arch().Name() == "x86_64"))
	return pipelines, nil
}

func edgeImagePipelines(t *imageType, customizations *blueprint.Customizations, filename string, options distro.ImageOptions, rng *rand.Rand) ([]osbuild.Pipeline, string, error) {
	pipelines := make([]osbuild.Pipeline, 0)
	ostreeRepoPath := "/ostree/repo"
	imgName := "image.raw"

	partitionTable, err := t.getPartitionTable(nil, options, rng)
	if err != nil {
		return nil, "", err
	}

	// prepare ostree deployment tree
	treePipeline := ostreeDeployPipeline(t, partitionTable, ostreeRepoPath, rng, customizations, options)
	pipelines = append(pipelines, *treePipeline)

	// make raw image from tree
	imagePipeline := liveImagePipeline(treePipeline.Name, imgName, partitionTable, t.arch, "")
	pipelines = append(pipelines, *imagePipeline)

	// compress image
	xzPipeline := xzArchivePipeline(imagePipeline.Name, imgName, filename)
	pipelines = append(pipelines, *xzPipeline)

	return pipelines, xzPipeline.Name, nil
}

func buildPipeline(repos []rpmmd.RepoConfig, buildPackageSpecs []rpmmd.PackageSpec, runner string) *osbuild.Pipeline {
	p := new(osbuild.Pipeline)
	p.Name = "build"
	p.Runner = runner
	p.AddStage(osbuild.NewRPMStage(osbuild.NewRPMStageOptions(repos), osbuild.NewRpmStageSourceFilesInputs(buildPackageSpecs)))
	p.AddStage(osbuild.NewSELinuxStage(selinuxStageOptions(true)))
	return p
}

func osPipeline(t *imageType,
	repos []rpmmd.RepoConfig,
	packages []rpmmd.PackageSpec,
	containers []container.Spec,
	c *blueprint.Customizations,
	options distro.ImageOptions,
	pt *disk.PartitionTable) (*osbuild.Pipeline, error) {
	imageConfig := t.getDefaultImageConfig()
	p := new(osbuild.Pipeline)
	if t.rpmOstree {
		p.Name = "ostree-tree"
	} else {
		p.Name = "os"
	}
	p.Build = "name:build"

	if t.rpmOstree && options.OSTree.FetchChecksum != "" && options.OSTree.URL != "" {
		p.AddStage(osbuild.NewOSTreePasswdStage("org.osbuild.source", options.OSTree.FetchChecksum))
	}

	rpmOptions := osbuild.NewRPMStageOptions(repos)
	rpmOptions.GPGKeysFromTree = imageConfig.GPGKeyFiles

	if imageConfig.ExcludeDocs != nil && *imageConfig.ExcludeDocs {
		if rpmOptions.Exclude == nil {
			rpmOptions.Exclude = &osbuild.Exclude{}
		}
		rpmOptions.Exclude.Docs = true
	}
	p.AddStage(osbuild.NewRPMStage(rpmOptions, osbuild.NewRpmStageSourceFilesInputs(packages)))

	// If the /boot is on a separate partition, the prefix for the BLS stage must be ""
	if pt == nil || pt.FindMountable("/boot") == nil {
		p.AddStage(osbuild.NewFixBLSStage(&osbuild.FixBLSStageOptions{}))
	} else {
		p.AddStage(osbuild.NewFixBLSStage(&osbuild.FixBLSStageOptions{Prefix: common.StringToPtr("")}))
	}

	if len(containers) > 0 {
		images := osbuild.NewContainersInputForSources(containers)

		var storagePath string

		// OSTree commits do not include data in `/var` since that is tied to the
		// deployment, rather than the commit. Therefore the containers need to be
		// stored in a different location, like `/usr/share`, and the container
		// storage engine configured accordingly.
		if t.rpmOstree {
			storagePath = "/usr/share/containers/storage"
			storageConf := "/etc/containers/storage.conf"

			containerStoreOpts := osbuild.NewContainerStorageOptions(storageConf, storagePath)
			p.AddStage(osbuild.NewContainersStorageConfStage(containerStoreOpts))
		}

		skopeo := osbuild.NewSkopeoStage(images, storagePath)
		p.AddStage(skopeo)
	}

	language, keyboard := c.GetPrimaryLocale()
	if language != nil {
		p.AddStage(osbuild.NewLocaleStage(&osbuild.LocaleStageOptions{Language: *language}))
	} else if imageConfig.Locale != nil {
		p.AddStage(osbuild.NewLocaleStage(&osbuild.LocaleStageOptions{Language: *imageConfig.Locale}))
	}
	if keyboard != nil {
		p.AddStage(osbuild.NewKeymapStage(&osbuild.KeymapStageOptions{Keymap: *keyboard}))
	} else if imageConfig.Keyboard != nil {
		p.AddStage(osbuild.NewKeymapStage(imageConfig.Keyboard))
	}

	if hostname := c.GetHostname(); hostname != nil {
		p.AddStage(osbuild.NewHostnameStage(&osbuild.HostnameStageOptions{Hostname: *hostname}))
	}

	timezone, ntpServers := c.GetTimezoneSettings()
	if timezone != nil {
		p.AddStage(osbuild.NewTimezoneStage(&osbuild.TimezoneStageOptions{Zone: *timezone}))
	} else if imageConfig.Timezone != nil {
		p.AddStage(osbuild.NewTimezoneStage(&osbuild.TimezoneStageOptions{Zone: *imageConfig.Timezone}))
	}

	if len(ntpServers) > 0 {
		servers := make([]osbuild.ChronyConfigServer, len(ntpServers))
		for idx, server := range ntpServers {
			servers[idx] = osbuild.ChronyConfigServer{Hostname: server}
		}
		p.AddStage(osbuild.NewChronyStage(&osbuild.ChronyStageOptions{Servers: servers}))
	} else if imageConfig.TimeSynchronization != nil {
		p.AddStage(osbuild.NewChronyStage(imageConfig.TimeSynchronization))
	}

	if !t.bootISO {
		// don't put users and groups in the payload of an installer
		// add them via kickstart instead
		if groups := c.GetGroups(); len(groups) > 0 {
			p.AddStage(osbuild.NewGroupsStage(osbuild.NewGroupsStageOptions(users.GroupsFromBP(groups))))
		}

		if userOptions, err := osbuild.NewUsersStageOptions(users.UsersFromBP(c.GetUsers()), false); err != nil {
			return nil, err
		} else if userOptions != nil {
			if t.rpmOstree {
				// for ostree, writing the key during user creation is
				// redundant and can cause issues so create users without keys
				// and write them on first boot
				userOptionsSansKeys, err := osbuild.NewUsersStageOptions(users.UsersFromBP(c.GetUsers()), true)
				if err != nil {
					return nil, err
				}
				p.AddStage(osbuild.NewUsersStage(userOptionsSansKeys))
				p.AddStage(osbuild.NewFirstBootStage(usersFirstBootOptions(userOptions)))
			} else {
				p.AddStage(osbuild.NewUsersStage(userOptions))
			}
		}
	}

	if services := c.GetServices(); services != nil || imageConfig.EnabledServices != nil ||
		imageConfig.DisabledServices != nil || imageConfig.DefaultTarget != nil {
		defaultTarget := ""
		if imageConfig.DefaultTarget != nil {
			defaultTarget = *imageConfig.DefaultTarget
		}
		p.AddStage(osbuild.NewSystemdStage(systemdStageOptions(
			imageConfig.EnabledServices,
			imageConfig.DisabledServices,
			services,
			defaultTarget,
		)))
	}

	var fwStageOptions *osbuild.FirewallStageOptions
	if firewallCustomization := c.GetFirewall(); firewallCustomization != nil {
		fwStageOptions = firewallStageOptions(firewallCustomization)
	}
	if firewallConfig := imageConfig.Firewall; firewallConfig != nil {
		// merge the user-provided firewall config with the default one
		if fwStageOptions != nil {
			fwStageOptions = &osbuild.FirewallStageOptions{
				// Prefer the firewall ports and services settings provided
				// via BP customization.
				Ports:            fwStageOptions.Ports,
				EnabledServices:  fwStageOptions.EnabledServices,
				DisabledServices: fwStageOptions.DisabledServices,
				// Default zone can not be set using BP customizations, therefore
				// default to the one provided in the default image configuration.
				DefaultZone: firewallConfig.DefaultZone,
			}
		} else {
			fwStageOptions = firewallConfig
		}
	}
	if fwStageOptions != nil {
		p.AddStage(osbuild.NewFirewallStage(fwStageOptions))
	}

	for _, sysconfigConfig := range imageConfig.Sysconfig {
		p.AddStage(osbuild.NewSysconfigStage(sysconfigConfig))
	}

	if t.arch.distro.isRHEL() {
		if options.Subscription != nil {
			commands := []string{
				fmt.Sprintf("/usr/sbin/subscription-manager register --org=%s --activationkey=%s --serverurl %s --baseurl %s", options.Subscription.Organization, options.Subscription.ActivationKey, options.Subscription.ServerUrl, options.Subscription.BaseUrl),
			}
			if options.Subscription.Insights {
				commands = append(commands, "/usr/bin/insights-client --register")
			}
			p.AddStage(osbuild.NewFirstBootStage(&osbuild.FirstBootStageOptions{
				Commands:       commands,
				WaitForNetwork: true,
			}))

			if rhsmConfig, exists := imageConfig.RHSMConfig[distro.RHSMConfigWithSubscription]; exists {
				p.AddStage(osbuild.NewRHSMStage(rhsmConfig))
			}
		} else {
			if rhsmConfig, exists := imageConfig.RHSMConfig[distro.RHSMConfigNoSubscription]; exists {
				p.AddStage(osbuild.NewRHSMStage(rhsmConfig))
			}
		}
	}

	for _, systemdLogindConfig := range imageConfig.SystemdLogind {
		p.AddStage(osbuild.NewSystemdLogindStage(systemdLogindConfig))
	}

	for _, cloudInitConfig := range imageConfig.CloudInit {
		p.AddStage(osbuild.NewCloudInitStage(cloudInitConfig))
	}

	for _, modprobeConfig := range imageConfig.Modprobe {
		p.AddStage(osbuild.NewModprobeStage(modprobeConfig))
	}

	for _, dracutConfConfig := range imageConfig.DracutConf {
		p.AddStage(osbuild.NewDracutConfStage(dracutConfConfig))
	}

	for _, systemdUnitConfig := range imageConfig.SystemdUnit {
		p.AddStage(osbuild.NewSystemdUnitStage(systemdUnitConfig))
	}

	if authselectConfig := imageConfig.Authselect; authselectConfig != nil {
		p.AddStage(osbuild.NewAuthselectStage(authselectConfig))
	}

	if seLinuxConfig := imageConfig.SELinuxConfig; seLinuxConfig != nil {
		p.AddStage(osbuild.NewSELinuxConfigStage(seLinuxConfig))
	}

	if tunedConfig := imageConfig.Tuned; tunedConfig != nil {
		p.AddStage(osbuild.NewTunedStage(tunedConfig))
	}

	for _, tmpfilesdConfig := range imageConfig.Tmpfilesd {
		p.AddStage(osbuild.NewTmpfilesdStage(tmpfilesdConfig))
	}

	for _, pamLimitsConfConfig := range imageConfig.PamLimitsConf {
		p.AddStage(osbuild.NewPamLimitsConfStage(pamLimitsConfConfig))
	}

	for _, sysctldConfig := range imageConfig.Sysctld {
		p.AddStage(osbuild.NewSysctldStage(sysctldConfig))
	}

	for _, dnfConfig := range imageConfig.DNFConfig {
		p.AddStage(osbuild.NewDNFConfigStage(dnfConfig))
	}

	if sshdConfig := imageConfig.SshdConfig; sshdConfig != nil {
		p.AddStage((osbuild.NewSshdConfigStage(sshdConfig)))
	}

	if authConfig := imageConfig.Authconfig; authConfig != nil {
		p.AddStage(osbuild.NewAuthconfigStage(authConfig))
	}

	if pwQuality := imageConfig.PwQuality; pwQuality != nil {
		p.AddStage(osbuild.NewPwqualityConfStage(pwQuality))
	}

	if waConfig := imageConfig.WAAgentConfig; waConfig != nil {
		p.AddStage(osbuild.NewWAAgentConfStage(waConfig))
	}

	if dnfAutomaticConfig := imageConfig.DNFAutomaticConfig; dnfAutomaticConfig != nil {
		p.AddStage(osbuild.NewDNFAutomaticConfigStage(dnfAutomaticConfig))
	}

	for _, yumRepo := range imageConfig.YUMRepos {
		p.AddStage(osbuild.NewYumReposStage(yumRepo))
	}

	if gcpGuestAgentConfig := imageConfig.GCPGuestAgentConfig; gcpGuestAgentConfig != nil {
		p.AddStage(osbuild.NewGcpGuestAgentConfigStage(gcpGuestAgentConfig))
	}

	if udevRules := imageConfig.UdevRules; udevRules != nil {
		p.AddStage(osbuild.NewUdevRulesStage(udevRules))
	}

	if pt != nil {
		kernelOptions := osbuild.GenImageKernelOptions(pt)
		if t.kernelOptions != "" {
			kernelOptions = append(kernelOptions, t.kernelOptions)
		}
		if bpKernel := c.GetKernel(); bpKernel.Append != "" {
			kernelOptions = append(kernelOptions, bpKernel.Append)
		}
		p = prependKernelCmdlineStage(p, strings.Join(kernelOptions, " "), pt)
		p.AddStage(osbuild.NewFSTabStage(osbuild.NewFSTabStageOptions(pt)))
		kernelVer := rpmmd.GetVerStrFromPackageSpecListPanic(packages, c.GetKernel().Name)
		bootloader := bootloaderConfigStage(t, *pt, kernelVer, false, false)

		if cfg := imageConfig.Grub2Config; cfg != nil {
			if grub2, ok := bootloader.Options.(*osbuild.GRUB2StageOptions); ok {
				// grub2.Config.Default is owned and set by `NewGrub2StageOptionsUnified`
				// and thus we need to preserve it
				if grub2.Config != nil {
					cfg.Default = grub2.Config.Default
				}

				grub2.Config = cfg
			}
		}

		p.AddStage(bootloader)
	}

	if oscapConfig := c.GetOpenSCAP(); oscapConfig != nil {
		if t.rpmOstree {
			return nil, fmt.Errorf("unexpected oscap options for ostree image type")
		}
		remediationOptions := osbuild.NewOscapRemediationStageOptions(
			osbuild.OscapConfig{
				Datastream: oscapConfig.DataStream,
				ProfileID:  oscapConfig.ProfileID,
			},
		)
		p.AddStage(osbuild.NewOscapRemediationStage(remediationOptions))
	}

	if t.arch.distro.isRHEL() && options.Facts != nil {
		p.AddStage(osbuild.NewRHSMFactsStage(&osbuild.RHSMFactsStageOptions{
			Facts: osbuild.RHSMFacts{
				ApiType: options.Facts.ApiType,
			},
		}))
	}

	// Relabel the tree, unless the `NoSElinux` flag is explicitly set to `true`
	if imageConfig.NoSElinux == nil || imageConfig.NoSElinux != nil && !*imageConfig.NoSElinux {
		p.AddStage(osbuild.NewSELinuxStage(selinuxStageOptions(false)))
	}

	if t.rpmOstree {
		p.AddStage(osbuild.NewOSTreePrepTreeStage(&osbuild.OSTreePrepTreeStageOptions{
			EtcGroupMembers: []string{
				// NOTE: We may want to make this configurable.
				"wheel", "docker",
			},
		}))
	}

	return p, nil
}

func edgeSimplifiedInstallerPipelines(t *imageType, customizations *blueprint.Customizations, options distro.ImageOptions, repos []rpmmd.RepoConfig, packageSetSpecs map[string][]rpmmd.PackageSpec, containers []container.Spec, rng *rand.Rand) ([]osbuild.Pipeline, error) {
	pipelines := make([]osbuild.Pipeline, 0)
	pipelines = append(pipelines, *buildPipeline(repos, packageSetSpecs[buildPkgsKey], t.arch.distro.runner.String()))
	installerPackages := packageSetSpecs[installerPkgsKey]
	kernelVer := rpmmd.GetVerStrFromPackageSpecListPanic(installerPackages, "kernel")
	imgName := "disk.img.xz"
	installDevice := customizations.GetInstallationDevice()

	// create the raw image
	imagePipelines, imgPipelineName, err := edgeImagePipelines(t, customizations, imgName, options, rng)
	if err != nil {
		return nil, err
	}

	pipelines = append(pipelines, imagePipelines...)

	// create boot ISO with raw image
	d := t.arch.distro
	archName := t.Arch().Name()
	installerTreePipeline := simplifiedInstallerTreePipeline(repos, installerPackages, kernelVer, archName, d.product, d.osVersion, "edge", customizations.GetFDO())
	isolabel := fmt.Sprintf(d.isolabelTmpl, archName)
	efibootTreePipeline := simplifiedInstallerEFIBootTreePipeline(installDevice, kernelVer, archName, d.vendor, d.product, d.osVersion, isolabel, customizations.GetFDO())
	bootISOTreePipeline := simplifiedInstallerBootISOTreePipeline(imgPipelineName, kernelVer, rng)

	pipelines = append(pipelines, *installerTreePipeline, *efibootTreePipeline, *bootISOTreePipeline)
	pipelines = append(pipelines, *bootISOPipeline(t.Filename(), d.isolabelTmpl, t.Arch().Name(), false))

	return pipelines, nil
}

func simplifiedInstallerBootISOTreePipeline(archivePipelineName, kver string, rng *rand.Rand) *osbuild.Pipeline {
	p := new(osbuild.Pipeline)
	p.Name = "bootiso-tree"
	p.Build = "name:build"

	p.AddStage(osbuild.NewCopyStageSimple(
		&osbuild.CopyStageOptions{
			Paths: []osbuild.CopyStagePath{
				{
					From: "input://file/disk.img.xz",
					To:   "tree:///disk.img.xz",
				},
			},
		},
		osbuild.NewFilesInputs(osbuild.NewFilesInputReferencesPipeline(archivePipelineName, "disk.img.xz")),
	))

	p.AddStage(osbuild.NewMkdirStage(
		&osbuild.MkdirStageOptions{
			Paths: []osbuild.Path{
				{
					Path: "images",
				},
				{
					Path: "images/pxeboot",
				},
			},
		},
	))

	pt := disk.PartitionTable{
		Size: 20971520,
		Partitions: []disk.Partition{
			{
				Start: 0,
				Size:  20971520,
				Payload: &disk.Filesystem{
					Type:       "vfat",
					Mountpoint: "/",
					UUID:       disk.NewVolIDFromRand(rng),
				},
			},
		},
	}

	filename := "images/efiboot.img"
	loopback := osbuild.NewLoopbackDevice(&osbuild.LoopbackDeviceOptions{Filename: filename})
	p.AddStage(osbuild.NewTruncateStage(&osbuild.TruncateStageOptions{Filename: filename, Size: fmt.Sprintf("%d", pt.Size)}))

	for _, stage := range osbuild.GenMkfsStages(&pt, loopback) {
		p.AddStage(stage)
	}

	inputName := "root-tree"
	copyInputs := osbuild.NewPipelineTreeInputs(inputName, "efiboot-tree")
	copyOptions, copyDevices, copyMounts := osbuild.GenCopyFSTreeOptions(inputName, "efiboot-tree", filename, &pt)
	p.AddStage(osbuild.NewCopyStage(copyOptions, copyInputs, copyDevices, copyMounts))

	inputName = "coi"
	copyInputs = osbuild.NewPipelineTreeInputs(inputName, "coi-tree")
	p.AddStage(osbuild.NewCopyStageSimple(
		&osbuild.CopyStageOptions{
			Paths: []osbuild.CopyStagePath{
				{
					From: fmt.Sprintf("input://%s/boot/vmlinuz-%s", inputName, kver),
					To:   "tree:///images/pxeboot/vmlinuz",
				},
				{
					From: fmt.Sprintf("input://%s/boot/initramfs-%s.img", inputName, kver),
					To:   "tree:///images/pxeboot/initrd.img",
				},
			},
		},
		copyInputs,
	))

	inputName = "efi-tree"
	copyInputs = osbuild.NewPipelineTreeInputs(inputName, "efiboot-tree")
	p.AddStage(osbuild.NewCopyStageSimple(
		&osbuild.CopyStageOptions{
			Paths: []osbuild.CopyStagePath{
				{
					From: fmt.Sprintf("input://%s/EFI", inputName),
					To:   "tree:///",
				},
			},
		},
		copyInputs,
	))

	return p
}

func simplifiedInstallerEFIBootTreePipeline(installDevice, kernelVer, arch, vendor, product, osVersion, isolabel string, fdo *blueprint.FDOCustomization) *osbuild.Pipeline {
	p := new(osbuild.Pipeline)
	p.Name = "efiboot-tree"
	p.Build = "name:build"
	p.AddStage(osbuild.NewGrubISOStage(grubISOStageOptions(installDevice, kernelVer, arch, vendor, product, osVersion, isolabel, fdo)))
	return p
}

func simplifiedInstallerTreePipeline(repos []rpmmd.RepoConfig, packages []rpmmd.PackageSpec, kernelVer, arch, product, osVersion, variant string, fdo *blueprint.FDOCustomization) *osbuild.Pipeline {
	p := new(osbuild.Pipeline)
	p.Name = "coi-tree"
	p.Build = "name:build"
	p.AddStage(osbuild.NewRPMStage(osbuild.NewRPMStageOptions(repos), osbuild.NewRpmStageSourceFilesInputs(packages)))
	p.AddStage(osbuild.NewBuildstampStage(buildStampStageOptions(arch, product, osVersion, variant)))
	p.AddStage(osbuild.NewLocaleStage(&osbuild.LocaleStageOptions{Language: "C.UTF-8"}))
	dracutStageOptions := dracutStageOptions(kernelVer, arch, []string{
		"coreos-installer",
		"fdo",
	})
	if fdo.HasFDO() && fdo.DiunPubKeyRootCerts != "" {
		p.AddStage(osbuild.NewFDOStageForRootCerts(fdo.DiunPubKeyRootCerts))
		dracutStageOptions.Install = []string{"/fdo_diun_pub_key_root_certs.pem"}
	}
	p.AddStage(osbuild.NewDracutStage(dracutStageOptions))
	return p
}

func ostreeDeployPipeline(
	t *imageType,
	pt *disk.PartitionTable,
	repoPath string,
	rng *rand.Rand,
	c *blueprint.Customizations,
	options distro.ImageOptions,
) *osbuild.Pipeline {

	p := new(osbuild.Pipeline)
	p.Name = "image-tree"
	p.Build = "name:build"
	osname := "redhat"
	remote := "rhel-edge"

	p.AddStage(osbuild.OSTreeInitFsStage())
	p.AddStage(osbuild.NewOSTreePullStage(
		&osbuild.OSTreePullStageOptions{Repo: repoPath, Remote: remote},
		osbuild.NewOstreePullStageInputs("org.osbuild.source", options.OSTree.FetchChecksum, options.OSTree.ImageRef),
	))
	p.AddStage(osbuild.NewOSTreeOsInitStage(
		&osbuild.OSTreeOsInitStageOptions{
			OSName: osname,
		},
	))
	p.AddStage(osbuild.NewOSTreeConfigStage(ostreeConfigStageOptions(repoPath, false)))
	p.AddStage(osbuild.NewMkdirStage(efiMkdirStageOptions()))
	kernelOpts := osbuild.GenImageKernelOptions(pt)
	p.AddStage(osbuild.NewOSTreeDeployStage(
		&osbuild.OSTreeDeployStageOptions{
			OsName: osname,
			Ref:    options.OSTree.ImageRef,
			Remote: remote,
			Mounts: []string{"/boot", "/boot/efi"},
			Rootfs: osbuild.Rootfs{
				Label: "root",
			},
			KernelOpts: kernelOpts,
		},
	))

	if options.OSTree.URL != "" {
		p.AddStage(osbuild.NewOSTreeRemotesStage(
			&osbuild.OSTreeRemotesStageOptions{
				Repo: "/ostree/repo",
				Remotes: []osbuild.OSTreeRemote{
					{
						Name: remote,
						URL:  options.OSTree.URL,
					},
				},
			},
		))
	}

	p.AddStage(osbuild.NewOSTreeFillvarStage(
		&osbuild.OSTreeFillvarStageOptions{
			Deployment: osbuild.OSTreeDeployment{
				OSName: osname,
				Ref:    options.OSTree.ImageRef,
			},
		},
	))

	fstabOptions := osbuild.NewFSTabStageOptions(pt)
	fstabOptions.OSTree = &osbuild.OSTreeFstab{
		Deployment: osbuild.OSTreeDeployment{
			OSName: osname,
			Ref:    options.OSTree.ImageRef,
		},
	}
	p.AddStage(osbuild.NewFSTabStage(fstabOptions))

	if bpUsers := c.GetUsers(); len(bpUsers) > 0 {
		usersStage, err := osbuild.GenUsersStage(users.UsersFromBP(bpUsers), false)
		if err != nil {
			panic(err)
		}
		usersStage.MountOSTree(osname, options.OSTree.ImageRef, 0)
		p.AddStage(usersStage)
	}
	if bpGroups := c.GetGroups(); len(bpGroups) > 0 {
		groupsStage := osbuild.GenGroupsStage(users.GroupsFromBP(bpGroups))
		groupsStage.MountOSTree(osname, options.OSTree.ImageRef, 0)
		p.AddStage(groupsStage)
	}

	p.AddStage(bootloaderConfigStage(t, *pt, "", true, true))

	p.AddStage(osbuild.NewOSTreeSelinuxStage(
		&osbuild.OSTreeSelinuxStageOptions{
			Deployment: osbuild.OSTreeDeployment{
				OSName: osname,
				Ref:    options.OSTree.ImageRef,
			},
		},
	))
	return p
}

func anacondaTreePipeline(repos []rpmmd.RepoConfig, packages []rpmmd.PackageSpec, kernelVer, arch, product, osVersion, variant string, users bool) *osbuild.Pipeline {
	p := new(osbuild.Pipeline)
	p.Name = "anaconda-tree"
	p.Build = "name:build"
	p.AddStage(osbuild.NewRPMStage(osbuild.NewRPMStageOptions(repos), osbuild.NewRpmStageSourceFilesInputs(packages)))
	p.AddStage(osbuild.NewBuildstampStage(buildStampStageOptions(arch, product, osVersion, variant)))
	p.AddStage(osbuild.NewLocaleStage(&osbuild.LocaleStageOptions{Language: "en_US.UTF-8"}))

	rootPassword := ""
	rootUser := osbuild.UsersStageOptionsUser{
		Password: &rootPassword,
	}

	installUID := 0
	installGID := 0
	installHome := "/root"
	installShell := "/usr/libexec/anaconda/run-anaconda"
	installPassword := ""
	installUser := osbuild.UsersStageOptionsUser{
		UID:      &installUID,
		GID:      &installGID,
		Home:     &installHome,
		Shell:    &installShell,
		Password: &installPassword,
	}
	usersStageOptions := &osbuild.UsersStageOptions{
		Users: map[string]osbuild.UsersStageOptionsUser{
			"root":    rootUser,
			"install": installUser,
		},
	}

	p.AddStage(osbuild.NewUsersStage(usersStageOptions))
	p.AddStage(osbuild.NewAnacondaStage(osbuild.NewAnacondaStageOptions(users, []string{})))
	p.AddStage(osbuild.NewLoraxScriptStage(loraxScriptStageOptions(arch)))
	dso := dracutStageOptions(kernelVer, arch, []string{
		"anaconda",
		"fcoe",
		"fcoe-uefi",
		"iscsi",
		"lunmask",
		"multipath",
		"nfs",
		"nvdimm", // non-volatile DIMM firmware (provides nfit, cuse, and nd_e820)
		"rdma",
		"rngd",
	})
	// drivers / kernel modules to add explicitly for parity with the official
	// RHEL 9.0 ISO
	dso.AddDrivers = []string{"cuse", "ipmi_devintf", "ipmi_msghandler"}
	p.AddStage(osbuild.NewDracutStage(dso))
	p.AddStage(osbuild.NewSELinuxConfigStage(&osbuild.SELinuxConfigStageOptions{State: osbuild.SELinuxStatePermissive}))

	return p
}

func bootISOTreePipeline(kernelVer, arch, vendor, product, osVersion, isolabel string, ksOptions *osbuild.KickstartStageOptions, payloadStages []*osbuild.Stage) *osbuild.Pipeline {
	p := new(osbuild.Pipeline)
	p.Name = "bootiso-tree"
	p.Build = "name:build"

	p.AddStage(osbuild.NewBootISOMonoStage(bootISOMonoStageOptions(kernelVer, arch, vendor, product, osVersion, isolabel), osbuild.NewBootISOMonoStagePipelineTreeInputs("anaconda-tree")))
	p.AddStage(osbuild.NewKickstartStage(ksOptions))
	p.AddStage(osbuild.NewDiscinfoStage(discinfoStageOptions(arch)))

	for _, stage := range payloadStages {
		p.AddStage(stage)
	}

	return p
}
func bootISOPipeline(filename, isolabel, arch string, isolinux bool) *osbuild.Pipeline {
	p := new(osbuild.Pipeline)
	p.Name = "bootiso"
	p.Build = "name:build"

	p.AddStage(osbuild.NewXorrisofsStage(xorrisofsStageOptions(filename, isolabel, arch, isolinux), "bootiso-tree"))
	p.AddStage(osbuild.NewImplantisomd5Stage(&osbuild.Implantisomd5StageOptions{Filename: filename}))

	return p
}

func liveImagePipeline(inputPipelineName string, outputFilename string, pt *disk.PartitionTable, arch *architecture, kernelVer string) *osbuild.Pipeline {
	p := new(osbuild.Pipeline)
	p.Name = "image"
	p.Build = "name:build"

	for _, stage := range osbuild.GenImagePrepareStages(pt, outputFilename, osbuild.PTSfdisk) {
		p.AddStage(stage)
	}

	inputName := "root-tree"
	copyOptions, copyDevices, copyMounts := osbuild.GenCopyFSTreeOptions(inputName, inputPipelineName, outputFilename, pt)
	copyInputs := osbuild.NewPipelineTreeInputs(inputName, inputPipelineName)
	p.AddStage(osbuild.NewCopyStage(copyOptions, copyInputs, copyDevices, copyMounts))

	for _, stage := range osbuild.GenImageFinishStages(pt, outputFilename) {
		p.AddStage(stage)
	}

	loopback := osbuild.NewLoopbackDevice(&osbuild.LoopbackDeviceOptions{Filename: outputFilename})
	p.AddStage(bootloaderInstStage(outputFilename, pt, arch, kernelVer, copyDevices, copyMounts, loopback))
	return p
}

func xzArchivePipeline(inputPipelineName, inputFilename, outputFilename string) *osbuild.Pipeline {
	p := new(osbuild.Pipeline)
	p.Name = "archive"
	p.Build = "name:build"

	p.AddStage(osbuild.NewXzStage(
		osbuild.NewXzStageOptions(outputFilename),
		osbuild.NewFilesInputs(osbuild.NewFilesInputReferencesPipeline(inputPipelineName, inputFilename)),
	))

	return p
}

func tarArchivePipeline(name, inputPipelineName string, tarOptions *osbuild.TarStageOptions) *osbuild.Pipeline {
	p := new(osbuild.Pipeline)
	p.Name = name
	p.Build = "name:build"
	p.AddStage(osbuild.NewTarStage(tarOptions, inputPipelineName))
	return p
}

func bootloaderConfigStage(t *imageType, partitionTable disk.PartitionTable, kernelVer string, install, greenboot bool) *osbuild.Stage {
	if t.Arch().Name() == distro.S390xArchName {
		return osbuild.NewZiplStage(new(osbuild.ZiplStageOptions))
	}

	uefi := t.supportsUEFI()
	legacy := t.arch.legacy

	options := osbuild.NewGrub2StageOptionsUnified(&partitionTable, kernelVer, uefi, legacy, t.arch.distro.vendor, install)
	options.Greenboot = greenboot

	return osbuild.NewGRUB2Stage(options)
}

func bootloaderInstStage(filename string, pt *disk.PartitionTable, arch *architecture, kernelVer string, devices *osbuild.Devices, mounts *osbuild.Mounts, disk *osbuild.Device) *osbuild.Stage {
	platform := arch.legacy
	if platform != "" {
		return osbuild.NewGrub2InstStage(osbuild.NewGrub2InstStageOption(filename, pt, platform))
	}

	if arch.name == distro.S390xArchName {
		return osbuild.NewZiplInstStage(osbuild.NewZiplInstStageOptions(kernelVer, pt), disk, devices, mounts)
	}

	return nil
}
