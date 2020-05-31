/* create schema mika collate utf8mb4_unicode_ci; */

create table torrent
(
    info_hash        binary(20)                     not null,
    release_name     varchar(255)                   not null,
    total_uploaded   int unsigned      default 0    not null,
    total_downloaded int unsigned      default 0    not null,
    total_completed  smallint unsigned default 0    not null,
    is_deleted       tinyint(1)        default 0    not null,
    is_enabled       tinyint(1)        default 1    not null,
    reason           varchar(255)      default ''   not null,
    multi_up         decimal(5, 2)     default 1.00 not null,
    multi_dn         decimal(5, 2)     default 1.00 not null,
    seeders          int               default 0    not null,
    leechers         int               default 0    not null,
    announces        int               default 0    not null,
    constraint pk_torrent primary key (info_hash),
    constraint uq_release_name unique (release_name)
);

create table users
(
    user_id          int unsigned auto_increment primary key,
    passkey          varchar(20)          not null,
    download_enabled tinyint(1) default 1 not null,
    is_deleted       tinyint(1) default 0 not null,
    downloaded       bigint     default 0 not null,
    uploaded         bigint     default 0 not null,
    announces        int        default 0 not null,
    constraint user_passkey_uindex unique (passkey)
);

create table peers
(
    peer_id          binary(20)             not null,
    info_hash        binary(20)             not null,
    user_id          int unsigned           not null,
    addr_ip          int unsigned           not null,
    addr_port        smallint unsigned      not null,
    total_downloaded int unsigned default 0 not null,
    total_uploaded   int unsigned default 0 not null,
    total_left       int unsigned default 0 not null,
    total_time       int unsigned default 0 not null,
    total_announces  int unsigned default 0 not null,
    speed_up         int unsigned default 0 not null,
    speed_dn         int unsigned default 0 not null,
    speed_up_max     int unsigned default 0 not null,
    speed_dn_max     int unsigned default 0 not null,
    announce_first   datetime               not null,
    announce_last    datetime               not null,
    location         point                  not null,
    constraint peers_pk primary key (info_hash, peer_id)
);


create table whitelist
(
    client_prefix varchar(10) not null primary key,
    client_name   varchar(20) not null
);


-- USERS

CREATE OR REPLACE PROCEDURE user_by_passkey(IN in_passkey varchar(40))
BEGIN
    SELECT user_id,
           passkey,
           download_enabled,
           is_deleted,
           downloaded,
           uploaded,
           announces
    FROM users
    WHERE passkey = in_passkey;
end;

CREATE OR REPLACE PROCEDURE user_by_id(IN in_user_id int)
BEGIN
    SELECT user_id,
           passkey,
           download_enabled,
           is_deleted,
           downloaded,
           uploaded,
           announces
    FROM users
    WHERE user_id = in_user_id;
end;

CREATE OR REPLACE PROCEDURE user_delete(IN in_user_id int)
BEGIN
    DELETE
    FROM users
    WHERE user_id = in_user_id;
end;

CREATE OR REPLACE PROCEDURE user_add(IN in_user_id int,
                                     IN in_passkey varchar(40),
                                     IN in_download_enabled bool,
                                     IN in_is_deleted bool,
                                     IN in_downloaded bigint,
                                     IN in_uploaded bigint,
                                     IN in_announces bigint)
BEGIN
    INSERT INTO users
    (user_id, passkey, download_enabled, is_deleted, downloaded, uploaded, announces)
    VALUES (in_user_id, in_passkey, in_download_enabled, in_is_deleted,
            in_downloaded, in_uploaded, in_announces);
end;

CREATE OR REPLACE PROCEDURE user_update(IN in_user_id int,
                                        IN in_passkey varchar(40),
                                        IN in_download_enabled bool,
                                        IN in_is_deleted bool,
                                        IN in_downloaded bigint,
                                        IN in_uploaded bigint,
                                        IN in_announces bigint,
                                        IN in_old_passkey varchar(40))
BEGIN
    UPDATE users
    SET user_id          = in_user_id,
        passkey          = in_passkey,
        download_enabled = in_download_enabled,
        is_deleted       = in_is_deleted,
        downloaded       = in_downloaded,
        uploaded         = in_uploaded,
        announces        = in_announces
    WHERE passkey = if(in_old_passkey = '', in_passkey, in_old_passkey);
end;

CREATE OR REPLACE PROCEDURE user_update_stats(IN in_passkey varchar(40),
                                              IN in_announces bigint,
                                              IN in_uploaded bigint,
                                              IN in_downloaded bigint)
BEGIN
    UPDATE users
    SET announces  = (announces + in_announces),
        uploaded   = (uploaded + in_uploaded),
        downloaded = (downloaded + in_downloaded)
    WHERE passkey = in_passkey;
END;

-- END USERS

-- TORRENTS

CREATE OR REPLACE PROCEDURE torrent_by_infohash(IN in_info_hash binary(20),
                                                IN in_deleted bool)
BEGIN
    SELECT info_hash,
           release_name,
           total_uploaded,
           total_downloaded,
           total_completed,
           is_deleted,
           is_enabled,
           reason,
           multi_up,
           multi_dn,
           seeders,
           leechers,
           announces
    FROM torrent
    WHERE info_hash = in_info_hash
      AND is_deleted = in_deleted;
