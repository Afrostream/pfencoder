#!/bin/sh

uuid=$1
source=$2
destination=$3
tmpdestination=`echo ${destination} | sed 's/\.[^\.]*$/\.tmp\.vob/g'`
urlfra=$4
urleng=$5
xmlfilefra="/tmp/$1-submuxconfig_fra.xml"
xmlfileeng="/tmp/$1-submuxconfig_eng.xml"

wget -O /tmp/${uuid}.fra.vtt ${urlfra}
if [ $? -ne 0 ]
then
  exit $?
fi
wget -O /tmp/${uuid}.eng.vtt ${urleng}
if [ $? -ne 0 ]
then
  exit $?
fi
echo "<subpictures format=\"pal\">" > ${xmlfilefra}
echo "   <stream>" >> ${xmlfilefra}
echo "      <textsub filename=\"/tmp/${uuid}.fra.vtt\" characterset=\"UTF-8\"" >> ${xmlfilefra}
echo "         fontsize=\"28.0\" font=\"arial.ttf\" horizontal-alignment=\"left\"" >> ${xmlfilefra} 
echo "         vertical-alignment=\"bottom\" left-margin=\"60\" right-margin=\"60\"" >> ${xmlfilefra} 
echo "         top-margin=\"20\" bottom-margin=\"30\" subtitle-fps=\"25\"" >> ${xmlfilefra} 
echo "         movie-fps=\"25\" movie-width=\"720\" movie-height=\"574\"" >> ${xmlfilefra}
echo "         force=\"yes\"" >> ${xmlfilefra}
echo "      />" >> ${xmlfilefra}
echo "   </stream>" >> ${xmlfilefra}
echo "</subpictures>" >> ${xmlfilefra}

echo "<subpictures format=\"pal\">" > ${xmlfileeng}
echo "   <stream>" >> ${xmlfileeng}
echo "      <textsub filename=\"/tmp/${uuid}.fra.vtt\" characterset=\"UTF-8\"" >> ${xmlfileeng}
echo "         fontsize=\"28.0\" font=\"arial.ttf\" horizontal-alignment=\"left\"" >> ${xmlfileeng} 
echo "         vertical-alignment=\"bottom\" left-margin=\"60\" right-margin=\"60\"" >> ${xmlfileeng} 
echo "         top-margin=\"20\" bottom-margin=\"30\" subtitle-fps=\"25\"" >> ${xmlfileeng} 
echo "         movie-fps=\"25\" movie-width=\"720\" movie-height=\"574\"" >> ${xmlfileeng}
echo "         force=\"yes\"" >> ${xmlfileeng}
echo "      />" >> ${xmlfileeng}
echo "   </stream>" >> ${xmlfileeng}
echo "</subpictures>" >> ${xmlfileeng}

/usr/bin/spumux -s3 "${xmlfilefra}" < "${source}" > "${tmpdestination}"
if [ $? -ne 0 ]
then
  exit $?
fi
/usr/bin/spumux -s4 "${xmlfileeng}" < "${tmpdestination}" > "${destination}"
if [ $? -ne 0 ]
then
  exit $?
fi

rm /tmp/${uuid}.fra.vtt
rm /tmp/${uuid}.eng.vtt
rm ${xmlfilefra}
rm ${xmlfileeng}

exit $?
