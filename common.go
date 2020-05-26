package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	maxLengthInterfaceNameForVmacNoShort = 10
	maxLengthVRRPIDForVmacNoShort        = 4
	maxVIPinVirtualIPaddress             = 20
	permissionFileCreated                = 0755
)

// function check if iface exist.
func checkIfaceExists(ifaceVrrp ifaceVrrpType) bool {
	_, err := os.Stat(strings.Join([]string{"/etc/network/interfaces.d/", ifaceVrrp.Iface}, ""))
	return !os.IsNotExist(err)
}

// generate /etc/network/ file for check/add.
func generateIfaceFile(ifaceVrrp ifaceVrrpType, postupAdd bool) string {
	var ifaceIn string
	if *isSlave {
		if strings.Contains(ifaceVrrp.IPSlave, ":") {
			ifaceIn = strings.Join([]string{"auto ", ifaceVrrp.Iface, "\n",
				"iface ", ifaceVrrp.Iface, " inet6 static\n",
				"\taddress ", ifaceVrrp.IPSlave, "/", ifaceVrrp.Mask, "\n"}, "")
		} else {
			if ifaceVrrp.IPSlave == "" {
				ifaceIn = strings.Join([]string{"auto ", ifaceVrrp.Iface, "\n",
					"iface ", ifaceVrrp.Iface, " inet manual\n",
					"\tup ifconfig ", ifaceVrrp.Iface, " up\n"}, "")
			} else {
				ifaceIn = strings.Join([]string{"auto ", ifaceVrrp.Iface, "\n",
					"iface ", ifaceVrrp.Iface, " inet static\n",
					"\taddress ", ifaceVrrp.IPSlave, "/", ifaceVrrp.Mask, "\n"}, "")
			}
		}
	} else {
		if strings.Contains(ifaceVrrp.IPMaster, ":") {
			ifaceIn = strings.Join([]string{"auto ", ifaceVrrp.Iface, "\n",
				"iface ", ifaceVrrp.Iface, " inet6 static\n",
				"\taddress ", ifaceVrrp.IPMaster, "/", ifaceVrrp.Mask, "\n"}, "")
		} else {
			if ifaceVrrp.IPMaster == "" {
				ifaceIn = strings.Join([]string{"auto ", ifaceVrrp.Iface, "\n",
					"iface ", ifaceVrrp.Iface, " inet manual\n",
					"\tup ifconfig ", ifaceVrrp.Iface, " up\n"}, "")
			} else {
				ifaceIn = strings.Join([]string{"auto ", ifaceVrrp.Iface, "\n",
					"iface ", ifaceVrrp.Iface, " inet static\n",
					"\taddress ", ifaceVrrp.IPMaster, "/", ifaceVrrp.Mask, "\n"}, "")
			}
		}
	}
	if (ifaceVrrp.VlanDevice != "") && (strings.Contains(ifaceVrrp.Iface, "vlan")) {
		ifaceIn = strings.Join([]string{ifaceIn, "\tvlan-raw-device ", ifaceVrrp.VlanDevice, "\n"}, "")
	}
	if ifaceVrrp.DefaultGW != "" {
		ifaceIn = strings.Join([]string{ifaceIn, "\tgateway ", ifaceVrrp.DefaultGW, "\n"}, "")
	}
	if *isSlave {
		if ifaceVrrp.LACPSlavesSlave != "" {
			ifaceIn = strings.Join([]string{ifaceIn, "\tslaves ", ifaceVrrp.LACPSlavesSlave, "\n",
				"\tbond_mode 802.3ad\n",
				"\tbond_miimon 50\n",
				"\tbond_downdelay 200\n",
				"\tbond_updelay 200\n"}, "")
			if postupAdd {
				ifaceIn = strings.Join([]string{ifaceIn,
					"\tpost-up echo layer3+4 > /sys/class/net/", ifaceVrrp.Iface, "/bonding/xmit_hash_policy\n"}, "")
			}
		}
	} else {
		if ifaceVrrp.LACPSlavesMaster != "" {
			ifaceIn = strings.Join([]string{ifaceIn, "\tslaves ", ifaceVrrp.LACPSlavesMaster, "\n",
				"\tbond_mode 802.3ad\n",
				"\tbond_miimon 50\n",
				"\tbond_downdelay 200\n",
				"\tbond_updelay 200\n"}, "")
			if postupAdd {
				ifaceIn = strings.Join([]string{ifaceIn,
					"\tpost-up echo layer3+4 > /sys/class/net/", ifaceVrrp.Iface, "/bonding/xmit_hash_policy\n"}, "")
			}
		}
	}
	if postupAdd && len(ifaceVrrp.PostUp) != 0 {
		for _, post := range ifaceVrrp.PostUp {
			ifaceIn = strings.Join([]string{ifaceIn, "\tpost-up ", post, "\n"}, "")
		}
	}
	return ifaceIn
}

