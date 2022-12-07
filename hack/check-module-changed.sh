#!/bin/bash

labels=""
function addTags(){
  if [[ -z "$labels" ]]; then
    labels=${1}
  else
    labels="${labels} && ${1}"
  fi
}

last_commit=$(git log --before="1 week ago" --pretty=format:"%h" -n 1)
files_changed=$(git diff $last_commit --name-only "cmd/" "pkg/" "test/" | grep ".go")
echo -e "File(s) Changed:\n$files_changed\n"

evs_tag=$(echo $files_changed | grep "evs" | wc -l)
if [[ $sfs_tag = "1" ]]; then
  echo "EVS has changed, required to run the EVS E2E tests"
  export E2E_EVS_TAG="EVS"
  addTags "EVS"
fi

sfs_tag=$(echo $files_changed | grep "sfs" | wc -l)
if [[ $sfs_tag = "1" ]]; then
  echo "SFS has changed, required to run the SFS E2E tests"
  export E2E_SFS_TAG="SFS"
  addTags "SFS"
fi

sfsturbo_tag=$(echo $files_changed | grep "sfsturbo" | wc -l)
if [[ $sfs_tag = "1" ]]; then
  echo "SFS TURBO has changed, required to run the SFS TURBO E2E tests"
  export E2E_SFS_TURBO_TAG="SFS_TURBO"
  addTags "SFS_TURBO"
fi

sfsturbo_tag=$(echo $files_changed | grep "obs" | wc -l)
if [[ $sfs_tag = "1" ]]; then
  echo "OBS has changed, required to run the OBS E2E tests"
  export E2E_OBS_TAG="OBS"
  addTags "OBS"
fi

export E2E_LABEL=labels

echo
echo "Module(s) changed: ${labels}"
echo
