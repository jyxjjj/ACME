package main

import (
	"sync/atomic"
	"time"
)

var managerRunning atomic.Int32

// serveManager runs manageCertificates once at startup and then attempts to run
// it once per day in the 01:00-01:05 time window. Uses an atomic flag to prevent reentrancy.
func serveManager() {
	// Load any upstream selected renewal time from existing domain json
	if err := loadSelectedRenewalTime(); err != nil {
		Log.Println("[Manager] No selected renewal time loaded:", err)
	}

	runManagerTaskIfNeeded()
	lastRunDate := time.Now().Add(-24 * time.Hour).Format("2006-01-02") // allow today's run
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		today := now.Format("2006-01-02")
		// run once per day in 01:00-01:05 window
		if lastRunDate != today {
			if now.Hour() == 1 && now.Minute() >= 0 && now.Minute() <= 5 {
				// If there is a selected renewal time queued, only run if the selected time
				// has been reached or if current time is within suggestedWindow. Also
				// avoid running if a selected-run is already in progress.
				if v := selectedRenewalTime.Load(); v != nil {
					if selTime, ok := v.(time.Time); ok && !selTime.IsZero() {
						// if not yet reached selected time, skip
						if time.Now().Before(selTime) {
							Log.Println("[Manager] Skipping ticked run because upstream-selected time not reached:", selTime)
							continue
						}
						// if reached, ensure we mark selectedInProgress to avoid double runs
						if !selectedInProgress.CompareAndSwap(0, 1) {
							Log.Println("[Manager] Selected renewal already in progress, skipping")
							continue
						}
						// perform run and clear selected flag
						if runManagerTaskIfNeeded() {
							lastRunDate = today
						}
						selectedInProgress.Store(0)
						// clear stored selected time after attempted run
						selectedRenewalTime.Store(time.Time{})
						continue
					}
				}
				if runManagerTaskIfNeeded() {
					lastRunDate = today
				}
			}
		}
	}
}

func runManagerTaskIfNeeded() bool {
	// attempt to set running flag from 0 to 1
	if !managerRunning.CompareAndSwap(0, 1) {
		Log.Println("[Manager] Task already running, skipping this run")
		return false
	}
	defer managerRunning.Store(0)
	Log.Println("[Manager] Running certificate management task...")
	if err := manageCertificates(); err != nil {
		Log.Println("[Manager] Error:", err)
		return false
	}
	Log.Println("[Manager] Task completed.")
	return true
}
