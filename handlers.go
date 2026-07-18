package main

import (
	"net/http"
	"strconv"
	"strings"
)

// ─── Helpers ───

func getCreds(userID int) map[string]string {
	cred := map[string]string{
		"api_token": "", "zone_id": "", "domain": "", "vps_ip": "",
		"ssh_host": "", "ssh_port": "22", "ssh_user": "root", "ssh_password": "",
	}
	var apiToken, zoneID, domain, vpsIP, sshHost, sshUser, sshPassword string
	var sshPort int
	err := db.QueryRow(`SELECT api_token, zone_id, domain, vps_ip, ssh_host, ssh_port, ssh_user, ssh_password 
		FROM credentials WHERE user_id = ?`, userID).Scan(
		&apiToken, &zoneID, &domain, &vpsIP, &sshHost, &sshPort, &sshUser, &sshPassword,
	)
	if err != nil {
		return cred
	}
	cred["api_token"] = apiToken
	cred["zone_id"] = zoneID
	cred["domain"] = domain
	cred["vps_ip"] = vpsIP
	cred["ssh_host"] = sshHost
	cred["ssh_port"] = strconv.Itoa(sshPort)
	cred["ssh_user"] = sshUser
	cred["ssh_password"] = sshPassword
	return cred
}

func saveCreds(userID int, cred map[string]string) {
	port, _ := strconv.Atoi(cred["ssh_port"])
	if port == 0 {
		port = 22
	}
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM credentials WHERE user_id = ?", userID).Scan(&exists)
	if exists > 0 {
		db.Exec(`UPDATE credentials SET api_token=?, zone_id=?, domain=?, vps_ip=?, ssh_host=?, ssh_port=?, ssh_user=?, ssh_password=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=?`,
			cred["api_token"], cred["zone_id"], cred["domain"], cred["vps_ip"],
			cred["ssh_host"], port, cred["ssh_user"], cred["ssh_password"], userID)
	} else {
		db.Exec(`INSERT INTO credentials (user_id, api_token, zone_id, domain, vps_ip, ssh_host, ssh_port, ssh_user, ssh_password) VALUES (?,?,?,?,?,?,?,?,?)`,
			userID, cred["api_token"], cred["zone_id"], cred["domain"], cred["vps_ip"],
			cred["ssh_host"], port, cred["ssh_user"], cred["ssh_password"])
	}
}

func getUserID(r *http.Request) int {
	id, _ := strconv.Atoi(r.Header.Get("X-User-Id"))
	return id
}

func getUsername(r *http.Request) string {
	return r.Header.Get("X-Username")
}

func getSSHConfig(r *http.Request) *SSHConfig {
	cred := getCreds(getUserID(r))
	port, _ := strconv.Atoi(cred["ssh_port"])
	if port == 0 {
		port = 22
	}
	return &SSHConfig{
		Host:     cred["ssh_host"],
		Port:     port,
		User:     cred["ssh_user"],
		Password: cred["ssh_password"],
	}
}

// ─── Page Handlers ───

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "dashboard.html", map[string]interface{}{
		"PageTitle":  "Dashboard",
		"ActiveMenu": "dashboard",
		"Username":   getUsername(r),
	})
}

func handleCredentialsPage(w http.ResponseWriter, r *http.Request) {
	cred := getCreds(getUserID(r))
	renderPage(w, "credentials.html", map[string]interface{}{
		"PageTitle":  "Credentials",
		"ActiveMenu": "credentials",
		"Username":   getUsername(r),
		"ApiToken":   cred["api_token"],
		"ZoneID":     cred["zone_id"],
		"Domain":     cred["domain"],
		"VpsIP":      cred["vps_ip"],
		"SSHHost":    cred["ssh_host"],
		"SSHPort":    cred["ssh_port"],
		"SSHUser":    cred["ssh_user"],
		"SSHPassword": cred["ssh_password"],
	})
}

func handleSetupFallbackPage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "setup-fallback.html", map[string]interface{}{
		"PageTitle":  "Setup Fallback",
		"ActiveMenu": "setup-fallback",
		"Username":   getUsername(r),
	})
}

func handleCreateHostnamePage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "create-hostname.html", map[string]interface{}{
		"PageTitle":  "Create Hostname",
		"ActiveMenu": "create-hostname",
		"Username":   getUsername(r),
	})
}

func handleBulkHostnamesPage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "bulk-hostnames.html", map[string]interface{}{
		"PageTitle":  "Bulk Hostnames",
		"ActiveMenu": "bulk-hostnames",
		"Username":   getUsername(r),
	})
}

