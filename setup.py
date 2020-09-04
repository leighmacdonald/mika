#!python3
import argparse
from os import system, popen, remove
from os.path import exists
from sys import argv
from time import sleep

STACK_NAME = "mika"
COMPOSE_FILE = "docker-compose.yml"
IMAGE_NAME = "ghcr.io/leighmacdonald/mika"
CONFIG_PATH = "mika.yaml"
DOCKERFILE = "docker/Dockerfile"
BINARY_PATH = "mika"


def config():
    if exists(CONFIG_PATH):
        print("{} exists already, please delete or rename it "
              "if you wish to recreate your config".format(CONFIG_PATH))
        exit(1)
    torrentStore = ""
    peerStore = ""
    userStore = ""
    print("""Mika supports 3 different storage setups which can be configured to best suite your specific needs.
The following questions will ask you which you want to use.

Most users starting with a fresh installation should use either the default postgres or mariadb options
for all but the peer store.

For more information on how this works please read docs/STORE_*.md files for more details.
""")
    """
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
  """


def install():
    config()
    build()
    up_d("db")
    print("Please wait while the database initializes, could take over a minute on slower hardware")
    ret_val = -1
    fmt = """docker-compose -f {} -p {} exec db mysql -uunit3d -punit3d -D unit3d -s -e "SELECT 1" >/dev/null 2>&1"""
    while ret_val != 0:
        sleep(2)
        ret_val = system(fmt.format(COMPOSE_FILE, STACK_NAME))
        print(".", end="")
    up_d()
    print("""services started. Run "{} logs" to check the console for any problems""".format(argv[0]))


def logs():
    system("docker-compose -f {} -p {} logs -f".format(COMPOSE_FILE, STACK_NAME))


def build():
    return system("docker-compose -f {} -p {} build".format(COMPOSE_FILE, STACK_NAME))


def build_image():
    return system("docker build -t {}:latest -f {} .".format(IMAGE_NAME, DOCKERFILE))


def up_d(service=""):
    return system("docker-compose -f {} -p {} up -d {}".format(COMPOSE_FILE, STACK_NAME, service))


def build_image_tag():
    return system("docker build -t {}:{} -f {} .".format(IMAGE_NAME, current_tag(), DOCKERFILE))


def publish_image():
    return system("docker push {}:latest".format(IMAGE_NAME))


def publish_image_tag():
    return system("docker push {}:{}".format(IMAGE_NAME, current_tag()))


def up():
    system("docker-compose -f {} -p {} build".format(COMPOSE_FILE, STACK_NAME))
    system("docker-compose -f {} -p {} up --remove-orphans".format(COMPOSE_FILE, STACK_NAME))


def reset():
    remove(CONFIG_PATH)
    remove(BINARY_PATH)


def latest_tag():
    system("git fetch")
    tags = popen("git describe --abbrev=0").read().split("\n")
    if not tags:
        raise Exception("No tags found")
    return tags[0]


def update():
    system("git checkout {}".format(latest_tag()))


def sql_shell():
    system("docker-compose -f {} -p {} up -d db".format(COMPOSE_FILE, STACK_NAME))
    system("docker-compose -f {} -p {} exec psql".format(COMPOSE_FILE, STACK_NAME))


def redis_shell():
    system("docker-compose -f {} -p {} up -d redis".format(COMPOSE_FILE, STACK_NAME))
    system("docker-compose -f {} -p {} exec redis redis-cli".format(COMPOSE_FILE, STACK_NAME))


def current_tag():
    return popen("git describe --abbrev=0").read().rstrip()


def parse_args():
    parser = argparse.ArgumentParser(description="mika tracker setup utility")
    parser.add_argument('command', metavar='C', type=str, nargs='+',
                        help='Command to execute')
    return parser.parse_args()


commands = {
    'build': build,
    'build_image': build_image,
    'build_image_tag': build_image_tag,
    'config': config,
    'install': install,
    'logs': logs,
    'publish_image': publish_image,
    'publish_image_tag': publish_image_tag,
    'redis_shell': redis_shell,
    'reset': reset,
    'sql_shell': sql_shell,
    'up': up,
    'up_d': up_d,
    'update': update,
}

if __name__ == "__main__":
    args = parse_args()
    if not args.command:
        print("No command specified")
        exit(1)

    command_arg = args.command[0].lower()
    if command_arg not in commands:
        print("Command doesnt exist: {}".format(command_arg))
        exit(1)
    exit(commands[command_arg]())
