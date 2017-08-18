package main

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"runtime"
	"strconv"
	"pfencoder/database"
	"pfencoder/tasks"
)

func registerEncoder() (id int, err error) {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	log.Printf("-- Registering encoder '%s'...", hostname)
	/* opening database */
	db, err := database.OpenGormDbOnce()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	encoder := database.Encoder{Hostname: hostname}
	db.Where(&encoder).FirstOrCreate(&encoder)
	//RESET activeTasks, load1 (encoder is starting !)
	encoder.ActiveTasks = 0
	encoder.Load1 = 0
	db.Save(&encoder)
	id = encoder.ID
	log.Printf("-- Registering encoder '%s' done successfully, id=%d", hostname, id)
	return
}

func main() {
	/** TESTING ZONE PURPOSE **/
	//log.Println("-- TESTING ZONE PURPOSE...")
	//
	//log.Println("-- TESTING ZONE PURPOSE DONE SUCCESSULLY")
	//return
	/** TESTING ZONE PURPOSE **/
	log.Println("-- pfencoder starting...")

	initGlobals()

	initChecks()

	/* create tasks */
	monitoringTask := createMonitoringTask()
	exchangerTask := createExchangerTask()
	
	/* all is ok, start tasks */

	monitoringTask.Start()
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
	mySqlDatabase := "video_encoding"
	if os.Getenv("MYSQL_DATABASE") != "" {
		mySqlDatabase = os.Getenv("MYSQL_DATABASE")
	}
	database.DbDsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", mysqlUser, mysqlPassword, mysqlHost, mySqlPort, mySqlDatabase)
	log.Println("-- initGlobals done successfully")
}

func initChecks() {
	log.Println("-- initChecks starting...")
	//TODO : binaries are functional
	//database check (blocked until database is started)
	db := database.OpenGormDb()
	defer db.Close()
	//rabbitMQ check (later)
	log.Println("-- initChecks done successfully")
}

func createMonitoringTask() tasks.MonitoringTask {
	log.Println("-- createMonitoringTask calling...")
	encoderId, err := registerEncoder()
	if err != nil {
		msg := "Cannot register encoder in database"
		log.Printf(msg)
		panic(msg)
	}
	log.Printf("-- Encoder database id is %d", encoderId)
	monitoringTask := tasks.NewMonitoringTask(encoderId)
	monitoringTask.Init()
	log.Println("-- createMonitoringTask calling done successfully")
	return monitoringTask
}

func createExchangerTask() tasks.ExchangerTask {
	log.Println("-- createExchangerTask calling...")
	rabbitmqHost := os.Getenv("RABBITMQ_HOST")
	rabbitmqUser := os.Getenv("RABBITMQ_USER")
	rabbitmqPassword := os.Getenv("RABBITMQ_PASSWORD")
	rabbitmqPort := 5672
	if os.Getenv("RABBITMQ_PORT") != "" {
		rabbitmqPort, _ = strconv.Atoi(os.Getenv("RABBITMQ_PORT"))
	}
	exchangerTask := tasks.NewExchangerTask(rabbitmqHost, rabbitmqPort, rabbitmqUser, rabbitmqPassword)
	exchangerTask.Init()
	log.Println("-- createExchangerTask calling done successfully")
	return exchangerTask
}

/*func forever() {
	for {
		fmt.Printf("%v+\n", time.Now())
		time.Sleep(time.Second)
	}
}*/
