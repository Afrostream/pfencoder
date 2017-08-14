package tasks

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"pfencoder/database"
	"regexp"
	"strconv"
	"strings"
)

var ffmpegProcesses = 0

var ffmpegPath = os.Getenv("FFMPEG_PATH")
var spumuxPath = os.Getenv("SPUMUX_PATH")

var encodedBasePath = os.Getenv("VIDEOS_ENCODED_BASE_PATH")

var pfSchedulerUrl = os.Getenv("PFSCHEDULER_BASE_URL")

type TranscoderTask struct {
	/* constructor */
	assetId int
	/**/
	initialized bool
}

func NewTranscoderTask(assetId int) TranscoderTask {
	return (TranscoderTask{assetId:assetId})
}

func (t *TranscoderTask) Init() bool {
	log.Printf("-- TranscoderTask init starting...")
	t.initialized = true
	log.Printf("-- TranscoderTask init done successfully")
	return t.initialized
}

func (t *TranscoderTask) DoEncoding() {
	log.Printf("-- [ %d ] Encoding task received", t.assetId)
	log.Printf("-- [ %d ] Get asset encoding configuration from database", t.assetId)
	db, _ := database.OpenDb()
	defer db.Close()
	query := "SELECT c.contentId,c.uuid,c.filename,a2.filename,a.filename,p.presetId,p.profileId,p.type,p.cmdLine,p.createdAt,p.updatedAt FROM assets AS a LEFT JOIN presets AS p ON a.presetId=p.presetId LEFT JOIN assets AS a2 ON a.assetIdDependance=a2.assetId LEFT JOIN contents AS c ON c.contentId=a.contentId WHERE a.assetId=?"
	stmt, err := db.Prepare(query)
	if err != nil {
		database.DbSetAssetStatus(db, t.assetId, "failed")
		log.Printf("XX [ %d ] Cannot prepare query %s: %s", t.assetId, query, err)
		return
	}
	defer stmt.Close()
	var ac database.AssetConfiguration
	var contentId *int
	var uuid *string
	var contentFilename *string
	err = stmt.QueryRow(t.assetId).Scan(&contentId, &uuid, &contentFilename, &ac.SrcFilename, &ac.DstFilename, &ac.P.ID, &ac.P.ProfileId, &ac.P.Type, &ac.P.CmdLine, &ac.P.CreatedAt, &ac.P.UpdatedAt)
	if err != nil {
		database.DbSetAssetStatus(db, t.assetId, "failed")
		log.Printf("XX [ %d ] Cannot query %s with assetId=%d and scan results: %s", t.assetId, query, t.assetId, err)
		return
	}
	if ac.SrcFilename == nil {
		ac.SrcFilename = contentFilename
	}
	dir := path.Dir(*ac.DstFilename)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Printf("XX Cannot create directory %s: %s", dir, err)
		database.DbSetAssetStatus(db, t.assetId, "failed")
		return
	}

	query = "SELECT lang,SUBSTRING_INDEX(url, '/', -1) AS vtt FROM subtitles WHERE contentId=?"
	stmt, err = db.Prepare(query)
	if err != nil {
		database.DbSetAssetStatus(db, t.assetId, "failed")
		log.Printf("XX [ %d ] Cannot prepare query %s: %s", t.assetId, query, err)
		return
	}
	defer stmt.Close()
	var rows *sql.Rows
	rows, err = stmt.Query(*contentId)
	if err != nil {
		database.DbSetAssetStatus(db, t.assetId, "failed")
		log.Printf("XX [ %d ] Cannot query %s with (%d): %s", t.assetId, query, contentId, err)
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
			database.DbSetAssetStatus(db, t.assetId, "failed")
			log.Printf("XX [ %d ] Cannot scan rows for query %s with (%d): %s", t.assetId, query, contentId, err)
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
		database.DbSetAssetStatus(db, t.assetId, "failed")
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
				database.DbSetAssetStatus(db, t.assetId, "failed")
				return
			}
			var filename string
			err = stmt.QueryRow(contentId, str[1]).Scan(&filename)
			if err != nil {
				log.Printf("XX Cannot QueryRow %s on query %s: %s", str[1], query, err)
				database.DbSetAssetStatus(db, t.assetId, "failed")
				return
			}
			cmdLine = strings.Replace(cmdLine, m, filename, -1)
		}
	}

	query = "SELECT parameter,value FROM profilesParameters WHERE assetId=?"
	stmt, err = db.Prepare(query)
	if err != nil {
		database.DbSetAssetStatus(db, t.assetId, "failed")
		log.Printf("XX [ %d ] Cannot prepare query %s: %s", query, err)
		return
	}
	defer stmt.Close()
	rows, err = stmt.Query(t.assetId)
	if err != nil {
		database.DbSetAssetStatus(db, t.assetId, "failed")
		log.Printf("XX [ %d ] Cannot query %s with (%d): %s", query, t.assetId, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var parameter string
		var value string
		err = rows.Scan(&parameter, &value)
		if err != nil {
			database.DbSetAssetStatus(db, t.assetId, "failed")
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
	log.Printf("-- [ %d ] Running command: %s %s", t.assetId, binaryPath, strings.Join(cmdArgs, " "))
	err = cmd.Start()
	if err != nil {
		database.DbSetAssetStatus(db, t.assetId, "failed")
		if contentId != nil {
			database.DbSetContentStatus(db, *contentId, "failed")
		}
		log.Printf("-- [ %d ] Cannot start command %s %s: %s", t.assetId, binaryPath, strings.Join(cmdArgs, " "), err)
		return
	}
	ffmpegProcesses++
	database.DbSetAssetStatus(db, t.assetId, "processing")

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
			database.DbSetAssetStatus(db, t.assetId, "failed")
			database.DbSetFFmpegLog(db, t.assetId, s)
			log.Printf("XX [ %d ] FFMpeg execution error, please consult logs in database table 'logs'", t.assetId)
			cmd.Wait()
			ffmpegProcesses--
			return
		}

		re, err := regexp.Compile(`frame= *([0-9]*) *fps= *([0-9]*) *q= *([-0-9\.]*)* *L?size= *([0-9]*)kB *time= *([0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{2}) *bitrate= *([0-9\.]*)kbits/s`)
		if err != nil {
			log.Printf("XX [ %d ] Cannot compile regexp frame=([0-9]*), progression will not be available: %s", t.assetId, err)
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
					var fp database.FfmpegProgressV0
					fp.Frame = v[1]
					fp.Fps = v[2]
					fp.Q = v[3]
					fp.Size = v[4]
					fp.Elapsed = strings.Split(v[5], ".")[0]
					fp.Bitrate = v[6]
					database.DbSetFFmpegProgression(db, t.assetId, fp)
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
		database.DbSetAssetStatus(db, t.assetId, "failed")
		log.Printf("XX [ %d ] FFMpeg execution error, please consult logs in database table 'logs'", t.assetId)
	} else {
		database.DbSetAssetStatus(db, t.assetId, "ready")
		if ac.P.Type == "ffmpeg" {
			t.updateAssetsStreams(t.assetId)
		}
	}
	ffmpegProcesses--
	database.DbSetFFmpegLog(db, t.assetId, fullLog)
	log.Printf("-- [ %d ] FFmpeg execution success", t.assetId)

	return
}

func (t *TranscoderTask) updateAssetsStreams(assetId int) {
	//url := fmt.Sprintf("http://p-afsmsch-001.afrostream.tv:4000/api/assetsStreams/%d", assetId)
	url := fmt.Sprintf("%s/api/assetsStreams/%d", pfSchedulerUrl, assetId)
	_, err := http.Post(url, "application/json", strings.NewReader("{}"))
	if err != nil {
		log.Printf("XX Cannot update assetsStreams with url %s: %s", url, err)
		return
	}

	return
}

func (t *TranscoderTask) StartEncoding() {
	//DATABASE -->
	db, err := database.OpenGormDb()
	if err != nil {
		log.Printf("Cannot connect to database, error=%s", err)
		return
	}
	defer db.Close()
	log.Printf("-- Transcoding assetId=%d started...", t.assetId)
	//Asset Informations (RESULT)
	asset := database.Asset{ID: t.assetId}
	if db.Where(&asset).First(&asset).RecordNotFound() {
		log.Printf("Cannot find asset with ID=%d, error=%s", t.assetId, err)
		t.setAssetState(&asset, "failed")
		return
	}
	t.setAssetState(&asset, "processing")
	//Content Informations (SOURCE)
	content := database.Content{ID: asset.ContentId}
	if db.Where(&content).First(&content).RecordNotFound() {
		log.Printf("Cannot find content with ID=%d, error=%s", asset.ContentId, err)
		t.setAssetState(&asset, "failed")
		return
	}
	//DependanceAsset Informations (SOURCE BIS)
	var dependanceAsset database.Asset
	if asset.AssetIdDependance != "" {
		dependanceAssetId, err := strconv.Atoi(asset.AssetIdDependance)
		if err != nil {
			t.setAssetState(&asset, "failed")
			return
		}
		dependanceAsset = database.Asset{ID: dependanceAssetId}
		if db.Where(&dependanceAsset).First(&dependanceAsset).RecordNotFound() {
			log.Printf("Cannot find dependanceAsset with ID=%d, error=%s", asset.AssetIdDependance, err)
			t.setAssetState(&asset, "failed")
			return
		}
	}
	//Preset Informations
	preset := database.Preset{ID: asset.PresetId}
	if db.Where(&preset).First(&preset).RecordNotFound() {
		log.Printf("Cannot find preset with ID=%d, error=%s", asset.PresetId, err)
		t.setAssetState(&asset, "failed")
		return
	}
	//ProfileParameters Informations
	profilesParameters := []database.ProfilesParameter{}
	db.Where(&database.ProfilesParameter{AssetId: asset.ID}).Find(&profilesParameters)
	//
	//AssetsFromSameContent Informations
	assetsFromSameContent := []database.Asset{}
	db.Where(&database.Asset{ContentId: content.ID}).Find(&assetsFromSameContent)
	//<-- DATABASE
	sourceFilename := dependanceAsset.Filename
	if sourceFilename == "" {
		sourceFilename = content.Filename
	}
	dir := path.Dir(asset.Filename)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Printf("Cannot create directory %s, error=%s", dir, err)
		t.setAssetState(&asset, "failed")
		return
	}
	subtitlesStr, subtitlesMap, err := t.generateSubtitles(content, dir)
	if err != nil {
		log.Printf("Cannot generate subtitles, error=%s", err)
		t.setAssetState(&asset, "failed")
		return
	}
	cmdLine, err := t.generateCommandLine(sourceFilename,
		content,
		asset,
		assetsFromSameContent,
		preset,
		profilesParameters,
		subtitlesStr,
		subtitlesMap)
	err = t.executeCommand(preset.Type, cmdLine)
	if err != nil {
		log.Printf("an error occurred while calling executeCommand, error=%s", err)
		t.setAssetState(&asset, "failed")
		//FIXME : Why should it be done in this case only (see old code...)
		/**
		if contentId != nil {
			database.DbSetContentStatus(db, *contentId, "failed")
		}
		**/
		return
	}
	t.setAssetState(&asset, "ready")
	if preset.Type == "ffmpeg" {
		t.updateAssetsStreams(t.assetId)
	}
	//TO BE CONTINUED
	log.Printf("-- Transcoding assetId=%d ended successfully", t.assetId)
}

