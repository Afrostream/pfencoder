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
	/* for testing purpose */
	//"github.com/jinzhu/gorm"
	//_ "github.com/jinzhu/gorm/dialects/mysql"
	"pfencoder/database"
	"pfencoder/tasks"
)

func registerEncoder() (id int, err error) {
	/** NEW **/
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	log.Printf("-- Registering encoder '%s'...", hostname)
	/* opening database */
	db, err := database.OpenGormDb()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	encoder := database.Encoder{Hostname: hostname}
	db.Where(&encoder).FirstOrCreate(&encoder)
	id = encoder.ID
	log.Printf("-- Registering encoder '%s' done successfully, id=%d", hostname, id)
	return
}

func main() {
	/** TESTING ZONE PURPOSE **/
	//log.Println("-- TESTING ZONE PURPOSE...")
	/*mysqlHost := os.Getenv(`MYSQL_HOST`)
	mysqlUser := os.Getenv(`MYSQL_USER`)
	mysqlPassword := os.Getenv(`MYSQL_PASSWORD`)
	mySqlPort := 3306
	if os.Getenv(`MYSQL_PORT`) != "" {
		mySqlPort, _ = strconv.Atoi(os.Getenv(`MYSQL_PORT`))
	}
	db, err := gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/video_encoding", mysqlUser, mysqlPassword, mysqlHost, mySqlPort))
	if err != nil {
		log.Println("PB DB")
	}
	defer db.Close()
	db.LogMode(true)
	var preset database.Preset
	db.First(&preset, 3)
	log.Println(fmt.Printf("%+v\n", preset))
	log.Println("-- TESTING ZONE PURPOSE DONE SUCCESSULLY")
	return*/
	/** TESTING ZONE PURPOSE **/
	log.Println("-- pfencoder starting...")
	var monitoringTask tasks.MonitoringTask
	var exchangerTask tasks.ExchangerTask

	initGlobals()

	initChecks()

	monitoringTask = createMonitoringTask()
	monitoringTask.Init()
	monitoringTask.Start()

	exchangerTask = createExchangerTask()
	exchangerTask.Init()
	exchangerTask.Start()

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
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPassword := os.Getenv("MYSQL_PASSWORD")
	mySqlPort := 3306
	if os.Getenv("MYSQL_PORT") != "" {
		mySqlPort, _ = strconv.Atoi(os.Getenv("MYSQL_PORT"))
	}
	database.DbDsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/video_encoding", mysqlUser, mysqlPassword, mysqlHost, mySqlPort)
	log.Println("-- initGlobals done successfully")
}

func initChecks() {
	log.Println("-- initChecks starting...")
	//TODO : binaries are functional
	//database is up
	var db *sql.DB
	var err error
	first := true
	for first == true || err != nil {
		db, err = database.OpenDb()
		defer db.Close()
		if err != nil {
			log.Printf("Cannot connect to DB, Waiting for MySQL...")
			time.Sleep(1 * time.Second)
		}
		first = false
	}
	log.Println("-- initChecks done successfully")
}

func createMonitoringTask() tasks.MonitoringTask {
	log.Println("-- createMonitoringTask starting...")
	encoderId, err := registerEncoder()
	if err != nil {
		msg := "Cannot register encoder in database"
		log.Printf(msg)
		panic(msg)
	}
	log.Printf("-- Encoder database id is %d", encoderId)
	monitoringTask := tasks.NewMonitoringTask(encoderId)
	log.Println("-- createMonitoringTask done successfully")
	return monitoringTask
}

func createExchangerTask() tasks.ExchangerTask {
	log.Println("-- createExchangerTask starting...")
	rabbitmqHost := os.Getenv("RABBITMQ_HOST")
	rabbitmqUser := os.Getenv("RABBITMQ_USER")
	rabbitmqPassword := os.Getenv("RABBITMQ_PASSWORD")
	rabbitmqPort := 5672
	if os.Getenv("RABBITMQ_PORT") != "" {
		rabbitmqPort, _ = strconv.Atoi(os.Getenv("RABBITMQ_PORT"))
	}
	exchangerTask := tasks.NewExchangerTask(rabbitmqHost, rabbitmqPort, rabbitmqUser, rabbitmqPassword)
	log.Println("-- createExchangerTask done successfully")
	return exchangerTask
}

/*func forever() {
	for {
		fmt.Printf("%v+\n", time.Now())
		time.Sleep(time.Second)
	}
}*/