func handleListHostnamesPage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "list-hostnames.html", map[string]interface{}{
		"PageTitle":  "List Hostnames",
		"ActiveMenu": "list-hostnames",
		"Username":   getUsername(r),
	})
}

func handleDnsManagerPage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "dns-manager.html", map[string]interface{}{
		"PageTitle":  "DNS Manager",
		"ActiveMenu": "dns-manager",
		"Username":   getUsername(r),
	})
}

// ─── API Handlers ───

func handleCheckStatusGlobal(w http.ResponseWriter, r *http.Request) {
	cred := getCreds(getUserID(r))
	connected := cred["api_token"] != "" && cred["zone_id"] != "" && cred["domain"] != ""
	jsonResp(w, 200, map[string]interface{}{"connected": connected})
}

func handleCheckConfig(w http.ResponseWriter, r *http.Request) {
	cred := map[string]string{
		"api_token":    r.FormValue("api_token"),
		"zone_id":      r.FormValue("zone_id"),
		"domain":       r.FormValue("domain"),
		"vps_ip":       r.FormValue("vps_ip"),
		"ssh_host":     r.FormValue("ssh_host"),
		"ssh_port":     r.FormValue("ssh_port"),
		"ssh_user":     r.FormValue("ssh_user"),
		"ssh_password": r.FormValue("ssh_password"),
	}
	if cred["api_token"] == "" || cred["zone_id"] == "" || cred["domain"] == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Semua field wajib diisi"})
		return
	}

	saveCreds(getUserID(r), cred)

	cf := NewCFClient(cred["api_token"], cred["zone_id"])
	zoneInfo, err := cf.GetZoneInfo()
	if err != nil {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Zone ID atau API Token tidak valid"})
		return
	}
	if success, _ := zoneInfo["success"].(bool); !success {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Zone ID atau API Token tidak valid"})
		return
	}

	zoneName := ""
	if result, ok := zoneInfo["result"].(map[string]interface{}); ok {
		zoneName, _ = result["name"].(string)
	}
	jsonResp(w, 200, map[string]interface{}{"success": true, "zone_name": zoneName})
}

func handleSetupFallback(w http.ResponseWriter, r *http.Request) {
	cred := getCreds(getUserID(r))
	vpsIP := r.FormValue("vps_ip")
	if vpsIP == "" {
		vpsIP = cred["vps_ip"]
	}
	if cred["api_token"] == "" || cred["zone_id"] == "" || cred["domain"] == "" || vpsIP == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Kredensial tidak lengkap. Isi dulu di menu Credentials."})
		return
	}

	cf := NewCFClient(cred["api_token"], cred["zone_id"])
	steps := cf.SetupFallback(vpsIP, cred["domain"])
	jsonResp(w, 200, map[string]interface{}{"success": true, "steps": steps})
}

func handleCreateHostnameAPI(w http.ResponseWriter, r *http.Request) {
	cred := getCreds(getUserID(r))
	targetDomain := r.FormValue("target_domain")
	method := r.FormValue("method")
	if method == "" {
		method = "http"
	}
	if cred["api_token"] == "" || cred["zone_id"] == "" || cred["domain"] == "" || targetDomain == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Kredensial tidak lengkap atau target domain kosong."})
		return
	}

	cf := NewCFClient(cred["api_token"], cred["zone_id"])
	result := cf.CreateWildcardHostname(targetDomain, cred["domain"], method)
	jsonResp(w, 200, map[string]interface{}{"success": true, "result": result})
}

func handleCheckStatus(w http.ResponseWriter, r *http.Request) {
	cred := getCreds(getUserID(r))
	hostnameID := r.FormValue("hostname_id")
	if cred["api_token"] == "" || cred["zone_id"] == "" || hostnameID == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Data tidak lengkap"})
		return
	}
	cf := NewCFClient(cred["api_token"], cred["zone_id"])
	result, _ := cf.GetCustomHostname(hostnameID)
	jsonResp(w, 200, result)
}

func handleListHostnames(w http.ResponseWriter, r *http.Request) {
	cred := getCreds(getUserID(r))
	if cred["api_token"] == "" || cred["zone_id"] == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Kredensial belum diisi"})
		return
	}
	cf := NewCFClient(cred["api_token"], cred["zone_id"])
	result, _ := cf.ListCustomHostnames()
	jsonResp(w, 200, result)
}

