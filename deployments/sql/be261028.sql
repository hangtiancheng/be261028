-- ============================================================
-- be261028 group-buy marketing system - database initialization script
-- Migrated from easy-group-buy (easy_group_buy); database renamed to be261028.
-- Defect fix: notify_task drops the team_id unique constraint (one team produces
--           multiple callback tasks: team settlement + each refund); rows are now
--           updated by auto-increment id, with a plain index on team_id.
-- ============================================================

CREATE DATABASE IF NOT EXISTS `be261028` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
USE `be261028`;

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ------------------------------------------------------------
-- Crowd tags
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `crowd_tags`;
CREATE TABLE `crowd_tags` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT 'Auto-increment id',
  `tag_id` varchar(32) NOT NULL COMMENT 'Crowd id',
  `tag_name` varchar(64) NOT NULL COMMENT 'Crowd name',
  `tag_desc` varchar(256) NOT NULL COMMENT 'Crowd description',
  `statistics` int NOT NULL COMMENT 'Crowd tag member count',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_tag_id` (`tag_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Crowd tags';

INSERT INTO `crowd_tags` (`id`, `tag_id`, `tag_name`, `tag_desc`, `statistics`, `create_time`, `update_time`)
VALUES (1,'RQ_KJHKL98UU78H66554GFDV','Potential buyers','Potential buyers',33,'2024-12-28 12:53:28','2025-01-28 08:23:57');

-- ------------------------------------------------------------
-- Crowd tag members
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `crowd_tags_detail`;
CREATE TABLE `crowd_tags_detail` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT 'Auto-increment id',
  `tag_id` varchar(32) NOT NULL COMMENT 'Crowd id',
  `user_id` varchar(16) NOT NULL COMMENT 'User id',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_tag_user` (`tag_id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Crowd tag members';

INSERT INTO `crowd_tags_detail` (`id`, `tag_id`, `user_id`, `create_time`, `update_time`)
VALUES
  (4,'RQ_KJHKL98UU78H66554GFDV','xiaofuge','2024-12-28 14:42:30','2024-12-28 14:42:30'),
  (5,'RQ_KJHKL98UU78H66554GFDV','liergou','2024-12-28 14:42:30','2024-12-28 14:42:30'),
  (9,'RQ_KJHKL98UU78H66554GFDV','xfg01','2025-01-25 15:44:55','2025-01-25 15:44:55'),
  (10,'RQ_KJHKL98UU78H66554GFDV','xfg02','2025-01-25 15:44:55','2025-01-25 15:44:55'),
  (11,'RQ_KJHKL98UU78H66554GFDV','xfg03','2025-01-25 15:44:55','2025-01-25 15:44:55'),
  (17,'RQ_KJHKL98UU78H66554GFDV','xfg04','2025-01-26 19:10:36','2025-01-26 19:10:36'),
  (18,'RQ_KJHKL98UU78H66554GFDV','xfg05','2025-01-26 19:10:36','2025-01-26 19:10:36'),
  (19,'RQ_KJHKL98UU78H66554GFDV','xfg06','2025-01-26 19:10:37','2025-01-26 19:10:37'),
  (20,'RQ_KJHKL98UU78H66554GFDV','xfg07','2025-01-26 19:10:37','2025-01-26 19:10:37'),
  (21,'RQ_KJHKL98UU78H66554GFDV','xfg08','2025-01-26 19:10:37','2025-01-26 19:10:37'),
  (22,'RQ_KJHKL98UU78H66554GFDV','xfg09','2025-01-26 19:10:37','2025-01-26 19:10:37');

-- ------------------------------------------------------------
-- Crowd tag batch jobs
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `crowd_tags_job`;
CREATE TABLE `crowd_tags_job` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT 'Auto-increment id',
  `tag_id` varchar(32) NOT NULL COMMENT 'Tag id',
  `batch_id` varchar(8) NOT NULL COMMENT 'Batch id',
  `tag_type` tinyint(1) NOT NULL DEFAULT '1' COMMENT 'Tag type (participation count, spend amount)',
  `tag_rule` varchar(8) NOT NULL COMMENT 'Tag rule (limit type, N times)',
  `stat_start_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Statistics window start',
  `stat_end_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Statistics window end',
  `status` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Status; 0 initial, 1 planned, 2 reset, 3 completed',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_batch_id` (`batch_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Crowd tag batch jobs';

