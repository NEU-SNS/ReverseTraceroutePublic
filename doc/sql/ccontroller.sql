-- MySQL dump 10.13  Distrib 5.6.30-76.3, for debian-linux-gnu (x86_64)
--
-- Host: localhost    Database: ccontroller
-- ------------------------------------------------------
-- Server version	5.6.30-76.3-log

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `ping_batch`
--

DROP TABLE IF EXISTS `ping_batch`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ping_batch` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int(10) unsigned NOT NULL,
  `created` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=69 DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `ping_batch_ping`
--

DROP TABLE IF EXISTS `ping_batch_ping`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ping_batch_ping` (
  `batch_id` int(10) unsigned NOT NULL,
  `ping_id` int(10) unsigned NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `ping_responses`
--

DROP TABLE IF EXISTS `ping_responses`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ping_responses` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `ping_id` bigint(20) NOT NULL,
  `from` int(10) unsigned NOT NULL,
  `seq` int(10) unsigned NOT NULL,
  `reply_size` int(10) unsigned NOT NULL,
  `reply_ttl` int(10) unsigned NOT NULL,
  `reply_proto` varchar(45) COLLATE utf8_unicode_ci NOT NULL,
  `rtt` int(10) unsigned NOT NULL,
  `probe_ipid` int(10) unsigned NOT NULL,
  `reply_ipid` int(10) unsigned NOT NULL,
  `icmp_type` int(10) unsigned NOT NULL,
  `icmp_code` int(10) unsigned NOT NULL,
  `tx` bigint(20) NOT NULL,
  `rx` bigint(20) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_ping_responses_1_idx` (`ping_id`),
  CONSTRAINT `pr_ping` FOREIGN KEY (`ping_id`) REFERENCES `pings` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB AUTO_INCREMENT=67779045 DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `ping_stats`
--

DROP TABLE IF EXISTS `ping_stats`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ping_stats` (
  `ping_id` bigint(20) NOT NULL,
  `loss` float NOT NULL,
  `min` float NOT NULL,
  `max` float NOT NULL,
  `avg` float NOT NULL,
  `std_dev` float NOT NULL,
  KEY `fk_ping_stats_1_idx` (`ping_id`),
  CONSTRAINT `ps_ping` FOREIGN KEY (`ping_id`) REFERENCES `pings` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `pings`
--

DROP TABLE IF EXISTS `pings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `pings` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `src` int(10) unsigned NOT NULL,
  `dst` int(10) unsigned NOT NULL,
  `start` bigint(20) DEFAULT NULL,
  `ping_sent` int(10) unsigned DEFAULT NULL,
  `probe_size` int(10) unsigned DEFAULT NULL,
  `user_id` int(10) unsigned DEFAULT NULL,
  `ttl` int(10) unsigned DEFAULT NULL,
  `wait` int(10) unsigned DEFAULT NULL,
  `spoofed_from` int(10) unsigned DEFAULT NULL,
  `version` varchar(45) COLLATE utf8_unicode_ci DEFAULT NULL,
  `spoofed` tinyint(1) unsigned NOT NULL,
  `record_route` tinyint(1) unsigned NOT NULL,
  `payload` tinyint(1) unsigned NOT NULL,
  `tsonly` tinyint(1) unsigned NOT NULL,
  `tsandaddr` tinyint(3) unsigned NOT NULL,
  `icmptimestamp` tinyint(1) unsigned NOT NULL,
  `icmpsum` tinyint(1) unsigned NOT NULL,
  `dl` tinyint(1) unsigned NOT NULL,
  `8` tinyint(1) unsigned NOT NULL,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `label` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `src_dst` (`src`,`dst`) USING BTREE,
  KEY `record_route` (`record_route`),
  KEY `label_created` (`label`,`created`),
  KEY `label` (`label`),
  KEY `created` (`created`) USING BTREE
  
) ENGINE=InnoDB AUTO_INCREMENT=118871350 DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `record_routes`
--