func handleDeleteHostname(w http.ResponseWriter, r *http.Request) {
	cred := getCreds(getUserID(r))
	hostnameID := r.FormValue("hostname_id")
	if cred["api_token"] == "" || cred["zone_id"] == "" || hostnameID == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Data tidak lengkap"})
		return
	}
	cf := NewCFClient(cred["api_token"], cred["zone_id"])

	// Get hostname before delete
	detail, _ := cf.GetCustomHostname(hostnameID)
	hostname := ""
	if result, ok := detail["result"].(map[string]interface{}); ok {
		hostname, _ = result["hostname"].(string)
	}

	result, _ := cf.DeleteCustomHostname(hostnameID)

	// Delete matching DNS records
	dnsDeleted := []string{}
	if hostname != "" {
		records, _ := cf.ListDnsRecords("")
		if recs, ok := records["result"].([]interface{}); ok {
			for _, rec := range recs {
				if rm, ok := rec.(map[string]interface{}); ok {
					if rm["name"] == hostname && rm["type"] == "A" {
						if id, ok := rm["id"].(string); ok {
							del, _ := cf.DeleteDnsRecord(id)
							if s, _ := del["success"].(bool); s {
								dnsDeleted = append(dnsDeleted, hostname)
							}
						}
					}
				}
			}
		}
	}
	result["dns_deleted"] = dnsDeleted
	jsonResp(w, 200, result)
}

func handleDeleteDnsRecord(w http.ResponseWriter, r *http.Request) {
	cred := getCreds(getUserID(r))
	recordID := r.FormValue("record_id")
	if cred["api_token"] == "" || cred["zone_id"] == "" || recordID == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Data tidak lengkap"})
		return
	}
	cf := NewCFClient(cred["api_token"], cred["zone_id"])
	result, _ := cf.DeleteDnsRecord(recordID)
	jsonResp(w, 200, result)
}

func handleUserInfo(w http.ResponseWriter, r *http.Request) {
	cred := getCreds(getUserID(r))
	jsonResp(w, 200, map[string]interface{}{
		"success":     true,
		"username":    getUsername(r),
		"credentials": cred,
	})
}

func handleGetDnsRecords(w http.ResponseWriter, r *http.Request) {
	cred := getCreds(getUserID(r))
	if cred["api_token"] == "" || cred["zone_id"] == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Kredensial belum diisi"})
		return
	}
	cf := NewCFClient(cred["api_token"], cred["zone_id"])
	result, _ := cf.ListDnsRecords("")

	// Filter out internal CF records
	if recs, ok := result["result"].([]interface{}); ok {
		filtered := []interface{}{}
		for _, rec := range recs {
			if rm, ok := rec.(map[string]interface{}); ok {
				if name, _ := rm["name"].(string); !strings.HasPrefix(name, "_cf-custom-hostname") {
					filtered = append(filtered, rec)
				}
			}
		}
		result["result"] = filtered
	}
	jsonResp(w, 200, result)
}

func handleUpdateDnsRecord(w http.ResponseWriter, r *http.Request) {
	cred := getCreds(getUserID(r))
	if cred["api_token"] == "" || cred["zone_id"] == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Kredensial belum diisi"})
		return
	}
	recordID := r.FormValue("record_id")
	dtype := r.FormValue("type")
	name := r.FormValue("name")
	content := r.FormValue("content")
	proxied := r.FormValue("proxied") == "true"
	ttl, _ := strconv.Atoi(r.FormValue("ttl"))
	if ttl == 0 {
		ttl = 1
	}

	if recordID == "" || dtype == "" || name == "" || content == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Data tidak lengkap"})
		return
	}

	cf := NewCFClient(cred["api_token"], cred["zone_id"])
	result, _ := cf.UpdateDnsRecord(recordID, map[string]interface{}{
		"type": dtype, "name": name, "content": content, "proxied": proxied, "ttl": ttl,
	})
	jsonResp(w, 200, result)
}

func handleCreateDnsRecord(w http.ResponseWriter, r *http.Request) {
	cred := getCreds(getUserID(r))
	if cred["api_token"] == "" || cred["zone_id"] == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Kredensial belum diisi"})
		return
	}
	dtype := r.FormValue("type")
	if dtype == "" {
		dtype = "A"
	}
	name := r.FormValue("name")
	content := r.FormValue("content")
	proxied := r.FormValue("proxied") == "true"

	if name == "" || content == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "Name dan content wajib"})
		return
	}

	cf := NewCFClient(cred["api_token"], cred["zone_id"])
	result, _ := cf.CreateDnsRecord(dtype, name, content, proxied)
	jsonResp(w, 200, result)
}

func handleCreateChallenge(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("challenge_token")
	body := r.FormValue("challenge_body")
	if token == "" || body == "" {
		jsonResp(w, 400, map[string]interface{}{"success": false, "error": "challenge_token dan challenge_body wajib"})
		return
	}
	ssh := getSSHConfig(r)
	result := ssh.CreateChallengeFile(token, body)
	jsonResp(w, 200, result)
}
