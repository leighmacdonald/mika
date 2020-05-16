# SQL Store (Postgres/MySQL/MariaDB)

This storage interface provides a standard SQL interface for querying and updating peer & torrent
data. All should function largely the same so they are rolled into one document. Notable differences
between instances will be shown as needed.

## Requirements

### MySQL/MariaDB

- MySQL 8.0+ / MariaDB 10.5+ (maybe 10.4?) due to the POINT column type used

Both are equally supported currently, but you should use MariaDB if you have the option (Larry
Ellison is a not so cool dude).

For proper time conversions this must be set in any configured mysql stores.
    
    store_*_properties: parseTime=true

### PostgreSQL

- PostgreSQL 10+
- PostGIS Extension for spatial column types (POINT) and queries
