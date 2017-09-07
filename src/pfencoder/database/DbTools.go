package database

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"log"
	"pfencoder/tools"
	"time"
)

var DbDsn string

func OpenGormDb() (db *gorm.DB) {
	for {
		//See : http://jinzhu.me/gorm/database.html#connecting-to-a-database
		db, err := gorm.Open("mysql", DbDsn + "?parseTime=true")
		if err == nil {
			return db
		}
		tools.LogOnError(err, "Failed to connect to the database %s, error=%s, retrying...", DbDsn, err)
		time.Sleep(3 * time.Second)
	}
}

func OpenGormDbOnce() (db *gorm.DB, err error) {
	//See : http://jinzhu.me/gorm/database.html#connecting-to-a-database
	db, err = gorm.Open("mysql", DbDsn + "?parseTime=true")
	if err != nil {
		tools.LogOnError(err, "Failed to connect to the database %s, error=%s", DbDsn, err)
	}
	return
}

//DEPRECATED
func OpenDb() (db *sql.DB, err error) {
	db, err = sql.Open("mysql", DbDsn + "?parseTime=true")
	if err != nil {
		tools.LogOnError(err, "Cannot open database %s", DbDsn)
	}
	err = db.Ping()
	if err != nil {
		tools.LogOnError(err, "Cannot ping database %s", DbDsn)
	}
	return
}

// DEPRECATED
func DbSetStatus(db *sql.DB, tableName string, id int, state string) (err error) {
	if db == nil {
		err = errors.New("db must not be nil, please set a database connection first")
		return
	}
	query := fmt.Sprintf("UPDATE %s SET state=? WHERE %sId=?", tableName, tableName[:len(tableName)-1])
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Printf("Cannot prepare query %s: %s", query, err)
		return
	}
	defer stmt.Close()
	_, err = stmt.Exec(state, id)
	if err != nil {
		log.Printf("Error during query execution %s with %sId=%d: %s", query, tableName[:len(tableName)-1], id, err)
		return
	}
	return
}

// DEPRECATED
func DbSetContentStatus(db *sql.DB, id int, state string) (err error) {
	log.Printf("-- [ %d ] Set content state to '%s'", id, state)
	err = DbSetStatus(db, "contents", id, state)

	return
}

// DEPRECATED
func DbSetAssetStatus(db *sql.DB, id int, state string) (err error) {
	log.Printf("-- [ %d ] Set asset state to '%s'", id, state)
	err = DbSetStatus(db, "assets", id, state)

	return
}

// DEPRECATED
func DbSetFFmpegProgression(db *sql.DB, assetId int, fp FfmpegProgressV0) (err error) {
	if db == nil {
		log.Printf("XX db must not be nil, please set a database connection first")
		err = errors.New("db must not be nil, please set a database connection first")
		return
	}
	query := "UPDATE ffmpegProgress SET frame=?,fps=?,q=?,size=?,elapsed=?,bitrate=? WHERE assetId=?"
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Printf("XX Cannot prepare query %s: %s", query, err)
		return
	}
	defer stmt.Close()
	var res sql.Result
	res, err = stmt.Exec(fp.Frame, fp.Fps, fp.Q, fp.Size, fp.Elapsed, fp.Bitrate, assetId)
	if err != nil {
		log.Printf("XX Error during query execution %s with fp=%#v: %s", query, fp, err)
		return
	}
	affect, err := res.RowsAffected()
	if err != nil {
		log.Printf("XX Can't get number of rows affected: %s", err)
		return
	}
	if affect == 0 {
		query := "INSERT INTO ffmpegProgress (`assetId`,`frame`,`fps`,`q`,`size`,`elapsed`,`bitrate`) VALUES (?,?,?,?,?,?,?)"
		stmt, err = db.Prepare(query)
		if err != nil {
			log.Printf("XX Can't prepare query %s: %s", query, err)
			return
		}
		_, err = stmt.Exec(assetId, fp.Frame, fp.Fps, fp.Q, fp.Size, fp.Elapsed, fp.Bitrate)
		if err != nil {
			log.Printf("XX Can't execute query %s with %d: %s", query, assetId, err)
			return
		}
	}

	return
}

// DEPRECATED
func DbSetFFmpegLog(db *sql.DB, assetId int, fullLog string) {
	query := "INSERT INTO ffmpegLogs (`assetId`,`log`) VALUES (?,?)"
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Printf("XX Can' prepare query %s: %s", query, err)
		return
	}
	defer stmt.Close()
	_, err = stmt.Exec(assetId, fullLog)
	if err != nil {
		log.Printf("XX Can't execute query %s with (%d,%s): %s", query, assetId, fullLog, err)
	}
}
