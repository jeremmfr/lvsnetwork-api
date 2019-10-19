package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// requestSlave : call HTTP request from MASTER to SLAVE
func requestSlave(url string, jsonBody interface{}) (int, string, error) {
	urlString := "http://" + *listenIPSlave + ":" + *listenPortSlave + url + "?&logname=lvsnetwork-master"
	tr := &http.Transport{
		DisableKeepAlives: true,
	}
	if *httpsSlave {
		urlString = strings.Replace(urlString, "http://", "https://", -1)
		tr = &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives: true,
		}
	}
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(jsonBody)
	if err != nil {
		return 500, "", err
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Post(urlString, "application/json; charset=utf-8", body)
	if err != nil {
		return 500, "", err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 500, "", err
	}
	return resp.StatusCode, string(respBody), err
}

// requestSlaveWithoutBody : call HTTP request from MASTER to SLAVE  without body
func requestSlaveWithoutBody(url string) (int, string, error) {
	urlString := "http://" + *listenIPSlave + ":" + *listenPortSlave + url + "?&logname=lvsnetwork-master"
	tr := &http.Transport{
		DisableKeepAlives: true,
	}
	if *httpsSlave {
		urlString = strings.Replace(urlString, "http://", "https://", -1)
		tr = &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives: true,
		}
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(urlString)
	if err != nil {
		return 500, "", err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 500, "", err
	}
	return resp.StatusCode, string(respBody), err
}

