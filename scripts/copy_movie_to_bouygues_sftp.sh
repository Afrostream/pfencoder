#!/bin/sh

sourceFile=$1

oldFilename=`basename $1`
newFilename=`echo -n $oldFilename | md5sum | awk '{ print $1 }'`.ts

date >> /tmp/copy-movie-to-bouygues.log
echo $1 >> /tmp/copy-movie-to-bouygues.log
echo $oldFilename >> /tmp/copy-movie-to-bouygues.log
echo $newFilename >> /tmp/copy-movie-to-bouygues.log
echo "rename $oldFilename $newFilename" >> /tmp/copy-movie-to-bouygues.log

tmp=`mktemp /tmp/batchftpXXXXXXXX`
ssh-keygen -f "/home/ubuntu/.ssh/known_hosts" -R 195.36.151.236
#echo "rm $1\n" >> ${tmp}
echo "mput $1\n" > ${tmp}
echo "rename $oldFilename $newFilename\n" >> ${tmp}
echo "bye\n" >> ${tmp}
sftp -oStrictHostKeyChecking=no -i /home/ubuntu/bouygues-sftp-rsa -b ${tmp} videovasrep@195.36.151.236:/public
rc=$?
if [ "X$rc" != "X0" ]
then
  echo "exit with return code $rc"
  exit $rc
fi
rm ${tmp}
