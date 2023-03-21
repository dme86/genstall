package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// Display welcome message and prompt to continue
	fmt.Println("Welcome to the Gentoo installer!")
	fmt.Println("This script will guide you through the process of installing Gentoo.")
	fmt.Println("Press Enter to continue, or Ctrl+C to exit.")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	// Prompt for disk to install Gentoo on and partition it
	fmt.Println("Please enter the disk you would like to install Gentoo on (e.g. /dev/sda):")
	disk, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	disk = strings.TrimSpace(disk)

	fmt.Println("Please enter the size of the boot partition (e.g. 512M):")
	bootSize, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	bootSize = strings.TrimSpace(bootSize)

	fmt.Println("Please enter the size of the root partition (e.g. 10G):")
	rootSize, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	rootSize = strings.TrimSpace(rootSize)

	fmt.Println("Please enter the size of the swap partition (e.g. 2G):")
	swapSize, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	swapSize = strings.TrimSpace(swapSize)

	fmt.Println("Please choose the filesystem for the root partition:")
	fmt.Println("1. ext4")
	fmt.Println("2. btrfs")
	fmt.Println("3. xfs")

	var filesystem string
	for {
		fmt.Print("Enter your choice (1-3): ")
		choice, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			filesystem = "ext4"
		case "2":
			filesystem = "btrfs"
		case "3":
			filesystem = "xfs"
		default:
			fmt.Println("Invalid choice, please try again.")
			continue
		}
		break
	}

	// Partition disk and format partitions
	fmt.Println("Partitioning disk and formatting partitions...")
	cmd := exec.Command("parted", "-s", disk, "mklabel", "gpt", "mkpart", "boot", "fat32", "1", bootSize, "mkpart", "root", filesystem, bootSize, fmt.Sprintf("+%s", rootSize), "mkpart", "swap", "linux-swap", fmt.Sprintf("+%s", bootSize+rootSize), fmt.Sprintf("+%s", swapSize), "set", "1", "boot", "on")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error partitioning and formatting disk:", err)
		return
	}

	// Mount partitions
	fmt.Println("Mounting partitions...")
	if err := os.MkdirAll("/mnt/gentoo", 0755); err != nil {
		fmt.Println("Error creating /mnt/gentoo directory:", err)
		return
	}

	cmd = exec.Command("mount", fmt.Sprintf("/dev/%s2", disk[5:]), "/mnt/gentoo")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error mounting root partition:", err)
		return
	}

	// Install Gentoo
	fmt.Println("Installing Gentoo...")

	// Download latest stage3 tarball and extract it
	stage3Url := "http://distfiles.gentoo.org/releases/amd64/autobuilds/latest-stage3-amd64.txt"
	stage3Tarball := ""
	resp, err := http.Get(stage3Url)
	if err != nil {
		fmt.Println("Error getting latest stage3 tarball URL:", err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		stage3Tarball = strings.Fields(line)[0]
		break
	}

	if stage3Tarball == "" {
		fmt.Println("Error finding latest stage3 tarball URL.")
		return
	}

	stage3Url = fmt.Sprintf("%s/releases/amd64/autobuilds/%s", strings.TrimSuffix(stage3Url, "latest-stage3-amd64.txt"), stage3Tarball)
	stage3Filename := fmt.Sprintf("/mnt/gentoo/%s", strings.TrimSuffix(stage3Tarball, ".DIGESTS.asc"))
	stage3DigestFilename := fmt.Sprintf("%s.DIGESTS.asc", stage3Filename)

	cmd = exec.Command("curl", "-L", "-o", stage3Filename, stage3Url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error downloading stage3 tarball:", err)
		return
	}

	cmd = exec.Command("curl", "-L", "-o", stage3DigestFilename, fmt.Sprintf("%s.DIGESTS.asc", stage3Url))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error downloading stage3 digest file:", err)
		return
	}

	cmd = exec.Command("grep", stage3Tarball, stage3DigestFilename, "|", "sha512sum", "-c")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error verifying stage3 tarball digest:", err)
		return
	}

	cmd = exec.Command("tar", "xpf", stage3Filename, "--xattrs-include='*.*'", "--numeric-owner", "-C", "/mnt/gentoo/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error extracting stage3 tarball:", err)
		return
	}

	// Copy DNS info to chroot environment
	cmd = exec.Command("cp", "/etc/resolv.conf", "/mnt/gentoo/etc/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error copying resolv.conf to chroot environment:", err)
		return
	}

	// Mount necessary filesystems inside chroot environment
	cmd = exec.Command("mount", "-t", "proc", "none", "/mnt/gentoo/proc")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error mounting proc filesystem inside chroot environment:", err)
		return
	}

	cmd = exec.Command("mount", "-o", "bind", "/dev", "/mnt/gentoo/dev")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Error mounting dev filesystem inside chroot environment:", err)
		return
	}

	cmd = exec.Command("mount", "-t", "sysfs", "none", "/mnt/gentoo/sys")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error mounting sys filesystem inside chroot environment:", err)
		return
	}

	// Copy the portage configuration file to the chroot environment
	portageConfSrc := "/etc/portage/make.conf"
	portageConfDst := "/mnt/gentoo/etc/portage/make.conf"
	cmd = exec.Command("cp", portageConfSrc, portageConfDst)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error copying portage configuration file to chroot environment:", err)
		return
	}

	// Set the timezone
	cmd = exec.Command("ln", "-sf", "/usr/share/zoneinfo/"+timezone, "/mnt/gentoo/etc/localtime")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error setting timezone:", err)
		return
	}

	// Set the hostname
	cmd = exec.Command("echo", hostname, ">", "/mnt/gentoo/etc/hostname")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error setting hostname:", err)
		return
	}

	// Generate fstab
	var filesystem string
	switch fs {
	case "btrfs":
		filesystem = "btrfs"
	case "ext4":
		filesystem = "ext4"
	default:
		fmt.Println("Unsupported filesystem type")
		return
	}

	cmd = exec.Command("genfstab", "-U", "/mnt/gentoo", fmt.Sprintf("/mnt/gentoo/etc/fstab"), "-p", "mnt/%s	%s	defaults	0	0", filesystem, "/dev/%s")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error generating fstab:", err)
		return
	}

	// Chroot into the new environment
	cmd = exec.Command("chroot", "/mnt/gentoo", "/bin/bash", "-c", fmt.Sprintf("env-update && source /etc/profile && emerge-webrsync && emerge -uDN @world"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Error running chroot command:", err)
		return
	}

	fmt.Println("Installation complete. Please reboot into your new Gentoo system.")

}
