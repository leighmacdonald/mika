#!/bin/env python2
from __future__ import unicode_literals
import binascii
import json
import argparse
from subprocess import call
import pymysql
import redis
import tracker
from redis.exceptions import ResponseError

config = {}


def bin2hex(bin_info_hash):
    return binascii.b2a_hex(bin_info_hash).lower()


def load_torrents(db_conn, redis_conn, force=False):
    print("> Loading torrents...")
    with db_conn.cursor() as cur:
        cur.execute("SELECT lower(hex(info_hash)), id, release_name  FROM torrents WHERE info_hash <> ''")
        hashes = cur.fetchall()
    for x in hashes:
        info_hash, torrent_id, release_name = x
        torrent_key = "t:t:{}".format(info_hash)
        # init default struct if not already exists
        redis_conn.hmset(torrent_key, {
            'info_hash': info_hash,
            'torrent_id': torrent_id,
            'name': release_name
        })


def load_stats(redis_conn):
    for stat in ['leechers', 'seeders', 'announces', 'scrapes']:
        key = "t:stats:{}".format(stat)
        if not redis_conn.exists(key):
            redis_conn.set(key, 0)


def load_users(db_conn, redis_conn):
    print("> Loading users... ")
    with db_conn.cursor() as cur:
        cur.execute("SELECT passkey, id, username FROM users")
        users = cur.fetchall()
    for user in users:
        redis_conn.set("t:user:{}".format(user[0]), user[1])
        redis_conn.hset("t:u:{}".format(user[1]), 'user_id', user[1])
        redis_conn.hset("t:u:{}".format(user[1]), 'passkey', user[0])
        redis_conn.hset("t:u:{}".format(user[1]), 'username', user[2])


def load_whitelist(db_conn, redis_conn):
    print("> Loading whitelist... ")
    key = "t:whitelist"
    redis_conn.delete(key)
    with db_conn.cursor() as cur:
        cur.execute("SELECT peer_id, vstring AS client FROM xbt_client_whitelist")
        clients = cur.fetchall()
    for peer_id, client in clients:
        redis_conn.hset(key, peer_id, client)


def make_db():
    conn = pymysql.connect(
        user=config.get("SQLUser"),
        passwd=config.get("SQLPass", ""),
        host=config.get("SQLHost", "localhost"),
        port=int(config.get("SQLPort", 3306)),
        db=config.get("SQLDB", ""))
    return conn


def make_redis():
    return redis.StrictRedis(
        host=config.get("redis_host", "localhost"),
        port=int(config.get("redis_port", 6379)),
        db=int(config.get("redis_db", 0)))


def warmup(**args):
    print("> Warming up redis data...")
    db_conn = make_db()
    redis_conn = make_redis()
    load_stats(redis_conn)
    load_whitelist(db_conn, redis_conn)
    load_users(db_conn, redis_conn)
    load_torrents(db_conn, redis_conn)


def gen_key(**args):
    print("> Generating new keys...")
    call("openssl req -x509 -nodes -days 365 -newkey rsa:1024 -keyout key_priv -out key_ca", shell=True)


def get_tracker():
    return tracker.TrackerClient(config['ListenHostAPI'], *config['RedisHost'].split(":"))


def wipe_torrent_stats(**args):
    redis_conn = make_redis()
    for tor in get_tracker().torrent_get_all_redis():
        key = "t:t:{}".format(tor[b'info_hash'].decode())
        redis_conn.hset(key, "downloaded", 0)
        redis_conn.hset(key, "uploaded", 0)


def wipe_user_stats(**args):
    redis_conn = make_redis()
    for user in get_tracker().users_get_all_redis():
        key = "t:u:{}".format(user['user_id'])
        redis_conn.hset(key, "downloaded", 0)
        redis_conn.hset(key, "uploaded", 0)
        redis_conn.hset(key, "corrupt", 0)
        redis_conn.hset(key, "snatches", 0)


def torrents_list(**args):
    for tor in get_tracker().torrent_get_all_redis():
        try:
            print("[{}] S: {} L: {} {}".format(
                tor[b'info_hash'].decode(),
                tor[b'seeders'].decode().rjust(4),
                tor[b'leechers'].decode().rjust(4),
                tor[b'name'].decode(),))
        except Exception:
            print(tor)


def unicode_keys(d):
    return


def users_list(sort="user_id", **args):
    for user in get_tracker().users_get_all_redis(sort):
        print("[{}] ID: {} Up: {} Dn: {} Enabled: {} Name: {}".format(
            user['passkey'].decode(),
            str(user['user_id']).rjust(5),
            str(user.get('uploaded', '?')).rjust(18),
            str(user.get('downloaded', '?')).rjust(18),
            str(user.get('enabled', '?')),
            str(user.get('username', '?'))))
        try:
            pass
        except Exception as err:
            print(err)
            print(user)


def tracker_cleanup(delete=False, **args):
    get_tracker().cleanup(delete=delete)


def parse_args():
    parser = argparse.ArgumentParser(description="Tracker management cli")
    parser.add_argument("-c", "--config", help="Config file path (config.json)", default="./config.json")
    subparsers = parser.add_subparsers(help='sub-command help')

    warmup_cmd = subparsers.add_parser('warmup', help='Warmup the redis cache')
    warmup_cmd.set_defaults(func=warmup)

    genkey_cmd = subparsers.add_parser("genkey", help="Generate a new set of SSL keys")
    genkey_cmd.set_defaults(func=gen_key)

    torrents_cmd = subparsers.add_parser("torrents", help="List torrents stored in redis")
    torrents_cmd.set_defaults(func=torrents_list)

    users_cmd = subparsers.add_parser("users", help="List users stored in redis")
    users_cmd.add_argument("-s", "--sort", help="Sort by: user_id, uploaded, downloaded", default="user_id")
    users_cmd.set_defaults(func=users_list)

    cleanup_cmd = subparsers.add_parser("cleanup", help="Clean up old deprecated keys")
    cleanup_cmd.add_argument("-d", "--delete", action="store_true", help="Deleted the keys from redis")
    cleanup_cmd.set_defaults(func=tracker_cleanup)

    wipetorstats_cmd = subparsers.add_parser("wipetorstats", help="Wipe all transfer/snatch counts from *ALL* torrents")
    wipetorstats_cmd.set_defaults(func=wipe_torrent_stats)

    wipeuserstats_cmd = subparsers.add_parser("wipeuserstats",
                                              help="Wipe all transfer/snatch counts from *ALL* users. DONT BE A HERO.")
    wipeuserstats_cmd.set_defaults(func=wipe_user_stats)


    return vars(parser.parse_args())


if __name__ == "__main__":
    options = parse_args()
    config = json.loads(open(options['config']).read())
    tracker.TrackerClient(config['ListenHostAPI'])
    options['func'](**options)
