#!/bin/bash

#Loop through all files. Ignore .FILENAME types
for f in `find . -type f -iname "*.go" ! -path "./build-harness/*" ! -path "./sslcert/*" ! -path "./test-data/*" ! -path "./vendor/*"`; do
  rhcommits=$(git --no-pager log --date=local --after="2020-03-01T16:36" --pretty=format:"%ad" $f) 
  if ! [ -z "$rhcommits" ]; 
    then
      echo $f 
  fi 


done