INSERT INTO `crowd_tags_job` (`id`, `tag_id`, `batch_id`, `tag_type`, `tag_rule`, `stat_start_time`, `stat_end_time`, `status`, `create_time`, `update_time`)
VALUES (1,'RQ_KJHKL98UU78H66554GFDV','10001',0,'100','2024-12-28 12:55:05','2024-12-28 12:55:05',0,'2024-12-28 12:55:05','2024-12-28 12:55:05');

-- ------------------------------------------------------------
-- Group-buy activities
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `group_buy_activity`;
CREATE TABLE `group_buy_activity` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Auto-increment',
  `activity_id` bigint NOT NULL COMMENT 'Activity id',
  `activity_name` varchar(128) NOT NULL COMMENT 'Activity name',
  `discount_id` varchar(8) NOT NULL COMMENT 'Discount id',
  `group_type` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Group type (0 auto-complete, 1 target-based)',
  `take_limit_count` int NOT NULL DEFAULT '1' COMMENT 'Max participations per user',
  `target` int NOT NULL DEFAULT '1' COMMENT 'Team target size',
  `valid_time` int NOT NULL DEFAULT '15' COMMENT 'Team validity duration (minutes)',
  `status` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Activity status (0 created, 1 effective, 2 overdue, 3 abandoned)',
  `start_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Activity start time',
  `end_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Activity end time',
  `tag_id` varchar(8) DEFAULT NULL COMMENT 'Crowd tag rule identifier',
  `tag_scope` varchar(4) DEFAULT NULL COMMENT 'Crowd tag rule scope (multi-select; 1 visibility restriction, 2 participation restriction)',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_activity_id` (`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Group-buy activities';

INSERT INTO `group_buy_activity` (`id`, `activity_id`, `activity_name`, `discount_id`, `group_type`, `take_limit_count`, `target`, `valid_time`, `status`, `start_time`, `end_time`, `tag_id`, `tag_scope`, `create_time`, `update_time`)
VALUES (1,100123,'Demo activity','25120207',0,1,1,15,1,'2024-12-07 10:19:40','2029-12-07 10:19:40','1','1','2024-12-07 10:19:40','2025-03-16 17:43:11');

-- ------------------------------------------------------------
-- Group-buy discounts
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `group_buy_discount`;
CREATE TABLE `group_buy_discount` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'Auto-increment id',
  `discount_id` varchar(8) NOT NULL COMMENT 'Discount id',
  `discount_name` varchar(64) NOT NULL COMMENT 'Discount title',
  `discount_desc` varchar(256) NOT NULL COMMENT 'Discount description',
  `discount_type` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Discount type (0: base, 1: crowd tag)',
  `market_plan` varchar(4) NOT NULL DEFAULT 'ZJ' COMMENT 'Marketing plan (ZJ: direct reduction, MJ: threshold reduction, N: fixed price, ZK: percentage discount)',
  `market_expr` varchar(32) NOT NULL COMMENT 'Marketing expression',
  `tag_id` varchar(8) DEFAULT NULL COMMENT 'Crowd tag restricting this discount',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_discount_id` (`discount_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO `group_buy_discount` (`id`, `discount_id`, `discount_name`, `discount_desc`, `discount_type`, `market_plan`, `market_expr`, `tag_id`, `create_time`, `update_time`)
VALUES (1,'25120207','Demo discount','Demo discount',0,'ZJ','20',NULL,'2024-12-07 10:20:15','2024-12-21 11:13:32');

