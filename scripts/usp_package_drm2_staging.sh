#!/bin/bash

set -e

KEY="dGVjaEBhZnJvc3RyZWFtLnR2fDIwMTYwODI2IDAwOjAwOjAwLDM5NXxwYWNrYWdlKGlzcyxtcDQpO3N0cmVhbSh2b2QsbGl2ZSk7ZHJtKGFlcyxwbGF5cmVhZHksZmF4cyx3aWRldmluZSxwaWZmMmNlbmMpO2lvX29wdCgpO2NhcHR1cmUoKTtjaGVjaygpO3N1cHBvcnQoMSk7b2VtKCk7dmVyc2lvbigxLjcuNCl8TGljZW5zZSBrZXkgMjAxNi0yMDE3fGU0OTk4YTczN2MyODRlOGZhZTc2ZWYyMjY0OWY1MGM2fGI0N2E0YzcxYmJmZTdhMGE3YzIzOGFlYjQ1Y2IwOGVjZjIyYTBkN2YyMTZlOTM5NDc0MDA2MmNkMDJkNDMxYjBlYzIwOWE0YzYzOWVkNWE2MDQyMDEwNmFkYTI3ZTYyYWQzYzc2NTViN2VmNjMwYmUyMDBhMTZhMTVhNDQ4MmQ1ODVjMDY3MDY0Y2E2NTYyOTY0NDQzMjc3M2I0MDM1YTllYWY0MmQzMjMwOWVhYTA4YzViOTgyYThiZWNiMTUyZWY1MTRmMzA4N2RkODllNTI3NTQ1ZDFhYTczM2E1MTQ3YzJiNjFjYTg3YWUzYjdlMDcwYzE3NmI3OWVkYTg1YzQ="

drmCENCKey=`java -jar /usr/local/bin/tools-drmtoday-cli-3.0.2.jar genKey`
drmCENCKeyId=`java -jar /usr/local/bin/tools-drmtoday-cli-3.0.2.jar genKeyId`
drmCENCKeyIV=$(cat /dev/urandom | tr -dc 'a-f0-9' | fold -w 16 | head -n 1)

drmAssetid=`echo $2 | sed "s/.*\///g" | sed 's/\.ism//g'`
drmMerchant="afrostream"
drmUser="ludovic.bostral@afrostream.tv"
drmPassword="msk3847djvm294mdslt"

drmIngestCENC="java -jar /usr/local/bin/tools-drmtoday-cli-3.0.2.jar ingestCENC \
          --environment PROD \
          --key ${drmCENCKey} \
          --keyid ${drmCENCKeyId} \
          --streamtype VIDEO_AUDIO \
          --assetid ${drmAssetid} \
          --merchant ${drmMerchant} \
          --user ${drmUser} \
          --password ${drmPassword}"

drmDeleteCENC="java -jar /usr/local/bin/tools-drmtoday-cli-3.0.2.jar deleteByAssetId \
          --assetid ${drmAssetid} \
          --merchant ${drmMerchant} \
          --user ${drmUser} \
          --password ${drmPassword} \
          --streamtype VIDEO_AUDIO \
          --environment PROD"

USPCENCKeyId=`java -jar /home/ubuntu/tools-drmtoday-cli-3.0.2.jar convertBinaryRepresentation --input ${drmCENCKeyId} --inputFormat UUID --outputFormat HEX`

echo "Executing ${drmDeleteCENC}"
${drmDeleteCENC}

echo "Executing ${drmIngestCENC}"

jsonAnswer=`${drmIngestCENC} | grep '{"systemId":'`

echo ${jsonAnswer}

psshBoxContentB64=`echo ${jsonAnswer} | python -c "import json,sys; obj=json.load(sys.stdin); print obj[\"systemId\"][\"edef8ba9-79d6-4ace-a3c8-27dcd51d21ed\"][\"psshBoxContent\"]"`
psshBoxContent=`java -jar /usr/local/bin/tools-drmtoday-cli-3.0.2.jar convertBinaryRepresentation --input ${psshBoxContentB64} --inputFormat B64 --outputFormat HEX`

video_re=".*_h264_.*"
audio_re=".*_aac(.*)"
audio_lang_re="_([a-z]{3}).mp4"
subtitles_re=".*\.vtt%.*"
ism_re=".*\.ism"
random_number=`echo $RANDOM | md5sum | cut -f 1 -d' '`
vttfiles=""

mp4split="mp4split \
	--license-key ${KEY} \
        --iss.key=${USPCENCKeyId}:${drmCENCKey} \
        --iss.key_iv=${drmCENCKeyIV} \
        --iss.license_server=https://lic.staging.drmtoday.com/license-proxy-headerauth/drmtoday/RightsManager.asmx \
	--iss.minimum_fragment_length=8 \
        --widevine.key=${USPCENCKeyId}:${drmCENCKey} \
        --widevine.license_server_url=https://lic.staging.drmtoday.com/license-proxy-widevine/cenc/ \
        --hls.playout=clear \
        --hls.minimum_fragment_length=8 \
        --hls.no_audio_only \
        --hls.client_manifest_version=4 \
        --hds.key=${USPCENCKeyId}:${drmCENCKey} \
        --hds.key_iv=${drmCENCKeyIV} \
        --hds.license_server=https://lic.staging.drmtoday.com/flashaccess/LicenseTrigger/v1 \
	--mpd.minimum_fragment_length=8 \
	--fragment-duration 8000 \
	-o "
        #--widevine.drm_specific_data=${psshBoxContent} \

