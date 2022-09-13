-- MySQL dump 10.13  Distrib 5.6.30-76.3, for debian-linux-gnu (x86_64)
--
-- Host: localhost    Database: vpservice
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
-- Table structure for table `backoff_strategies`
--

DROP TABLE IF EXISTS `backoff_strategies`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `backoff_strategies` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `initial_backoff` bigint(20) NOT NULL,
  `multiplier` int(11) NOT NULL,
  `maximum_backoff` bigint(20) NOT NULL,
  `next_init_backoff` bigint(20) NOT NULL,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dist_to_dest`
--

DROP TABLE IF EXISTS `dist_to_dest`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dist_to_dest` (
  `src` int(10) unsigned NOT NULL,
  `dst` int(10) unsigned NOT NULL,
  `dist` int(10) unsigned NOT NULL,
  `slash_24` int(10) unsigned NOT NULL,
  `date` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`src`,`dst`,`slash_24`),
  KEY `index1` (`src`,`dst`),
  KEY `slash` (`slash_24`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `dist_to_dest_temp`
--

DROP TABLE IF EXISTS `dist_to_dest_temp`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `dist_to_dest_temp` (
  `src` int(10) unsigned NOT NULL,
  `dst` int(10) unsigned NOT NULL,
  `dist` int(10) unsigned NOT NULL,
  `slash_24` int(10) unsigned NOT NULL,
  `date` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY `index1` (`src`,`dst`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `quarantine_events`
--

DROP TABLE IF EXISTS `quarantine_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `quarantine_events` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `type` varchar(45) COLLATE utf8_unicode_ci NOT NULL,
  `ip` int(10) unsigned NOT NULL,
  `hostname` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `site` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `quarantine_type` varchar(25) COLLATE utf8_unicode_ci NOT NULL,
  `quarantine` blob NOT NULL,
  `time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=12 DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `quarantined_vps`
--

DROP TABLE IF EXISTS `quarantined_vps`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `quarantined_vps` (
  `ip` int(10) unsigned NOT NULL,
  `hostname` varchar(255) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',
  `site` varchar(255) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',
  `type` varchar(25) COLLATE utf8_unicode_ci NOT NULL DEFAULT 'DEFAULT',
  `added` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `quarantine` blob NOT NULL,
  PRIMARY KEY (`ip`),
  UNIQUE KEY `hostname_UNIQUE` (`hostname`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `unresponsive_targets`
--

DROP TABLE IF EXISTS `unresponsive_targets`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `unresponsive_targets` (
  `ip` int(10) unsigned NOT NULL,
  PRIMARY KEY (`ip`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `vantage_points`
--

DROP TABLE IF EXISTS `vantage_points`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `vantage_points` (
  `ip` int(10) unsigned NOT NULL,
  `hostname` varchar(255) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',
  `site` varchar(255) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',
  `timestamp` tinyint(4) NOT NULL DEFAULT '0',
  `record_route` tinyint(4) NOT NULL DEFAULT '0',
  `spoof` tinyint(4) NOT NULL DEFAULT '0',
  `rec_spoof` tinyint(4) NOT NULL DEFAULT '0',
  `ping` tinyint(4) NOT NULL DEFAULT '0',
  `trace` tinyint(4) NOT NULL DEFAULT '0',
  `last_check` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`ip`),
  KEY `rr` (`record_route`),
  KEY `spoof` (`spoof`),
  KEY `ts` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `vp_events`
--

DROP TABLE IF EXISTS `vp_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `vp_events` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `type` varchar(45) COLLATE utf8_unicode_ci NOT NULL,
  `hostname` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `site` varchar(255) COLLATE utf8_unicode_ci NOT NULL,
  `time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `hostname_time` (`hostname`,`time`)
) ENGINE=InnoDB AUTO_INCREMENT=11330 DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2016-07-19  9:21:08
