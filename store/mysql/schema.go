package mysql

// TODO force use of utf8mb4_general_ci ?
const schema = `
create schema mika collate latin1_swedish_ci;

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
    created_on datetime not null,
    updated_on datetime not null,
    constraint pk_torrent  primary key (info_hash),
    constraint uq_release_name  unique (release_name)
);


create table user
(
	user_id int unsigned auto_increment
		primary key,
	passkey varchar(20) not null,
	download_enabled tinyint(1) default 1 not null,
	is_deleted tinyint(1) default 0 not null,
	constraint user_passkey_uindex
		unique (passkey)
);

create table peers
(
	peer_id binary(20) not null,
	info_hash binary(20) not null,
	user_id int unsigned not null,
	torrent_id int unsigned not null,
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
	location point not null,
	created_on datetime not null,
	updated_on datetime not null,
	constraint peers_pk primary key (info_hash, peer_id)
	constraint peers_user_user_id_fk
		foreign key (user_id) references user (user_id)
			on update cascade on delete cascade,
);
`