mp4splitwithoutsub=${mp4split}

cd $1
shift

for i in $*
do
  if [[ ${i} =~ ${subtitles_re} ]]
  then
    vttsub=`echo ${i} | cut -f1 -d'%'`
    if [ "x${vttfiles}" = "x" ]
    then
      vttfiles="${i}"
    else
      vttfiles="${vttfiles} ${i}"
    fi
    lang=`echo ${i} | cut -f2 -d'%'`
    dfxpsub=`echo ${vttsub} | sed 's/\.vtt/\.dfxp/g'`
    dfxpsubtmp=`echo ${vttsub} | sed "s/\.vtt/-${random_number}\.dfxp/g"`
    ismtsub=`echo ${vttsub} | sed 's/\.vtt/\.ismt/g' | sed 's/.*\///g'`
    ismtsubtmp=`echo ${vttsub} | sed "s/\.vtt/-${random_number}\.ismt/g" | sed 's/.*\///g'`
    #mp4split --license-key ${KEY} -o ${dfxpsubtmp} --fragment_duration=1000000000 ${vttsub} --track_language=${lang}
    mp4split --license-key ${KEY} -o ${dfxpsubtmp} --fragment_duration=8000 ${vttsub} --track_language=${lang}
    cat ${dfxpsubtmp} | sed 's/tts:fontSize="16px"/tts:fontSize="22px"/g' > ${dfxpsubtmp}.new
    mv ${dfxpsubtmp}.new ${dfxpsubtmp}
    #mp4split --license-key ${KEY} -o ${ismtsubtmp} --fragment_duration=1000000000 ${dfxpsubtmp} --track_language=${lang}
    mp4split --license-key ${KEY} -o ${ismtsubtmp} --fragment_duration=8000 ${dfxpsubtmp} --track_language=${lang}
    mv ${ismtsubtmp} ${ismtsub}
    rm ${dfxpsubtmp}
    mp4split="${mp4split} ${ismtsub} --track_language=${lang}"
  elif [[ ${i} =~ ${video_re} ]]
  then
    mp4split="${mp4split} ${i} --track_type=video"
    mp4splitwithoutsub="${mp4splitwithoutsub} ${i} --track_type=video"
  elif [[ ${i} =~ ${audio_re} ]]
  then
    if [[ ${BASH_REMATCH[1]} =~ ${audio_lang_re} ]]
    then
      mp4split="${mp4split} ${i} --track_type=audio --track_language=${BASH_REMATCH[1]} "
      mp4splitwithoutsub="${mp4splitwithoutsub} ${i} --track_type=audio --track_language=${BASH_REMATCH[1]} "
    else
      mp4split="${mp4split} ${i} --track_type=audio "
      mp4splitwithoutsub="${mp4splitwithoutsub} ${i} --track_type=audio "
    fi
  else
    mp4split="${mp4split} ${i}"
    mp4splitwithoutsub="${mp4splitwithoutsub} ${i}"
  fi
done

echo ${mp4splitwithoutsub}
${mp4splitwithoutsub}

# Create specific mpd dash manifest with vtt references if needed
if [ "x${vttfiles}" != "x" ]
then
  echo "vttfiles are ${vttfiles}"
  mpd=`echo "${1}" | sed 's/\.ism/\.mpd/g'`
  mpdtmp=`echo "${1}" | sed "s/\.ism/-${random_number}\.mpd/g"`
  name=`pwd | cut -f7 -d'/'`
  rm -f "${mpd}"
  wget -O "${mpdtmp}" "http://origin.cdn.afrostream.net/vod/${name}/${1}/.mpd"
  adaptationsets=""
  for i in ${vttfiles}
  do
    url=`echo "https:\/\/hw.cdn.afrostream.net\/vod\/${name}\/${i}" | cut -f1 -d'%'`
    lang=`echo "${i}" | cut -f2 -d'%'`
    adaptationset="    <AdaptationSet mimeType=\"text\/vtt\" lang=\"${lang}\">\n      <Representation id=\"caption_${lang}\" bandwidth=\"256\">\n        <BaseURL>${url}<\/BaseURL>\n      <\/Representation>\n    <\/AdaptationSet>"
    if [ "x${adaptationsets}" = "x" ]
    then
      adaptationsets="${adaptationset}"
    else
      adaptationsets="${adaptationsets}\n${adaptationset}"
    fi
  done

  echo ${adaptationsets}

  cat "${mpdtmp}" | sed "s/ *<BaseURL>dash\/<\/BaseURL>/${adaptationsets}\n    <BaseURL>dash\/<\/BaseURL>/g" > "${mpd}"
  rm -f "${mpdtmp}"
fi

echo ${mp4split}
${mp4split}