func (t *TranscoderTask) setAssetState(asset *database.Asset, state string) {
	db, err := database.OpenGormDb()
	if err != nil {
		log.Printf("setAssetState : Cannot connect to database, error=%s", err)
		//TODO => FAILED
	}
	defer db.Close()
	asset.State = state
	db.Save(asset)
}

func(t* TranscoderTask) setFfmpegProgress(v []string) (ffmpegProgress database.FfmpegProgress, err error) {
	db, err := database.OpenGormDb()
	if err != nil {
		log.Printf("setFfmpegProgress : Cannot connect to database, error=%s", err)
		//TODO => FAILED
	}
	defer db.Close()
	/* NCO : Cannot use "primary_key because there's no one...*/
	ffmpegProgress = database.FfmpegProgress{AssetId:t.assetId}
		//GET
	if db.Where(&ffmpegProgress).First(&ffmpegProgress).RecordNotFound() {
		//CREATE
		db.Create(&ffmpegProgress)
	}
	//UPDATE
	frame, err := strconv.Atoi(v[1])
	if err == nil {
		ffmpegProgress.Frame = frame
	} else {
		log.Printf("setFfmpegProgress : conversion failed for frame, string=%s", v[1])
	}
	fps, err := strconv.Atoi(v[2])
	if err == nil {
		ffmpegProgress.Fps = fps
	} else {
		log.Printf("setFfmpegProgress : conversion failed for fps, string=%s", v[2])
	}
	q, err := strconv.ParseFloat(v[3], 32)
	if err == nil {
		ffmpegProgress.Q = float32(q)
	} else {
		log.Printf("setFfmpegProgress : conversion failed for q, string=%s", v[3])
	}
	size, err := strconv.Atoi(v[4])
	if err == nil {
		ffmpegProgress.Size = size
	} else {
		log.Printf("setFfmpegProgress : conversion failed for size, string=%s", v[4])
	}
	ffmpegProgress.Elapsed = strings.Split(v[5], ".")[0]
	bitrate, err := strconv.ParseFloat(v[6], 32)
	if err == nil {
		ffmpegProgress.Bitrate = float32(bitrate)
	} else {
		log.Printf("setFfmpegProgress : conversion failed for bitrate, string=%s", v[6])
	}
	db.Save(&ffmpegProgress)
	return
}

