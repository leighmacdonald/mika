-- USERS
CREATE OR REPLACE PROCEDURE user_by_passkey(
    IN in_passkey varchar(32)
)
BEGIN
    SELECT id                             as user_id,
           passkey                        as passkey,
           can_download                   as download_enabled,
           if(active = true, false, true) as is_deleted,
           downloaded                     as downloaded,
           uploaded                       as uploaded,
           0                              as announces
    FROM users
    WHERE passkey = in_passkey collate utf8mb4_unicode_ci;
end;

CREATE OR REPLACE PROCEDURE user_by_id(
    IN in_user_id int
)
BEGIN
    SELECT `id`                           as user_id,
           passkey                        as passkey,
           can_download                   as download_enabled,
           if(active = true, false, true) as is_deleted,
           downloaded                     as downloaded,
           uploaded                       as uploaded,
           0                              as announces
    FROM users
    WHERE `id` = in_user_id;
end;

CREATE OR REPLACE PROCEDURE user_delete(IN in_user_id int)
BEGIN
    DELETE
    FROM users
    WHERE `id` = in_user_id;
end;


CREATE OR REPLACE PROCEDURE user_add(IN in_user_id int,
                                     IN in_passkey varchar(40),
                                     IN in_download_enabled bool,
                                     IN in_is_deleted bool,
                                     IN in_downloaded bigint unsigned,
                                     IN in_uploaded bigint unsigned,
                                     IN in_announces bigint)
BEGIN
    SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = 'not compatible';
end;

CREATE OR REPLACE PROCEDURE user_update(IN in_user_id int,
                                        IN in_passkey varchar(40),
                                        IN in_download_enabled bool,
                                        IN in_is_deleted bool,
                                        IN in_downloaded bigint unsigned,
                                        IN in_uploaded bigint unsigned,
                                        IN in_announces bigint,
                                        IN in_old_passkey varchar(40))
BEGIN
    UPDATE users
    SET `id`         = in_user_id,
        passkey      = in_passkey,
        can_download = in_download_enabled,
        active       = NOT in_is_deleted,
        downloaded   = in_downloaded,
        uploaded     = in_uploaded
    WHERE passkey = if(in_old_passkey = '', in_passkey, in_old_passkey) collate utf8mb4_unicode_ci;
end;

CREATE OR REPLACE PROCEDURE user_update_stats(IN in_passkey varchar(40),
                                              IN in_announces bigint,
                                              IN in_uploaded bigint unsigned,
                                              IN in_downloaded bigint unsigned)
BEGIN
    UPDATE users
    SET uploaded   = (uploaded + in_uploaded),
        downloaded = (downloaded + in_downloaded)
    WHERE passkey = in_passkey collate utf8mb4_unicode_ci;
END;

-- END USERS

-- TORRENTS
-- TODO sub queries to get total uploaded/downloaded?
CREATE OR REPLACE PROCEDURE torrent_by_infohash(IN in_info_hash binary(20),
                                                IN in_deleted bool)
BEGIN
    SELECT UNHEX(info_hash)          as info_hash,
           0                         as total_uploaded,
           0                         as total_downloaded,
           times_completed           as total_completed,
           false                     as is_deleted,
           true                      as is_enabled,
           'Invalid torrent'         as reason,
           if(doubleup = true, 2, 1) as multi_up,
           if(free = true, 0, 1)     as multi_dn,
           seeders                   as seeders,
           leechers                  as leechers,
           0                         as announces
    FROM torrents
    WHERE info_hash = HEX(in_info_hash);
end;

CREATE OR REPLACE PROCEDURE torrent_delete(IN in_info_hash binary(20))
BEGIN
    DELETE FROM torrents WHERE info_hash = HEX(in_info_hash);
end;

CREATE OR REPLACE PROCEDURE torrent_disable(IN in_info_hash binary(20))
BEGIN
    DELETE FROM torrents WHERE info_hash = HEX(in_info_hash);
end;

CREATE OR REPLACE PROCEDURE torrent_add(IN in_info_hash binary(20))
BEGIN
    SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = 'not compatible';
end;

CREATE OR REPLACE PROCEDURE torrent_update_stats(IN in_info_hash binary(20),
                                                 IN in_total_downloaded bigint unsigned,
                                                 IN in_total_uploaded bigint unsigned,
                                                 IN in_announces bigint,
                                                 IN in_total_completed int,
                                                 IN in_seeders int,
                                                 IN in_leechers int)
BEGIN
    UPDATE
        torrents
    SET times_completed = (times_completed + in_total_completed),
        seeders         = in_seeders,
        leechers        = in_leechers
    WHERE info_hash = HEX(in_info_hash);
END;

-- END TORRENTS