-- ------------------------------------------------------------
-- Group-buy teams (user group orders)
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `group_buy_order`;
CREATE TABLE `group_buy_order` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT 'Auto-increment id',
  `team_id` varchar(8) NOT NULL COMMENT 'Team id',
  `activity_id` bigint NOT NULL COMMENT 'Activity id',
  `source` varchar(8) NOT NULL COMMENT 'Source',
  `channel` varchar(8) NOT NULL COMMENT 'Channel',
  `original_price` decimal(8,2) NOT NULL COMMENT 'Original price',
  `deduction_price` decimal(8,2) NOT NULL COMMENT 'Discount amount',
  `pay_price` decimal(8,2) NOT NULL COMMENT 'Payable price',
  `target_count` int NOT NULL COMMENT 'Target member count',
  `complete_count` int NOT NULL COMMENT 'Completed (paid) member count',
  `lock_count` int NOT NULL COMMENT 'Locked (in-progress) member count',
  `status` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Status (0 in progress, 1 completed, 2 failed, 3 completed with refunds)',
  `valid_start_time` datetime NOT NULL COMMENT 'Team validity start',
  `valid_end_time` datetime NOT NULL COMMENT 'Team validity end',
  `notify_type` varchar(8) NOT NULL DEFAULT 'HTTP' COMMENT 'Callback type (HTTP, MQ)',
  `notify_url` varchar(512) DEFAULT NULL COMMENT 'Callback URL (required for HTTP callbacks)',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_team_id` (`team_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ------------------------------------------------------------
-- Group-buy order details
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `group_buy_order_list`;
CREATE TABLE `group_buy_order_list` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT 'Auto-increment id',
  `user_id` varchar(64) NOT NULL COMMENT 'User id',
  `team_id` varchar(8) NOT NULL COMMENT 'Team id',
  `order_id` varchar(12) NOT NULL COMMENT 'Order id',
  `activity_id` bigint NOT NULL COMMENT 'Activity id',
  `start_time` datetime NOT NULL COMMENT 'Activity start time',
  `end_time` datetime NOT NULL COMMENT 'Activity end time',
  `goods_id` varchar(16) NOT NULL COMMENT 'Goods id',
  `source` varchar(8) NOT NULL COMMENT 'Source',
  `channel` varchar(8) NOT NULL COMMENT 'Channel',
  `original_price` decimal(8,2) NOT NULL COMMENT 'Original price',
  `deduction_price` decimal(8,2) NOT NULL COMMENT 'Discount amount',
  `pay_price` decimal(8,2) NOT NULL COMMENT 'Paid amount',
  `status` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Status; 0 initially locked, 1 paid, 2 refunded by user',
  `out_trade_no` varchar(32) NOT NULL COMMENT 'External trade number - guarantees caller idempotency',
  `out_trade_time` datetime DEFAULT NULL COMMENT 'External trade time',
  `biz_id` varchar(64) NOT NULL COMMENT 'Business unique id',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_order_id` (`order_id`),
  UNIQUE KEY `uq_biz_id` (`biz_id`),
  KEY `idx_user_id_activity_id` (`user_id`,`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ------------------------------------------------------------