func (t* TranscoderTask) addFfmpegLog(msg string) (ffmpegLog database.FfmpegLog, err error) {
	db, err := database.OpenGormDb()
	if err != nil {
		log.Printf("addFfmpegLog : Cannot connect to database, error=%s", err)
	}
	ffmpegLog = database.FfmpegLog{AssetId:t.assetId, Log:msg}
	db.Create(&ffmpegLog)
	return
}

func (t *TranscoderTask) generateSubtitles(content database.Content, dir string) (subtitlesStr string, subtitlesMap map[string]string, err error) {
	db, err := database.OpenGormDb()
	if err != nil {
		log.Printf("generateCommandLine : Cannot connect to database, error=%s", err)
		return
	}
	defer db.Close()
	subtitles := []database.Subtitle{}
	db.Where(&database.Subtitle{ContentId:content.ID}).Find(&subtitles)
	rowsEmpty := true
	for _, subtitle := range subtitles {
		lang := subtitle.Lang
		rowUrl := subtitle.Url
		// TODO : emulate : lang,SUBSTRING_INDEX(url, '/', -1) AS vtt
		vtt := rowUrl
		subtitlesMap[lang] = encodedBasePath + "/origin/vod/" + path.Base(dir) + "/" + strings.Replace(vtt, " ", "_", -1)
		subtitlesStr += strings.Replace(vtt, " ", "_", -1) + "%" + lang + " "
		rowsEmpty = false
	}
	//TODO : NCO : Why do we remove last caracter ?
	if rowsEmpty == false {
		subtitlesStr = subtitlesStr[:len(subtitlesStr)-1]
	}
	return
}