-- PEERS
CREATE OR REPLACE PROCEDURE peer_update_stats(IN in_info_hash binary(20),
                                              IN in_peer_id binary(20),
                                              IN in_total_downloaded bigint unsigned,
                                              IN in_total_uploaded bigint unsigned,
                                              IN in_total_announces bigint,
                                              IN in_announce_last datetime,
                                              IN in_speed_dn bigint,
                                              IN in_speed_up bigint,
                                              IN in_speed_dn_max bigint,
                                              IN in_speed_up_max bigint)
BEGIN
    UPDATE
        peers
    SET downloaded   = (downloaded + in_total_downloaded),
        uploaded     = (uploaded + in_total_uploaded),
        updated_at   = in_announce_last,
        speed_up     = in_speed_up,
        speed_dn     = in_speed_dn,
        speed_up_max = GREATEST(speed_up_max, in_speed_up_max),
        speed_dn_max = GREATEST(speed_dn_max, in_speed_dn_max)
    WHERE info_hash = HEX(in_info_hash)
      AND peer_id = HEX(in_peer_id);
END;

CREATE OR REPLACE PROCEDURE peer_reap(IN in_expiry_time datetime)
BEGIN
    DELETE FROM peers WHERE updated_at <= in_expiry_time;
end;

CREATE OR REPLACE PROCEDURE peer_add(IN in_info_hash binary(20),
                                     IN in_peer_id binary(20),
                                     IN in_user_id int,
                                     IN in_ipv6 boolean,
                                     IN in_addr_ip varchar(255),
                                     IN in_addr_port int,
                                     IN in_location varchar(255),
                                     IN in_announce_first datetime,
                                     IN in_announce_last datetime,
                                     IN in_downloaded bigint unsigned,
                                     IN in_uploaded bigint unsigned,
                                     IN in_left bigint unsigned,
                                     IN in_client varchar(255),
                                     IN in_country_code char(2),
                                     IN in_asn varchar(10),
                                     IN in_as_name varchar(255))
BEGIN
    SELECT @int_torrent_id := `id` FROM torrents WHERE info_hash = HEX(in_info_hash);
    INSERT INTO peers
    (peer_id, md5_peer_id, info_hash, user_id, ipv6, ip, port, location, created_at, updated_at, torrent_id,
     downloaded, uploaded, `left`, agent, seeder, country_code, asn, as_name)
    VALUES (HEX(in_peer_id),
            md5(HEX(in_peer_id)),
            HEX(in_info_hash),
            in_user_id,
            in_ipv6,
            in_addr_ip,
            in_addr_port,
            ST_PointFromText(in_location),
            in_announce_first,
            in_announce_last,
            @int_torrent_id,
            in_downloaded,
            in_uploaded,
            in_left,
            in_client,
            if(in_left = 0, 1, 0),
            in_country_code,
            in_asn,
            in_as_name);
end;

CREATE OR REPLACE PROCEDURE peer_delete(IN in_info_hash binary(20),
                                        IN in_peer_id binary(20))
BEGIN
    DELETE FROM peers WHERE info_hash = HEX(in_info_hash) AND peer_id = HEX(in_peer_id);
end;

CREATE OR REPLACE PROCEDURE peer_get(IN in_info_hash BINARY(20),
                                     IN in_peer_id BINARY(20))
BEGIN
    SELECT UNHEX(peer_id)      as peer_id,
           UNHEX(info_hash)    as info_hash,
           user_id             as user_id,
           ipv6                as ipv6,
           ip                  as addr_ip,
           port                as addr_port,
           downloaded          as total_downloaded,
           uploaded            as total_uploaded,
           `left`              as total_left,
           0                   as total_time,
           total_announces     as total_announces,
           speed_up            as speed_up,
           speed_dn            as speed_dn,
           speed_up_max        as speed_up_max,
           speed_dn_max        as speed_dn_max,
           ST_AsText(location) as location,
           updated_at          as announce_last,
           created_at          as announce_first,
           country_code        as country_code,
           asn                 as asn,
           as_name             as as_name
    FROM peers
    WHERE info_hash = HEX(in_info_hash)
      and peer_id = HEX(in_peer_id);
END;

CREATE OR REPLACE PROCEDURE peer_get_n(IN in_info_hash binary(20), IN in_limit int)
BEGIN
    SELECT UNHEX(peer_id)      as peer_id,
           UNHEX(info_hash)    as info_hash,
           user_id             as user_id,
           ipv6                as ipv6,
           ip                  as addr_ip,
           port                as addr_port,
           downloaded          as total_downloaded,
           uploaded            as total_uploaded,
           `left`              as total_left,
           0                   as total_time,
           total_announces     as total_announces,
           speed_up            as speed_up,
           speed_dn            as speed_dn,
           speed_up_max        as speed_up_max,
           speed_dn_max        as speed_dn_max,
           ST_AsText(location) as location,
           updated_at          as announce_last,
           created_at          as announce_first,
           country_code        as country_code,
           asn                 as asn,
           as_name             as as_name
    FROM peers
    WHERE info_hash = HEX(in_info_hash)
    LIMIT in_limit;
end;

-- END PEERS