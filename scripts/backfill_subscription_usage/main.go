package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/joho/godotenv"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: go run ./scripts/backfill_subscription_usage [--start 0] [--end 0] [--batch-size 1000]\n")
	fmt.Fprintf(os.Stderr, "Reads SQL_DSN and LOG_SQL_DSN from the environment or .env files. Run once during the deployment window.\n")
	flag.PrintDefaults()
}

func main() {
	var startTimestamp int64
	var endTimestamp int64
	var batchSize int
	flag.Int64Var(&startTimestamp, "start", 0, "inclusive log start timestamp; 0 means no lower bound")
	flag.Int64Var(&endTimestamp, "end", 0, "inclusive log end timestamp; 0 means no upper bound")
	flag.IntVar(&batchSize, "batch-size", 1000, "log batch size")
	flag.Usage = usage
	flag.Parse()
	if common.PrintHelp != nil && *common.PrintHelp {
		usage()
		return
	}

	_ = godotenv.Load(".env", ".env.prod", ".env.dev")
	if sqlitePath := os.Getenv("SQLITE_PATH"); sqlitePath != "" {
		common.SQLitePath = sqlitePath
	}
	common.IsMasterNode = os.Getenv("NODE_TYPE") != "slave"

	if err := model.InitDB(); err != nil {
		fmt.Fprintf(os.Stderr, "init database failed: %v\n", err)
		os.Exit(1)
	}
	if err := model.InitLogDB(); err != nil {
		fmt.Fprintf(os.Stderr, "init log database failed: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = model.CloseDB()
	}()

	stats, err := model.BackfillSubscriptionUsageFromLogs(startTimestamp, endTimestamp, batchSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "backfill failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("subscription usage backfill completed\n")
	fmt.Printf("scanned_logs=%d applied_logs=%d aggregated_rows=%d total_quota=%d\n",
		stats.ScannedLogs, stats.AppliedLogs, stats.AggregatedRows, stats.TotalQuota)
	fmt.Printf("skipped_missing_fields=%d skipped_invalid_json=%d skipped_missing_subscription=%d\n",
		stats.SkippedMissingFields, stats.SkippedInvalidJSON, stats.SkippedMissingSubscription)
}
