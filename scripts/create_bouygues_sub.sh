#!/bin/bash

export VIDEO_FORMAT=PAL

uuid=$1
source=$2
destination=$3
urlfra=$4
urleng=$5
xmlfilefra="/tmp/$1-submuxconfig_fra.xml"

wget -O /tmp/${uuid}.fra.vtt ${urlfra}
#if [ "$?" -ne "0" ]
#then
#  exit $?
#fi
cat /tmp/${uuid}.fra.vtt | sed 's/\([0-9][0-9]\)\.\([0-9][0-9][0-9]\)/\1,\2/g' | sed 's/WEBVTT//g' | sed 's/NOTE Paragraph//g' | sed '1,4d' > /tmp/${uuid}.fra.srt

echo "<subpictures format=\"PAL\">" > ${xmlfilefra}
echo "   <stream>" >> ${xmlfilefra}
echo "      <textsub filename=\"/tmp/${uuid}.fra.srt\" characterset=\"UTF-8\"" >> ${xmlfilefra}
echo "         fontsize=\"28.0\" font=\"arial.ttf\" horizontal-alignment=\"left\"" >> ${xmlfilefra} 
echo "         vertical-alignment=\"bottom\" left-margin=\"60\" right-margin=\"60\"" >> ${xmlfilefra} 
echo "         top-margin=\"20\" bottom-margin=\"30\" subtitle-fps=\"25\"" >> ${xmlfilefra} 
echo "         movie-fps=\"25\" movie-width=\"720\" movie-height=\"574\"" >> ${xmlfilefra}
echo "         force=\"yes\"" >> ${xmlfilefra}
echo "      />" >> ${xmlfilefra}
echo "   </stream>" >> ${xmlfilefra}
echo "</subpictures>" >> ${xmlfilefra}

/usr/bin/spumux -s3 "${xmlfilefra}" < "${source}" > "${destination}"
#if [ "$?" -ne "0" ]
#then
#  exit $?
#fi

rm /tmp/${uuid}.fra.srt
rm /tmp/${uuid}.fra.vtt
rm ${xmlfilefra}

exit $?
