-- MySQL dump 10.13  Distrib 5.6.30-76.3, for debian-linux-gnu (x86_64)
--
-- Host: localhost    Database: traceroute_atlas
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
-- Table structure for table `atlas_traceroute_hops`
--

DROP TABLE IF EXISTS `atlas_traceroute_hops`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `atlas_traceroute_hops` (
  `trace_id` int(10) unsigned NOT NULL,
  `hop` int(10) unsigned NOT NULL DEFAULT '0',
  `ttl` int(10) unsigned NOT NULL,
  `mpls`tinyint(1), 
  KEY `fk_atlas_traceroute_hops_1_idx` (`trace_id`),
  KEY `index2` (`hop`) USING BTREE,
  CONSTRAINT `atlas_traceroute` FOREIGN KEY (`trace_id`) REFERENCES `atlas_traceroutes` (`Id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `atlas_traceroutes`
--

DROP TABLE IF EXISTS `atlas_traceroutes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `atlas_traceroutes` (
  `Id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `dest` int(10) unsigned NOT NULL DEFAULT '0',
  `src` int(10) unsigned NOT NULL DEFAULT '0',
  `source_asn` int(10) unsigned NOT NULL DEFAULT '0',
  `platform` varchar(255) COLLATE utf8_unicode_ci NOT NULL DEFAULT '',
  `date` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `stale` tinyint(1) NOT NULL DEFAULT 0, 
  PRIMARY KEY (`Id`),
  KEY `index2` (`dest`,`date`) USING BTREE,
  KEY `index3` (`src`,`dest`,`date`),
  KEY `index4` (`platform`) USING BTREE,
  KEY `index5` (`stale`) USING HASH
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `ip_aliases`
--

DROP TABLE IF EXISTS `ip_aliases`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ip_aliases` (
  `cluster_id` int(11) NOT NULL,
  `ip_address` int(10) unsigned NOT NULL,
  PRIMARY KEY (`ip_address`),
  KEY `cluster_idx` (`cluster_id`) USING BTREE,
  KEY `ip_indx` (`ip_address`) 
) ENGINE=MyISAM DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2016-07-19  9:20:49

-- Source ASes of the RIPE Atlas probes
DROP TABLE IF EXISTS `atlas_source_as`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `atlas_source_as` (
  `Id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `ip` int(10) unsigned NOT NULL DEFAULT '0',
  `asn` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`Id`),
  KEY `index2` (`asn`) USING BTREE
) ENGINE=InnoDB ;




DROP TABLE IF EXISTS `atlas_rr_pings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `atlas_rr_pings` (
  `traceroute_id` int(10) unsigned NOT NULL,
  `ping_id` int(10) unsigned NOT NULL,
  `tr_hop` int(10) unsigned NOT NULL,
  `rr_hop` int(10) unsigned NOT NULL,
  `rr_hop_index` int(10) unsigned NOT NULL,
  `date` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY `index1` (`traceroute_id`) USING BTREE,
  KEY `date` (`date`)
  CONSTRAINT `atlas_traceroutes` FOREIGN KEY (`traceroute_id`) REFERENCES `atlas_traceroutes` (`Id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;



DROP TABLE IF EXISTS `atlas_rr_intersection`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `atlas_rr_intersection` (
  `traceroute_id` int(10) unsigned NOT NULL,
  `rr_hop` int(10) unsigned NOT NULL,
  `tr_ttl_start` int(10) unsigned NOT NULL,
  `tr_ttl_end` int(10) unsigned NOT NULL,
  KEY `index1` (`traceroute_id`) USING BTREE,
  KEY `index_rr_hop`(`rr_hop`) USING HASH, 
  CONSTRAINT `atlas_traceroutes` FOREIGN KEY (`traceroute_id`) REFERENCES `atlas_traceroutes` (`Id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;