-- Callback tasks (defect fix: team_id unique constraint removed; one team may
-- have multiple callback tasks)
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `notify_task`;
CREATE TABLE `notify_task` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT 'Auto-increment id',
  `activity_id` bigint NOT NULL COMMENT 'Activity id',
  `team_id` varchar(8) NOT NULL COMMENT 'Team id',
  `notify_category` varchar(64) DEFAULT NULL COMMENT 'Callback category',
  `notify_type` varchar(8) NOT NULL DEFAULT 'HTTP' COMMENT 'Callback type (HTTP, MQ)',
  `notify_mq` varchar(64) DEFAULT NULL COMMENT 'Callback message target (Kafka topic)',
  `notify_url` varchar(512) DEFAULT NULL COMMENT 'Callback endpoint',
  `notify_count` int NOT NULL DEFAULT '0' COMMENT 'Callback attempt count',
  `notify_status` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'Callback status [0 pending, 1 success, 2 retry, 3 error]',
  -- Defect fix: parameter_json widened from varchar(256) (the team-completion
  -- message grows linearly with member count; large teams exceeded 256 and rolled
  -- back the whole settlement transaction); notify_url aligned with
  -- group_buy_order's varchar(512) so long callback URLs cannot fail to write.
  `parameter_json` varchar(1024) NOT NULL COMMENT 'Parameter payload',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
  PRIMARY KEY (`id`),
  KEY `idx_notify_status` (`notify_status`),
  KEY `idx_team_id` (`team_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ------------------------------------------------------------
-- Source/channel goods-to-activity mapping
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `sc_sku_activity`;
CREATE TABLE `sc_sku_activity` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT 'Auto-increment id',
  `source` varchar(8) NOT NULL COMMENT 'Source',
  `channel` varchar(8) NOT NULL COMMENT 'Channel',
  `activity_id` bigint NOT NULL COMMENT 'Activity id',
  `goods_id` varchar(16) NOT NULL COMMENT 'Goods id',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_sc_goodsid` (`source`,`channel`,`goods_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Source/channel goods-to-activity mapping';

INSERT INTO `sc_sku_activity` (`id`, `source`, `channel`, `activity_id`, `goods_id`, `create_time`, `update_time`)
VALUES (1,'s01','c01',100123,'9890001','2025-01-01 13:15:54','2025-01-01 13:15:54');

-- ------------------------------------------------------------
-- Goods (SKU) catalog
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `sku`;
CREATE TABLE `sku` (
  `id` int unsigned NOT NULL AUTO_INCREMENT COMMENT 'Auto-increment id',
  `source` varchar(8) NOT NULL COMMENT 'Source',
  `channel` varchar(8) NOT NULL COMMENT 'Channel',
  `goods_id` varchar(16) NOT NULL COMMENT 'Goods id',
  `goods_name` varchar(128) NOT NULL COMMENT 'Goods name',
  `original_price` decimal(10,2) NOT NULL COMMENT 'Goods price',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create time',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_goods_id` (`goods_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Goods (SKU) catalog';

INSERT INTO `sku` (`id`, `source`, `channel`, `goods_id`, `goods_name`, `original_price`, `create_time`, `update_time`)
VALUES (1,'s01','c01','9890001','Handwritten MyBatis: A Progressive Source-Code Practice',100.00,'2024-12-21 11:10:06','2024-12-21 11:10:06');

-- ------------------------------------------------------------
-- Demo data: two-person team activity (goodsId=9890002, used to showcase the
-- team list). The home page can load it via /?goodsId=9890002; the
-- default goods remains 9890001.
-- ------------------------------------------------------------
INSERT INTO `group_buy_discount` (`id`, `discount_id`, `discount_name`, `discount_desc`, `discount_type`, `market_plan`, `market_expr`, `tag_id`, `create_time`, `update_time`)
VALUES (2,'D300001','Limited-time price cut','10 off instantly when the two-person team forms',0,'ZJ','10',NULL,'2025-01-01 00:00:00','2025-01-01 00:00:00');

INSERT INTO `group_buy_activity` (`id`, `activity_id`, `activity_name`, `discount_id`, `group_type`, `take_limit_count`, `target`, `valid_time`, `status`, `start_time`, `end_time`, `tag_id`, `tag_scope`, `create_time`, `update_time`)
VALUES (2,300001,'Two-person group-buy demo','D300001',1,5,2,60,1,'2025-01-01 00:00:00','2029-12-31 23:59:59',NULL,NULL,'2025-01-01 00:00:00','2025-01-01 00:00:00');

INSERT INTO `sku` (`id`, `source`, `channel`, `goods_id`, `goods_name`, `original_price`, `create_time`, `update_time`)
VALUES (2,'s01','c01','9890002','Handwritten MyBatis (two-person team edition)',120.00,'2025-01-01 00:00:00','2025-01-01 00:00:00');

INSERT INTO `sc_sku_activity` (`id`, `source`, `channel`, `activity_id`, `goods_id`, `create_time`, `update_time`)
VALUES (2,'s01','c01',300001,'9890002','2025-01-01 00:00:00','2025-01-01 00:00:00');

SET FOREIGN_KEY_CHECKS = 1;
