package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	auth "github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"
)

// checkVlanCom : ping IP 'ipslave' (in json) from master server (check if L2 ok)
func checkVlanCom(ifaceVrrp ifaceVrrpType) error {
	sleep()
	if strings.Contains(ifaceVrrp.IPSlave, ":") {
		err := exec.Command("ping6", "-c1", "-t1", ifaceVrrp.IPSlave).Run()
		if err != nil {
			return fmt.Errorf("master don't ping slave %v", ifaceVrrp.IPSlave)
		}
	} else {
		err := exec.Command("ping", "-c1", "-t1", ifaceVrrp.IPSlave).Run()
		if err != nil {
			return fmt.Errorf("master don't ping slave %v", ifaceVrrp.IPSlave)
		}
	}
	return nil
}

//validate missing or incompatibility parameters
func (ifaceVrrp ifaceVrrpType) validate() string {
	if !ifaceVrrp.IPVipOnly && len(ifaceVrrp.IPVip) != 0 {
		if ifaceVrrp.IPMaster == "" {
			return "missing IP_master"
		}
		if ifaceVrrp.IPSlave == "" {
			return "missing IP_slave"
		}
		if ifaceVrrp.Mask == "" {
			return "missing Mask"
		}
		if (strings.Contains(ifaceVrrp.Iface, "vlan")) && (ifaceVrrp.VlanDevice == "") {
			return "missing Vlan_device with iface vlan"
		}
	}
	if ifaceVrrp.IPMaster != "" {
		if ifaceVrrp.Mask == "" {
			return "missing Mask"
		}
		_, ipnet, err := net.ParseCIDR(strings.Join([]string{ifaceVrrp.IPMaster, "/", ifaceVrrp.Mask}, ""))
		if err != nil {
			return strings.Join([]string{"Error CIDR ", ifaceVrrp.IPMaster, "/", ifaceVrrp.Mask}, "")
		}
		if ifaceVrrp.IPSlave != "" {
			if !ipnet.Contains(net.ParseIP(ifaceVrrp.IPSlave)) {
				return strings.Join([]string{"IP_master network don't include IP slave : ", ifaceVrrp.IPSlave}, "")
			}
		} else {
			return "missing IP_slave"
		}
	}
	if (ifaceVrrp.DefaultGW != "") && (ifaceVrrp.IPMaster == "" || ifaceVrrp.IPSlave == "") {
		return "missing IP_master || IP_slave with Default_GW"
	}
	if len(ifaceVrrp.IPVip) != 0 && !ifaceVrrp.IPVipOnly {
		for _, vip := range ifaceVrrp.IPVip {
			_, ipnet, _ := net.ParseCIDR(strings.Join([]string{ifaceVrrp.IPMaster, "/", ifaceVrrp.Mask}, ""))
			if !ipnet.Contains(net.ParseIP(vip)) {
				return strings.Join([]string{"IP_master network don't include VIP : ", vip}, "")
			}
		}
	}
	if len(ifaceVrrp.IPVip) != 0 {
		if ifaceVrrp.VrrpGroup == "" {
			return "missing Vrrp_group for VIP"
		}
		if ifaceVrrp.IDVrrp == "" {
			return "missing ID_vrrp for VIP"
		}
		IDVrrpInt, err := strconv.Atoi(ifaceVrrp.IDVrrp)
		if err != nil {
			return "Error on Id_vrrp integer"
		}
		if IDVrrpInt < 1 || IDVrrpInt > 255 {
			return "Id_vrrp must be in the range from 1 to 255"
		}
		if ifaceVrrp.PrioMaster == "" {
			return "missing Prio_master for VIP"
		}
		if ifaceVrrp.PrioSlave == "" {
			return "missing Prio_slave for VIP"
		}
	}
	if ((ifaceVrrp.AuthType != "") && (ifaceVrrp.AuthPass == "")) ||
		((ifaceVrrp.AuthPass != "") && (ifaceVrrp.AuthType == "")) {
		return "missing Auth_type or Auth_pass"
	}
	return ""
}

