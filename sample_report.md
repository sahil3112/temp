## **Globomantics SOC Escalation Report**

**Date of Analysis:** June 19, 2026  
**Analyst:** SOC Analyst  
**Incident Type:** Suspected Malware Infection & Command and Control (C2) Activity  
**Network Segment:** Isolated Monitoring Segment  
**Confidence Level:** High (Multiple static signature matches for known malware)  

---

### **1. Executive Summary**

During an alert analysis of the isolated network segment, the Maltrail sensor detected severe, high-confidence malicious activity originating from multiple internal hosts. A total of four unique internal IP addresses were analyzed based on the sensor logs. Three of these IPs are actively infected endpoints involving multiple malware families (Emotet, Cobalt Strike, Ficker, Chanitor, Virut, and Zeus), while the fourth IP was identified as local DNS infrastructure actively routing this malicious traffic. Immediate containment of the three infected hosts is required.

---

### **2. Analyzed IP Addresses & Threat Details**

Based on the Maltrail reporting interface, the following four internal IP addresses were identified and analyzed:

#### **IP 1: 10.0.0.101 [LAN] — COMPROMISED ENDPOINT**

* **Overall Severity:** High
* **Analysis:** This host is exhibiting classic initial-infection behaviors. It was observed querying an IP-check service (`api.ipify.org`), commonly used by malware to identify the public IP of its victim. Furthermore, the host initiated a direct executable download (`57umant.ru/6huy67tgk.exe`) and matched static trails for three distinct malware families.
* **Key Indicators of Compromise (IoCs):**
* **Chanitor (Malware):** DNS requests to `chnicallimigue.com` and HTTP requests to `95.47.161.162` (`/8/forum.php`).
* **Ficker (Malware):** DNS requests to `57umant.ru` and `sweyblidian.com`, followed by an HTTP executable download from `8.211.5.232`.
* **Cobalt Strike (Malware):** Direct HTTP connection to `103.207.42.11`.



#### **IP 2: 10.0.0.115 [LAN] — COMPROMISED ENDPOINT**

* **Overall Severity:** High
* **Analysis:** This host demonstrates heavy beaconing activity strongly associated with the Emotet botnet, utilizing various ports (80, 443, 8080) to contact multiple external IP addresses. It also exhibits signs of a Cobalt Strike payload.
* **Key Indicators of Compromise (IoCs):**
* **Emotet (Malware):** High-volume outbound connections to numerous IPs including `103.98.188.50`, `212.83.184.188`, `159.65.163.220`, `128.199.93.156`, and `68.183.62.61` across standard and alternative HTTP/HTTPS ports.
* **Cobalt Strike (Malware):** DNS request to `lentgenn.com`.



#### **IP 3: 147.32.84.165 — COMPROMISED ENDPOINT**

* **Overall Severity:** High
* **Analysis:** This host generated the highest volume of alerts (49 total threats). It is generating DNS requests to bad history and parked domains, downloading executables (e.g., `shabi.coolnuff.com:2012/p/out/kp.exe`), and matching static trails for legacy/widespread trojans.
* **Key Indicators of Compromise (IoCs):**
* **Palevo (Malware):** DNS requests to `hcuewgbbnfdu1ew.com` and `88.perfectexe.com`.
* **Ursnif & Zeus (Malware):** DNS requests to `statusline.ru` and `mcbt.ru`.
* **Virut (Malware):** DNS queries to `irc.zief.pl` and `www.lddwj.com`.



#### **IP 4: 147.32.80.9 — NON-COMPROMISED INFRASTRUCTURE (DNS)**

* **Overall Severity:** Medium (Alerts triggered by response payloads, not host intent)
* **Analysis:** This IP is operating as the local DNS resolver for the infected segment. It appears in the `source` column of the Maltrail logs originating from UDP Port 53, confirming it is sending DNS *responses* back to ephemeral ports on the infected client `147.32.84.165`. Maltrail flagged this host because the domains inside the returned payloads triggered heuristic warnings. **This server is not infected and is operating normally.**
* **Key Alerts:**
* **Suspicious Parked Sites:** Returned DNS resolutions for `parking2.nic.ru` and `compulink.gr` to the infected client.



---

### **4. Analyst Recommendations**

1. **Immediate Isolation:** Confirm that `10.0.0.101`, `10.0.0.115`, and `147.32.84.165` are strictly isolated from the production network to prevent lateral movement. **Do not isolate `147.32.80.9`, as this will disable DNS resolution for the segment.**
2. **Firewall Blocking:** Update perimeter egress firewalls to explicitly block the High-Severity IP addresses and domains listed in the IoCs above.
3. **Endpoint Forensics:** Initiate full forensic captures on the three compromised hosts to locate the downloaded executables (e.g., `6huy67tgk.exe`, `kp.exe`).
4. **Threat Hunting:** Pull the DNS query logs from `147.32.80.9`, `10.0.0.2`, and `10.0.0.10` to cross-reference against other potential internal clients that may have beaconed out to the known-bad domains.
