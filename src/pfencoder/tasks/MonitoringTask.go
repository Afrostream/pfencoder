package tasks

import (
	"log"
	"os"
	"os/exec"
	"pfencoder/database"
	"regexp"
	"strconv"
	"time"
)

type MonitoringTask struct {
	/* constructor */
	instanceId int
	/**/
	initialized bool
	uptimePath  string
}

func NewMonitoringTask(instanceId int) MonitoringTask {
	return (MonitoringTask{instanceId: instanceId})
}

func (m *MonitoringTask) Init() bool {
	m.uptimePath = os.Getenv("UPTIME_PATH")
	m.initialized = true
	return m.initialized
}

func (m *MonitoringTask) Start() {
	if m.initialized == false {
		log.Printf("MonitoringTask not initialized, Thread cannot start...")
		return
	}
	log.Printf("-- MonitoringTask Thread starting...")
	ticker := time.NewTicker(time.Second * 5)
	go func() {
		log.Printf("-- MonitoringTask Thread started")
		for _ = range ticker.C {
			//log.Printf("-- MonitoringTask Thread ticker...")
			load1, err := m.load1()
			if err != nil {
				log.Printf("Cannot parse load1 as float32, error=%s", err)
				continue
			}
			//log.Printf("-- MonitoringTask Thread ticker, load1=%f", load1)
			db, err := database.OpenGormDb()
			if err != nil {
				log.Printf("Cannot connect to database, error=%s", err)
				continue
			}
			encoder := database.Encoder{ID: m.instanceId}
			if db.Where(&encoder).First(&encoder).RecordNotFound() {
				log.Printf("Cannot find encoder in database with ID=%d", m.instanceId)
				continue
			}
			encoder.Load1 = load1
			encoder.ActiveTasks = ffmpegProcesses
			db.Save(&encoder)
			db.Close()
			//log.Printf("-- MonitoringTask Thread ticker done successfully")
		}
		log.Printf("MonitoringTask Thread stopped")
	}()
}

func (m *MonitoringTask) load1WithInputToCompile(input string) (load1 float32, err error) {
	s, err := exec.Command(m.uptimePath).Output()
	if err != nil {
		log.Printf("Cannot exec cmd %s: %s", m.uptimePath, err)
		return
	}
	//log.Printf("Uptime output=%s", string(s))
	re, err := regexp.Compile(input)
	if err != nil {
		log.Printf("Cannot compile regexp: %s", err)
		return
	}
	matches := re.FindAllStringSubmatch(string(s), -1)
	var load1AsStr string
	for _, v := range matches {
		load1AsStr = v[1]
	}
	value, err := strconv.ParseFloat(load1AsStr, 32)
	if err != nil {
		//log.Printf("Cannot parse load1 as float32, value=%s, error=%s", load1AsStr, err)
		return
	}
	load1 = float32(value)
	return
}

func (m *MonitoringTask) load1() (load1 float32, err error) {
	//TODO : one load, two parses
	//PRODUCTION (1ST TRY)
	load1, err = m.load1WithInputToCompile("load average: *([0-9\\.]*), *")
	//MAC (2ND TRY)
	if err != nil {
		load1, err = m.load1WithInputToCompile("load averages: *([0-9\\.]*) *")
	}
	return
}
