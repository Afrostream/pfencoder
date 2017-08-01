package main

import (
	"time"
	"log"
	"os/exec"
	"regexp"
)

type MonitoringTask struct {
}

func (m MonitoringTask) startMonitoringLoad(encoderId int64) {
	ticker := time.NewTicker(time.Second * 1)
	log.Printf("-- Starting load monitoring thread")
	go func() {
		for _ = range ticker.C {
			s, err := exec.Command(uptimePath).Output()
			if err != nil {
				log.Printf("XX Can't exec cmd %s: %s", uptimePath, err)
				continue
			}
			re, err := regexp.Compile("load average: *([0-9\\.]*), *")
			if err != nil {
				log.Printf("XX Can't compile regexp: %s", err)
				continue
			}
			matches := re.FindAllStringSubmatch(string(s), -1)
			var load1 string
			for _, v := range matches {
				load1 = v[1]
			}
			db := openDb()
			query := "UPDATE encoders SET load1=?,activeTasks=? WHERE encoderId=?"
			stmt, err := db.Prepare(query)
			if err != nil {
				log.Printf("XX Can't prepare query %s, cannot report encoder load in database: %s", query, err)
				db.Close()
				continue
			}
			log.Printf("-- Inserting load value %s into database", load1)
			_, err = stmt.Exec(load1, ffmpegProcesses, encoderId)
			if err != nil {
				log.Printf("XX Can't exec query %s with (%s): %s", query, load1, err)
				stmt.Close()
				db.Close()
				continue
			}
			stmt.Close()
			db.Close()
		}
	}()
}