end;

CREATE OR REPLACE PROCEDURE torrent_delete(IN in_info_hash binary(20))
BEGIN
    DELETE FROM torrent WHERE info_hash = in_info_hash;
end;

CREATE OR REPLACE PROCEDURE torrent_disable(IN in_info_hash binary(20))
BEGIN
    UPDATE torrent SET is_deleted = true WHERE info_hash = in_info_hash;
end;

CREATE OR REPLACE PROCEDURE torrent_add(IN in_info_hash binary(20),
                                        IN in_release_name varchar(255))
BEGIN
    INSERT INTO torrent (info_hash, release_name)
    VALUES (in_info_hash, in_release_name);
end;

CREATE OR REPLACE PROCEDURE torrent_update_stats(IN in_info_hash binary(20),
                                                 IN in_total_downloaded bigint,
                                                 IN in_total_uploaded bigint,
                                                 IN in_announces bigint,
                                                 IN in_total_completed int,
                                                 IN in_seeders int,
                                                 IN in_leechers int)
BEGIN
    UPDATE
        torrent
    SET total_downloaded = (total_downloaded + in_total_downloaded),
        total_uploaded   = (total_uploaded + in_total_uploaded),
        announces        = (announces + in_announces),
        total_completed  = (total_completed + in_total_completed),
        seeders          = in_seeders,
        leechers         = in_leechers
    WHERE info_hash = in_info_hash;
END;


CREATE OR REPLACE PROCEDURE whitelist_all()
BEGIN
    SELECT * FROM whitelist;
end;

CREATE OR REPLACE PROCEDURE whitelist_add(IN in_client_prefix varchar(255),
                                          IN in_client_name varchar(255))
BEGIN
    INSERT INTO whitelist (client_prefix, client_name)
    VALUES (in_client_prefix, in_client_name);
end;

CREATE OR REPLACE PROCEDURE whitelist_delete_by_prefix(IN in_client_prefix varchar(255))
BEGIN
    DELETE FROM whitelist WHERE client_prefix = in_client_prefix;
end;

-- END TORRENTS

-- PEERS
CREATE OR REPLACE PROCEDURE peer_update_stats(IN in_info_hash binary(20),
                                              IN in_peer_id binary(20),
                                              IN in_total_downloaded bigint,
                                              IN in_total_uploaded bigint,
                                              IN in_total_announces bigint,
                                              IN in_announce_last datetime)
BEGIN
    UPDATE
        peers
    SET total_announces  = (total_announces + in_total_announces),
        total_downloaded = (total_downloaded + in_total_downloaded),
        total_uploaded   = (total_uploaded + in_total_uploaded),
        announce_last    = in_announce_last
    WHERE info_hash = in_info_hash
      AND peer_id = in_peer_id;
END;

CREATE OR REPLACE PROCEDURE peer_reap(IN in_expiry_time datetime)
BEGIN
    DELETE FROM peers WHERE announce_last <= in_expiry_time;
end;
-- TODO Add left field
CREATE OR REPLACE PROCEDURE peer_add(IN in_info_hash binary(20),
                                     IN in_peer_id binary(20),
                                     IN in_user_id int,
                                     IN in_addr_ip varchar(255),
                                     IN in_addr_port int,
                                     IN in_location varchar(255),
                                     IN in_announce_first datetime,
                                     IN in_announce_last datetime)
BEGIN
    INSERT INTO peers
    (peer_id, info_hash, user_id, addr_ip, addr_port, location, announce_first, announce_last)
    VALUES (in_peer_id, in_info_hash, in_user_id,
            INET_ATON(in_addr_ip), in_addr_port, ST_PointFromText(in_location),
            in_announce_first, in_announce_last);
end;

CREATE OR REPLACE PROCEDURE peer_delete(IN in_info_hash binary(20),
                                        IN in_peer_id binary(20))
BEGIN
    DELETE FROM peers WHERE info_hash = in_info_hash AND peer_id = in_peer_id;
end;

CREATE OR REPLACE PROCEDURE peer_get(IN in_info_hash binary(20), IN in_peer_id binary(20))
BEGIN
    SELECT peer_id,
           info_hash,
           user_id,
           INET_NTOA(addr_ip)  as addr_ip,
           addr_port,
           total_downloaded,
           total_uploaded,
           total_left,
           total_time,
           total_announces,
           speed_up,
           speed_dn,
           speed_up_max,
           speed_dn_max,
           ST_AsText(location) as location,
           announce_last,
           announce_first
    FROM peers
    WHERE info_hash = in_info_hash
      AND peer_id = in_peer_id;
end;

CREATE OR REPLACE PROCEDURE peer_get_n(IN in_info_hash binary(20), IN in_limit int)
BEGIN
    SELECT peer_id,
           info_hash,
           user_id,
           INET_NTOA(addr_ip)  as addr_ip,
           addr_port,
           total_downloaded,
           total_uploaded,
           total_left,
           total_time,
           total_announces,
           speed_up,
           speed_dn,
           speed_up_max,
           speed_dn_max,
           ST_AsText(location) as location,
           announce_last,
           announce_first
    FROM peers
    WHERE info_hash = in_info_hash
    LIMIT in_limit;
end;
-- END PEERS