// addIfaceVrrp : on master API for add configuration (network + vrrp) on master & slave server
func addIfaceVrrp(w http.ResponseWriter, r *http.Request) {
	if *htpasswdfile != "" {
		htpasswd := auth.HtpasswdFileProvider(*htpasswdfile)
		authenticator := auth.BasicAuth{
			Realm:   "Basic Realm",
			Secrets: htpasswd,
		}
		usercheck := authenticator.CheckAuth(r)
		if usercheck == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	var ifaceVrrp ifaceVrrpType
	vars := mux.Vars(r)
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&ifaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	ifaceVrrp.Iface = vars["iface"]
	sort.Strings(ifaceVrrp.IPVip)
	sort.Strings(ifaceVrrp.PostUp)

	validate := ifaceVrrp.validate()
	if validate != "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, validate)
		return
	}
	// iface configuration
	if !ifaceVrrp.IPVipOnly {
		ifaceExistsMaster := checkIfaceExists(ifaceVrrp)
		if ifaceExistsMaster {
			ifaceOkMaster, err := checkIfaceOk(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if !ifaceOkMaster {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "iface already exist on master with different config or not up")
				return
			}
		}

		ifaceExistsSlave, err := checkIfaceSlaveExists(ifaceVrrp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if ifaceExistsSlave {
			ifaceOkSlave, err := checkIfaceSlaveOk(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if !ifaceOkSlave {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "iface already exist on slave with different config or not up")
				return
			}
		}
		if !ifaceExistsMaster {
			err := addIface(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
		if !ifaceExistsSlave {
			err := addIfaceSlave(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
		if ifaceVrrp.IPMaster != "" {
			err = checkVlanCom(ifaceVrrp)
			if err != nil {
				sleep()
				err2 := checkVlanCom(ifaceVrrp)
				if err2 != nil {
					errReturn := fmt.Errorf("%v %v", err, err2)
					err3 := removeIfaceSlave(ifaceVrrp)
					if err3 != nil {
						errReturn = fmt.Errorf("%v %v", errReturn, err3)
					}
					err3 = removeIface(ifaceVrrp)
					if err3 != nil {
						errReturn = fmt.Errorf("%v %v", errReturn, err3)
					}
					http.Error(w, errReturn.Error(), 500)
					return
				}
			}
		}
	} else {
		if !checkIfaceExists(ifaceVrrp) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Iface", ifaceVrrp.Iface, "does not exist on master")
			return
		}
		ifaceExistsSlave, err := checkIfaceSlaveExists(ifaceVrrp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if !ifaceExistsSlave {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Iface", ifaceVrrp.Iface, "does not exist on slave")
			return
		}
	}
	// vrrp configuration
	if len(ifaceVrrp.IPVip) != 0 {
		mutex.Lock()
		addIfaceVrrpKeepalived(ifaceVrrp, true, w)
		sleep()

		addIfaceVrrpKeepalived(ifaceVrrp, false, w)
		sleep()

		mutex.Unlock()
	}
}

func addIfaceVrrpKeepalived(ifaceVrrp ifaceVrrpType, master bool, w http.ResponseWriter) {
	var err error
	var vrrpExists bool
	if master {
		vrrpExists = checkVrrpExists(ifaceVrrp)
	} else {
		vrrpExists, err = checkVrrpSlaveExists(ifaceVrrp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			mutex.Unlock()
			return
		}
	}
	if vrrpExists {
		var vrrpOk bool
		if master {
			vrrpOk, err = checkVrrpOk(ifaceVrrp)
		} else {
			vrrpOk, err = checkVrrpSlaveOk(ifaceVrrp)
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			mutex.Unlock()
			return
		}
		if !vrrpOk {
			w.WriteHeader(http.StatusBadRequest)
			if master {
				fmt.Fprintln(w, "vrrp already exist on master with different config")
			} else {
				fmt.Fprintln(w, "vrrp already exist on slave with different config")
			}
			mutex.Unlock()
			return
		}
		if master {
			err = syncGroupAndReload()
		} else {
			err = syncGroupAndReloadSlave()
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			mutex.Unlock()
			return
		}
	} else {
		if master {
			err = addVrrp(ifaceVrrp)
		} else {
			err = addVrrpSlave(ifaceVrrp)
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			mutex.Unlock()
			return
		}
		if master {
			err = reloadVrrp()
		} else {
			err = reloadVrrpSlave()
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			mutex.Unlock()
			return
		}

		// reload twice for vmac up before add IP (bug keepalived)
		// reload twice for new vrrp comme up
		sleep()
		if master {
			err = syncGroupAndReload()
		} else {
			err = syncGroupAndReloadSlave()
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			mutex.Unlock()
			return
		}
	}
}

// removeIfaceVrrp on master API for remove all configuration (network + vrrp) on master & slave server
func removeIfaceVrrp(w http.ResponseWriter, r *http.Request) {
	if *htpasswdfile != "" {
		htpasswd := auth.HtpasswdFileProvider(*htpasswdfile)
		authenticator := auth.BasicAuth{
			Realm:   "Basic Realm",
			Secrets: htpasswd,
		}
		usercheck := authenticator.CheckAuth(r)
		if usercheck == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	var ifaceVrrp ifaceVrrpType
	vars := mux.Vars(r)
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&ifaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	ifaceVrrp.Iface = vars["iface"]
	sort.Strings(ifaceVrrp.IPVip)
	sort.Strings(ifaceVrrp.PostUp)

	validate := ifaceVrrp.validate()
	if validate != "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, validate)
		return
	}
	mutex.Lock()
	// vrrp configuration
	if len(ifaceVrrp.IPVip) != 0 {
		vrrpExistsSlave, err := checkVrrpSlaveExists(ifaceVrrp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			mutex.Unlock()
			return
		}
		if vrrpExistsSlave {
			err := removeVrrpSlave(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
			err = syncGroupAndReloadSlave()
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
		} else {
			err := syncGroupAndReloadSlave()
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
		}
		sleep()

		vrrpExistsMaster := checkVrrpExists(ifaceVrrp)
		if vrrpExistsMaster {
			err := removeVrrp(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
			err = syncGroupAndReload()
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
		} else {
			err := syncGroupAndReload()
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
		}
		sleep()
	}
	// iface configuration
	if !ifaceVrrp.IPVipOnly {
		ifaceExistsSlave, err := checkIfaceSlaveExists(ifaceVrrp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			mutex.Unlock()
			return
		}
		if ifaceExistsSlave {
			err := removeIfaceSlave(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
		}
		if checkIfaceExists(ifaceVrrp) {
			err := removeIface(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
		}
	}
	mutex.Unlock()
}

// checkIfaceVrrp on master API for check all configuration (network + vrrp) on master & slave server
func checkIfaceVrrp(w http.ResponseWriter, r *http.Request) {
	if *htpasswdfile != "" {
		htpasswd := auth.HtpasswdFileProvider(*htpasswdfile)
		authenticator := auth.BasicAuth{
			Realm:   "Basic Realm",
			Secrets: htpasswd,
		}
		usercheck := authenticator.CheckAuth(r)
		if usercheck == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	var ifaceVrrp ifaceVrrpType
	vars := mux.Vars(r)
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&ifaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	ifaceVrrp.Iface = vars["iface"]
	sort.Strings(ifaceVrrp.IPVip)
	sort.Strings(ifaceVrrp.PostUp)

	validate := ifaceVrrp.validate()
	if validate != "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, validate)
		return
	}
	ifaceVrrpResponse := ifaceVrrp
	// iface configuration
	if !ifaceVrrp.IPVipOnly {
		if checkIfaceExists(ifaceVrrp) {
			ifaceOkMaster, err := checkIfaceOk(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if !ifaceOkMaster {
				ifaceOkWithoutPostup, err := checkIfaceWithoutPostup(ifaceVrrp)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				if !ifaceOkWithoutPostup {
					w.WriteHeader(http.StatusPartialContent)
					ifaceVrrpResponse.IPMaster = "?"
					ifaceVrrpResponse.Mask = "?"
					ifaceVrrpResponse.PostUp = []string{"?"}
					ifaceVrrpResponse.DefaultGW = ""
					ifaceVrrpResponse.LACPSlavesMaster = ""
					ifaceVrrpResponse.VlanDevice = ""
				} else {
					w.WriteHeader(http.StatusPartialContent)
					ifaceVrrpResponse.PostUp = []string{"?"}
				}
			}
		} else {
			ifaceExistsSlave, err := checkIfaceSlaveExists(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if ifaceExistsSlave {
				ifaceOkSlave, err := checkIfaceSlaveOk(ifaceVrrp)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				if !ifaceOkSlave {
					ifaceOkWithoutPostup, err := checkIfaceSlaveWithoutPostup(ifaceVrrp)
					if err != nil {
						http.Error(w, err.Error(), 500)
						return
					}
					if !ifaceOkWithoutPostup {
						w.WriteHeader(http.StatusPartialContent)
						ifaceVrrpResponse.IPSlave = "?"
						ifaceVrrpResponse.LACPSlavesSlave = ""
					} else {
						w.WriteHeader(http.StatusPartialContent)
						ifaceVrrpResponse.PostUp = []string{"?"}
					}
				} else {
					w.WriteHeader(http.StatusPartialContent)
					ifaceVrrpResponse.IPMaster = "?"
					ifaceVrrpResponse.LACPSlavesMaster = ""
				}
			} else {
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}
	}
	// vrrp configuration
	if len(ifaceVrrp.IPVip) != 0 {
		vrrpExistsMaster := checkVrrpExists(ifaceVrrp)
		if vrrpExistsMaster {
			vrrpOkMaster, err := checkVrrpOk(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if !vrrpOkMaster {
				w.WriteHeader(http.StatusPartialContent)
				ifaceVrrpResponse.IPVip = []string{"?"}
				ifaceVrrpResponse.PrioMaster = "?"
				ifaceVrrpResponse.AuthType = ""
				ifaceVrrpResponse.AuthPass = ""
				ifaceVrrpResponse.SyncIface = ""
				ifaceVrrpResponse.GarpMDelay = ""
				ifaceVrrpResponse.AdvertInt = ""
			}
		} else {
			w.WriteHeader(http.StatusPartialContent)
			ifaceVrrpResponse.IPVip = []string{"?"}
			ifaceVrrpResponse.IDVrrp = "?"
			ifaceVrrpResponse.PrioMaster = "?"
			ifaceVrrpResponse.AuthType = ""
			ifaceVrrpResponse.AuthPass = ""
			ifaceVrrpResponse.SyncIface = ""
			ifaceVrrpResponse.GarpMDelay = ""
			ifaceVrrpResponse.AdvertInt = ""
		}

		vrrpExistsSlave, err := checkVrrpSlaveExists(ifaceVrrp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if vrrpExistsSlave {
			vrrpOkSlave, err := checkVrrpSlaveOk(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if !vrrpOkSlave {
				w.WriteHeader(http.StatusPartialContent)
				ifaceVrrpResponse.PrioSlave = "?"
				ifaceVrrpResponse.IPVip = []string{"?"}
				ifaceVrrpResponse.IDVrrp = "?"
				ifaceVrrpResponse.AuthType = ""
				ifaceVrrpResponse.AuthPass = ""
				ifaceVrrpResponse.SyncIface = ""
				ifaceVrrpResponse.GarpMDelay = ""
				ifaceVrrpResponse.AdvertInt = ""
			}
		} else {
			w.WriteHeader(http.StatusPartialContent)
			ifaceVrrpResponse.PrioSlave = "?"
			ifaceVrrpResponse.IPVip = []string{"?"}
			ifaceVrrpResponse.IDVrrp = "?"
			ifaceVrrpResponse.AuthType = ""
			ifaceVrrpResponse.AuthPass = ""
			ifaceVrrpResponse.SyncIface = ""
			ifaceVrrpResponse.GarpMDelay = ""
			ifaceVrrpResponse.AdvertInt = ""
		}
	}
	js, err := json.Marshal(ifaceVrrpResponse)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

// changeIfaceVrrp on master API for change configuration needed (network + vrrp) on master & slave server
func changeIfaceVrrp(w http.ResponseWriter, r *http.Request) {
	if *htpasswdfile != "" {
		htpasswd := auth.HtpasswdFileProvider(*htpasswdfile)
		authenticator := auth.BasicAuth{
			Realm:   "Basic Realm",
			Secrets: htpasswd,
		}
		usercheck := authenticator.CheckAuth(r)
		if usercheck == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	var ifaceVrrp ifaceVrrpType
	vars := mux.Vars(r)
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&ifaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	ifaceVrrp.Iface = vars["iface"]
	sort.Strings(ifaceVrrp.IPVip)
	sort.Strings(ifaceVrrp.PostUp)

	validate := ifaceVrrp.validate()
	if validate != "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, validate)
		return
	}
	// iface configuration
	if !ifaceVrrp.IPVipOnly {
		ifaceExistsMaster := checkIfaceExists(ifaceVrrp)
		if !ifaceExistsMaster {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Iface", ifaceVrrp.Iface, "does not exist on master")
			return
		}
		ifaceExistsSlave, err := checkIfaceSlaveExists(ifaceVrrp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if !ifaceExistsSlave {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Iface", ifaceVrrp.Iface, "does not exist on slave")
			return
		}
		ifaceOkMaster, err := checkIfaceOk(ifaceVrrp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if !ifaceOkMaster {
			ifaceOkWithoutPostup, err := checkIfaceWithoutPostup(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if !ifaceOkWithoutPostup {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "[MASTER] Change IP_master, IP_slave, Mask, Default_GW,"+
					" LACP_slaves_master, LACP_slaves_slave or Vlan_device isn't possible")
				return
			}
			err = changeIfacePostup(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			err = removeIfaceFile(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			err = addIfaceFile(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
		ifaceOkSlave, err := checkIfaceSlaveOk(ifaceVrrp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if !ifaceOkSlave {
			ifaceOkWithoutPostup, err := checkIfaceSlaveWithoutPostup(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if !ifaceOkWithoutPostup {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "[SLAVE] Change IP_master, IP_slave, Mask, Default_GW,"+
					" LACP_slaves_master, LACP_slaves_slave or Vlan_device isn't possible")
				return
			}
			err = changeIfaceSlavePostup(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			err = removeIfaceSlaveFile(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			err = addIfaceSlaveFile(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
	}
	// vrrp configuration
	if len(ifaceVrrp.IPVip) != 0 {
		vrrpExistsMaster := checkVrrpExists(ifaceVrrp)
		vrrpExistsMasterOtherVG, err := checkVrrpExistsOtherVG(ifaceVrrp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		vrrpExistsSlave, err := checkVrrpSlaveExists(ifaceVrrp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		vrrpExistsSlaveOtherVG, err := checkVrrpSlaveExistsOtherVG(ifaceVrrp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var vrrpOkMaster bool
		var vrrpOkSlave bool
		var ifaceVrrpRmMaster ifaceVrrpType
		var ifaceVrrpRmSlave ifaceVrrpType

		switch {
		case vrrpExistsMaster:
			vrrpOkMaster, err = checkVrrpOk(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			ifaceVrrpRmMaster = ifaceVrrp
		case vrrpExistsMasterOtherVG != "":
			vrrpOkMaster = false
			ifaceVrrpRmMaster = ifaceVrrp
			ifaceVrrpRmMaster.VrrpGroup = vrrpExistsMasterOtherVG
		default:
			vrrpOkMaster = false
		}

		switch {
		case vrrpExistsSlave:
			vrrpOkSlave, err = checkVrrpSlaveOk(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			ifaceVrrpRmSlave = ifaceVrrp
		case vrrpExistsSlaveOtherVG != "":
			vrrpOkSlave = false
			ifaceVrrpRmSlave = ifaceVrrp
			ifaceVrrpRmSlave.VrrpGroup = vrrpExistsSlaveOtherVG
		default:
			vrrpOkSlave = false
		}

		if !vrrpOkMaster {
			mutex.Lock()
			if ifaceVrrpRmMaster.VrrpGroup != "" {
				err = removeVrrp(ifaceVrrpRmMaster)
				if err != nil {
					http.Error(w, err.Error(), 500)
					mutex.Unlock()
					return
				}
			}
			err = addVrrp(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
			err = reloadVrrp()
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
			// reload twice for vmac up before add IP (bug keepalived)
			// reload twice for new vrrp comme up

			sleep()
			err = syncGroupAndReload()
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
			sleep()

			if !vrrpOkSlave {
				if ifaceVrrpRmSlave.VrrpGroup != "" {
					err = removeVrrpSlave(ifaceVrrpRmSlave)
					if err != nil {
						http.Error(w, err.Error(), 500)
						return
					}
				}
				err = addVrrpSlave(ifaceVrrp)
				if err != nil {
					http.Error(w, err.Error(), 500)
					mutex.Unlock()
					return
				}
				err = syncGroupAndReloadSlave()
				if err != nil {
					http.Error(w, err.Error(), 500)
					mutex.Unlock()
					return
				}
				// reload twice for vmac up before add IP (bug keepalived)
				if ifaceVrrp.UseVmac {
					sleep()
					err = syncGroupAndReloadSlave()
					if err != nil {
						http.Error(w, err.Error(), 500)
						mutex.Unlock()
						return
					}
				}
				sleep()
			}
			mutex.Unlock()
		} else if !vrrpOkSlave {
			mutex.Lock()
			if ifaceVrrpRmSlave.VrrpGroup != "" {
				err := removeVrrpSlave(ifaceVrrpRmSlave)
				if err != nil {
					http.Error(w, err.Error(), 500)
					mutex.Unlock()
					return
				}
			}
			err = addVrrpSlave(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
			err = syncGroupAndReloadSlave()
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
			// reload twice for vmac up before add IP (bug keepalived)
			if ifaceVrrp.UseVmac {
				sleep()
				err = syncGroupAndReloadSlave()
				if err != nil {
					http.Error(w, err.Error(), 500)
					mutex.Unlock()
					return
				}
			}
			sleep()
			mutex.Unlock()
		}
	} else {
		mutex.Lock()
		vrrpExistsSlave, err := checkVrrpSlaveExists(ifaceVrrp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			mutex.Unlock()
			return
		}
		if vrrpExistsSlave {
			err := removeVrrpSlave(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
			err = syncGroupAndReloadSlave()
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
			sleep()
		}

		vrrpExistsMaster := checkVrrpExists(ifaceVrrp)
		if vrrpExistsMaster {
			err := removeVrrp(ifaceVrrp)
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
			err = syncGroupAndReload()
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
			sleep()
		}
		mutex.Unlock()
	}
}

// moveIDIfaceVrrp on master API for change ID vrrp without vrrp flap on slave
func moveIDIfaceVrrp(w http.ResponseWriter, r *http.Request) {
	if *htpasswdfile != "" {
		htpasswd := auth.HtpasswdFileProvider(*htpasswdfile)
		authenticator := auth.BasicAuth{
			Realm:   "Basic Realm",
			Secrets: htpasswd,
		}
		usercheck := authenticator.CheckAuth(r)
		if usercheck == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	var ifaceVrrp ifaceVrrpType
	var ifaceVrrpOldID ifaceVrrpType
	vars := mux.Vars(r)
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&ifaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	ifaceVrrp.Iface = vars["iface"]
	sort.Strings(ifaceVrrp.IPVip)
	sort.Strings(ifaceVrrp.PostUp)

	validate := ifaceVrrp.validate()
	if validate != "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, validate)
		return
	}
	ifaceVrrpOldID = ifaceVrrp
	ifaceVrrpOldID.IDVrrp = vars["old_Id_vrrp"]
	if len(ifaceVrrp.IPVip) != 0 {
		mutex.Lock()
		vrrpExistsMaster := checkVrrpExists(ifaceVrrpOldID)
		if vrrpExistsMaster {
			vrrpOkMaster, err := checkVrrpWithoutSync(ifaceVrrpOldID)
			if err != nil {
				http.Error(w, err.Error(), 500)
				mutex.Unlock()
				return
			}
			if vrrpOkMaster {
				vrrpExistsSlave, err := checkVrrpSlaveExists(ifaceVrrpOldID)
				if err != nil {
					http.Error(w, err.Error(), 500)
					mutex.Unlock()
					return
				}
				if vrrpExistsSlave {
					vrrpOkSlave, err := checkVrrpSlaveWithoutSync(ifaceVrrpOldID)
					if err != nil {
						http.Error(w, err.Error(), 500)
						mutex.Unlock()
						return
					}
					if vrrpOkSlave {
						err := removeVrrpSlave(ifaceVrrpOldID)
						if err != nil {
							http.Error(w, err.Error(), 500)
							mutex.Unlock()
							return
						}
						err = syncGroupAndReloadSlave()
						if err != nil {
							http.Error(w, err.Error(), 500)
							mutex.Unlock()
							return
						}
						sleep()

						err = removeVrrp(ifaceVrrpOldID)
						if err != nil {
							http.Error(w, err.Error(), 500)
							mutex.Unlock()
							return
						}
						err = addVrrp(ifaceVrrp)
						if err != nil {
							http.Error(w, err.Error(), 500)
							mutex.Unlock()
							return
						}
						err = reloadVrrp()
						if err != nil {
							http.Error(w, err.Error(), 500)
							mutex.Unlock()
							return
						}
						// reload twice for vmac up before add IP (bug keepalived)
						// reload twice for new vrrp comme up
						sleep()
						err = syncGroupAndReload()
						if err != nil {
							http.Error(w, err.Error(), 500)
							mutex.Unlock()
							return
						}
						sleep()

						err = addVrrpSlave(ifaceVrrp)
						if err != nil {
							http.Error(w, err.Error(), 500)
							mutex.Unlock()
							return
						}
						err = syncGroupAndReloadSlave()
						if err != nil {
							http.Error(w, err.Error(), 500)
							mutex.Unlock()
							return
						}
						// reload twice for vmac up before add IP (bug keepalived)
						if ifaceVrrp.UseVmac {
							sleep()
							err = syncGroupAndReloadSlave()
							if err != nil {
								http.Error(w, err.Error(), 500)
								mutex.Unlock()
								return
							}
						}
						sleep()

						mutex.Unlock()
					} else {
						mutex.Unlock()
						w.WriteHeader(http.StatusBadRequest)
						fmt.Fprintln(w, "different vrrp on slave => you can't change Id_vrrp and others options at the same time")
						return
					}
				} else {
					mutex.Unlock()
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, "unknown old vrrp id on slave")
					return
				}
			} else {
				mutex.Unlock()
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "different vrrp on master => you can't change Id_vrrp and others options at the same time")
				return
			}
		} else {
			mutex.Unlock()
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "unknown old vrrp id on master")
			return
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "IP_vip empty, no move needed")
		return
	}
}

// add vrrp script file and reload keepalived on master and slave
func addVrrpScript(w http.ResponseWriter, r *http.Request) {
	if *htpasswdfile != "" {
		htpasswd := auth.HtpasswdFileProvider(*htpasswdfile)
		authenticator := auth.BasicAuth{
			Realm:   "Basic Realm",
			Secrets: htpasswd,
		}
		usercheck := authenticator.CheckAuth(r)
		if usercheck == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	var vrrpScript vrrpScriptType
	vars := mux.Vars(r)
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&vrrpScript)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if vrrpScript.Name != vars["name"] {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "name in url and json are not same")
		return
	}

	validate := vrrpScript.validate()
	if validate != "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, validate)
		return
	}
	mutex.Lock()
	if checkVrrpScriptExists(vrrpScript.Name) {
		vrrpScriptOk, err := checkVrrpScriptOk(vrrpScript)
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
		if !vrrpScriptOk {
			mutex.Unlock()
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "vrrp_script already exist on master with different config")
		}
	} else {
		err := addVrrpScriptFile(vrrpScript)
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
		err = reloadVrrp()
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
		sleep()
	}
	vrrpScriptSlaveExists, err := checkVrrpScriptExistsSlave(vrrpScript)
	if err != nil {
		mutex.Unlock()
		http.Error(w, err.Error(), 500)
		return
	}
	if vrrpScriptSlaveExists {
		vrrpScriptOk, err := vrrpScriptOkSlave(vrrpScript)
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
		if !vrrpScriptOk {
			mutex.Unlock()
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "vrrp_script already exist on slave with different config")
			return
		}
	} else {
		err := addVrrpScriptSlave(vrrpScript)
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
		err = reloadVrrpSlave()
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
		sleep()
	}
	mutex.Unlock()
}

// remove vrrp script file and reload keepalived on master and slave
func removeVrrpScript(w http.ResponseWriter, r *http.Request) {
	if *htpasswdfile != "" {
		htpasswd := auth.HtpasswdFileProvider(*htpasswdfile)
		authenticator := auth.BasicAuth{
			Realm:   "Basic Realm",
			Secrets: htpasswd,
		}
		usercheck := authenticator.CheckAuth(r)
		if usercheck == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	var vrrpScript vrrpScriptType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&vrrpScript)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	mutex.Lock()
	if checkVrrpScriptExists(vrrpScript.Name) {
		err := removeVrrpScriptFile(vrrpScript)
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
		err = reloadVrrp()
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
		sleep()
	} else {
		err = reloadVrrp()
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
		sleep()
	}
	vrrpScriptSlaveExists, err := checkVrrpScriptExistsSlave(vrrpScript)
	if err != nil {
		mutex.Unlock()
		http.Error(w, err.Error(), 500)
		return
	}
	if vrrpScriptSlaveExists {
		err := removeVrrpScriptSlave(vrrpScript)
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
		err = reloadVrrpSlave()
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
		sleep()
	} else {
		err = reloadVrrpSlave()
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
		sleep()
	}
	mutex.Unlock()
}
func changeVrrpScript(w http.ResponseWriter, r *http.Request) {
	if *htpasswdfile != "" {
		htpasswd := auth.HtpasswdFileProvider(*htpasswdfile)
		authenticator := auth.NewBasicAuthenticator("Basic Realm", htpasswd)
		usercheck := authenticator.CheckAuth(r)
		if usercheck == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	var vrrpScript vrrpScriptType
	vars := mux.Vars(r)
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&vrrpScript)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if vrrpScript.Name != vars["name"] {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "name in url and json are not same")
		return
	}

	validate := vrrpScript.validate()
	if validate != "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, validate)
		return
	}
	mutex.Lock()
	if checkVrrpScriptExists(vrrpScript.Name) {
		err = removeVrrpScriptFile(vrrpScript)
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
	}
	err = addVrrpScriptFile(vrrpScript)
	if err != nil {
		mutex.Unlock()
		http.Error(w, err.Error(), 500)
		return
	}
	err = reloadVrrp()
	if err != nil {
		mutex.Unlock()
		http.Error(w, err.Error(), 500)
		return
	}
	sleep()

	vrrpScriptSlaveExists, err := checkVrrpScriptExistsSlave(vrrpScript)
	if err != nil {
		mutex.Unlock()
		http.Error(w, err.Error(), 500)
		return
	}
	if vrrpScriptSlaveExists {
		err = removeVrrpScriptSlave(vrrpScript)
		if err != nil {
			mutex.Unlock()
			http.Error(w, err.Error(), 500)
			return
		}
	}
	err = addVrrpScriptSlave(vrrpScript)
	if err != nil {
		mutex.Unlock()
		http.Error(w, err.Error(), 500)
		return
	}
	err = reloadVrrpSlave()
	if err != nil {
		mutex.Unlock()
		http.Error(w, err.Error(), 500)
		return
	}
	sleep()
	mutex.Unlock()
}

// read vrrp file on master and check if same on slave
func checkVrrpScript(w http.ResponseWriter, r *http.Request) {
	if *htpasswdfile != "" {
		htpasswd := auth.HtpasswdFileProvider(*htpasswdfile)
		authenticator := auth.BasicAuth{
			Realm:   "Basic Realm",
			Secrets: htpasswd,
		}
		usercheck := authenticator.CheckAuth(r)
		if usercheck == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	vars := mux.Vars(r)
	var vrrpScriptRead vrrpScriptType
	var err error
	if checkVrrpScriptExists(vars["name"]) {
		vrrpScriptRead, err = readVrrpScriptFile(vars["name"])
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	vrrpScriptSlaveExists, err := checkVrrpScriptExistsSlave(vrrpScriptType{
		Name: vars["name"],
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if !vrrpScriptSlaveExists {
		http.Error(w, "script exists on master but not find on slave", 500)
	} else {
		vrrpScriptOk, err := vrrpScriptOkSlave(vrrpScriptRead)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if !vrrpScriptOk {
			http.Error(w, "script master/slave not same", 500)
			return
		}
	}
	js, err := json.Marshal(vrrpScriptRead)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

// check vrrpScriptType parameters
func (vrrpScript vrrpScriptType) validate() string {
	if vrrpScript.Interval < 1 {
		return "interval too small"
	}
	if vrrpScript.Timeout > vrrpScript.Interval {
		return "timeout too long with this interval"
	}
	if vrrpScript.Weight < -253 || vrrpScript.Weight > 253 {
		return "weight is not in valid range"
	}
	if vrrpScript.Script == "" {
		return "missing script"
	}
	if vrrpScript.Fall < 1 {
		return "fall too small"
	}
	if vrrpScript.Rise < 1 {
		return "fall too small"
	}
	return ""
}
