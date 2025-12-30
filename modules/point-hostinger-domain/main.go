// To install dependencies, run:
// go get github.com/joho/godotenv
package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	// Ensure dependency is installed
	cmd := exec.Command("go", "get", "github.com/joho/godotenv")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Failed to install dependency github.com/joho/godotenv:", err)
		return
	}

	// Load .env file
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	} 
	apiKey := os.Getenv("API_KEY_HOSTINGER")
	if apiKey == "" {
		fmt.Println("API_KEY_HOSTINGER not found in .env")
		return
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your domain (e.g. mydomain.tld): ")
	domain, _ := reader.ReadString('\n')
	domain = strings.TrimSpace(domain)

	fmt.Print("Enter the SSH IP to set as A record: ")
	sshIP, _ := reader.ReadString('\n')
	sshIP = strings.TrimSpace(sshIP)

	url := fmt.Sprintf("https://developers.hostinger.com/api/dns/v1/zones/%s", domain)

	payload := fmt.Sprintf(`{
  "overwrite": true,
  "zone": [
    {
      "name": "@",
      "records": [
        {"content": "%s"}
      ],
      "ttl": 14400,
      "type": "A"
    }
  ]
}`, sshIP)

	req, err := http.NewRequest("PUT", url, strings.NewReader(payload))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	fmt.Println("Status:", res.Status)
	fmt.Println("Response:", string(body))
}
