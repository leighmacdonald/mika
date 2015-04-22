"""
Simple client to interact with the backend tracker instance.
"""


class TrackerClient(object):

    def __init__(self):
        pass

    def torrent_get(self, torrent_id):
        pass

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

