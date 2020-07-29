#1/bin/bash
stack_name="mika"
compose_file="docker-compose.yml"

run_mysql () {
  docker-compose -f ${compose_file} -p ${stack_name} run --rm mariadb mysql -uunit3d -punit3d -D unit3d
}

run_redis() {
  docker-compose -f ${compose_file} -p ${stack_name} run --rm redis redis-cli
}

run_up() {
  docker-compose -f ${compose_file} -p ${stack_name} up --remove-orphans -d
}

run_usage() {
  echo "Usage: $0 {artisan|build|clean|cleanall|config|down|exec|install|logs|prune|redis|run|sql}"
  exit 1
}

run_docker_build() {
  docker build -t leighmacdonald/mika:latest .
}

run_build_release() {
  docker build -t "leighmacdonald/mika:$(git describe --abbrev=0)" .
}

run_docker_publish_latest() {
  docker push leighmacdonald/mika:latest
}
run_docker_publish_release() {
  docker push "leighmacdonald/mika:$(git describe --abbrev=0)"
}

case "$1" in
  build)
    run_docker_build
    ;;
  build_release)
    run_build_release
    ;;
  docker_publish_latest)
    run_docker_publish_latest
    ;;
  docker_publish_release)
    run_docker_publish_release
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
  redis)
    run_redis
    ;;
  run)
    docker-compose run --rm mika
    ;;
  sql)
    run_sql
    ;;
  up)
    shift
    docker-compose -f ${compose_file} -p ${stack_name} up -d "$@"
    docker-compose -f ${compose_file} -p ${stack_name} logs -f "$@"
    ;;
  *)
    run_usage
    ;;
esac