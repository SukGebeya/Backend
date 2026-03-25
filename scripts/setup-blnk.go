package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"

	blnkgo "github.com/blnkfinance/blnk-go"
)

const (
	etbLedgerName = "ETB Ledger"
	currency      = "ETB"
)

func main() {
	fmt.Println("Setting up Blnk for SukGebeya...")
	fmt.Println()

	baseURL := getEnvOrDefault("BLNK_BASE_URL", "http://localhost:5001")
	var apiKey *string

	if key := os.Getenv("BLNK_SECRET_KEY"); key != "" {
		apiKey = &key
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalf("❌ Invalid BLNK_BASE_URL: %v", err)
	}

	client := blnkgo.NewClient(parsedURL, apiKey,
		blnkgo.WithTimeout(30e9),
		blnkgo.WithRetry(3),
	)

	fmt.Printf("📡 Connecting to Blnk at %s...\n", baseURL)
	fmt.Println()

	ledgers, _, err := client.Ledger.List()
	if err != nil {
		log.Fatalf("Failed to connect to Blnk: %v", err)
	}

	fmt.Println("✅ Connected to Blnk successfully!")
	fmt.Println()

	var etbLedger *blnkgo.Ledger
	for _, l := range ledgers {
		if l.Name == etbLedgerName {
			etbLedger = &l
			fmt.Printf("📌 Found existing ETB Ledger: %s\n", l.LedgerID)
			break
		}
	}

	if etbLedger == nil {
		fmt.Println("Creating ETB Ledger...")
		ledger, resp, err := client.Ledger.Create(blnkgo.CreateLedgerRequest{
			Name: etbLedgerName,
			MetaData: map[string]interface{}{
				"currency": currency,
				"app":      "sukgebeya",
			},
		})
		if err != nil {
			log.Fatalf("❌ Failed to create ledger: %v", err)
		}
		etbLedger = ledger
		fmt.Printf("✅ Created ETB Ledger: %s (status: %d)\n", ledger.LedgerID, resp.StatusCode)
	} else {
		fmt.Println("✅ Using existing ETB Ledger")
	}
	fmt.Println()

	var escrowBalanceID, revenueBalanceID string

	escrowBalanceID, err = findOrCreateBalance(client, etbLedger.LedgerID, "escrow", "Escrow Balance")
	if err != nil {
		log.Fatalf("❌ Failed to setup escrow balance: %v", err)
	}

	revenueBalanceID, err = findOrCreateBalance(client, etbLedger.LedgerID, "revenue", "Revenue Balance")
	if err != nil {
		log.Fatalf("❌ Failed to setup revenue balance: %v", err)
	}

	fmt.Println()
	fmt.Println("📊 Blnk Setup Summary:")
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Printf("│ %-60s │\n", "Blnk Resources Created")
	fmt.Println("├─────────────────────────────────────────────────────────────┤")
	fmt.Printf("│ %-25s │ %-32s │\n", "ETB Ledger ID", etbLedger.LedgerID)
	fmt.Printf("│ %-25s │ %-32s │\n", "Escrow Balance ID", escrowBalanceID)
	fmt.Printf("│ %-25s │ %-32s │\n", "Revenue Balance ID", revenueBalanceID)
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Println()

	envFile := ".env.example"
	if err := updateEnvFile(envFile, map[string]string{
		"BLNK_ETB_LEDGER_ID":      etbLedger.LedgerID,
		"BLNK_ESCROW_BALANCE_ID":  escrowBalanceID,
		"BLNK_REVENUE_BALANCE_ID": revenueBalanceID,
	}); err != nil {
		log.Fatalf("❌ Failed to update %s: %v", envFile, err)
	}

	fmt.Printf("✅ Updated %s with Blnk IDs\n", envFile)
	fmt.Println()
	fmt.Println("🎉 Blnk setup complete!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Copy the values from .env.example to your .env file")
	fmt.Println("  2. Restart your application")
	fmt.Println("  3. Run 'make seed' to seed the database")
}

func findOrCreateBalance(client *blnkgo.Client, ledgerID, balanceType, displayName string) (string, error) {
	fmt.Printf("🔍 Looking for existing %s...\n", displayName)

	balances, _, err := client.LedgerBalance.Filter(blnkgo.FilterParams{
		Filters: []blnkgo.Filter{
			{
				Field:    "ledger_id",
				Operator: blnkgo.OpEqual,
				Value:    ledgerID,
			},
		},
		Limit: 100,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list balances: %w", err)
	}

	if balances.Data != nil {
		dataSlice, ok := balances.Data.([]interface{})
		if ok {
			for _, item := range dataSlice {
				balanceMap, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				metaData, _ := balanceMap["meta_data"].(map[string]interface{})
				if metaData != nil {
					if t, _ := metaData["type"].(string); t == balanceType {
						balanceID, _ := balanceMap["balance_id"].(string)
						if balanceID != "" {
							fmt.Printf("📌 Found existing %s: %s\n", displayName, balanceID)
							return balanceID, nil
						}
					}
				}
			}
		}
	}

	fmt.Printf("📝 Creating %s...\n", displayName)
	balance, resp, err := client.LedgerBalance.Create(blnkgo.CreateLedgerBalanceRequest{
		LedgerID: ledgerID,
		Currency: currency,
		MetaData: map[string]interface{}{
			"type": balanceType,
			"app":  "sukgebeya",
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create balance: %w", err)
	}

	fmt.Printf("✅ Created %s: %s (status: %d)\n", displayName, balance.BalanceID, resp.StatusCode)
	return balance.BalanceID, nil
}

func updateEnvFile(filename string, values map[string]string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file %s does not exist", filename)
		}
		return err
	}

	lines := strings.Split(string(content), "\n")
	updated := make(map[string]bool)

	re := regexp.MustCompile(`^([A-Z_]+)=.*$`)

	for i, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) == 2 {
			key := matches[1]
			if value, exists := values[key]; exists {
				lines[i] = fmt.Sprintf("%s=%s", key, value)
				updated[key] = true
			}
		}
	}

	for key, value := range values {
		if !updated[key] {
			lines = append(lines, fmt.Sprintf("%s=%s", key, value))
		}
	}

	output := strings.Join(lines, "\n")
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return os.WriteFile(filename, []byte(output), 0644)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
