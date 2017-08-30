#!/bin/bash

if [ "$1" == "" ]
then
    echo "missing input file or/and profileId, exiting"
    exit 255
fi


FILENAME=$(basename "$1")
PROFILEID=`echo "${FILENAME}" | cut -f 4 -d '_' | cut -f1 -d'.'`
SOURCE_FILENAME="/space/videos/sources/$FILENAME"

#logger -p user.notice "moving $1 to ${SOURCE_FILENAME}"
mv $1 ${SOURCE_FILENAME}

logger -p user.notice "file ${FILENAME}: ensure permissions are set to 0644"
chmod 0644 ${SOURCE_FILENAME}

logger -p user.notice "file ${FILENAME}: calling upload"
#source /usr/local/rvm/scripts/rvm \
#    && cd /space/apps/afrostream_mama/current \
#    && RAILS_ENV=prd bin/rake mama:ingest source_file="${SOURCE_FILENAME}" 1>>/tmp/ingest.log 2>&1
result=`curl -X POST -d "{\"filename\":\"${SOURCE_FILENAME}\"}" http://p-afsmsch-001.afrostream.tv:4000/api/contents`
uuid=`echo ${result} | cut -f2 -d, | cut -f2 -d: | sed 's/"//g' | sed 's/}//g'`
curl -X POST -d "{\"uuid\":\"${uuid}\",\"profileId\":${PROFILEID}}" http://p-afsmsch-001.afrostream.tv:4000/api/transcode

if [ $? -eq 0 ]
then
    logger -p user.notice "file ${FILENAME}: successfully called pfscheduler api"
else
    logger -p user.error "file ${FILENAME}: error while calling pfscheduler api"
fi
