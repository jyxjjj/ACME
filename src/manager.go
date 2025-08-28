package main

import (
	"time"
)

func serveManager() {
	err := manageCertificates()
	if err != nil {
		Log.Println("[Manager] Error:", err)
	} else {
		Log.Println("[Manager] Task completed.")
	}
	lastRunDate := time.Now().Format("2006-01-02")
	for {
		now := time.Now()
		today := now.Format("2006-01-02")
		if lastRunDate != today {
			if now.Hour() == 1 {
				if now.Minute() == 0 {
					Log.Println("[Manager] Running certificate management task...")
					err := manageCertificates()
					if err != nil {
						Log.Println("[Manager] Error:", err)
					} else {
						lastRunDate = today
						Log.Println("[Manager] Task completed.")
					}
				}
			}
		}
		time.Sleep(30 * time.Second)
	}
}
