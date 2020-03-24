# Licensed Materials - Property of IBM
# (c) Copyright IBM Corporation 2018, 2019. All Rights Reserved.
# Note to U.S. Government Users Restricted Rights:
# Use, duplication or disclosure restricted by GSA ADP Schedule
# Contract with IBM Corp.

#!/bin/bash

#Project start year
origin_year=2018
#Back up year if system time is null or incorrect
back_up_year=2019
#Currrent year
current_year=$(date +"%Y")

if [ -z "$current_year" ] || [ $current_year -lt $origin_year ]; then
  echo "Can't get correct system time\n  >>Use back_up_year=$back_up_year as current_year to check copyright in the file $f\n"
  current_year=$back_up_year
fi

#lic_year to scan for year formart single line's correctness
lic_year=()
#All possible combination within [origin_year, current_year] range is valid format
#seq isn't recommanded after bash version 3.0
for ((start_year=origin_year;start_year<=current_year;start_year++)); 
do
  lic_year+=(" (c) Copyright IBM Corporation ${start_year}. All Rights Reserved.")
  for ((end_year=start_year+1;end_year<=current_year;end_year++)); 
  do
    lic_year+=(" (c) Copyright IBM Corporation ${start_year}, ${end_year}. All Rights Reserved.")
  done
done
lic_year_size=${#lic_year[@]}

#lic_rest to scan for rest copyright format's correctness
lic_rest=()
lic_rest+=(" Licensed Materials - Property of IBM")
lic_rest+=(" Note to U.S. Government Users Restricted Rights:")
lic_rest+=(" Use, duplication or disclosure restricted by GSA ADP Schedule")
lic_rest+=(" Contract with IBM Corp.")
lic_rest_size=${#lic_rest[@]}

#Used to signal an exit
ERROR=0


echo "##### Copyright check #####"
#Loop through all files. Ignore .FILENAME types
for f in `find . -type f ! -iname ".*" ! -path "./pkg/hcmetcdqueue/*" ! -path "./vendor/*" ! -path "./swagger/*" ! -path "./node_modules/*" ! -path "./coverage/*"`; do
  if [ ! -f "$f" ] || [ "$f" = "./build-tools/copyright-check.sh" ]; then
    continue
  fi

  FILETYPE=$(basename ${f##*.})
  case "${FILETYPE}" in
  	js | sh | go | java | rb)
  		COMMENT_PREFIX=""
  		;;
  	*)
      continue
  esac

  #Read the first 10 lines, most Copyright headers use the first 6 lines.
  header=`head -10 $f`
  printf " Scanning $f . . . "

  #Check for year copyright single line
  year_line_count=0
  for ((i=0;i<${lic_year_size};i++));
  do
    #Validate year formart within [origin_year, current_year] range
    if [[ "$header" == *"${lic_year[$i]}"* ]]; then
      year_line_count=$((year_line_count + 1))
    fi
  done

  #Must find and only find one line valid year, otherwise invalid copyright formart
  if [[ $year_line_count != 1 ]]; then
    printf "Missing copyright\n  >>Could not find correct copyright year in the file $f\n"
    ERROR=1
    break
  fi

  #Check for rest copyright lines
  for ((i=0;i<${lic_rest_size};i++));
  do
    #Validate the copyright line being checked is present
    if [[ "$header" != *"${lic_rest[$i]}"* ]]; then
      printf "Missing copyright\n  >>Could not find [${lic_rest[$i]}] in the file $f\n"
      ERROR=1
      break
    fi
  done

  #Add a status message of OK, if all copyright lines are found
  if [[ "$ERROR" == 0 ]]; then
    printf "OK\n"
  fi
done

echo "##### Copyright check ##### ReturnCode: ${ERROR}"
exit $ERROR
