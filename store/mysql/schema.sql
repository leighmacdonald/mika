create table role
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

create table user
(
    user_id int unsigned auto_increment
        primary key,
    role_id int unsigned not null,
    is_deleted tinyint(1) default 0 not null,
    downloaded bigint unsigned default 0 not null,
    uploaded bigint unsigned default 0 not null,
    announces int default 0 not null,
    passkey varchar(40) not null,
    download_enabled tinyint(1) default 1 not null,
    created_on datetime default current_timestamp() not null,
    updated_on datetime default current_timestamp() not null,
    constraint user_passkey_uindex
        unique (passkey),
    constraint users_roles_role_id_fk
        foreign key (role_id) references role (role_id)
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
        foreign key (user_id) references user (user_id)
);

create table whitelist
(
    client_prefix char(8) not null
        primary key,
    client_name varchar(20) not null
);

