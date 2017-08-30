#!/bin/bash

set -e

KEY="dGVjaEBhZnJvc3RyZWFtLnR2fDIwMTUwODIxIDAwOjAwOjAwLDM5NXxwYWNrYWdlKGlzcyk7c3RyZWFtKHZvZCxsaXZlKTtkcm0oYWVzLHBsYXlyZWFkeSxmYXhzLHdpZGV2aW5lKTtpb19vcHQoKTtjYXB0dXJlKCk7Y2hlY2soKTtzdXBwb3J0KDEpO29lbSgpfHZlcnNpb24oMTc0MCl8ODliZmM3Y2ZmZDhhNDFmNWJkZmRlMWY4YzRjMmU3NTZ8OWRiMzJkOTJmMTEyMmVkODQ1YWJmODA5ZTFmMGNmZTdhMzQxN2M2OTY5NTNlYTAwODQwZDM1NjI2ZDIwMTM1NmM3MTk5ODliNmExNmI0ZTUyNTc3MWM1MWNhZTVjOGZlMDQ2Y2IwZjRhNDUxZDM4ZmFmYjdhYWU4MjI2YjVhMWFiZWQyMDhhY2ZjNDI0YzFhN2JmNDBjNzY2YzUxMjAwMWIzNzcyNTQwYmRmMjk2Y2M3NzAxNDdjNzY2ZDdmMWQ4NTJjNDFkYzQyNmNjN2M3MWE2OTQyZDg4NzZlMmFiNzNiMGY0YTExNThhNjMzMjAwZDIwZGRhMDE1NDM3M2M1OQ=="

video_re=".*_h264_.*"
audio_re=".*_aac(.*)"
audio_lang_re="_([a-z]{3}).mp4"
subtitles_re=".*\.vtt%.*"

mp4split="mp4split \
	--license-key ${KEY} \
	--iss.playout clear --iss.minimum_fragment_length=8 \
	--hls.playout clear --hls.minimum_fragment_length=8 --hls.no_audio_only --hls.client_manifest_version=4 \
	--hds.playout false \
	--mpd.playout clear --mpd.minimum_fragment_length=8 \
	--fragment-duration 8000 \
	-o "

cd $1
shift

for i in $*
do
  if [[ ${i} =~ ${subtitles_re} ]]
  then
    vttsub=`echo ${i} | cut -f1 -d'%'`
    lang=`echo ${i} | cut -f2 -d'%'`
    dfxpsub=`echo ${vttsub} | sed 's/\.vtt/\.dfxp/g'`
    ismtsub=`echo ${vttsub} | sed 's/\.vtt/\.ismt/g' | sed 's/.*\///g'`
    mp4split --license-key ${KEY} -o ${dfxpsub} --fragment_duration=1000000000 ${vttsub} --track_language=${lang}
    cat ${dfxpsub} | sed 's/tts:fontSize="16px"/tts:fontSize="22px"/g' > ${dfxpsub}.new
    mv ${dfxpsub}.new ${dfxpsub}
    mp4split --license-key ${KEY} -o ${ismtsub} --fragment_duration=1000000000 ${dfxpsub} --track_language=${lang}
    mp4split="${mp4split} ${ismtsub} --track_language=${lang}"
  elif [[ ${i} =~ ${video_re} ]]
  then
    mp4split="${mp4split} ${i} --track_type=video"
  elif [[ ${i} =~ ${audio_re} ]]
  then
    if [[ ${BASH_REMATCH[1]} =~ ${audio_lang_re} ]]
    then
      mp4split="${mp4split} ${i} --track_type=audio --track_language=${BASH_REMATCH[1]} "
    else
      mp4split="${mp4split} ${i} --track_type=audio "
    fi
  else
    mp4split="${mp4split} ${i}"
  fi
done

echo ${mp4split}

${mp4split}

