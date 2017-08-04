package main

import (
	"log"
	"os/exec"
	"regexp"
	"time"
)

type MonitoringTask struct {
	/* constructor */
	instanceId int64
	/**/
	initialized bool
}

func newMonitoringTask(instanceId int64) MonitoringTask {
	return (MonitoringTask{instanceId: instanceId})
}

func (m *MonitoringTask) init() {
	m.initialized = true
}

func (m *MonitoringTask) start() {
	if m.initialized == false {
		log.Printf("MonitoringTask not initialized, Thread cannot start...")
		return
	}
	log.Printf("-- MonitoringTask Thread starting...")
	ticker := time.NewTicker(time.Second * 5)
	go func() {
		log.Printf("-- MonitoringTask Thread started")
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
			db, _ := openDb()
			query := "UPDATE encoders SET load1=?,activeTasks=? WHERE encoderId=?"
			stmt, err := db.Prepare(query)
			if err != nil {
				log.Printf("XX Can't prepare query %s, cannot report encoder load in database: %s", query, err)
				db.Close()
				continue
			}
			log.Printf("-- Inserting load value %s into database", load1)
			_, err = stmt.Exec(load1, ffmpegProcesses, m.instanceId)
			if err != nil {
				log.Printf("XX Can't exec query %s with (%s): %s", query, load1, err)
				stmt.Close()
				db.Close()
				continue
			}
			stmt.Close()
			db.Close()
		}
		log.Printf("-- MonitoringTask Thread stopped")
	}()
}
