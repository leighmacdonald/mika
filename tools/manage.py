#!/bin/env python2
import binascii
import json
import argparse
import pymysql
import redis

config = {}


def bin2hex(bin_info_hash):
    return binascii.b2a_hex(bin_info_hash).lower()


def load_torrents(db_conn, redis_conn, force=False):
    print("> Loading torrents...")
    with db_conn.cursor() as cur:
        cur.execute("SELECT info_hash, id  FROM torrents where info_hash <> ''")
        hashes = cur.fetchall()
    for info_hash, torrent_id in hashes:
        torrent_key = "t:t:{}".format(torrent_id)
        # init default struct if not already exists
        if not redis_conn.exists(torrent_key) or force:
            redis_conn.hmset(torrent_key, {
                'announces': 0,
                'leechers': 0,
                'seeders': 0,
                'snatches': 0,
                'uploaded': 0,
                'downloaded': 0
            })
        # Set info_hash -> torrent_id mapping
        redis_conn.set("t:info_hash:{}".format(bin2hex(info_hash).decode("utf8")), torrent_id)


def load_stats(redis_conn):
    for stat in ['leechers', 'seeders', 'announces', 'scrapes']:
        key = "t:stats:{}".format(stat)
        if not redis_conn.exists(key):
            redis_conn.set(key, 0)


def load_users(db_conn, redis_conn):
    print("> Loading users... ")
    with db_conn.cursor() as cur:
        cur.execute("SELECT passkey, id, download, upload FROM users")
        users = cur.fetchall()
    for user in users:
        redis_conn.set("t:user:{}".format(user[0]), user[1])
        redis_conn.hmset("t:u:{}".format(user[1]), {'downloaded': user[2], 'uploaded': user[3]})


def load_whitelist(db_conn, redis_conn):
    print("> Loading whitelist... ")
    key = "t:whitelist"
    redis_conn.delete(key)
    with db_conn.cursor() as cur:
        cur.execute("SELECT peer_id, vstring as client FROM xbt_client_whitelist")
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


def parse_args():
    parser = argparse.ArgumentParser(description="Tracker management cli")
    parser.add_argument("-c", "--config", help="Config file path (config.json)", default="./config.json")
    subparsers = parser.add_subparsers(help='sub-command help')

    warmup_cmd = subparsers.add_parser('warmup', help='Warmup the redis cache')
    warmup_cmd.set_defaults(func=warmup)

    return vars(parser.parse_args())


if __name__ == "__main__":
    options = parse_args()
    config = json.loads(open(options['config']).read())
    options['func'](**options)