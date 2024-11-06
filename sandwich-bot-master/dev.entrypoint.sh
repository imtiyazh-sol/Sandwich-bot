#!/bin/bash

root=$(pwd)
child_directories=$(find "$root" -maxdepth 1 -type d)
sought_env_file=".env.dev"
var_env_arr=("POSTGRES_USER" "POSTGRES_PORT" "POSTGRES_DB" "APP_PORT")

result=""

for dir in $child_directories
do
  if [ "$root" == "$dir" ]; then
    continue
  fi

  env_file=$(find "$dir" -maxdepth 1 -type f -iname "$sought_env_file")

  if [ ! -e "$env_file" ]; then
    continue
  fi

  dir_name=$(basename $dir)

  
  for var in "${var_env_arr[@]}"
  do
    var_value=$(grep -ri "$var" $env_file)
    if [ -z "$var_value" ]; then
      continue
    fi

    dir_name=$(echo "$dir_name" | tr '[:lower:]' '[:upper:]')

    new_value="${dir_name}_$var=${var_value##*=}"

    result+="${new_value} "
  done
done

# echo "$result docker compose -f dev.docker-compose.yml $1 $2"
eval "$result docker compose -f dev.docker-compose.yml $*"


