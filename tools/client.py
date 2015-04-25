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
        elif method == "post":
            resp = requests.post(self._make_url(path), json=payload)
        elif method == "delete":
            resp = requests.delete(self._make_url(path))
        else:
            raise Exception("no")
        return resp

    def _make_url(self, path):
        return "http://{}:{}/api{}".format(self._host, self._port, path)

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

    def whitelist_update(self, prefix_old, prefix_new, client_name):
        pass

    def whitelist_del(self, prefix):
        resp = self._request("/whitelist/{}".format(prefix), method='delete')
        return resp.ok

    def whitelist_add(self, prefix, client_name):
        resp = self._request("/whitelist", method='post', payload={
            'prefix': prefix,
            'client': client_name
        })
        return resp.ok

    def _key(self, *args):
        pass
