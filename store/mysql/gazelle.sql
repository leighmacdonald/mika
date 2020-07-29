-- USERS
DROP PROCEDURE IF EXISTS user_by_passkey;
CREATE PROCEDURE user_by_passkey(IN in_passkey varchar(40))
BEGIN
    SELECT `ID`         as user_id,
           torrent_pass as passkey,
           can_leech    as download_enabled,
           Enabled      as is_deleted,
           Downloaded   as downloaded,
           Uploaded     as uploaded,
           0            as announces
    FROM users
    WHERE torrent_pass = in_passkey;
end;

DROP PROCEDURE IF EXISTS user_by_id;
CREATE PROCEDURE user_by_id(IN in_user_id int)
BEGIN
    SELECT `ID`         as user_id,
           torrent_pass as passkey,
           can_leech    as download_enabled,
           Enabled      as is_deleted,
           Downloaded   as downloaded,
           Uploaded     as uploaded,
           0            as announces
    FROM users
    WHERE `ID` = in_user_id;
end;
