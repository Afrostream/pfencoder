package main

import (
	"os"
	"os/exec"
        "path"
	"errors"
	"fmt"
	"strings"
        "net/http"
	"log"
	"time"
	"regexp"
	"encoding/json"
	"github.com/streadway/amqp"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

var ffmpegProcesses int

const ffmpegPath = "/usr/bin/ffmpeg"
const spumuxPath = "/usr/bin/spumux"
const uptimePath = "/usr/bin/uptime"
const dbDsn = "pfencoder:Fhd4elKd0UxCd43gVHu5@tcp(10.91.83.18:3306)/video_encoding"

type OrderMessage struct {
  Hostname string
  AssetId int
}

type Preset struct {
  Id			int
  ProfileId		int
  Type			string
  CmdLine		string
  CreatedAt		string
  UpdatedAt		string
}

type AssetConfiguration struct {
  SrcFilename	*string
  DstFilename	*string
  P		Preset
}

type FFMpegProgression struct {
  Frame		string
  Fps		string
  Q		string
  Size		string
  Elapsed	string
  Bitrate	string
}

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
    _, err = stmt.Exec(assetId,fp.Frame, fp.Fps, fp.Q, fp.Size, fp.Elapsed, fp.Bitrate)
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

func updateAssetsStreams(assetId int) {
  url := fmt.Sprintf("http://p-afsmsch-001.afrostream.tv:4000/api/assetsStreams/%d", assetId)
  _, err := http.Post(url, "application/json", strings.NewReader("{}"))
  if err != nil {
    log.Printf("XX Cannot update assetsStreams with url %s: %s", url, err)
    return
  }

  return
}

func doEncoding(assetId int) {
  log.Printf("-- [ %d ] Encoding task received", assetId)
  log.Printf("-- [ %d ] Get asset encoding configuration from database", assetId)
  db := openDb()
  defer db.Close()
  query := "SELECT c.contentId,c.uuid,c.filename,a2.filename,a.filename,p.presetId,p.profileId,p.type,p.cmdLine,p.createdAt,p.updatedAt FROM assets AS a LEFT JOIN presets AS p ON a.presetId=p.presetId LEFT JOIN assets AS a2 ON a.assetIdDependance=a2.assetId LEFT JOIN contents AS c ON c.contentId=a.contentId WHERE a.assetId=?"
  stmt, err := db.Prepare(query)
  if err != nil {
    dbSetAssetStatus(db, assetId, "failed")
    log.Printf("XX [ %d ] Cannot prepare query %s: %s", assetId, query, err)
    return
  }
  defer stmt.Close()
  var ac AssetConfiguration
  var contentId *int
  var uuid *string
  var contentFilename *string
  err = stmt.QueryRow(assetId).Scan(&contentId, &uuid, &contentFilename, &ac.SrcFilename, &ac.DstFilename, &ac.P.Id, &ac.P.ProfileId, &ac.P.Type, &ac.P.CmdLine, &ac.P.CreatedAt, &ac.P.UpdatedAt)
  if err != nil {
    dbSetAssetStatus(db, assetId, "failed")
    log.Printf("XX [ %d ] Cannot query %s with assetId=%d and scan results: %s", assetId, query, assetId, err)
    return
  }
  if ac.SrcFilename == nil {
    ac.SrcFilename = contentFilename
  }
  dir := path.Dir(*ac.DstFilename)
  err = os.MkdirAll(dir, 0755)
  if err != nil {
    log.Printf("XX Cannot create directory %s: %s", dir, err)
    dbSetAssetStatus(db, assetId, "failed")
    return
  }

  query = "SELECT lang,SUBSTRING_INDEX(url, '/', -1) FROM subtitles WHERE contentId=?"
  var stmt2 *sql.Stmt
  stmt2, err = db.Prepare(query)
  if err != nil {
    dbSetAssetStatus(db, assetId, "failed")
    log.Printf("XX [ %d ] Cannot prepare query %s: %s", assetId, query, err)
    return
  }
  defer stmt2.Close()
  var rows *sql.Rows
  rows, err = stmt.Query(*contentId)
  if err != nil {
    dbSetAssetStatus(db, assetId, "failed")
    log.Printf("XX [ %d ] Cannot query %s with (%d): %s", assetId, query, contentId, err)
    return
  }
  defer rows.Close()
  subtitlesStr := ``
  rowsEmpty := true
  for rows.Next() {
    var lang string
    var vtt string
    err = rows.Scan(&lang, &vtt)
    if err != nil {
      dbSetAssetStatus(db, assetId, "failed")
      log.Printf("XX [ %d ] Cannot scan rows for query %s with (%d): %s", assetId, query, contentId, err)
      return
    }
    subtitlesStr += vtt + `%` + lang + ` `
    rowsEmpty = false
  }
  if rowsEmpty == false {
    subtitlesStr = subtitlesStr[:len(subtitlesStr)-1]
  }

  var cmdArgs []string
  cmdLine := strings.Replace(ac.P.CmdLine, "%SOURCE%", *ac.SrcFilename, -1)
  cmdLine = strings.Replace(cmdLine, "%DESTINATION%", *ac.DstFilename, -1)
  cmdLine = strings.Replace(cmdLine, "%UUID%", *uuid, -1)
  cmdLine = strings.Replace(cmdLine, "%BASEDIR%", path.Dir(*ac.DstFilename), -1)
  cmdLine = strings.Replace(cmdLine, "%SUBTITLES%", subtitlesStr, -1)
  log.Printf("uuid is %s", *uuid)
  var re *regexp.Regexp
  re, err = regexp.Compile("%SOURCE_[0-9]+%")
  if err != nil {
    log.Printf("XX Cannot compile regexp %SOURCE_[0-9]+%: %s", err)
    dbSetAssetStatus(db, assetId, "failed")
    return
  }
  matches := re.FindAllString(cmdLine, -1)
  if matches != nil {
    for _, m := range matches {
      str := strings.Split(m, `_`)
      query := "SELECT filename FROM assets WHERE contentId=? AND presetId=?"
      stmt, err := db.Prepare(query)
      if err != nil {
        log.Printf("XX Cannot prepare query %s: %s", query, err)
        dbSetAssetStatus(db, assetId, "failed")
        return
      }
      var filename string
      err = stmt.QueryRow(contentId, str[1]).Scan(&filename)
      if err != nil {
        log.Printf("XX Cannot QueryRow %s on query %s: %s", str[1], query, err)
        dbSetAssetStatus(db, assetId, "failed")
        return
      }
      cmdLine = strings.Replace(cmdLine, m, filename, -1)
    }
  }

  query = "SELECT parameter,value FROM profilesParameters WHERE assetId=?"
  stmt, err = db.Prepare(query)
  if err != nil {
    dbSetAssetStatus(db, assetId, "failed")
    log.Printf("XX [ %d ] Cannot prepare query %s: %s", query, err)
    return
  }
  defer stmt.Close()
  rows, err = stmt.Query(assetId)
  if err != nil {
    dbSetAssetStatus(db, assetId, "failed")
    log.Printf("XX [ %d ] Cannot query %s with (%d): %s", query, assetId, err)
    return
  }
  defer rows.Close()
  for rows.Next() {
    var parameter string
    var value string
    err = rows.Scan(&parameter, &value)
    if err != nil {
      dbSetAssetStatus(db, assetId, "failed")
      log.Printf("XX [ %d ] Cannot scan result for query %s: %s", query, err)
      return
    }
    cmdLine = strings.Replace(cmdLine, `%` + parameter + `%`, value, -1)
  }
  cmdArgs = strings.Split(cmdLine, " ")
  var binaryPath string
  switch(ac.P.Type) {
    case "ffmpeg":
      binaryPath = ffmpegPath
    case "spumux":
      binaryPath = spumuxPath
    case "script":
      binaryPath = cmdArgs[0]
      cmdArgs = cmdArgs[1:]
  }

  cmd := exec.Command(binaryPath, cmdArgs...)
  stderr, err := cmd.StderrPipe()
  log.Printf("-- [ %d ] Running command: %s %s", assetId, binaryPath, strings.Join(cmdArgs, " "))
  err = cmd.Start()
  if err != nil {
    dbSetAssetStatus(db, assetId, "failed")
    if contentId != nil {
      dbSetContentStatus(db, *contentId, "failed")
    }
    log.Printf("-- [ %d ] Cannot start command %s %s: %s", assetId, binaryPath, strings.Join(cmdArgs, " "), err)
    return
  }
  ffmpegProcesses++
  dbSetAssetStatus(db, assetId, "processing")

  fullLog := ""
  switch(ac.P.Type) {
    case "ffmpeg":
      var s string
      b := make([]byte, 32)
      ffmpegStartOK := false
      for {
        bytesRead, err := stderr.Read(b);
        if err != nil {
          break
        }
        if strings.Contains(s, `Press [q] to stop, [?] for help`) == true {
          ffmpegStartOK = true
          break
        }
        s += string(b[:bytesRead])
      }

      // If FFMpeg exit with error
      if ffmpegStartOK == false {
        dbSetAssetStatus(db, assetId, "failed")
        dbSetFFmpegLog(db, assetId, s)
        log.Printf("XX [ %d ] FFMpeg execution error, please consult logs in database table 'logs'", assetId)
        cmd.Wait()
        ffmpegProcesses--
        return
      }

      re, err := regexp.Compile(`frame= *([0-9]*) *fps= *([0-9]*) *q= *([-0-9\.]*)* *L?size= *([0-9]*)kB *time= *([0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{2}) *bitrate= *([0-9\.]*)kbits/s`)
      if err != nil {
        log.Printf("XX [ %d ] Cannot compile regexp frame=([0-9]*), progression will not be available: %s", assetId, err)
      }

      fullLog = ""
      for {
        bytesRead, err := stderr.Read(b)
        if err != nil {
          s += string([]byte{ 0x0d })
        }
        s += string(b[:bytesRead])
        if strings.Contains(s, string([]byte{ 0x0d })) == true {
          str := strings.Split(s, string([]byte{ 0x0d }))
          fullLog += str[0] + string([]byte{ 0x0a }) + str[1]
          s = str[1]
          matches := re.FindAllStringSubmatch(str[0], -1)
          for _, v := range matches {
            var fp FFMpegProgression
            fp.Frame = v[1]
            fp.Fps = v[2]
            fp.Q = v[3]
            fp.Size = v[4]
            fp.Elapsed = strings.Split(v[5], ".")[0]
            fp.Bitrate = v[6]
            dbSetFFmpegProgression(db, assetId, fp)
          }
        }
        if err != nil {
          break
        }
      }
    case "spumux":
      fallthrough
    case "script":
      b := make([]byte, 4096)
      fullLog = ""
      for {
        bytesRead, err := stderr.Read(b)
        if err != nil {
          break;
        }
        fullLog += string(b[:bytesRead])
      }
  }
  err = cmd.Wait()
  if err != nil {
    dbSetAssetStatus(db, assetId, "failed")
    log.Printf("XX [ %d ] FFMpeg execution error, please consult logs in database table 'logs'", assetId)
  } else {
    dbSetAssetStatus(db, assetId, "ready")
    if ac.P.Type == "ffmpeg" {
      updateAssetsStreams(assetId)
    }
  }
  ffmpegProcesses--
  dbSetFFmpegLog(db, assetId, fullLog)
  log.Printf("-- [ %d ] FFmpeg execution success", assetId)

  return
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

func startMonitoringLoad(encoderId int64) {
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

func main() {
  var encoderId int64
  ffmpegProcesses = 0

  conn, err := amqp.Dial("amqp://p-afsmsch-001.afrostream.tv/")
  failOnError(err, "Failed to connect to RabbitMQ")
  defer conn.Close()

  ch, err := conn.Channel()
  failOnError(err, "Failed to open a channel")
  defer ch.Close()

  err = ch.ExchangeDeclare(
    "afsm-encoders",   // name
    "fanout", // type
    true,     // durable
    false,    // auto-deleted
    false,    // internal
    false,    // no-wait
    nil,      // arguments
  )
  failOnError(err, "Failed to declare an exchange")

  q, err := ch.QueueDeclare(
    "",
    false,
    false,
    true,
    false,
    nil,
  )
  failOnError(err, "Failed to declare a queue")

  err = ch.QueueBind(
    q.Name, // queue name
    "",     // routing key
    "afsm-encoders", // exchange
    false,
    nil,
  )
  failOnError(err, "Failed to bind a queue")

  msgs, err := ch.Consume(
    q.Name,
    "",
    true,
    false,
    false,
    false,
    nil,
  )
  failOnError(err, "Failed to register a consumer")

  encoderId, err = registerEncoder()
  if err != nil {
    panic(err)
  }
  log.Printf("-- Encoder database id is %d", encoderId)

  startMonitoringLoad(encoderId)

  forever := make(chan bool)

  go func() {
    for d := range msgs {
      log.Printf("Received a message: %s", d.Body)
      var oMessage OrderMessage
      err = json.Unmarshal([]byte(d.Body), &oMessage)
      hostname, err := os.Hostname()
      if err != nil {
        log.Fatal(err)
      } else {
        if oMessage.Hostname == hostname {
          if ffmpegProcesses < 4 {
            log.Printf("Start running ffmpeg process")
            go doEncoding(oMessage.AssetId)
            log.Printf("Func doEncoding() thread created")
          } else {
            log.Printf("Cannot start one more ffmpeg process (encoding queue full)")
          }
        }
      }
    }
  }()

  log.Printf(" [*] Waiting for messages, To exit press CTRL+C")
  <-forever
}
