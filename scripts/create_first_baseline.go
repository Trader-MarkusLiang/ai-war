package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"nofx/store"
)

func main() {
	// Database path
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/opt/nofx/data/data.db"
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create baseline strategy store
	baselineStore := store.NewBaselineStrategyStore(db)

	// Define baseline config from bt_20260103_092738
	config := store.BaselineConfig{
		RSIPeriod:      14,
		MACDFast:       12,
		MACDSlow:       26,
		MACDSignal:     9,
		EMAPeriod:      20,
		StochRSIPeriod: 14,
		ATRPeriod:      14,
		SignalThresholds: store.BaselineSignalThresholds{
			RSIOversold:     30,
			RSIOverbought:   70,
			StochOversold:   20,
			StochOverbought: 80,
			MinSignalCount:  2,
		},
		RiskManagement: store.BaselineRiskManagement{
			EquityMultiplier:  5.0,
			Leverage:          5,
			HardStopLossPct:   2.0,
			TrailingTP1Pct:    2.0,
			TrailingTP1Lock:   0.5,
			TrailingTP2Pct:    4.0,
			TrailingTP2Lock:   1.0,
			TrailingTP3Pct:    6.0,
			TrailingTP3Lock:   1.5,
			TrailingSL1Pct:    3.0,
			TrailingSL1Lock:   1.0,
			TrailingSL2Pct:    5.0,
			TrailingSL2Lock:   1.5,
		},
	}

	// Create baseline strategy
	strategy := &store.BaselineStrategy{
		ID:              uuid.New().String(),
		UserID:          "default",
		Name:            "StochRSI_EMA_MACD_4h",
		Description:     "从回测 bt_20260103_092738 提取的baseline策略，使用StochRSI作为核心信号，配合EMA趋势和MACD动能确认",
		Config:          config,
		IsSystemDefault: false,
	}

	// Check if strategy already exists
	existing, err := baselineStore.GetByName("default", "StochRSI_EMA_MACD_4h")
	if err == nil && existing != nil {
		fmt.Printf("⚠️  Baseline strategy '%s' already exists (ID: %s)\n", existing.Name, existing.ID)
		fmt.Println("Skipping creation.")
		return
	}

	// Create the strategy
	if err := baselineStore.Create(strategy); err != nil {
		log.Fatalf("Failed to create baseline strategy: %v", err)
	}

	// Print success message
	fmt.Println("✅ Successfully created baseline strategy!")
	fmt.Printf("   ID: %s\n", strategy.ID)
	fmt.Printf("   Name: %s\n", strategy.Name)
	fmt.Printf("   Description: %s\n", strategy.Description)

	// Print config summary
	configJSON, _ := json.MarshalIndent(strategy.Config, "   ", "  ")
	fmt.Printf("   Config:\n%s\n", string(configJSON))
}
