package mysql

const schema = `
create table torrent
(
    info_hash binary(20) not null,
    release_name varchar(255) not null,
    total_uploaded int unsigned default 0 not null,
    total_downloaded int unsigned default 0 not null,
    total_completed smallint unsigned default 0 not null,
    is_deleted tinyint(1) default 0 not null,
    is_enabled tinyint(1) default 1 not null,
    reason varchar(255) default '' not null,
    multi_up decimal(5,2) default 1.00 not null,
    multi_dn decimal(5,2) default 1.00 not null,
    constraint pk_torrent  primary key (info_hash),
    constraint uq_release_name  unique (release_name)
);


create table users
(
	user_id int unsigned auto_increment	primary key,
	passkey varchar(20) not null,
	download_enabled tinyint(1) default 1 not null,
	is_deleted tinyint(1) default 0 not null,
	downloaded bigint default 0 not null,
	uploaded bigint default 0 not null,
	announces int default 0 not null,
	constraint user_passkey_uindex unique (passkey)
);

create table peers
(
	peer_id binary(20) not null,
	info_hash binary(20) not null,
	user_id int unsigned not null,
	addr_ip int unsigned not null,
	addr_port smallint unsigned not null,
	total_downloaded int unsigned default 0 not null,
	total_uploaded int unsigned default 0 not null,
	total_left int unsigned default 0 not null,
	total_time int unsigned default 0 not null,
	total_announces int unsigned default 0 not null,
	speed_up int unsigned default 0 not null,
	speed_dn int unsigned default 0 not null,
	speed_up_max int unsigned default 0 not null,
	speed_dn_max int unsigned default 0 not null,
	announce_first datetime not null,
	announce_last datetime not null,
	location point not null,
	constraint peers_pk primary key (info_hash, peer_id)
);


create table whitelist
(
	client_prefix varchar(10) not null primary key,
	client_name varchar(20) not null
);
`
