package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"time"
	"runtime"
	"strconv"
)

var ffmpegProcesses int

var ffmpegPath string
var spumuxPath string
var uptimePath string
var dbDsn string
var encodedBasePath string

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
	db := openDb()
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
	log.Println("pfencoder starting...")
	var encoderId int64
	ffmpegProcesses = 0
	var err error

	uptimePath = os.Getenv(`UPTIME_PATH`)
	spumuxPath = os.Getenv(`SPUMUX_PATH`)
	ffmpegPath = os.Getenv(`FFMPEG_PATH`)
	encodedBasePath = os.Getenv(`VIDEOS_ENCODED_BASE_PATH`)
	mysqlHost := os.Getenv(`MYSQL_HOST`)
	mysqlUser := os.Getenv(`MYSQL_USER`)
	mysqlPassword := os.Getenv(`MYSQL_PASSWORD`)
	mySqlPort := 3306
	if os.Getenv(`MYSQL_PORT`) != ""  {
		mySqlPort,_ = strconv.Atoi(os.Getenv(`MYSQL_PORT`))
	}
	dbDsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/video_encoding", mysqlUser, mysqlPassword, mysqlHost, mySqlPort)
	rabbitmqHost := os.Getenv(`RABBITMQ_HOST`)
	rabbitmqUser := os.Getenv(`RABBITMQ_USER`)
	rabbitmqPassword := os.Getenv(`RABBITMQ_PASSWORD`)
	rabbitmqPort := 5672
	if os.Getenv(`RABBITMQ_PORT`) != ""  {
		rabbitmqPort,_ = strconv.Atoi(os.Getenv(`RABBITMQ_PORT`))
	}
	first := true
	for first == true || err != nil {
		encoderId, err = registerEncoder()
		if err != nil {
			log.Printf("Cannot register encoder in database, Waiting MySQL...")
			time.Sleep(1 * time.Second)
		}
		first = false
	}
	log.Printf("-- Encoder database id is %d", encoderId)

	monitoringTask := MonitoringTask{}
	monitoringTask.startMonitoringLoad(encoderId)

	exchangerTask := newExchangerTask(rabbitmqHost, rabbitmqPort, rabbitmqUser, rabbitmqPassword)
	exchangerTask.init()
	exchangerTask.startTask()
	log.Println("pfencoder started, To exit press CTRL+C")
	runtime.Goexit()
	/* NCO : ? Goexit or forever : What is the best.. ? */
	/*done := make(chan bool)
	go forever()
	log.Println("pfencoder started, To exit press CTRL+C")
	<-done // Block forever
	*/
	log.Println("pfencoder stopped")
}

/*func forever() {
	for {
		fmt.Printf("%v+\n", time.Now())
		time.Sleep(time.Second)
	}
}*/
