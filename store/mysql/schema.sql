create table peers
(
    peer_id binary(20) not null,
    info_hash binary(20) not null,
    user_id int unsigned not null,
    ipv6 tinyint(1) not null,
    addr_ip int unsigned not null,
    addr_port smallint unsigned not null,
    total_downloaded bigint unsigned default 0 not null,
    total_uploaded bigint unsigned default 0 not null,
    total_left bigint unsigned default 0 not null,
    total_time int unsigned default 0 not null,
    total_announces int unsigned default 0 not null,
    speed_up int unsigned default 0 not null,
    speed_dn int unsigned default 0 not null,
    speed_up_max int unsigned default 0 not null,
    speed_dn_max int unsigned default 0 not null,
    announce_first datetime not null,
    announce_last datetime not null,
    announce_prev datetime not null,
    location point not null,
    country_code char(2) default '' not null,
    asn int unsigned default 0 not null,
    as_name varchar(255) default '' not null,
    agent varchar(100) not null,
    crypto_level int unsigned default 0 not null,
    primary key (info_hash, peer_id)
);

create table roles
(
    role_id int unsigned auto_increment
        primary key,
    role_name varchar(64) not null,
    priority int not null,
    multi_up decimal(5,2) default -1.00 not null,
    multi_down decimal(5,2) default -1.00 not null,
    download_enabled tinyint(1) default 1 not null,
    upload_enabled tinyint(1) default 1 not null,
    created_on timestamp default current_timestamp() not null,
    updated_on timestamp default current_timestamp() not null,
    constraint role_priority_uindex
        unique (priority),
    constraint roles_role_name_uindex
        unique (role_name)
);

create table torrent
(
    info_hash binary(20) not null
        primary key,
    total_uploaded bigint unsigned default 0 not null,
    total_downloaded bigint unsigned default 0 not null,
    total_completed smallint unsigned default 0 not null,
    is_deleted tinyint(1) default 0 not null,
    is_enabled tinyint(1) default 1 not null,
    reason varchar(255) default '' not null,
    multi_up decimal(5,2) default 1.00 not null,
    multi_dn decimal(5,2) default 1.00 not null,
    seeders int default 0 not null,
    leechers int default 0 not null,
    announces int default 0 not null
);

create table users
(
    user_id int unsigned auto_increment
        primary key,
    passkey varchar(40) not null,
    download_enabled tinyint(1) default 1 not null,
    is_deleted tinyint(1) default 0 not null,
    downloaded bigint unsigned default 0 not null,
    uploaded bigint unsigned default 0 not null,
    announces int default 0 not null,
    constraint user_passkey_uindex
        unique (passkey)
);

create table user_multi
(
    user_id int unsigned not null,
    info_hash binary(20) not null,
    multi_up decimal(5,2) default -1.00 not null,
    multi_down decimal(5,2) default -1.00 not null,
    created_on timestamp default current_timestamp() not null,
    updated_on timestamp default current_timestamp() not null,
    valid_until timestamp null,
    constraint user_multi_uindex
        unique (user_id, info_hash),
    constraint user_multi_user_id_fk
        foreign key (user_id) references users (user_id)
);

create table user_roles
(
    user_id int unsigned not null,
    role_id int unsigned not null,
    created_on timestamp default current_timestamp() not null on update current_timestamp(),
    constraint user_roles_uindex
        unique (user_id, role_id),
    constraint user_roles_roles_role_id_fk
        foreign key (role_id) references roles (role_id),
    constraint user_roles_users_user_id_fk
        foreign key (user_id) references users (user_id)
            on delete cascade
);

create table whitelist
(
    client_prefix char(8) not null
        primary key,
    client_name varchar(20) not null
);

