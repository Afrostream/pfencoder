package main

import (
	"fmt"
	"log"
	"database/sql"
	"errors"
)

func openDb() (db *sql.DB) {
	db, err := sql.Open("mysql", dbDsn)
	logOnError(err, "Cannot open database %s", dbDsn)
	err = db.Ping()
	logOnError(err, "Cannot ping database %s", dbDsn)

	return
}

func dbSetStatus(db *sql.DB, tableName string, id int, state string) (err error) {
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

func dbSetContentStatus(db *sql.DB, id int, state string) (err error) {
	log.Printf("-- [ %d ] Set content state to '%s'", id, state)
	err = dbSetStatus(db, "contents", id, state)

	return
}

func dbSetAssetStatus(db *sql.DB, id int, state string) (err error) {
	log.Printf("-- [ %d ] Set asset state to '%s'", id, state)
	err = dbSetStatus(db, "assets", id, state)

	return
}

func dbSetFFmpegProgression(db *sql.DB, assetId int, fp FFMpegProgression) (err error) {
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

func dbSetFFmpegLog(db *sql.DB, assetId int, fullLog string) {
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