DROP TABLE IF EXISTS `record_routes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `record_routes` (
  `response_id` bigint(20) NOT NULL,
  `hop` tinyint(3) unsigned NOT NULL,
  `ip` int(10) unsigned NOT NULL,
  KEY `fk_record_routes_1_idx` (`response_id`),
  CONSTRAINT `rr_ping_responses` FOREIGN KEY (`response_id`) REFERENCES `ping_responses` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `timestamp_addrs`
--

DROP TABLE IF EXISTS `timestamp_addrs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `timestamp_addrs` (
  `response_id` bigint(20) NOT NULL,
  `order` tinyint(3) unsigned NOT NULL,
  `ip` int(10) unsigned NOT NULL,
  `ts` int(10) unsigned DEFAULT NULL,
  KEY `fk_timestamp_addrs_1_idx` (`response_id`),
  CONSTRAINT `tsa_ping_responses` FOREIGN KEY (`response_id`) REFERENCES `ping_responses` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `timestamps`
--

DROP TABLE IF EXISTS `timestamps`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `timestamps` (
  `response_id` bigint(20) NOT NULL,
  `order` tinyint(3) unsigned NOT NULL,
  `ts` int(10) unsigned NOT NULL,
  KEY `fk_timestamps_1_idx` (`response_id`),
  CONSTRAINT `ts_ping_responses` FOREIGN KEY (`response_id`) REFERENCES `ping_responses` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `icmp_timestamps`
--

DROP TABLE IF EXISTS `icmp_timestamps`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
origin_ts, receive_ts, transmit_ts
CREATE TABLE `icmp_timestamps` (
  `response_id` bigint(20) NOT NULL,
  `origin_ts` int(10) unsigned NOT NULL,
  `receive_ts` int(10) unsigned NOT NULL,
  `transmit_ts` int(10) unsigned NOT NULL,
  KEY `fk_icmp_timestamps_1_idx` (`response_id`),
  CONSTRAINT `icmp_ts_ping_responses` FOREIGN KEY (`response_id`) REFERENCES `ping_responses` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ;

--
-- Table structure for table `trace_batch`
--

DROP TABLE IF EXISTS `trace_batch`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `trace_batch` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int(10) unsigned NOT NULL,
  `created` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `trace_batch_trace`
--

DROP TABLE IF EXISTS `trace_batch_trace`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `trace_batch_trace` (
  `batch_id` int(10) unsigned NOT NULL,
  `trace_id` int(10) unsigned NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `traceroute_hops`
--

DROP TABLE IF EXISTS `traceroute_hops`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `traceroute_hops` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `traceroute_id` bigint(20) NOT NULL,
  `hop` int(10) unsigned NOT NULL,
  `addr` int(10) unsigned NOT NULL,
  `probe_ttl` int(10) unsigned NOT NULL,
  `probe_id` int(10) unsigned NOT NULL,
  `probe_size` int(10) unsigned NOT NULL,
  `rtt` int(10) unsigned NOT NULL,
  `reply_ttl` int(10) unsigned NOT NULL,
  `reply_tos` int(10) unsigned DEFAULT NULL,
  `reply_size` int(10) unsigned DEFAULT NULL,
  `reply_ipid` int(10) unsigned DEFAULT NULL,
  `icmp_type` int(10) unsigned DEFAULT NULL,
  `icmp_code` int(10) unsigned DEFAULT NULL,
  `icmp_q_ttl` int(10) unsigned DEFAULT NULL,
  `icmp_q_ipl` int(10) unsigned DEFAULT NULL,
  `icmp_q_tos` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_traceroute_hops_1_idx` (`traceroute_id`),
  KEY `traceroute_id` (`traceroute_id`),
  CONSTRAINT `traceroute` FOREIGN KEY (`traceroute_id`) REFERENCES `traceroutes` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB AUTO_INCREMENT=81005906 DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `traceroutes`
--

DROP TABLE IF EXISTS `traceroutes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `traceroutes` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `src` int(10) unsigned NOT NULL,
  `dst` int(10) unsigned NOT NULL,
  `type` varchar(45) COLLATE utf8_unicode_ci DEFAULT NULL,
  `user_id` int(10) unsigned NOT NULL,
  `method` varchar(45) COLLATE utf8_unicode_ci DEFAULT NULL,
  `sport` int(10) unsigned NOT NULL,
  `dport` int(10) unsigned NOT NULL,
  `stop_reason` varchar(45) COLLATE utf8_unicode_ci DEFAULT NULL,
  `stop_data` int(10) unsigned NOT NULL,
  `start` datetime NOT NULL,
  `version` varchar(45) COLLATE utf8_unicode_ci DEFAULT NULL,
  `hop_count` int(10) unsigned NOT NULL,
  `attempts` int(10) unsigned NOT NULL,
  `hop_limit` int(10) unsigned NOT NULL,
  `first_hop` int(10) unsigned NOT NULL,
  `wait` int(10) unsigned NOT NULL,
  `wait_probe` int(10) unsigned NOT NULL,
  `tos` int(10) unsigned NOT NULL,
  `probe_size` int(10) unsigned NOT NULL,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `label` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `src_dst` (`src`,`dst`) USING BTREE,
  KEY `label` (`label`),
  KEY `start` (`start`),
  KEY `created` (`created`)
) ENGINE=InnoDB AUTO_INCREMENT=34137068 DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `users` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',
  `email` varchar(255) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',
  `max` int(10) unsigned NOT NULL DEFAULT '0',
  `delay` int(10) unsigned NOT NULL DEFAULT '0',
  `key` varchar(100) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `index2` (`key`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2016-07-19  9:20:04