// checkIfaceSlaveExists : call /check_iface_exists/ on slave => onslaveCheckIfaceExists()
func checkIfaceSlaveExists(ifaceVrrp ifaceVrrpType) (bool, error) {
	statuscode, body, err := requestSlave(strings.Join([]string{"/check_iface_exists/",
		ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if (err != nil) || (statuscode == 500) {
		return false, err
	}
	if statuscode == 404 {
		return false, nil
	}
	if statuscode == 200 {
		return true, nil
	}
	return false, fmt.Errorf("error on slave => %v", body)
}

// checkIfaceSlaveOk : call /check_iface_ok/ on slave => onslaveCheckIfaceOk()
func checkIfaceSlaveOk(ifaceVrrp ifaceVrrpType) (bool, error) {
	statuscode, body, err := requestSlave(strings.Join([]string{"/check_iface_ok/",
		ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if (err != nil) || (statuscode == 500) {
		return false, err
	}
	if statuscode == 404 {
		return false, nil
	}
	if statuscode == 200 {
		return true, nil
	}
	return false, fmt.Errorf("error on slave => %v", body)
}

// checkIfaceSlaveWithoutPostup : call /check_iface_without_postup/ on slave => onslaveCheckIfaceWithoutPostup()
func checkIfaceSlaveWithoutPostup(ifaceVrrp ifaceVrrpType) (bool, error) {
	statuscode, body, err := requestSlave(strings.Join([]string{"/check_iface_without_postup/",
		ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if (err != nil) || (statuscode == 500) {
		return false, err
	}
	if statuscode == 404 {
		return false, nil
	}
	if statuscode == 200 {
		return true, nil
	}
	return false, fmt.Errorf("error on slave => %v", body)
}

// addIfaceSlave : call /add_iface/ on slave => onslaveAddIface()
func addIfaceSlave(ifaceVrrp ifaceVrrpType) error {
	statuscode, body, err := requestSlave(strings.Join([]string{"/add_iface/", ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if err != nil {
		return err
	}
	if statuscode == 200 {
		return nil
	}
	return fmt.Errorf("error on slave => %v", body)
}

// addIfaceSlaveFile : call /add_iface_file/ on slave => onslaveAddIfaceFile()
func addIfaceSlaveFile(ifaceVrrp ifaceVrrpType) error {
	statuscode, body, err := requestSlave(strings.Join([]string{"/add_iface_file/",
		ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if err != nil {
		return err
	}
	if statuscode == 200 {
		return nil
	}
	return fmt.Errorf("error on slave => %v", body)
}

// removeIfaceSlave : call /remove_iface/ on slave => onslaveRemoveIface()
func removeIfaceSlave(ifaceVrrp ifaceVrrpType) error {
	statuscode, body, err := requestSlave(strings.Join([]string{"/remove_iface/",
		ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if err != nil {
		return err
	}
	if statuscode == 200 {
		return nil
	}
	return fmt.Errorf("error on slave => %v", body)
}

// removeIfaceSlaveFile : call /remove_iface_file/ on slave => onslaveRemoveIfaceFile()
func removeIfaceSlaveFile(ifaceVrrp ifaceVrrpType) error {
	statuscode, body, err := requestSlave(strings.Join([]string{"/remove_iface_file/",
		ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if err != nil {
		return err
	}
	if statuscode == 200 {
		return nil
	}
	return fmt.Errorf("error on slave => %v", body)
}

// changeIfaceSlavePostup : call /change_iface_postup/ on slave  => onslaveChangeIfacePostup()
func changeIfaceSlavePostup(ifaceVrrp ifaceVrrpType) error {
	statuscode, body, err := requestSlave(strings.Join([]string{"/change_iface_postup/",
		ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if err != nil {
		return err
	}
	if statuscode == 200 {
		return nil
	}
	return fmt.Errorf("error on slave => %v", body)
}

// checkVrrpSlaveExists : call /check_vrrp_exists/ on slave => onslaveCheckVrrpExists()
func checkVrrpSlaveExists(ifaceVrrp ifaceVrrpType) (bool, error) {
	statuscode, body, err := requestSlave(strings.Join([]string{"/check_vrrp_exists/",
		ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if (err != nil) || (statuscode == 500) {
		return false, err
	}
	if statuscode == 404 {
		return false, nil
	}
	if statuscode == 200 {
		return true, nil
	}
	return false, fmt.Errorf("error on slave => %v", body)
}

// checkVrrpSlaveExistsOtherVG : call /check_vrrp_exists_otherVG/ on slave => onslaveCheckVrrpExistsOtherVG()
func checkVrrpSlaveExistsOtherVG(ifaceVrrp ifaceVrrpType) (string, error) {
	statuscode, body, err := requestSlave(strings.Join([]string{"/check_vrrp_exists_otherVG/",
		ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if (err != nil) || (statuscode == 500) {
		return "", err
	}
	if statuscode == 404 {
		return "", nil
	}
	if statuscode == 200 {
		return strings.Join(strings.Fields(body), ""), nil
	}
	return "", fmt.Errorf("error on slave => %v", body)
}

// checkVrrpSlaveOk : call /check_vrrp_ok/ on slave => onslaveCheckVrrpOk()
func checkVrrpSlaveOk(ifaceVrrp ifaceVrrpType) (bool, error) {
	statuscode, body, err := requestSlave(strings.Join([]string{"/check_vrrp_ok/",
		ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if (err != nil) || (statuscode == 500) {
		return false, err
	}
	if statuscode == 404 {
		return false, nil
	}
	if statuscode == 200 {
		return true, nil
	}
	return false, fmt.Errorf("error on slave => %v", body)
}

// checkVrrpSlaveWithoutSync : call /check_vrrp_without_sync/ on slave => onslaveCheckVrrpWithoutSync()
func checkVrrpSlaveWithoutSync(ifaceVrrp ifaceVrrpType) (bool, error) {
	statuscode, body, err := requestSlave(strings.Join([]string{"/check_vrrp_without_sync/",
		ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if (err != nil) || (statuscode == 500) {
		return false, err
	}
	if statuscode == 404 {
		return false, nil
	}
	if statuscode == 200 {
		return true, nil
	}
	return false, fmt.Errorf("error on slave => %v", body)
}

// syncGroupAndReloadSlave : call /sync_group_reload_vrrp/ on slave => onslaveSyncGroupAndReload()
func syncGroupAndReloadSlave() error {
	statuscode, body, err := requestSlaveWithoutBody("/sync_group_reload_vrrp/")
	if err != nil {
		return err
	}
	if statuscode == 200 {
		return nil
	}
	return fmt.Errorf("error on slave => %v", body)
}

// reloadVrrpSlave : call /reload_vrrp/ on slave => onslaveReloadVrrp()
func reloadVrrpSlave() error {
	statuscode, body, err := requestSlaveWithoutBody("/reload_vrrp/")
	if err != nil {
		return err
	}
	if statuscode == 200 {
		return nil
	}
	return fmt.Errorf("error on slave => %v", body)
}

// addVrrpSlave : call /add_vrrp/ on slave => onslaveAddVrrp()
func addVrrpSlave(ifaceVrrp ifaceVrrpType) error {
	statuscode, body, err := requestSlave(strings.Join([]string{"/add_vrrp/",
		ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if err != nil {
		return err
	}
	if statuscode == 200 {
		return nil
	}
	return fmt.Errorf("error on slave => %v", body)
}

// removeVrrpSlave : call /remove_vrrp/ on slave => onslaveRemoveVrrp()
func removeVrrpSlave(ifaceVrrp ifaceVrrpType) error {
	statuscode, body, err := requestSlave(strings.Join([]string{"/remove_vrrp/",
		ifaceVrrp.Iface, "/"}, ""), ifaceVrrp)
	if err != nil {
		return err
	}
	if statuscode == 200 {
		return nil
	}
	return fmt.Errorf("error on slave => %v", body)
}

// checkVrrpScriptExistsSlave : call /check_vrrp_script_exists/ on slave => onslaveCheckVrrpScriptExists()
func checkVrrpScriptExistsSlave(vrrpScript vrrpScriptType) (bool, error) {
	statuscode, body, err := requestSlave(strings.Join([]string{"/check_vrrp_script_exists/",
		vrrpScript.Name, "/"}, ""), vrrpScript)
	if (err != nil) || (statuscode == 500) {
		return false, err
	}
	if statuscode == 404 {
		return false, nil
	}
	if statuscode == 200 {
		return true, nil
	}
	return false, fmt.Errorf("error on slave => %v", body)
}

// vrrpScriptOkSlave : call /check_vrrp_script_ok/ on slave => onslaveCheckVrrpScriptOk()
func vrrpScriptOkSlave(vrrpScript vrrpScriptType) (bool, error) {
	statuscode, body, err := requestSlave(strings.Join([]string{"/check_vrrp_script_ok/",
		vrrpScript.Name, "/"}, ""), vrrpScript)
	if (err != nil) || (statuscode == 500) {
		return false, err
	}
	if statuscode == 404 {
		return false, nil
	}
	if statuscode == 200 {
		return true, nil
	}
	return false, fmt.Errorf("error on slave => %v", body)
}

// addVrrpScriptSlave : call /add_vrrp_script/ on slave => onslaveAddVrrpScript()
func addVrrpScriptSlave(vrrpScript vrrpScriptType) error {
	statuscode, body, err := requestSlave(strings.Join([]string{"/add_vrrp_script/",
		vrrpScript.Name, "/"}, ""), vrrpScript)
	if err != nil {
		return err
	}
	if statuscode == 200 {
		return nil
	}
	return fmt.Errorf("error on slave => %v", body)
}

// removeVrrpScriptSlave : call /remove_vrrp_script/ on slave => onslaveRemoveVrrpScript()
func removeVrrpScriptSlave(vrrpScript vrrpScriptType) error {
	statuscode, body, err := requestSlave(strings.Join([]string{"/remove_vrrp_script/",
		vrrpScript.Name, "/"}, ""), vrrpScript)
	if err != nil {
		return err
	}
	if statuscode == 200 {
		return nil
	}
	return fmt.Errorf("error on slave => %v", body)
}
