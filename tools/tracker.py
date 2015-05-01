"""
Simple client to interact with the backend tracker instance.
"""
from __future__ import print_function
import requests
import redis
from redis.exceptions import ResponseError


class TrackerClient(object):
    def __init__(self, api_uri, redis_host="localhost", redis_port=6379, redis_db=0):
        self._api_uri = api_uri
        self._redis_host = redis_host
        self._redis_port = redis_port
        self._redis_db = redis_db
        self._redis = redis.StrictRedis(host=redis_host, port=int(redis_port), db=int(redis_db))

    def _request(self, path, method='get', payload=None):
        if method == "get":
            resp = requests.get(self._make_url(path))
        elif method == "post":
            resp = requests.post(self._make_url(path), json=payload)
        elif method == "delete":
            resp = requests.delete(self._make_url(path))
        else:
            raise Exception("no")
        return resp

    def _make_url(self, path):
        return "".join([self._api_uri, path])

    def torrent_get(self, torrent_id):
        resp = self._request("/torrent/{}".format(torrent_id))
        if resp.ok:
            return resp.json()
        return None

    def torrent_get_all(self, torrent_ids):
        pass

    def torrent_add(self, info_hash, torrent_id):
        return self._request("/torrent", method='post', payload={
            'info_hash': info_hash,
            'torrent_id': torrent_id
        })

    def torrent_del(self, torrent_id):
        return self._request("/torrent/{}".format(torrent_id), method='delete').ok

    def user_get_active(self, user_id):
        pass

    def user_get_incomplete(self, user_id):
        pass

    def user_get_complete(self, user_id):
        pass

    def user_get_hnr(self, user_id):
        pass

    def user_update(self, user_id, uploaded=None, downloaded=None, passkey=None, can_leech=None):
        user = self.user_get(user_id)
        if not user:
            return False
        updated_data = {
            'uploaded': uploaded if uploaded is not None else user['uploaded'],
            'downloaded': downloaded if downloaded is not None else user['downloaded'],
            'can_leech': can_leech if can_leech is not None else user['can_leech'],
            'passkey': passkey if passkey is not None else user['passkey'],
        }

        resp = self._request("/user/{}".format(user_id), 'post', payload=updated_data)
        return resp.ok

    def user_get(self, user_id):
        resp = self._request("/user/{}".format(user_id))
        return resp.json() if resp.ok else None

    def user_add(self, user_id, passkey):
        resp = self._request("/user", method='post', payload={
            'user_id': user_id,
            'passkey': passkey
        })
        return resp.ok

    def whitelist_del(self, prefix):
        resp = self._request("/whitelist/{}".format(prefix), method='delete')
        return resp.ok

    def whitelist_add(self, prefix, client_name):
        resp = self._request("/whitelist", method='post', payload={
            'prefix': prefix,
            'client': client_name
        })
        return resp.ok

    def torrent_get_all_redis(self):
        torrents = []
        keys = self._redis.keys("t:t:*")
        for key in keys:
            try:
                key = key.decode()
            except UnicodeDecodeError:
                pass
            try:
                if len(key) != 44:
                    continue
                tor = self._redis.hgetall(key)
            except ResponseError as err:
                print(err)
                print(key)
                break
            else:
                torrents.append(tor)
        return torrents

    def users_get_all_redis(self, sort="user_id"):
        users = []
        keys = [k for k in self._redis.keys("t:u:*")]
        for k in keys:
            try:
                data = self._redis.hgetall(k)
                user = {
                    'passkey': data.get(b'passkey', "ERROR: PASSKEY NOT SET"),
                    'user_id': int(data.get(b'user_id', b"-1")),
                    'downloaded': int(data.get(b'downloaded', b"-1")),
                    'uploaded': int(data.get(b'uploaded', b"-1")),
                    'username': data.get(b'username', b"ERROR: NO USER!").decode(),
                    'enabled': data.get(b'enabled', b"0").decode()
                }
                users.append(user)
            except Exception as err:

                print(err)
                print(data)
                print(k)
                break

        users.sort(key=lambda u: u[sort])
        return users

    def cleanup(self, delete=False):
        # Look for active/inactive user suffix keys etc. t:u:$id:*
        keys = [k for k in self._redis.keys("t:u:*")]
        old_keys = []
        for key in keys:
            if len(key.split(b":")) != 3:
                old_keys.append(key)
        for key in old_keys:
            print(key)
            if delete:
                self._redis.delete(key)
        # Look for peer suffix keys t:t:$ih:*
        keys = [k for k in self._redis.keys("t:t:*")]
        for key in keys:
            k = key.split(b":")
            if len(k) != 3:
                old_keys.append(key)
            else:
                # Look for old int based keys
                try:
                    int(k[2])
                except Exception:
                    pass
                else:
                    keys.append(key)
        for key in old_keys:
            print(key)
            if delete:
                self._redis.delete(key)