// function check if conf up and equal to ifaceVrrp.
func checkIfaceOk(ifaceVrrp ifaceVrrpType) (bool, error) {
	ifaceIn := generateIfaceFile(ifaceVrrp, true)

	ifaceReadByte, err := ioutil.ReadFile(strings.Join([]string{"/etc/network/interfaces.d/", ifaceVrrp.Iface}, ""))
	ifaceRead := string(ifaceReadByte)
	if err != nil {
		return false, err
	}
	if ifaceIn == ifaceRead {
		err := exec.Command("ifquery", ifaceVrrp.Iface, "--state").Run()
		if err != nil {
			if *debug {
				log.Printf("ifquery %v --state failed", ifaceVrrp.Iface)
			}
			return false, nil
		}
		return true, nil
	}
	if *debug {
		log.Printf("File from json : %#v", ifaceIn)
		log.Printf("File read : %#v", ifaceRead)
	}
	return false, nil
}

// checkIfaceWithoutPostup : check network config without post-up line.
func checkIfaceWithoutPostup(ifaceVrrp ifaceVrrpType) (bool, error) {
	ifaceIn := generateIfaceFile(ifaceVrrp, false)

	ifaceReadByte, err := ioutil.ReadFile(strings.Join([]string{"/etc/network/interfaces.d/", ifaceVrrp.Iface}, ""))
	ifaceRead := string(ifaceReadByte)
	if err != nil {
		return false, err
	}
	re := regexp.MustCompile("\tpost-up.*\n")
	ifaceRead = re.ReplaceAllString(ifaceRead, "")
	if ifaceIn == ifaceRead {
		return true, nil
	}
	if *debug {
		log.Printf("File from json : %#v", ifaceIn)
		log.Printf("File read : %#v", ifaceRead)
	}
	return false, nil
}

// addIfaceFile : write network config file.
func addIfaceFile(ifaceVrrp ifaceVrrpType) error {
	ifaceIn := generateIfaceFile(ifaceVrrp, true)

	ipVersCmd := "-4"
	if strings.Contains(ifaceVrrp.IPMaster, ":") {
		ipVersCmd = "-6"
	}
	if ifaceVrrp.DefaultGW != "" {
		cmd := exec.Command("ip", ipVersCmd, "route")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			return err
		}
		returnCmd, _ := ioutil.ReadAll(stdout)
		if (strings.Contains(string(returnCmd), "default")) &&
			(!strings.Contains(string(returnCmd), strings.Join([]string{
				"default", "via", ifaceVrrp.DefaultGW}, " "))) {
			return fmt.Errorf("default gateway already exist")
		}
	}

	err := ioutil.WriteFile(strings.Join([]string{"/etc/network/interfaces.d/", ifaceVrrp.Iface}, ""),
		[]byte(ifaceIn), 0644)
	if err != nil {
		return err
	}
	return nil
}

// addIface : call addIfaceFile() and ifup interface.
func addIface(ifaceVrrp ifaceVrrpType) error {
	err := addIfaceFile(ifaceVrrp)
	if err != nil {
		return err
	}
	cmdOut, err := exec.Command("ifup", ifaceVrrp.Iface).CombinedOutput()
	if err != nil {
		return fmt.Errorf(string(cmdOut), err.Error())
	}
	err = exec.Command("ifquery", ifaceVrrp.Iface, "--state").Run()
	if err != nil {
		return fmt.Errorf("error on ifup %v", ifaceVrrp.Iface)
	}
	return nil
}

// removeIfaceFile : remove network config file.
func removeIfaceFile(ifaceVrrp ifaceVrrpType) error {
	err := os.Remove(strings.Join([]string{"/etc/network/interfaces.d/", ifaceVrrp.Iface}, ""))
	if err != nil {
		return err
	}
	return nil
}

