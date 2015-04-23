import unittest
from tools import client


class ClientTest(unittest.TestCase):
    def setUp(self):
        self.client = client.TrackerClient("localhost", 34000)

    def test_torrent_get(self):
        t = self.client.torrent_get(1)
        self.assertTrue(t.ok)
