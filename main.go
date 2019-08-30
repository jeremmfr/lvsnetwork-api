package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type ifaceVrrpType struct {
	IPVipOnly         bool     `json:"IP_vip_only"`
	UseVmac           bool     `json:"Use_vmac"`
	Iface             string   `json:"iface"`
	IPMaster          string   `json:"IP_master"`
	IPSlave           string   `json:"IP_slave"`
	Mask              string   `json:"Mask"`
	PrioMaster        string   `json:"Prio_master"`
	PrioSlave         string   `json:"Prio_slave"`
	VlanDevice        string   `json:"Vlan_device"`
	VrrpGroup         string   `json:"Vrrp_group"`
	IfaceForVrrp      string   `json:"Iface_vrrp"`
	IDVrrp            string   `json:"Id_vrrp"`
	AuthType          string   `json:"Auth_type"`
	AuthPass          string   `json:"Auth_pass"`
	DefaultGW         string   `json:"Default_GW"`
	LACPSlavesMaster  string   `json:"LACP_slaves_master"`
	LACPSlavesSlave   string   `json:"LACP_slaves_slave"`
	SyncIface         string   `json:"Sync_iface"`
	GarpMDelay        string   `json:"Garp_m_delay"`
	GarpMasterRefresh string   `json:"Garp_master_refresh"`
	AdvertInt         string   `json:"Advert_int"`
	IPVip             []string `json:"IP_vip"`
	PostUp            []string `json:"Post_up"`
}

var (
	htpasswdfile            *string
	isSlave                 *bool
	listenIPSlave           *string
	listenPortSlave         *string
	httpsSlave              *bool
	timeSleep               *int
	reloadKeepalivedCommand *string
	debug                   *bool
	mutex                   = &sync.Mutex{}
)

const ipv4str string = "ipv4"
const ipv6str string = "ipv6"

func main() {
	listenIP := flag.String("ip", "127.0.0.1", "listen on IP")
	listenPort := flag.String("port", "8080", "listen on port")
	https := flag.Bool("https", false, "https = true or false")
	cert := flag.String("cert", "", "file of certificat for https")
	key := flag.String("key", "", "file of key for https")
	accessLogFile := flag.String("log", "/var/log/lvsnetwork-api.access.log", "file for access log")
	htpasswdfile = flag.String("htpasswd", "", "htpasswd file for login:password")
	isSlave = flag.Bool("is_slave", false, "slave ?")
	listenIPSlave = flag.String("ip_slave", "172.17.197.82", "listen slave on IP")
	listenPortSlave = flag.String("port_slave", "8080", "listen slave on port")
	httpsSlave = flag.Bool("https_slave", false, "https for request from master to slave ?")
	timeSleep = flag.Int("sleep", 10, "time for sleep before check iface communicate")
	reloadKeepalivedCommand = flag.String("reload_cmd", "/etc/init.d/keepalived-vrrp reload", "command for reload vrrp keepalived process")
	debug = flag.Bool("debug", false, "debug for file comparison")

	flag.Parse()

	// accesslog file open
	accessLog, err := os.OpenFile(*accessLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open access log: %s", err)
	}

	checkIfupdownVersion()

	// create router
	router := mux.NewRouter().StrictSlash(true)
	if *isSlave {
		router.HandleFunc("/check_iface_exists/{iface}/", onslaveCheckIfaceExists)
		router.HandleFunc("/check_iface_ok/{iface}/", onslaveCheckIfaceOk)
		router.HandleFunc("/check_iface_without_postup/{iface}/", onslaveCheckIfaceWithoutPostup)
		router.HandleFunc("/add_iface/{iface}/", onslaveAddIface)
		router.HandleFunc("/add_iface_file/{iface}/", onslaveAddIfaceFile)
		router.HandleFunc("/remove_iface/{iface}/", onslaveRemoveIface)
		router.HandleFunc("/remove_iface_file/{iface}/", onslaveRemoveIfaceFile)
		router.HandleFunc("/change_iface_postup/{iface}/", onslaveChangeIfacePostup)
		router.HandleFunc("/check_vrrp_exists/{iface}/", onslaveCheckVrrpExists)
		router.HandleFunc("/check_vrrp_exists_otherVG/{iface}/", onslaveCheckVrrpExistsOtherVG)
		router.HandleFunc("/check_vrrp_ok/{iface}/", onslaveCheckVrrpOk)
		router.HandleFunc("/check_vrrp_without_sync/{iface}/", onslaveCheckVrrpWithoutSync)
		router.HandleFunc("/add_vrrp/{iface}/", onslaveAddVrrp)
		router.HandleFunc("/remove_vrrp/{iface}/", onslaveRemoveVrrp)
		router.HandleFunc("/reload_vrrp/", onslaveReloadVrrp)

		loggedRouter := handlers.CombinedLoggingHandler(accessLog, router)

		if *https {
			if (*cert == "") || (*key == "") {
				log.Fatalf("HTTPS true but no cert and key defined")
			} else {
				log.Fatal(http.ListenAndServeTLS(strings.Join([]string{*listenIPSlave, ":", *listenPortSlave}, ""), *cert, *key, loggedRouter))
			}
		} else {
			log.Fatal(http.ListenAndServe(strings.Join([]string{*listenIPSlave, ":", *listenPortSlave}, ""), loggedRouter))
		}
	} else {
		router.HandleFunc("/add_iface_vrrp/{iface}/", addIfaceVrrp)
		router.HandleFunc("/remove_iface_vrrp/{iface}/", removeIfaceVrrp)
		router.HandleFunc("/check_iface_vrrp/{iface}/", checkIfaceVrrp)
		router.HandleFunc("/change_iface_vrrp/{iface}/", changeIfaceVrrp)
		router.HandleFunc("/moveid_iface_vrrp/{iface}/{old_Id_vrrp}/", moveIDIfaceVrrp)

		loggedRouter := handlers.CombinedLoggingHandler(accessLog, router)

		if *https {
			if (*cert == "") || (*key == "") {
				log.Fatalf("HTTPS true but no cert and key defined")
			} else {
				log.Fatal(http.ListenAndServeTLS(strings.Join([]string{*listenIP, ":", *listenPort}, ""), *cert, *key, loggedRouter))
			}
		} else {
			log.Fatal(http.ListenAndServe(strings.Join([]string{*listenIP, ":", *listenPort}, ""), loggedRouter))
		}
	}
}

// checkIfupdownVersion : test ifquery version for --state options
func checkIfupdownVersion() {
	cmd := exec.Command("ifquery", "--help")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	returnCmd, _ := ioutil.ReadAll(stdout)

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}
	if !strings.Contains(string(returnCmd), "state") {
		log.Fatalf("no state for ifquery, go upgrade ifupdown package")
	}
}

// sleep : just wait
func sleep() {
	time.Sleep(time.Duration(*timeSleep) * time.Second)
}
