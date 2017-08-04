package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"
)

/* GLOBALS --> */

var ffmpegProcesses int

var ffmpegPath string
var spumuxPath string
var uptimePath string

var encodedBasePath string

var dbDsn string

/* <-- GLOBALS */

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s\n", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func logOnError(err error, format string, v ...interface{}) {
	format = format + ": %s"
	if err != nil {
		log.Printf(format, v, err)
	}
}

func registerEncoder() (id int64, err error) {
	id = -1
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	log.Printf("-- Register encoder '%s' for processing encoding tasks", hostname)
	db, _ := openDb()
	defer db.Close()

	query := "SELECT encoderId FROM encoders WHERE hostname=?"
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Printf("XX Cannot prepare query %s: %s", query, err)
		return
	}
	err = stmt.QueryRow(hostname).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		stmt.Close()
		query = "INSERT INTO encoders (`hostname`) VALUES (?)"
		stmt, err = db.Prepare(query)
		if err != nil {
			log.Printf("Cannot prepare query %s: %s", query, err)
			return
		}
		defer stmt.Close()
		var result sql.Result
		result, err = stmt.Exec(hostname)
		if err != nil {
			log.Printf("Error during query execution %s with hostname=%s: %s", query, hostname, err)
			return
		}
		id, err = result.LastInsertId()
		if err != nil {
			log.Printf("XX Cannot get the last insert id: %s", err)
			return
		}
	case err != nil:
		stmt.Close()
		log.Printf("Error during query execution %s with hostname=%s: %s", query, hostname, err)
	}

	return
}

func main() {
	log.Println("-- pfencoder starting...")
	var monitoringTask MonitoringTask
	var exchangerTask ExchangerTask

	initGlobals()

	initChecks()

	monitoringTask = createMonitoringTask()
	monitoringTask.init()
	monitoringTask.start()

	exchangerTask = createExchangerTask()
	exchangerTask.init()
	exchangerTask.start()

	log.Println("-- pfencoder started, To exit press CTRL+C")
	runtime.Goexit()
	/* NCO : ? Goexit or forever : What is the best.. ? */
	/*done := make(chan bool)
	go forever()
	log.Println("pfencoder started, To exit press CTRL+C")
	<-done // Block forever
	*/
	log.Println("-- pfencoder stopped")
}

func initGlobals() {
	log.Println("-- initGlobals starting...")
	ffmpegProcesses = 0

	ffmpegPath = os.Getenv(`FFMPEG_PATH`)
	spumuxPath = os.Getenv(`SPUMUX_PATH`)
	uptimePath = os.Getenv(`UPTIME_PATH`)

	encodedBasePath = os.Getenv(`VIDEOS_ENCODED_BASE_PATH`)

	mysqlHost := os.Getenv(`MYSQL_HOST`)
	mysqlUser := os.Getenv(`MYSQL_USER`)
	mysqlPassword := os.Getenv(`MYSQL_PASSWORD`)
	mySqlPort := 3306
	if os.Getenv(`MYSQL_PORT`) != "" {
		mySqlPort, _ = strconv.Atoi(os.Getenv(`MYSQL_PORT`))
	}
	dbDsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/video_encoding", mysqlUser, mysqlPassword, mysqlHost, mySqlPort)
	log.Println("-- initGlobals done successfully")
}

func initChecks() {
	log.Println("-- initChecks starting...")
	//binaries are functional
	//database is up
	var db *sql.DB
	var err error
	first := true
	for first == true || err != nil {
		db, err = openDb()
		defer db.Close()
		if err != nil {
			log.Printf("Cannot connect to DB, Waiting for MySQL...")
			time.Sleep(1 * time.Second)
		}
		first = false
	}
	log.Println("-- initChecks done successfully")
}

func createMonitoringTask() MonitoringTask {
	log.Println("-- createMonitoringTask starting...")
	encoderId, err := registerEncoder()
	if err != nil {
		msg := "Cannot register encoder in database"
		log.Printf(msg)
		panic(msg)
	}
	log.Printf("-- Encoder database id is %d", encoderId)
	monitoringTask := newMonitoringTask(encoderId)
	log.Println("-- createMonitoringTask done successfully")
	return monitoringTask
}

func createExchangerTask() ExchangerTask {
	log.Println("-- createExchangerTask starting...")
	rabbitmqHost := os.Getenv(`RABBITMQ_HOST`)
	rabbitmqUser := os.Getenv(`RABBITMQ_USER`)
	rabbitmqPassword := os.Getenv(`RABBITMQ_PASSWORD`)
	rabbitmqPort := 5672
	if os.Getenv(`RABBITMQ_PORT`) != "" {
		rabbitmqPort, _ = strconv.Atoi(os.Getenv(`RABBITMQ_PORT`))
	}
	exchangerTask := newExchangerTask(rabbitmqHost, rabbitmqPort, rabbitmqUser, rabbitmqPassword)
	log.Println("-- createExchangerTask done successfully")
	return exchangerTask
}

/*func forever() {
	for {
		fmt.Printf("%v+\n", time.Now())
		time.Sleep(time.Second)
	}
}*/
