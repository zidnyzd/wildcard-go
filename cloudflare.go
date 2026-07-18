package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type CFClient struct {
	APIToken string
	ZoneID   string
	BaseURL  string
	Client   *http.Client
}

func NewCFClient(token, zoneID string) *CFClient {
	return &CFClient{
		APIToken: token,
		ZoneID:   zoneID,
		BaseURL:  "https://api.cloudflare.com/client/v4",
		Client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (cf *CFClient) request(method, path string, body interface{}) (map[string]interface{}, error) {
	reqURL := cf.BaseURL + path
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cf.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := cf.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CF API error: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

func (cf *CFClient) GetZoneInfo() (map[string]interface{}, error) {
	return cf.request("GET", "/zones/"+cf.ZoneID, nil)
}

func (cf *CFClient) ListDnsRecords(name string) (map[string]interface{}, error) {
	path := "/zones/" + cf.ZoneID + "/dns_records"
	if name != "" {
		path += "?name=" + url.QueryEscape(name)
	}
	return cf.request("GET", path, nil)
}

func (cf *CFClient) CreateDnsRecord(dtype, name, content string, proxied bool) (map[string]interface{}, error) {
	return cf.request("POST", "/zones/"+cf.ZoneID+"/dns_records", map[string]interface{}{
		"type": dtype, "name": name, "content": content, "proxied": proxied, "ttl": 1,
	})
}

func (cf *CFClient) UpdateDnsRecord(recordID string, data map[string]interface{}) (map[string]interface{}, error) {
	return cf.request("PUT", "/zones/"+cf.ZoneID+"/dns_records/"+recordID, data)
}

func (cf *CFClient) DeleteDnsRecord(recordID string) (map[string]interface{}, error) {
	return cf.request("DELETE", "/zones/"+cf.ZoneID+"/dns_records/"+recordID, nil)
}

func (cf *CFClient) SetFallbackOrigin(origin string) (map[string]interface{}, error) {
	return cf.request("PUT", "/zones/"+cf.ZoneID+"/custom_hostnames/fallback_origin", map[string]interface{}{"origin": origin})
}

func (cf *CFClient) GetFallbackOrigin() (map[string]interface{}, error) {
	return cf.request("GET", "/zones/"+cf.ZoneID+"/custom_hostnames/fallback_origin", nil)
}

func (cf *CFClient) CreateCustomHostname(hostname, method string) (map[string]interface{}, error) {
	return cf.request("POST", "/zones/"+cf.ZoneID+"/custom_hostnames", map[string]interface{}{
		"hostname": hostname,
		"ssl":      map[string]interface{}{"method": method, "type": "dv"},
	})
}

func (cf *CFClient) GetCustomHostname(id string) (map[string]interface{}, error) {
	return cf.request("GET", "/zones/"+cf.ZoneID+"/custom_hostnames/"+id, nil)
}

func (cf *CFClient) ListCustomHostnames() (map[string]interface{}, error) {
	return cf.request("GET", "/zones/"+cf.ZoneID+"/custom_hostnames", nil)
}

func (cf *CFClient) DeleteCustomHostname(id string) (map[string]interface{}, error) {
	return cf.request("DELETE", "/zones/"+cf.ZoneID+"/custom_hostnames/"+id, nil)
}

func (cf *CFClient) SetupFallback(vpsIP, domain string) []map[string]interface{} {
	steps := []map[string]interface{}{}

	// 1. Fallback A record
	fallbackName := "fallback." + domain
	existing, _ := cf.ListDnsRecords(fallbackName)
	if result, ok := existing["result"].([]interface{}); ok && len(result) > 0 {
		steps = append(steps, map[string]interface{}{"step": "DNS Fallback Record", "result": map[string]interface{}{"skipped": true, "record": result[0]}})
	} else {
		res, _ := cf.CreateDnsRecord("A", "fallback", vpsIP, true)
		steps = append(steps, map[string]interface{}{"step": "DNS Fallback Record", "result": res})
	}

	// 2. Wildcard A record — search in all records since CF filter may not match "*.domain"
	wildcardName := "*." + domain
	wcExisting, _ := cf.ListDnsRecords(wildcardName)
	if result, ok := wcExisting["result"].([]interface{}); ok && len(result) > 0 {
		steps = append(steps, map[string]interface{}{"step": "DNS Wildcard Record", "result": map[string]interface{}{"skipped": true, "record": result[0]}})
	} else {
		// Fallback: search all records for wildcard
		allRecs, _ := cf.ListDnsRecords("")
		found := false
		if result, ok := allRecs["result"].([]interface{}); ok {
			for _, rec := range result {
				if rm, ok := rec.(map[string]interface{}); ok {
					if rm["type"] == "A" && rm["name"] == wildcardName {
						steps = append(steps, map[string]interface{}{"step": "DNS Wildcard Record", "result": map[string]interface{}{"skipped": true, "record": rm}})
						found = true
						break
					}
				}
			}
		}
		if !found {
			res, _ := cf.CreateDnsRecord("A", "*", vpsIP, false)
			steps = append(steps, map[string]interface{}{"step": "DNS Wildcard Record", "result": res})
		}
	}

	// 3. Set fallback origin
	res, _ := cf.SetFallbackOrigin("fallback." + domain)
	steps = append(steps, map[string]interface{}{"step": "Set Fallback Origin", "result": res})

	return steps
}

func (cf *CFClient) CreateWildcardHostname(targetDomain, yourDomain, method string) map[string]interface{} {
	hostname := targetDomain + "." + yourDomain

	// Check if already exists
	existing, _ := cf.ListCustomHostnames()
	if result, ok := existing["result"].([]interface{}); ok {
		for _, h := range result {
			if hm, ok := h.(map[string]interface{}); ok && hm["hostname"] == hostname {
				return map[string]interface{}{"success": true, "skipped": true, "hostname": hostname, "existing": hm}
			}
		}
	}

	cfResult, _ := cf.CreateCustomHostname(hostname, method)

	// Create DNS A record for multi-level subdomain
	dnsExisting, _ := cf.ListDnsRecords(hostname)
	dnsExists := false
	if result, ok := dnsExisting["result"].([]interface{}); ok {
		for _, rec := range result {
			if rm, ok := rec.(map[string]interface{}); ok && rm["name"] == hostname {
				dnsExists = true
				break
			}
		}
	}

	if !dnsExists {
		// Find VPS IP from wildcard record
		allRecs, _ := cf.ListDnsRecords("")
		if result, ok := allRecs["result"].([]interface{}); ok {
			for _, rec := range result {
				if rm, ok := rec.(map[string]interface{}); ok {
					if rm["name"] == "*."+yourDomain {
						if ip, ok := rm["content"].(string); ok && ip != "" {
							dnsRes, _ := cf.CreateDnsRecord("A", hostname, ip, false)
							cfResult["_dns_record"] = dnsRes
						}
						break
					}
				}
			}
		}
	}

	return cfResult
}
