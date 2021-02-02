-- MariaDB dump 10.18  Distrib 10.5.8-MariaDB, for Win64 (AMD64)
--
-- Host: localhost    Database: mika
-- ------------------------------------------------------
-- Server version	10.5.8-MariaDB-1:10.5.8+maria~focal
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `role`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `role` (
  `role_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `remote_id` bigint(20) unsigned NOT NULL DEFAULT 0,
  `role_name` varchar(64) NOT NULL,
  `priority` int(11) NOT NULL,
  `multi_up` decimal(5,2) NOT NULL DEFAULT -1.00,
  `multi_down` decimal(5,2) NOT NULL DEFAULT -1.00,
  `download_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `upload_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `created_on` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_on` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`role_id`),
  UNIQUE KEY `role_priority_uindex` (`priority`),
  UNIQUE KEY `roles_role_name_uindex` (`role_name`)
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `torrent`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `torrent` (
  `info_hash` binary(20) NOT NULL,
  `total_uploaded` bigint(20) unsigned NOT NULL DEFAULT 0,
  `total_downloaded` bigint(20) unsigned NOT NULL DEFAULT 0,
  `total_completed` smallint(5) unsigned NOT NULL DEFAULT 0,
  `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
  `is_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `reason` varchar(255) NOT NULL DEFAULT '',
  `multi_up` decimal(5,2) NOT NULL DEFAULT 1.00,
  `multi_dn` decimal(5,2) NOT NULL DEFAULT 1.00,
  `seeders` int(11) NOT NULL DEFAULT 0,
  `leechers` int(11) NOT NULL DEFAULT 0,
  `announces` int(11) NOT NULL DEFAULT 0,
  `title` varchar(255) NOT NULL,
  `created_on` datetime NOT NULL DEFAULT current_timestamp(),
  `updated_on` datetime NOT NULL DEFAULT current_timestamp(),
  `total_downloaded_real` bigint(20) unsigned NOT NULL DEFAULT 0,
  `total_uploaded_real` bigint(20) unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (`info_hash`),
  CONSTRAINT `chk_ih_len` CHECK (octet_length(`info_hash`) = 20)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `user` (
  `user_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `role_id` int(10) unsigned NOT NULL,
  `remote_id` bigint(20) unsigned NOT NULL DEFAULT 0,
  `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
  `downloaded` bigint(20) unsigned NOT NULL DEFAULT 0,
  `uploaded` bigint(20) unsigned NOT NULL DEFAULT 0,
  `announces` int(11) NOT NULL DEFAULT 0,
  `passkey` varchar(40) NOT NULL,
  `download_enabled` tinyint(1) NOT NULL DEFAULT 1,
  `created_on` datetime NOT NULL DEFAULT current_timestamp(),
  `updated_on` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`user_id`),
  UNIQUE KEY `user_passkey_uindex` (`passkey`),
  KEY `users_roles_role_id_fk` (`role_id`),
  CONSTRAINT `users_roles_role_id_fk` FOREIGN KEY (`role_id`) REFERENCES `role` (`role_id`),
  CONSTRAINT `pk_exists` CHECK (octet_length(`passkey`) >= 20 and octet_length(`passkey`) <= 40)
) ENGINE=InnoDB AUTO_INCREMENT=1396 DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user_multi`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `user_multi` (
  `user_id` int(10) unsigned NOT NULL,
  `info_hash` binary(20) NOT NULL,
  `multi_up` decimal(5,2) NOT NULL DEFAULT -1.00,
  `multi_down` decimal(5,2) NOT NULL DEFAULT -1.00,
  `created_on` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_on` timestamp NOT NULL DEFAULT current_timestamp(),
  `valid_until` timestamp NULL DEFAULT NULL,
  UNIQUE KEY `user_multi_uindex` (`user_id`,`info_hash`),
  CONSTRAINT `user_multi_user_id_fk` FOREIGN KEY (`user_id`) REFERENCES `user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `whitelist`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `whitelist` (
  `client_prefix` char(8) NOT NULL,
  `client_name` varchar(20) NOT NULL,
  PRIMARY KEY (`client_prefix`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed
