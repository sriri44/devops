package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runCmd(cmdStr string, args ...string) error {
	cmd := exec.Command(cmdStr, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	country := flag.String("country", "AU", "Country Name (2 letter code)")
	state := flag.String("state", "Some-State", "State or Province Name (full name)")
	city := flag.String("city", "", "Locality Name (eg, city)")
	org := flag.String("org", "Internet Widgits Pty Ltd", "Organization Name (eg, company)")
	unit := flag.String("unit", "", "Organizational Unit Name (eg, section)")
	common := flag.String("common", "", "Common Name (e.g. server FQDN or YOUR name)")
	email := flag.String("email", "", "Email Address")
	flag.Parse()

	if *common == "" {
		fmt.Println("Error: --common is required (e.g. server FQDN or YOUR name)")
		flag.Usage()
		os.Exit(1)
	}

	// 1. openssl req
	opensslArgs := []string{
		"req", "-x509", "-nodes", "-days", "365", "-newkey", "rsa:2048",
		"-keyout", "/etc/ssl/private/nginx-selfsigned.key",
		"-out", "/etc/ssl/certs/nginx-selfsigned.crt",
		"-subj",
		fmt.Sprintf("/C=%s/ST=%s/L=%s/O=%s/OU=%s/CN=%s/emailAddress=%s", *country, *state, *city, *org, *unit, *common, *email),
	}
	fmt.Println("\nRunning openssl req...")
	if err := runCmd("sudo", append([]string{"openssl"}, opensslArgs...)...); err != nil {
		fmt.Println("Error running openssl req:", err)
		return
	}

	// 2. openssl dhparam
	fmt.Println("\nGenerating dhparam...")
	if err := runCmd("sudo", "openssl", "dhparam", "-dsaparam", "-out", "/etc/nginx/dhparam.pem", "4096"); err != nil {
		fmt.Println("Error running openssl dhparam:", err)
		return
	}

	// 3. Write self-signed.conf
	selfSignedConf := `ssl_certificate /etc/ssl/certs/nginx-selfsigned.crt;
ssl_certificate_key /etc/ssl/private/nginx-selfsigned.key;`
	fmt.Println("\nWriting /etc/nginx/snippets/self-signed.conf...")
	if err := runCmd("sudo", "bash", "-c", fmt.Sprintf("echo '%s' > /etc/nginx/snippets/self-signed.conf", selfSignedConf)); err != nil {
		fmt.Println("Error writing self-signed.conf:", err)
		return
	}

	// 4. Write ssl-params.conf
	sslParamsConf := `ssl_protocols TLSv1.3;
ssl_prefer_server_ciphers on;
ssl_dhparam /etc/nginx/dhparam.pem;
ssl_ciphers EECDH+AESGCM:EDH+AESGCM;
ssl_ecdh_curve secp384r1;
ssl_session_timeout  10m;
ssl_session_cache shared:SSL:10m;
ssl_session_tickets off;
ssl_stapling on;
ssl_stapling_verify on;
resolver 8.8.8.8 8.8.4.4 valid=300s;
resolver_timeout 5s;
# Disable strict transport security for now. You can uncomment the following
# line if you understand the implications.
#add_header Strict-Transport-Security \"max-age=63072000; includeSubDomains; preload\";
add_header X-Frame-Options DENY;
add_header X-Content-Type-Options nosniff;
add_header X-XSS-Protection "1; mode=block";`
	fmt.Println("\nWriting /etc/nginx/snippets/ssl-params.conf...")
	if err := runCmd("sudo", "bash", "-c", fmt.Sprintf("echo '%s' > /etc/nginx/snippets/ssl-params.conf", sslParamsConf)); err != nil {
		fmt.Println("Error writing ssl-params.conf:", err)
		return
	}

	// 5. Backup site config
	domain := strings.Split(*common, ":")[0]
	if domain == "" {
		domain = "default"
	}
	fmt.Println("\nBacking up /etc/nginx/sites-available/" + domain + "...")
	runCmd("sudo", "cp", "/etc/nginx/sites-available/"+domain, "/etc/nginx/sites-available/"+domain+".bak")

	// 6. Write new site config
	siteConf := fmt.Sprintf(`server {
	listen 80;
	listen [::]:80;
	server_name %s www.%s;
	return 302 https://$server_name$request_uri;
}

server {
	listen 443 ssl;
	listen [::]:443 ssl;
	include snippets/self-signed.conf;
	include snippets/ssl-params.conf;
	server_name %s www.%s;
	
	location / {
		proxy_pass http://localhost:3000;
		proxy_set_header Host $host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_set_header X-Forwarded-Proto $scheme;
	}
}
`, domain, domain, domain, domain)
	fmt.Println("\nWriting /etc/nginx/sites-available/" + domain + "...")
	if err := runCmd("sudo", "bash", "-c", fmt.Sprintf("echo '%s' > /etc/nginx/sites-available/%s", siteConf, domain)); err != nil {
		fmt.Println("Error writing site config:", err)
		return
	}

	runCmd("sudo", "cp", "/etc/nginx/sites-available/"+domain, "/etc/nginx/sites-enabled/")

	// 7. UFW allow Nginx Full
	fmt.Println("\nAllowing Nginx Full in UFW...")
	runCmd("sudo", "ufw", "allow", "'Nginx Full'")
	fmt.Println("Deleting Nginx HTTP from UFW...")
	runCmd("sudo", "ufw", "delete", "allow", "'Nginx HTTP'")

	// 8. nginx -t
	fmt.Println("\nTesting nginx config...")
	if err := runCmd("sudo", "nginx", "-t"); err != nil {
		fmt.Println("nginx config test failed:", err)
		return
	}

	// 9. Restart nginx
	fmt.Println("\nRestarting nginx...")
	if err := runCmd("sudo", "systemctl", "restart", "nginx"); err != nil {
		fmt.Println("Failed to restart nginx:", err)
		return
	}

	fmt.Println("\nAll done!")
}