func (t *TranscoderTask) generateCommandLine(sourceFilename string,
	content database.Content,
	asset database.Asset,
	assetsFromSameContent []database.Asset,
	preset database.Preset,
	profilesParameters []database.ProfilesParameter,
	subtitlesStr string,
	subtitlesMap map[string]string) (cmdLine string, err error) {
	cmdLine = strings.Replace(preset.CmdLine, "%SOURCE%", sourceFilename, -1)
	cmdLine = strings.Replace(cmdLine, "%DESTINATION%", asset.Filename, -1)
	cmdLine = strings.Replace(cmdLine, "%UUID%", content.Uuid, -1)
	cmdLine = strings.Replace(cmdLine, "%BASEDIR%", path.Dir(asset.Filename), -1)
	cmdLine = strings.Replace(cmdLine, "%SUBTITLES%", subtitlesStr, -1)
	for k, l := range subtitlesMap {
		log.Printf("k is %s", k)
		cmdLine = strings.Replace(cmdLine, "%SUBTITLE_"+strings.ToUpper(k)+"%", l, -1)
	}
	re, err := regexp.Compile("%SOURCE_[0-9]+%")
	if err != nil {
		log.Printf("generateCommandLine : Cannot compile regexp %SOURCE_[0-9]+%: %s", err)
		return
	}
	matches := re.FindAllString(cmdLine, -1)
	if matches != nil {
		for _, m := range matches {
			str := strings.Split(m, "_")
			var str1AsInt int
			str1AsInt, err = strconv.Atoi(str[1])
			if err != nil {
				log.Printf("generateCommandLine : conversion failed, error=%s", err)
				return
			}
			//TODO : NCO : MAYBE BETTER WAY (LATER)
			assetFromSameContentFound := false
			for _, assetFromSameContent := range assetsFromSameContent {
				if assetFromSameContent.PresetId == str1AsInt {
					cmdLine = strings.Replace(cmdLine, m, assetFromSameContent.Filename, -1)
					assetFromSameContentFound = true
					break
				}
			}
			if assetFromSameContentFound == false {
				log.Printf("generateCommandLine : no assetFromSameContent found for assetId=%d with contentId=%d presetId=%d", asset.ID, content.ID, str1AsInt)
				return
			}
		}
	}
	//Parameters
	for _, profilesParameter := range profilesParameters {
		cmdLine = strings.Replace(cmdLine, "%"+profilesParameter.Parameter+"%", profilesParameter.Value, -1)
	}
	return
}

