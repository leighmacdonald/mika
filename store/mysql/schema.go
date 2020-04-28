package mysql

// TODO force use of utf8mb4_general_ci ?
const schema = `
create schema mika collate latin1_swedish_ci;

create table torrent
(
	torrent_id int unsigned auto_increment
		primary key,
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
	constraint torrent_info_hash_uindex
		unique (info_hash),
	constraint torrent_release_name_uindex
		unique (release_name)
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
	user_peer_id int unsigned auto_increment
		primary key,
	peer_id binary(20) not null,
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
	constraint peer_id_torrent_id_uindex
		unique (peer_id, torrent_id),
	constraint peers_user_user_id_fk
		foreign key (user_id) references user (user_id)
			on update cascade on delete cascade,
	constraint torrent_id_fk
		foreign key (torrent_id) references torrent (torrent_id)
			on update cascade on delete cascade
);
`
