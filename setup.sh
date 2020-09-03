#!/bin/bash
stack_name="mika"
compose_file="docker-compose.yml"
image_name="ghcr.io/leighmacdonald/mika"

run_config() {
  if test -f mika.yaml; then
    echo "mika.yaml exists already, please delete or rename it if you wish to recreate your config"
    exit 1
  fi
  torrentStore=""
  peerStore=""
  userStore=""

  printf "Mika supports 3 different storage setups which can be configured to best suite your specific needs.\n"
  printf "The following 3 questions will ask you which you want to use."
  printf "\nMost users starting with a fresh installation should use either the default postgres or mariadb options"
  printf "for all but the peer store.\n"
  printf "For more information on how this works please read docs/STORE_*.md files for more details.\n"

  while [[ $torrentStore == "" ]]; do
    printf "\nTorrent store config\n"
    read -rp "Which store type are you using for torrents? ([postgres], mysql, redis, memory): " ts
    if [ "$ts" == "" ]; then
      ts="postgres"
    fi
    read -rp "Using $ts for torrent store, Is this correct? [Y/n]" correct
    if [ "$correct" == "" ] || [ "$correct" == "y" ] || [ "$correct" == "Y" ]; then
      torrentStore=$ts
    fi
  done

  while [[ $userStore == "" ]]; do
    printf "\nUser store config\n"
    printf "Its strongly recommended to use the same settings as the torrent store for most users.\n"
    printf "This option is only meant to cover some rare use cases when integrating into existing authentication systems\n"
    read -rp "Which store type are you using for users? ([postgres], mysql, redis, memory): " us
    if [ "$us" == "" ]; then
      us="postgres"
    fi
    read -rp "Using $us for user store, Is this correct? [Y/n]" correct
    if [ "$correct" == "" ] || [ "$correct" == "y" ] || [ "$correct" == "Y" ]; then
      userStore=$us
    fi
  done

  while [[ $peerStore == "" ]]; do
    printf "\nPeer store config\n"
    printf "Memory is *strongly* recommended for most use cases\n"
    read -rp "Which store type are you using for active peers?  (postgres, mysql, redis, [memory]): " ps
    read -rp "Using $ps for user store, Is this correct? [Y/n]" correct
    if [ "$correct" == "" ] || [ "$correct" == "y" ] || [ "$correct" == "Y" ]; then
      peerStore=$us
    fi
  done

  while [[ $emailValid == false ]]; do
    read -rp "Enter your email for Lets Encrypt SSL Certs (eg: user@host.com): " email
    echo "Using email for registration: $email"
    read -rp "Is this correct? [Y/n]" correctE
    if [ "$correctE" == "" ] || [ "$correctE" == "y" ] || [ "$correctE" == "Y" ]; then
      emailValid=true
    fi
  done

  cp mika.yaml.dist mika.yaml
  echo "Enter your IP2Location API Key if you have one. Without this you will not be " \
    "able to use any geo lookup functionality."
  read -rp ": " geoenable
  if [ "$geoenable" != "" ]; then
    echo "Enabling geo database functionality using key: $geoenable"
    sed -i -r "s/geodb_api_key:/geodb_api_key: ${geoenable}/g" docker/mika.yaml
    sed -i -r "s/geodb_enabled: false/geodb_enabled: true/g" docker/mika.yaml
  fi
  echo "Created mika.yaml"
  echo "If you are using docker, you can build your image using $0 build_image"
}

run_install() {
  if ! test -f mika.yaml; then
    run_config
  fi
  docker-compose -f ${compose_file} -p ${stack_name} build
  docker-compose -f ${compose_file} -p ${stack_name} up -d db
  echo "Please wait while the database initializes, could take over a minute on slower hardware"
  ret_val=-1
  until [ $ret_val -eq 0 ]; do
    docker-compose -f ${compose_file} -p ${stack_name} exec mariadb mysql -uunit3d -punit3d -D unit3d -s -e "SELECT 1" >/dev/null 2>&1
    ret_val=$?
    printf "."
    sleep 2
  done
  echo ""
  docker-compose -f ${compose_file} -p ${stack_name} up -d
  docker-compose -f ${compose_file} -p ${stack_name} logs -f
}

run_clean_config() {
  rm -rf mika.yaml
}

run_psql() {
  docker-compose -f ${compose_file} -p ${stack_name} up -d db
  docker-compose -f ${compose_file} -p ${stack_name} exec psql
}

run_redis() {
  docker-compose -f ${compose_file} -p ${stack_name} up -d redis
  docker-compose -f ${compose_file} -p ${stack_name} exec redis redis-cli
}

run_update() {
  git fetch
  latest=$(git tag -l | tail -n1)
  git checkout "${latest}"
}

run_clean() {
  rm -f mika

}

run_up() {
  docker-compose -f ${compose_file} -p ${stack_name} build
  docker-compose -f ${compose_file} -p ${stack_name} up --remove-orphans -d
}

run_usage() {
  echo "Usage: $0 {build|clean|cleanall|config|down|exec|install|logs|psql|publish_image|publish_image_tag|redis|run}"
  exit 1
}

run_build_image_tag() {
  tag=$(git describe --abbrev=0)
  docker build -t "$image_name":"$tag" -f docker/Dockerfile .
}

run_build_image() {
  docker build -t "$image_name":latest -f docker/Dockerfile .
}

run_publish_image() {
  docker push ${image_name}:latest
}

run_publish_image_tag() {
  docker push ${image_name}:"$tag"
}

case "$1" in
build)
  shift
  docker-compose -f ${compose_file} -p ${stack_name} build "$@"
  ;;
build_image)
  run_build_image
  ;;
build_image_tag)
  run_build_image_tag
  ;;
clean)
  run_clean
  ;;
cleanall)
  run_clean
  run_clean_config
  ;;
config)
  run_config "$2" "$3"
  ;;
down)
  shift
  docker-compose -f ${compose_file} -p ${stack_name} down "$@"
  ;;
exec)
  shift
  docker-compose -f ${compose_file} -p ${stack_name} exec "$@"
  ;;
install)
  run_install
  ;;
logs)
  shift
  docker-compose -f ${compose_file} -p ${stack_name} logs "$@"
  ;;
prune)
  run_prune
  ;;
publish_image)
  run_publish_image
  ;;
publish_image_tag)
  run_publish_image_tag
  ;;
redis)
  run_redis
  ;;
run)
  shift
  docker-compose -f ${compose_file} -p ${stack_name} run --rm "$@"
  ;;
sql)
  run_sql
  ;;
up)
  shift
  docker-compose -f ${compose_file} -p ${stack_name} up -d "$@"
  docker-compose -f ${compose_file} -p ${stack_name} logs -f
  ;;
update)
  run_update
  ;;
*)
  run_usage
  ;;
esac
