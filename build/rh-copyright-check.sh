#!/bin/bash

# IBM Confidential
# OCO Source Materials

# (C) Copyright IBM Corporation 2019 All Rights Reserved
# The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.

# Check for redhat copyright and only for redhat copyright
# defensive check - if this passess AND the old copyright-check passes then we are good for sure

YEAR=2019

CHECK6="Copyright (c) 2020 Red Hat, Inc."

#LIC_ARY to scan for
LIC_ARY=("$CHECK6")
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

  # Read the first 10 lines, most Copyright headers use the first 6 lines.
  HEADER=`head -10 $f`
  # printf " Scanning $f . . . "
  # lastcommit=$(git --no-pager log -n -1 --date=local --after="2020-03-01T16:36" --pretty=format:"%ad" $f) # the last commit of this file
  printf "Last changed: `git --no-pager log -n 1 --pretty=format:"%ad"` $f \n"
  echo $(git diff --name-only HEAD...$TRAVIS_BRANCH )
  #Check for all copyright lines
  for i in `seq 0 $((${LIC_ARY_SIZE}+1))`; do
    #Add a status message of OK, if all copyright lines are found
    if [ $i -eq ${LIC_ARY_SIZE} ]; then
      # printf "Last changed: $lastcommit $f OK \n"
      printf "  "
    else
      #Validate the copyright line being checked is present
      if [[ "$HEADER" != *"${LIC_ARY[$i]}"* ]]; then
          # printf "Last changed: $lastcommit $f BAD \n"
          # printf "missing copyright line: [${LIC_ARY[$i]}]"
          if ! [ -z "$lastcommit" ]; # if there are new commits, then we need the rh copyright
            then
              ERROR=1
              break
            fi 
      fi
    fi
  done
done

echo "##### Copyright check ##### ReturnCode: ${ERROR}"
exit $ERROR
