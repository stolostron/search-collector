#!/bin/bash

#IBM Confidential
#OCO Source Materials

#(C) Copyright IBM Corporation 2019 All Rights Reserved
#The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.

YEAR=2019

#LINE1="${COMMENT_PREFIX}IBM Confidential"
CHECK1="IBM Confidential"
#LINE2="${COMMENT_PREFIX}OCO Source Materials"
CHECK2="OCO Source Materials"


#LINE4="${COMMENT_PREFIX}(C) Copyright IBM Corporation 2019 All Rights Reserved"
CHECK4="(C) Copyright IBM Corporation 2019 All Rights Reserved"
#LINE5="${COMMENT_PREFIX}The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office."
CHECK5="The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office."

CHECK6="Copyright (c) 2020 Red Hat, Inc."

#LIC_ARY to scan for
LIC_ARY=("$CHECK1" "$CHECK2" "$CHECK3" "$CHECK4" "$CHECK5" "$CHECK6")
LIC_ARY_SIZE=${#LIC_ARY[@]}

#Used to signal an exit
ERROR=0


echo "##### Copyright check #####"
#Loop through all files. Ignore .FILENAME types
for f in `find . -type f -iname "*.go" ! -path "./build-harness/*" ! -path "./sslcert/*" ! -path "./test-data/*" ! -path "./vendor/*"`; do
  if [ ! -f "$f" ] || [ "$f" = "./build-tools/copyright-check.sh" ]; then
    continue
  fi

  FILETYPE=$(basename ${f##*.})
  case "${FILETYPE}" in
  	sh | go)
  		COMMENT_PREFIX=""
  		;;
  	*)
      continue
  esac

  #Read the first 10 lines, most Copyright headers use the first 6 lines.
  HEADER=`head -10 $f`
  printf " Scanning $f . . . "

  #Check for all copyright lines
  for i in `seq 0 $((${LIC_ARY_SIZE}+1))`; do
    #Add a status message of OK, if all copyright lines are found
    if [ $i -eq ${LIC_ARY_SIZE} ]; then
      printf "OK\n"
    else
      #Validate the copyright line being checked is present
      if [[ "$HEADER" != *"${LIC_ARY[$i]}"* ]]; then
        printf "Missing copyright\n  >>Could not find [${LIC_ARY[$i]}] in the file $f\n"
        ERROR=1
        break
      fi
    fi
  done
done

echo "##### Copyright check ##### ReturnCode: ${ERROR}"
exit $ERROR