// removeIface : ifdown iface and network config file.
func removeIface(ifaceVrrp ifaceVrrpType) error {
	VGs, err := ioutil.ReadDir("/etc/keepalived/keepalived-vrrp.d/")
	if err != nil {
		return fmt.Errorf("error for readdir /etc/keepalived/keepalived-vrrp.d/")
	}
	for _, VG := range VGs {
		if strings.HasSuffix(VG.Name(), ".conf") {
			continue
		}
		files, err := filepath.Glob(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/", VG.Name(), "/*.conf"}, ""))
		if err != nil {
			return err
		}
		for _, file := range files {
			vrrpFileByte, err := ioutil.ReadFile(file)
			vrrpFile := string(vrrpFileByte)
			if err != nil {
				return fmt.Errorf("read file %v error", file)
			}
			if strings.Contains(vrrpFile, strings.Join([]string{"dev ", ifaceVrrp.Iface, "\n"}, "")) {
				return fmt.Errorf("iface %v used for an other vrrp", ifaceVrrp.Iface)
			}
			if strings.Contains(vrrpFile, strings.Join([]string{"dev vmac_", ifaceVrrp.Iface, "_"}, "")) {
				return fmt.Errorf("iface %v used for an other vrrp", ifaceVrrp.Iface)
			}
			if strings.Contains(vrrpFile, strings.Join([]string{"dev vc_", ifaceVrrp.Iface, "_"}, "")) {
				return fmt.Errorf("iface %v used for an other vrrp", ifaceVrrp.Iface)
			}
		}
	}
	if len(ifaceVrrp.PostUp) != 0 {
		for _, post := range ifaceVrrp.PostUp {
			err := reversePostUp(post)
			if err != nil {
				return err
			}
		}
	}
	err = exec.Command("ifdown", ifaceVrrp.Iface, "--force").Run()
	if err != nil {
		return err
	}
	err = removeIfaceFile(ifaceVrrp)
	if err != nil {
		return err
	}
	return nil
}

// checkVrrpExists: check if vrrp config file exist.
func checkVrrpExists(ifaceVrrp ifaceVrrpType) bool {
	_, err := os.Stat(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/",
		ifaceVrrp.VrrpGroup, "/", ifaceVrrp.Iface, "_", ifaceVrrp.IDVrrp, ".conf"}, ""))
	return !os.IsNotExist(err)
}

// checkVrrpExistsOtherVG : check if vrrp config file exist in other vrrp group directory.
func checkVrrpExistsOtherVG(ifaceVrrp ifaceVrrpType) (string, error) {
	VGReturn := ""
	VGs, err := ioutil.ReadDir("/etc/keepalived/keepalived-vrrp.d/")
	if err != nil {
		return VGReturn, fmt.Errorf("readdir /etc/keepalived/keepalived-vrrp.d/ error")
	}
	for _, VG := range VGs {
		if strings.HasSuffix(VG.Name(), ".conf") {
			continue
		}
		_, err := os.Stat(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/",
			VG.Name(), "/", ifaceVrrp.Iface, "_", ifaceVrrp.IDVrrp, ".conf"}, ""))

		if !os.IsNotExist(err) {
			VGReturn = VG.Name()
		}
	}
	return VGReturn, nil
}

// function generate vrrp file string.
func generateVrrpFile(ifaceVrrp ifaceVrrpType, syncAdd bool) (string, error) {
	version := ipv4str
	for _, vip := range ifaceVrrp.IPVip {
		if strings.Contains(vip, ":") {
			version = ipv6str
		}
	}
	ifaceCut := strings.Split(ifaceVrrp.Iface, ":")[0]
	vrrpIn := strings.Join([]string{"vrrp_instance network_", ifaceVrrp.Iface, "_id_", ifaceVrrp.IDVrrp, " {\n",
		"\tstate BACKUP\n"}, "")
	if syncAdd {
		if ifaceVrrp.IfaceForVrrp != "" {
			vrrpIn = strings.Join([]string{vrrpIn, "\tinterface ", ifaceVrrp.IfaceForVrrp, "\n"}, "")
		} else {
			vrrpIn = strings.Join([]string{vrrpIn, "\tinterface ", ifaceCut, "\n"}, "")
		}
	}
	vrrpIn = strings.Join([]string{vrrpIn, "\ttrack_interface {\n",
		"\t\t", ifaceCut, "\n",
		"\t}\n"}, "")
	if len(ifaceVrrp.TrackScript) > 0 {
		vrrpIn = strings.Join([]string{vrrpIn, "\ttrack_script {\n"}, "")
		for _, script := range ifaceVrrp.TrackScript {
			vrrpIn = strings.Join([]string{vrrpIn, "\t\t", script, "\n"}, "")
		}
		vrrpIn = strings.Join([]string{vrrpIn, "\t}\n"}, "")
	}
	if (ifaceVrrp.UseVmac) && (version != ipv6str) {
		switch {
		case (strings.Count(ifaceCut, "") < maxLengthInterfaceNameForVmacNoShort-1) &&
			(strings.Count(ifaceVrrp.IDVrrp, "") < maxLengthVRRPIDForVmacNoShort):
			vrrpIn = strings.Join([]string{vrrpIn, "\tuse_vmac vmac_", ifaceCut, "_", ifaceVrrp.IDVrrp, "\n"}, "")
		case strings.Count(ifaceCut, "") < maxLengthInterfaceNameForVmacNoShort:
			vrrpIn = strings.Join([]string{vrrpIn, "\tuse_vmac vc_", ifaceCut, "_", ifaceVrrp.IDVrrp, "\n"}, "")
		default:
			return "", fmt.Errorf("interface %s too long", ifaceCut)
		}
		vrrpIn = strings.Join([]string{vrrpIn, "\tvmac_xmit_base\n"}, "")
	}
	if ifaceVrrp.GarpMDelay != "" {
		vrrpIn = strings.Join([]string{vrrpIn, "\tgarp_master_delay ", ifaceVrrp.GarpMDelay, "\n"}, "")
		vrrpIn = strings.Join([]string{vrrpIn, "\tgarp_lower_prio_delay ", ifaceVrrp.GarpMDelay, "\n"}, "")
	} else {
		vrrpIn = strings.Join([]string{vrrpIn, "\tgarp_master_delay 5\n"}, "")
		vrrpIn = strings.Join([]string{vrrpIn, "\tgarp_lower_prio_delay 5\n"}, "")
	}
	if ifaceVrrp.GarpMasterRefresh != "" {
		vrrpIn = strings.Join([]string{vrrpIn, "\tgarp_master_refresh ", ifaceVrrp.GarpMasterRefresh, "\n"}, "")
	}
	vrrpIn = strings.Join([]string{vrrpIn, "\tvirtual_router_id ", ifaceVrrp.IDVrrp, "\n"}, "")
	if *isSlave {
		vrrpIn = strings.Join([]string{vrrpIn, "\tpriority ", ifaceVrrp.PrioSlave, "\n"}, "")
	} else {
		vrrpIn = strings.Join([]string{vrrpIn, "\tpriority ", ifaceVrrp.PrioMaster, "\n"}, "")
	}
	if ifaceVrrp.AdvertInt != "" {
		vrrpIn = strings.Join([]string{vrrpIn, "\tadvert_int ", ifaceVrrp.AdvertInt, "\n"}, "")
	} else {
		vrrpIn = strings.Join([]string{vrrpIn, "\tadvert_int 1\n"}, "")
	}
	if (ifaceVrrp.AuthType != "") && (version != ipv6str) {
		vrrpIn = strings.Join([]string{vrrpIn, "\tauthentication {\n",
			"\t\tauth_type ", ifaceVrrp.AuthType, "\n",
			"\t\tauth_pass ", ifaceVrrp.AuthPass, "\n",
			"\t}\n"}, "")
	}
	vrrpIn = strings.Join([]string{vrrpIn, "\tvirtual_ipaddress {\n"}, "")
	if (ifaceVrrp.UseVmac) && (version != ipv6str) {
		for i, vip := range ifaceVrrp.IPVip {
			if i == maxVIPinVirtualIPaddress {
				break
			}
			if (strings.Count(ifaceCut, "") < 9) && (strings.Count(ifaceVrrp.IDVrrp, "") < 4) {
				vrrpIn = strings.Join([]string{vrrpIn, "\t\t", vip, " dev vmac_", ifaceCut, "_", ifaceVrrp.IDVrrp, "\n"}, "")
			} else {
				vrrpIn = strings.Join([]string{vrrpIn, "\t\t", vip, " dev vc_", ifaceCut, "_", ifaceVrrp.IDVrrp, "\n"}, "")
			}
		}
		vrrpIn = strings.Join([]string{vrrpIn, "\t}\n", ""}, "")
		if len(ifaceVrrp.IPVip) >= maxVIPinVirtualIPaddress {
			vrrpIn = strings.Join([]string{vrrpIn, "\tvirtual_ipaddress_excluded {\n"}, "")
			for i, vip := range ifaceVrrp.IPVip {
				if i < maxVIPinVirtualIPaddress {
					continue
				}
				if (strings.Count(ifaceCut, "") < 9) && (strings.Count(ifaceVrrp.IDVrrp, "") < 4) {
					vrrpIn = strings.Join([]string{vrrpIn, "\t\t", vip, " dev vmac_", ifaceCut, "_", ifaceVrrp.IDVrrp, "\n"}, "")
				} else {
					vrrpIn = strings.Join([]string{vrrpIn, "\t\t", vip, " dev vc_", ifaceCut, "_", ifaceVrrp.IDVrrp, "\n"}, "")
				}
			}
			vrrpIn = strings.Join([]string{vrrpIn, "\t}\n", ""}, "")
		}
	} else {
		for i, vip := range ifaceVrrp.IPVip {
			if i == maxVIPinVirtualIPaddress {
				break
			}
			vrrpIn = strings.Join([]string{vrrpIn, "\t\t", vip, " dev ", ifaceCut, "\n"}, "")
		}
		vrrpIn = strings.Join([]string{vrrpIn, "\t}\n", ""}, "")
		if len(ifaceVrrp.IPVip) >= maxVIPinVirtualIPaddress {
			vrrpIn = strings.Join([]string{vrrpIn, "\tvirtual_ipaddress_excluded {\n"}, "")
			for i, vip := range ifaceVrrp.IPVip {
				if i < maxVIPinVirtualIPaddress {
					continue
				}
				vrrpIn = strings.Join([]string{vrrpIn, "\t\t", vip, " dev ", ifaceCut, "\n"}, "")
			}
			vrrpIn = strings.Join([]string{vrrpIn, "\t}\n", ""}, "")
		}
	}
	vrrpIn = strings.Join([]string{vrrpIn, "}\n", ""}, "")
	if ifaceVrrp.SyncIface != "" {
		vrrpIn = strings.Join([]string{vrrpIn, "global_defs {\n",
			"\tlvs_sync_daemon ", ifaceVrrp.SyncIface,
			" network_", ifaceVrrp.Iface, "_id_", ifaceVrrp.IDVrrp, " id ", ifaceVrrp.IDVrrp, "\n",
			"}\n"}, "")
	}
	return vrrpIn, nil
}

// checkVrrpOk : check vrrp config file.
func checkVrrpOk(ifaceVrrp ifaceVrrpType) (bool, error) {
	vrrpIn, err := generateVrrpFile(ifaceVrrp, true)
	if err != nil {
		return false, err
	}
	vrrpReadByte, err := ioutil.ReadFile(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/",
		ifaceVrrp.VrrpGroup, "/", ifaceVrrp.Iface, "_", ifaceVrrp.IDVrrp, ".conf"}, ""))

	vrrpRead := string(vrrpReadByte)
	if err != nil {
		return false, err
	}
	if vrrpIn == vrrpRead {
		return true, nil
	}
	if *debug {
		log.Printf("File from json : %#v", vrrpIn)
		log.Printf("File read : %#v", vrrpRead)
	}
	return false, nil
}

// checkVrrpWithoutSync : check vrrp config file without interface line (move interface vrrp packet).
func checkVrrpWithoutSync(ifaceVrrp ifaceVrrpType) (bool, error) {
	vrrpIn, err := generateVrrpFile(ifaceVrrp, false)
	if err != nil {
		return false, err
	}
	vrrpReadByte, err := ioutil.ReadFile(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/",
		ifaceVrrp.VrrpGroup, "/", ifaceVrrp.Iface, "_", ifaceVrrp.IDVrrp, ".conf"}, ""))

	vrrpRead := string(vrrpReadByte)
	if err != nil {
		return false, err
	}
	re := regexp.MustCompile("\tinterface.*\n")
	vrrpRead = re.ReplaceAllString(vrrpRead, "")
	if vrrpIn == vrrpRead {
		return true, nil
	}
	if *debug {
		log.Printf("File from json : %#v", vrrpIn)
		log.Printf("File read : %#v", vrrpRead)
	}
	return false, nil
}

// addVrrp : add vrrp configuration file.
func addVrrp(ifaceVrrp ifaceVrrpType) error {
	vrrpIn, err := generateVrrpFile(ifaceVrrp, true)
	if err != nil {
		return err
	}

	_, err = os.Stat(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/", ifaceVrrp.VrrpGroup}, ""))
	if os.IsNotExist(err) {
		err := os.Mkdir(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/",
			ifaceVrrp.VrrpGroup}, ""), os.FileMode(permissionFileCreated))
		if err != nil {
			return err
		}
	}
	err = ioutil.WriteFile(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/",
		ifaceVrrp.VrrpGroup, "/", ifaceVrrp.Iface, "_", ifaceVrrp.IDVrrp, ".conf"}, ""), []byte(vrrpIn), 0644)
	if err != nil {
		return err
	}
	return nil
}

// remove vrrp configuration file.
func removeVrrp(ifaceVrrp ifaceVrrpType) error {
	err := os.Remove(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/",
		ifaceVrrp.VrrpGroup, "/", ifaceVrrp.Iface, "_", ifaceVrrp.IDVrrp, ".conf"}, ""))
	if err != nil {
		return err
	}
	return nil
}

// create vrrp_sync_group configuration and reload keepalived daemon.
func syncGroupAndReload() error {
	VGs, err := ioutil.ReadDir("/etc/keepalived/keepalived-vrrp.d/")
	if err != nil {
		return fmt.Errorf("readdir /etc/keepalived/keepalived-vrrp.d/ error")
	}
	for _, VG := range VGs {
		if strings.HasSuffix(VG.Name(), ".conf") {
			continue
		}
		var instances []string
		files, err := filepath.Glob(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/", VG.Name(), "/*.conf"}, ""))
		if err != nil {
			return err
		}
		if files != nil {
			for _, file := range files {
				vrrpFileByte, err := ioutil.ReadFile(file)
				vrrpFile := string(vrrpFileByte)
				if err != nil {
					return fmt.Errorf("read file %v error", file)
				}
				vrrpFileWords := strings.Fields(vrrpFile)
				if len(vrrpFileWords) > 0 {
					if vrrpFileWords[0] == "vrrp_instance" {
						instances = append(instances, vrrpFileWords[1])
					}
				}
			}
			if len(instances) == 0 {
				err := os.RemoveAll(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/", VG.Name()}, ""))
				if err != nil {
					return fmt.Errorf("error when remove VG empty")
				}
			} else {
				vrrpSyncGroupIn := strings.Join([]string{"vrrp_sync_group ", VG.Name(), " {\n\tgroup {\n"}, "")
				for _, instance := range instances {
					vrrpSyncGroupIn = strings.Join([]string{vrrpSyncGroupIn, "\t\t", instance, "\n"}, "")
				}
				vrrpSyncGroupIn = strings.Join([]string{vrrpSyncGroupIn, "\t}\n}\n"}, "")
				err := ioutil.WriteFile(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/",
					VG.Name(), "/vrrp_sync_group"}, ""), []byte(vrrpSyncGroupIn), 0644)
				if err != nil {
					return err
				}
			}
		} else {
			err := os.RemoveAll(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/", VG.Name()}, ""))
			if err != nil {
				return fmt.Errorf("error when remove VG empty")
			}
		}
	}
	err = reloadVrrp()
	if err != nil {
		return err
	}
	return nil
}

// func reloadVrrp.
func reloadVrrp() error {
	reloadKeepalivedCommandParts := strings.Fields(*reloadKeepalivedCommand)
	reloadKeepalivedCommandBin := reloadKeepalivedCommandParts[0]
	reloadKeepalivedCommandArgs := reloadKeepalivedCommandParts[1:]
	cmdOut, err := exec.Command(reloadKeepalivedCommandBin, reloadKeepalivedCommandArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf(string(cmdOut), err.Error())
	}
	return nil
}

// reversePostUp : del route/rule if remove post-up route/rule add.
func reversePostUp(post string) error {
	if (strings.Contains(post, "route add")) || (strings.Contains(post, "ip rule add")) {
		postdown := strings.Replace(post, "post-up", "", 1)
		postdown = strings.Replace(postdown, "route add", "route del", 1)
		postdown = strings.Replace(postdown, "ip rule add", "ip rule del", 1)
		postdownParts := strings.Fields(postdown)
		postdownCommand := postdownParts[0]
		postdownArgs := postdownParts[1:]

		cmdOut, err := exec.Command(postdownCommand, postdownArgs...).CombinedOutput()
		if err != nil {
			return fmt.Errorf(postdown, string(cmdOut), err.Error())
		}
	}
	return nil
}

// addPostUp : execute command post-up if iface already up.
func addPostUp(postup string) error {
	postupParts := strings.Fields(postup)
	postupCommand := postupParts[0]
	postupArgs := postupParts[1:]
	cmdOut, err := exec.Command(postupCommand, postupArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf(postup, string(cmdOut), err.Error())
	}
	return nil
}

// changeIfacePostup : change different post-up line with respect to the configuration.
func changeIfacePostup(ifaceVrrp ifaceVrrpType) error {
	ifaceReadByte, err := ioutil.ReadFile(strings.Join([]string{"/etc/network/interfaces.d/", ifaceVrrp.Iface}, ""))
	ifaceRead := string(ifaceReadByte)
	if err != nil {
		return err
	}
	re := regexp.MustCompile("\tpost-up .*\n")
	postupLine := re.FindAllString(ifaceRead, -1)

	if len(postupLine) != 0 {
		if len(ifaceVrrp.PostUp) != 0 {
			for _, postupIn := range ifaceVrrp.PostUp {
				addPost := true
				for _, postupRead := range postupLine {
					postupReadShort := strings.Replace(postupRead, "\tpost-up ", "", -1)
					postupReadShort = strings.Replace(postupReadShort, "\n", "", -1)
					if postupIn == postupReadShort {
						addPost = false
					}
				}
				if addPost {
					err := addPostUp(postupIn)
					if err != nil {
						return err
					}
				}
			}
			for _, postupRead := range postupLine {
				removePost := true
				postupReadShort := strings.Replace(postupRead, "\tpost-up ", "", -1)
				postupReadShort = strings.Replace(postupReadShort, "\n", "", -1)
				if postupReadShort == strings.Join([]string{"echo layer3+4 > /sys/class/net/",
					ifaceVrrp.Iface, "/bonding/xmit_hash_policy"}, "") {
					continue
				}
				for _, postupIn := range ifaceVrrp.PostUp {
					if postupReadShort == postupIn {
						removePost = false
					}
				}
				if removePost {
					err := reversePostUp(postupReadShort)
					if err != nil {
						return err
					}
				}
			}
		} else {
			for _, postupRead := range postupLine {
				postupReadShort := strings.Replace(postupRead, "\tpost-up ", "", -1)
				postupReadShort = strings.Replace(postupReadShort, "\n", "", -1)
				if postupReadShort == strings.Join([]string{"echo layer3+4 > /sys/class/net/",
					ifaceVrrp.Iface, "/bonding/xmit_hash_policy"}, "") {
					continue
				}
				err := reversePostUp(postupReadShort)
				if err != nil {
					return err
				}
			}
		}
	} else {
		if len(ifaceVrrp.PostUp) != 0 {
			for _, postupIn := range ifaceVrrp.PostUp {
				err := addPostUp(postupIn)
				if err != nil {
					return err
				}
			}
		} else {
			return nil
		}
	}
	return nil
}

// check if vrrp script file exists.
func checkVrrpScriptExists(vrrpScriptName string) bool {
	_, err := os.Stat(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/",
		"script_", vrrpScriptName, ".conf"}, ""))
	return !os.IsNotExist(err)
}

// compare vrrp script file with a vrrpScriptType.
func checkVrrpScriptOk(vrrpScript vrrpScriptType) (bool, error) {
	scriptIn := generateScriptFile(vrrpScript)
	scriptReadByte, err := ioutil.ReadFile(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/",
		"script_", vrrpScript.Name, ".conf"}, ""))

	scriptRead := string(scriptReadByte)
	if err != nil {
		return false, err
	}
	if scriptIn == scriptRead {
		return true, nil
	}
	if *debug {
		log.Printf("File from json : %#v", scriptIn)
		log.Printf("File read : %#v", scriptRead)
	}
	return false, nil
}

// add vrrp script file on system.
func addVrrpScriptFile(vrrpScript vrrpScriptType) error {
	scriptIn := generateScriptFile(vrrpScript)
	err := ioutil.WriteFile(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/",
		"script_", vrrpScript.Name, ".conf"}, ""), []byte(scriptIn), 0644)
	if err != nil {
		return err
	}
	return nil
}

// remove vrrp script file on system.
func removeVrrpScriptFile(vrrpScript vrrpScriptType) error {
	err := os.Remove(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/",
		"script_", vrrpScript.Name, ".conf"}, ""))
	if err != nil {
		return err
	}
	return nil
}

// generate vrrp script file string.
func generateScriptFile(vrrpScript vrrpScriptType) string {
	scriptIn := strings.Join([]string{"vrrp_script ", vrrpScript.Name, " {\n",
		"\tscript \"", vrrpScript.Script, "\"\n"}, "")
	if vrrpScript.Fall != 0 {
		scriptIn = strings.Join([]string{scriptIn, "\tfall ", strconv.Itoa(vrrpScript.Fall), "\n"}, "")
	}
	if vrrpScript.Interval != 0 {
		scriptIn = strings.Join([]string{scriptIn, "\tinterval ", strconv.Itoa(vrrpScript.Interval), "\n"}, "")
	}
	if vrrpScript.Rise != 0 {
		scriptIn = strings.Join([]string{scriptIn, "\trise ", strconv.Itoa(vrrpScript.Rise), "\n"}, "")
	}
	if vrrpScript.Timeout != 0 {
		scriptIn = strings.Join([]string{scriptIn, "\ttimeout ", strconv.Itoa(vrrpScript.Timeout), "\n"}, "")
	}
	if vrrpScript.WeightReverse {
		if vrrpScript.Weight != 0 {
			scriptIn = strings.Join([]string{scriptIn, "\tweight ", strconv.Itoa(vrrpScript.Weight), " reverse\n"}, "")
		} else {
			scriptIn = strings.Join([]string{scriptIn, "\tweight 0 reverse\n"}, "")
		}
	} else {
		if vrrpScript.Weight != 0 {
			scriptIn = strings.Join([]string{scriptIn, "\tweight ", strconv.Itoa(vrrpScript.Weight), "\n"}, "")
		}
	}
	if vrrpScript.User != "" {
		scriptIn = strings.Join([]string{scriptIn, "\tuser ", vrrpScript.User, "\n"}, "")
	}
	if vrrpScript.InitFail {
		scriptIn = strings.Join([]string{scriptIn, "\tinit_fail\n"}, "")
	}
	scriptIn = strings.Join([]string{scriptIn, "}\n"}, "")
	return scriptIn
}

// read vrpp script file on system and fill a vrrpScriptType.
func readVrrpScriptFile(scriptName string) (vrrpScriptType, error) {
	var scriptRead vrrpScriptType
	var err error
	scriptReadByte, err := ioutil.ReadFile(strings.Join([]string{"/etc/keepalived/keepalived-vrrp.d/",
		"script_", scriptName, ".conf"}, ""))
	if err != nil {
		return scriptRead, err
	}
	if !strings.HasPrefix(string(scriptReadByte), "vrrp_script "+scriptName+" {") ||
		!strings.HasSuffix(string(scriptReadByte), "\n}\n") {
		return scriptRead, fmt.Errorf("the file is bad (not start or end with good character) ")
	}
	for _, line := range strings.Split(string(scriptReadByte), "\n") {
		switch {
		case strings.HasPrefix(line, "vrrp_script "):
			scriptRead.Name = strings.TrimSuffix(strings.TrimPrefix(line, "vrrp_script "), " {")
		case strings.HasPrefix(line, "\tscript "):
			scriptRead.Script = strings.Trim(strings.TrimPrefix(line, "\tscript "), "\"")
		case strings.HasPrefix(line, "\tuser "):
			scriptRead.User = strings.TrimPrefix(line, "\tuser ")
		case line == "\tinit_fail":
			scriptRead.InitFail = true
		case strings.HasPrefix(line, "\tfall "):
			scriptRead.Fall, err = strconv.Atoi(strings.TrimPrefix(line, "\tfall "))
			if err != nil {
				return scriptRead, err
			}
		case strings.HasPrefix(line, "\tinterval "):
			scriptRead.Interval, err = strconv.Atoi(strings.TrimPrefix(line, "\tinterval "))
			if err != nil {
				return scriptRead, err
			}
		case strings.HasPrefix(line, "\trise "):
			scriptRead.Rise, err = strconv.Atoi(strings.TrimPrefix(line, "\trise "))
			if err != nil {
				return scriptRead, err
			}
		case strings.HasPrefix(line, "\ttimeout "):
			scriptRead.Timeout, err = strconv.Atoi(strings.TrimPrefix(line, "\ttimeout "))
			if err != nil {
				return scriptRead, err
			}
		case strings.HasPrefix(line, "\tweight "):
			weightLine := strings.TrimPrefix(line, "\tweight ")
			weightSplit := strings.Split(weightLine, " ")
			if len(weightSplit) > 1 {
				scriptRead.WeightReverse = true
			}
			scriptRead.Weight, err = strconv.Atoi(weightSplit[0])
			if err != nil {
				return scriptRead, err
			}
		case line == "}":
			continue
		case line == "":
			continue
		default:
			return scriptRead, fmt.Errorf("vrrp script file has unknown line %q", line)
		}
	}
	return scriptRead, nil
}
