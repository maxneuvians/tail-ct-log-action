package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type entry struct {
	Id        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Domain    string    `json:"domain"`
	Issuer    string    `json:"issuer"`
	From      string    `json:"from"`
	To        string    `json:"to"`
}

type issuer struct {
	FriendlyName string `json:"friendly_name"`
	PublicKeySha string `json:"public_key_sha256"`
}

type result struct {
	Id           string   `json:"id"`
	TbsSha256    string   `json:"tbs_sha256"`
	CertSha256   string   `json:"cert_sha256"`
	DnsNames     []string `json:"dns_names"`
	PubkeySha256 string   `json:"pubkey_sha256"`
	NotBefore    string   `json:"not_before"`
	NotAfter     string   `json:"not_after"`
	Revoked      bool     `json:"revoked"`
	Issuer       issuer   `json:"issuer"`
}

func main() {
	var after string
	domain := os.Getenv("DOMAIN")

	if domain == "" {
		log.Fatal("DOMAIN environment variable is required")
	}

	filename := domain_file(domain)

	existing_records, err := read_domain_file(filename)

	if err != nil {
		after = ""
	}

	// Get Id of the last record
	if existing_records.Id != "" {
		after = existing_records.Id
	}

	log.Printf("Looking for certificates for domain %s after %s", domain, after)

	for {
		records, err := request_records(domain, after)

		if err != nil {
			log.Default().Printf("Error: %s", err)
			break
		}

		if len(records) == 0 {
			log.Default().Print("No more records found")
			break
		}

		log.Printf("Found %d records", len(records))

		entries := make([]entry, 0)

		for _, record := range records {
			for _, domainName := range record.DnsNames {
				entries = append(entries, entry{
					Id:        record.Id,
					Timestamp: time.Now(),
					Domain:    domainName,
					Issuer:    record.Issuer.FriendlyName,
					From:      record.NotBefore,
					To:        record.NotAfter,
				})
			}
		}

		err = append_to_domain_file(filename, entries)

		if err != nil {
			log.Fatal(err)
		}

		after = records[len(records)-1].Id
	}

	log.Printf("Done")
}

func append_to_domain_file(filename string, entries []entry) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer file.Close()

	encoder := json.NewEncoder(file)

	for _, entry := range entries {
		err = encoder.Encode(entry)

		if err != nil {
			return err
		}
	}

	return nil
}

func domain_file(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), ".", "-") + ".jsonl"
}

func read_domain_file(name string) (entry, error) {
	file, err := os.Open(name)

	if err != nil {
		return entry{}, err
	}

	// Get last line of file and read it as an entry
	var last_entry entry

	decoder := json.NewDecoder(file)

	for {
		err := decoder.Decode(&last_entry)

		if err != nil {
			break
		}
	}

	return last_entry, nil
}

func request_records(domain string, after string) ([]result, error) {
	endpoint := fmt.Sprintf("https://api.certspotter.com/v1/issuances?domain=%s&include_subdomains=true&expand=dns_names&match_wildcards=true&expand=issuer&after=%s", domain, after)

	resp, err := http.Get(endpoint)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	var results []result

	err = json.NewDecoder(resp.Body).Decode(&results)

	if err != nil {
		return nil, err
	}

	return results, nil
}
