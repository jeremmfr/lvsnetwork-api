# lvsnetwork-api
![GitHub release (latest by date)](https://img.shields.io/github/v/release/jeremmfr/lvsnetwork-api)
[![Go Status](https://github.com/jeremmfr/lvsnetwork-api/workflows/Go%20Tests/badge.svg)](https://github.com/jeremmfr/lvsnetwork-api/actions)
[![Lint Status](https://github.com/jeremmfr/lvsnetwork-api/workflows/GolangCI-Lint/badge.svg)](https://github.com/jeremmfr/lvsnetwork-api/actions)
[![GoDoc](https://godoc.org/github.com/jeremmfr/lvsnetwork-api?status.svg)](https://godoc.org/github.com/jeremmfr/lvsnetwork-api)
[![Go Report Card](https://goreportcard.com/badge/github.com/jeremmfr/lvsnetwork-api)](https://goreportcard.com/report/github.com/jeremmfr/lvsnetwork-api)

Create API REST for create/configure network interface and keepalived VRRP configuration


Compile:
--------

go build -o lvsnetwork-api

Run:
----
	./lvsnetwork-api -h
		Usage of ./lvsnetwork-api:
		  -cert string
		        file of certificat for https
		  -htpasswd string
		        htpasswd file for login:password
		  -https
		        https = true or false
		  -https_slave
		        https for request from master to slave ?
		  -ip string
		        listen on IP (default "127.0.0.1")
		  -ip_slave string
		        listen slave on IP (default "172.17.197.82")
		  -is_slave
		        slave ?
		  -key string
		        file of key for https
		  -log string
		        file for access log (default "/var/log/lvsnetwork-api.access.log")
		  -port string
		        listen on port (default "8080")
		  -port_slave string
		        listen slave on port (default "8080")
		  -reload_cmd string
		        command for reload vrrp keepalived process (default "/etc/init.d/keepalived-vrrp reload")
		  -sleep int
		        time for sleep between ifup master/slave and keepalived reload master/slave (default 10)

By default, lvsnetwork-api communicate with same application on other server with is_slave true.  
Iface configuration is set in directory **/etc/network/interfaces.d/**.  
Vrrp configuration is set in directory **/etc/keepalived/keepalived-vrrp.d/** with one directory per vrrp_sync_group.  
***
API List :
---------

**ADD ifacevrp**  
	`/add_iface_vrrp/{iface}/`  
**REMOVE ifacevrp**  
	`/remove_iface_vrrp/{iface}/`  
**CHECK ifacevrp**  
	`/check_iface_vrrp/{iface}/`  
**MODIFY ifacevrp** expect Id_vrrp  
	`/change_iface_vrrp/{iface}/`  
**MODIFY ifacevrp Id_vrrp**  
	`/moveid_iface_vrrp/{iface}/{old_Id_vrrp}/`  
**ADD vrrp_script**  
	`/add_vrrp_script/{name}/`  
**REMOVE vrrp_script**  
	`/remove_vrrp_script/{name}/`  
**CHECK vrrp_script**  
	`/check_vrrp_script/{name}/`  
**MODIFY vrrp_script**  
	`/change_vrrp_script/{name}/`  


All requests need json in body with parameters
* for ifacevrrp :
  * **IP_vip_only** (Optional) [Def: false] configure only vrrp configuration
  * **IP_vip** (Optional) list of IPv4 for vrrp configuration
  * **Id_vrrp** (Optional if IP_vip empty) id for vrrp configuration [between 1-255]
  * **Prio_master** (Optional if IP_vip empty) priority on master vrrp configuration
  * **Prio_slave** (Optional if IP_vip empty) priority on slave vrrp configuration
  * **Vrrp_group** (Optional if IP_vip empty) group for vrrp configuration (automatic create/delete directory in /etc/keepalived/keepalived-vrrp.d/)
  * **Iface_vrrp** (Optional) [Default: $iface] vrrp parameter : interface
  * **Garp_m_delay** (Optional) [Default: 5] vrrp paramter : garp_master_delay
  * **Garp_master_refresh** (Optional) vrrp paramter : garp_master_refresh
  * **Sync_iface** (Optional) vrrp parameter : lvs_sync_daemon_interface
  * **Auth_type** (Optional) vrrp parameter :  authentication auth_type
  * **Auth_pass** (Optional) vrrp parameter : authentication auth_pass
  * **Advert_int** (Optional) vrrp parameter : advert_int
  * **IP_master** (Optional if IP_vip_only=true or IP_vip empty) IPv4 for iface configuration on master server
  * **IP_slave** (Optional if IP_vip_only=true or IP_vip empty) IPv4 for iface configuration on slave server
  * **Mask** (Optional if IP_vip_only=true or IP_vip empty) short netmask for iface configuration on master/slave server
  * **Vlan_device** (Optional if iface != vlan* ) device for vlan configuration (vlan-raw-device)
  * **LACP_slaves_master** (Optional) add bonding 802.3ad configuration with slaves interfaces for master
  * **LACP_slaves_slave** (Optional) add bonding 802.3ad configuration with slaves interfaces for slave
  * **Default_GW** (Optional) gateway configuration for iface
  * **Post_up** (Optional) post-up line in iface configuration
  * **Use_vmac** (Optional) use vmac for vrrp configuration
  * **TrackScript** (Optional) List of track_script


* for vrrp_script:
  * **script** (Required) script with arguments if needed
  * **rise** (Required) number of successes for OK transition
  * **fall** (Required) number of successes for KO transition
  * **init_fail** (Optional) assume script initially is in failed state
  * **weight** (Optional) adjust priority by this weight
  * **weight_reverse** (Optional) reverse causes the direction of the adjustment of the priority to be reversed
  * **interval** (Optional) seconds between script invocations, default 1 if no set
  * **timeout** (Optional) seconds after which script is considered to have failed
  * **user** (Optional) user to run script under
