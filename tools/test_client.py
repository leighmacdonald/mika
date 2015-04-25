import random
import unittest
from tools import client


class ClientTest(unittest.TestCase):
    hash_1 = "40b8b386a0c2f03d492399b9aa7297aefdb84641"
    id_1 = 9999999999999

    def setUp(self):
        self.client = client.TrackerClient("localhost", 34000)

    def test_torrent_get(self):
        # TODO this will pass once torrent client get primed onstartup
        # t1 = self.client.torrent_get(3886)
        # self.assertIsNotNone(t1)
        # self.assertEqual(t1['torrent_id'], 3886)

        t2 = self.client.torrent_get(999999999999999)
        self.assertIsNone(t2)

    def test_torrent_add(self):
        resp = self.client.torrent_add(self.hash_1, self.id_1)
        self.assertTrue(resp)

    def test_torrent_del(self):
        resp = self.client.torrent_del(self.id_1)
        self.assertTrue(resp)

    def test_user_get(self):
        resp = self.client.user_get(94)
        self.assertEqual(resp['user_id'], 94)

    def test_user_update(self):
        a = random.randint(0, 1000)
        resp = self.client.user_update(94, downloaded=a, uploaded=a)
        self.assertTrue(resp)

        user = self.client.user_get(94)
        self.assertEqual(user['downloaded'], a)
        self.assertEqual(user['uploaded'], a)

    def test_user_add(self):
        resp = self.client.user_add(999999, "asdfadfasdfasdfasdfasd")
        self.assertTrue(resp)

    def test_whitelist_add(self):
        self.assertTrue(self.client.whitelist_add("test2", "moo"))

    def test_whitelist_del(self):
        resp = self.client.whitelist_del("test2")
        self.assertTrue(resp)
