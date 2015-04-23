"""
Simple client to interact with the backend tracker instance.
"""
from __future__ import print_function
import requests


class TrackerClient(object):
    def __init__(self, host, port):
        self._host = host
        self._port = port

    def _request(self, path, method='get', payload=None):
        if method == "get":
            resp = requests.get(self._make_url(path))
        elif method == "port":
            resp = requests.post(self._make_url(path), json=payload)
        else:
            raise Exception("no")
        if resp.ok:
            return resp.json()
        return False

    def _make_url(self, path):
        return "http://{}:{}/api{}".format(self._host, self._port, path)

    def torrent_get(self, torrent_id):
        return self._request("/torrent/{}".format(torrent_id))

    def torrent_get_all(self, torrent_ids):
        pass

    def torrent_add(self, info_hash, torrent_id):
        pass

    def torrent_del(self, info_hash):
        pass

    def user_get_active(self, user_id):
        pass

    def user_get_incomplete(self, user_id):
        pass

    def user_get_complete(self, user_id):
        pass

    def user_get_hnr(self, user_id):
        pass

    def user_update(self, user_id, payload):
        pass

    def user_get(self, user_id):
        pass

    def user_update_passkey(self, passkey_old, passkey_new, user_id):
        pass

    def passkey_del(self, passkey):
        pass

    def passkey_add(self, passkey, user_id):
        pass

    def whitelist_update(self, prefix_old, prefix_new, client_name):
        pass

    def whitelist_del(self, prefix):
        pass

    def whitelist_add(self, prefix, client_name):
        pass

    def get_torrent_peer(self, torrent_id, peer_id):
        pass

    def _key(self, *args):
        pass
