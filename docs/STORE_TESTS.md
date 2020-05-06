# Store Testing Info & Configuration

This document describes how to fully test the backing store systems used. All stores will be tested against the 
same test cases located in store.TestTorrentStore, store.TestUsersStore, store.TestPeersStore. 

Mika mostly relies on integration testing for the stores so each setup will need to have configs specified for your
backend servers being tested against.

The tests currently try to load the following configs by default before starting the tests. These can be overridden
by specifying the environment variable: `MIKA_CONFIG=/path/to/config.yaml` 

- `mika_testing_mysql.yaml` Default MySQL/MariaDB configuration file
- `mika_testing_postgres.yaml` Default postgres configuration file
- `mika_testing_redis.yaml` Default redis configuration file 

If the configs are not found, those tests will be skipped.

As a safeguard, the test will only run when the configured run mode is `general_run_mode: test`.  All tables are
dropped and schemas recreated for each run, so take care when running these.
