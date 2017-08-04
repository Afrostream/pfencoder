package main

import (
	"fmt"
	"log"
	"path"
	"strings"
	"os"
	"os/exec"
	"regexp"
	"database/sql"
	"net/http"
)

type TranscoderTask struct {
	/* constructor */
	/**/	
}

func (t *TranscoderTask) doEncoding(assetId int) {
	log.Printf("-- [ %d ] Encoding task received", assetId)
	log.Printf("-- [ %d ] Get asset encoding configuration from database", assetId)
	db,_ := openDb()
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

	query = "SELECT lang,SUBSTRING_INDEX(url, '/', -1) AS vtt FROM subtitles WHERE contentId=?"
	stmt, err = db.Prepare(query)
	if err != nil {
		dbSetAssetStatus(db, assetId, "failed")
		log.Printf("XX [ %d ] Cannot prepare query %s: %s", assetId, query, err)
		return
	}
	defer stmt.Close()
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
	subtitles := make(map[string]string)
	for rows.Next() {
		var lang string
		var vtt string
		err = rows.Scan(&lang, &vtt)
		if err != nil {
			dbSetAssetStatus(db, assetId, "failed")
			log.Printf("XX [ %d ] Cannot scan rows for query %s with (%d): %s", assetId, query, contentId, err)
			return
		}
		subtitles[lang] = encodedBasePath + `/origin/vod/` + path.Base(dir) + `/` + strings.Replace(vtt, ` `, `_`, -1)
		subtitlesStr += strings.Replace(vtt, ` `, `_`, -1) + `%` + lang + ` `
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
	for k, l := range subtitles {
		log.Printf("k is %s", k)
		cmdLine = strings.Replace(cmdLine, "%SUBTITLE_"+strings.ToUpper(k)+"%", l, -1)
	}
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
		cmdLine = strings.Replace(cmdLine, `%`+parameter+`%`, value, -1)
	}
	cmdArgs = strings.Split(cmdLine, " ")
	var binaryPath string
	switch ac.P.Type {
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
	switch ac.P.Type {
	case "ffmpeg":
		var s string
		b := make([]byte, 32)
		ffmpegStartOK := false
		for {
			bytesRead, err := stderr.Read(b)
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
				s += string([]byte{0x0d})
			}
			s += string(b[:bytesRead])
			if strings.Contains(s, string([]byte{0x0d})) == true {
				str := strings.Split(s, string([]byte{0x0d}))
				fullLog += str[0] + string([]byte{0x0a}) + str[1]
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
				break
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
			t.updateAssetsStreams(assetId)
		}
	}
	ffmpegProcesses--
	dbSetFFmpegLog(db, assetId, fullLog)
	log.Printf("-- [ %d ] FFmpeg execution success", assetId)

	return
}

func (t *TranscoderTask) updateAssetsStreams(assetId int) {
	url := fmt.Sprintf("http://p-afsmsch-001.afrostream.tv:4000/api/assetsStreams/%d", assetId)
	_, err := http.Post(url, "application/json", strings.NewReader("{}"))
	if err != nil {
		log.Printf("XX Cannot update assetsStreams with url %s: %s", url, err)
		return
	}

	return
}