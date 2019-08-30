package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// onslaveCheckIfaceExists : request received on slave for check network config file exists => checkIfaceExists()
func onslaveCheckIfaceExists(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	ifaceExists := checkIfaceExists(IfaceVrrp)
	if !ifaceExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

// onslaveCheckIfaceOk : request received on slave to slave for check network config file => checkIfaceOk()
func onslaveCheckIfaceOk(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	ifaceOk, err := checkIfaceOk(IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
	if !ifaceOk {
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

// onslaveCheckIfaceOk : request received on slave for check network config file without PostUp parameter => checkIfaceWithoutPostup()
func onslaveCheckIfaceWithoutPostup(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	ifaceOk, err := checkIfaceWithoutPostup(IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
	if !ifaceOk {
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

// onslaveAddIface : request received on slave for create network config file and ifup => addIface()
func onslaveAddIface(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	err = addIface(IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

// onslaveAddIfaceFile : request received on slave for create network config file (no ifup) => addIfaceFile()
func onslaveAddIfaceFile(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	err = addIfaceFile(IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

// onslaveAddIfaceFile : request received on slave for ifdown and remove network config file => removeIface()
func onslaveRemoveIface(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	err = removeIface(IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

// onslaveAddIfaceFile : request received on slave for remove network config file => removeIfaceFile()
func onslaveRemoveIfaceFile(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	err = removeIfaceFile(IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

// onslaveChangeIfacePostup : request received on slave for rewrite post-up line and apply/revert modification => changeIfacePostup()
func onslaveChangeIfacePostup(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	err = changeIfacePostup(IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

}

// onslaveCheckVrrpExists : request received on slave for check vrrp config file exists => checkVrrpExists()
func onslaveCheckVrrpExists(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	vrrpExists := checkVrrpExists(IfaceVrrp)
	if !vrrpExists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

// onslaveCheckVrrpExistsOtherVG : request received on slave for check vrrp config file exists in other VG (in json) => checkVrrpExistsOtherVG()
func onslaveCheckVrrpExistsOtherVG(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	VG, err := checkVrrpExistsOtherVG(IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if VG == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	fmt.Fprintln(w, VG)
}

// onslaveCheckVrrpOk : request received on slave for check vrrp config file => checkVrrpOk()
func onslaveCheckVrrpOk(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	vrrpOk, err := checkVrrpOk(IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
	if !vrrpOk {
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

// onslaveCheckVrrpWithoutSync : request received on slave for check vrrp config file without interface line => checkVrrpWithoutSync()
func onslaveCheckVrrpWithoutSync(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	vrrpOk, err := checkVrrpWithoutSync(IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
	if !vrrpOk {
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

// onslaveCheckVrrpWithoutSync : request received on slave for reload keepalived service => reloadVrrp()
func onslaveReloadVrrp(w http.ResponseWriter, r *http.Request) {
	err := reloadVrrp()
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// onslaveAddVrrp : request received on slave for add vrrp config file => addVrrp()
func onslaveAddVrrp(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	err = addVrrp(IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// onslaveRemoveVrrp : request received on slave for remove vrrp config file => removeVrrp()
func onslaveRemoveVrrp(w http.ResponseWriter, r *http.Request) {
	var IfaceVrrp ifaceVrrpType
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	vars := mux.Vars(r)
	IfaceVrrp.Iface = vars["iface"]
	err = removeVrrp(IfaceVrrp)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}