func (t *TranscoderTask) executeCommand(Type string, cmdLine string) (generalErr error) {
	log.Printf("-- [ %d ] executeCommand starting...", t.assetId)
	var cmdArgs []string
	var binaryPath string
	cmdArgs = strings.Split(cmdLine, " ")
	switch Type {
	case "ffmpeg":
		binaryPath = ffmpegPath
	case "spumux":
		binaryPath = spumuxPath
	case "script":
		binaryPath = cmdArgs[0]
		cmdArgs = cmdArgs[1:]
	default:
		msg := fmt.Sprintf("executeCommand : unknown Type=%s", Type)
		log.Printf(msg)
		generalErr = errors.New(msg)
		return
	}
	cmd := exec.Command(binaryPath, cmdArgs...)
	stderr, generalErr := cmd.StderrPipe()
	if generalErr != nil {
		log.Printf("-- [ %d ] executeCommand : stdErrPipe failed : %s %s: %s", t.assetId, binaryPath, strings.Join(cmdArgs, " "), generalErr)
		return
	}
	log.Printf("-- [ %d ] executeCommand : command starting... : %s %s", t.assetId, binaryPath, strings.Join(cmdArgs, " "))
	generalErr = cmd.Start()
	if generalErr != nil {
		//NCO : putted outside, but why content status does changes ? 
		/*database.DbSetAssetStatus(db, t.assetId, "failed")
		if contentId != nil {
			database.DbSetContentStatus(db, *contentId, "failed")
		}*/
		log.Printf("-- [ %d ] executeCommand : command starting failed : %s %s: %s", t.assetId, binaryPath, strings.Join(cmdArgs, " "), generalErr)
		return
	}
	ffmpegProcesses++

	fullLog := ""
	switch Type {
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
		// If ffmpeg exit on ERROR
		if ffmpegStartOK == false {
			t.addFfmpegLog(s)
			generalErr = errors.New(fmt.Sprintf("[ %d ] executeCommand : ffmpeg execution failed, please consult logs in database", t.assetId))
			cmd.Wait()
			ffmpegProcesses--
			return
		}
		re, err := regexp.Compile(`frame= *([0-9]*) *fps= *([0-9]*) *q= *([-0-9\.]*)* *L?size= *([0-9]*)kB *time= *([0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{2}) *bitrate= *([0-9\.]*)kbits/s`)
		if err != nil {
			log.Printf("[ %d ] executeCommand : Cannot compile regexp frame=([0-9]*), progression will not be available: %s", t.assetId, err)
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
					_, err := t.setFfmpegProgress(v)
					if err != nil {
						//just log the error
						log.Printf("-- [ %d ] progression update failed, error=%s", t.assetId, err)
					}
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
	generalErr = cmd.Wait()
	ffmpegProcesses--
	t.addFfmpegLog(fullLog)
	log.Printf("-- [ %d ] executeCommand done successfully", t.assetId)
	return
}
