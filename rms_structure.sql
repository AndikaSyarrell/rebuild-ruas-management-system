-- --------------------------------------------------------
-- Host:                         127.0.0.1
-- Server version:               8.4.3 - MySQL Community Server - GPL
-- Server OS:                    Win64
-- HeidiSQL Version:             12.8.0.6908
-- --------------------------------------------------------

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET NAMES utf8 */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

-- Dumping structure for table rms_migrated.t_access
CREATE TABLE IF NOT EXISTS `t_access` (
  `access_id` int NOT NULL AUTO_INCREMENT,
  `access_title` varchar(156) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `access_slug` varchar(156) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `access_module` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `access_sort` int NOT NULL DEFAULT '1',
  `access_create_date` datetime NOT NULL,
  `access_modify_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`access_id`),
  UNIQUE KEY `uq_access_slug` (`access_slug`)
) ENGINE=InnoDB AUTO_INCREMENT=38 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Data exporting was unselected.

-- Dumping structure for table rms_migrated.t_activity
CREATE TABLE IF NOT EXISTS `t_activity` (
  `activity_id` int NOT NULL AUTO_INCREMENT,
  `activity_ref_po` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `activity_ref_user` varchar(144) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `activity_type` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'open/progress/prepared/complete/cancel/quarantine/edit',
  `activity_notes` varchar(4048) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `activity_create_date` datetime NOT NULL,
  `activity_modify_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`activity_id`),
  KEY `idx_activity_po` (`activity_ref_po`),
  KEY `idx_activity_user` (`activity_ref_user`),
  CONSTRAINT `fk_activity_po` FOREIGN KEY (`activity_ref_po`) REFERENCES `t_po` (`po_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_activity_user` FOREIGN KEY (`activity_ref_user`) REFERENCES `t_admin` (`admin_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=52813 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Data exporting was unselected.

-- Dumping structure for table rms_migrated.t_admin
CREATE TABLE IF NOT EXISTS `t_admin` (
  `admin_id` varchar(144) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `admin_ref_region` int NOT NULL,
  `admin_ref_role` int DEFAULT NULL COMMENT 'FK ke T_Role, menggantikan admin_role bertipe teks bebas',
  `admin_token` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `admin_reset_code` varchar(250) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `admin_email` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `admin_name` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `admin_password` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `admin_active` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'active',
  `admin_pic` varchar(14) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'yes',
  `admin_pic_client` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'no',
  `admin_img` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `admin_img_thmb` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `admin_create_date` datetime NOT NULL,
  `admin_modify_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`admin_id`),
  UNIQUE KEY `uq_admin_email` (`admin_email`),
  KEY `idx_admin_region` (`admin_ref_region`),
  KEY `idx_admin_role` (`admin_ref_role`),
  CONSTRAINT `fk_admin_region` FOREIGN KEY (`admin_ref_region`) REFERENCES `t_region` (`region_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_admin_role` FOREIGN KEY (`admin_ref_role`) REFERENCES `t_role` (`role_id`) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Data exporting was unselected.

-- Dumping structure for table rms_migrated.t_client
CREATE TABLE IF NOT EXISTS `t_client` (
  `client_id` int NOT NULL AUTO_INCREMENT,
  `client_ref_region` int NOT NULL,
  `client_name` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `client_phone` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `client_email` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `client_password` varchar(296) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `client_reset_code` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `client_token` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `client_active` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'no',
  `client_address` varchar(4048) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `client_create_date` datetime NOT NULL,
  `client_modify_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`client_id`),
  UNIQUE KEY `uq_client_email` (`client_email`),
  KEY `idx_client_region` (`client_ref_region`),
  CONSTRAINT `fk_client_region` FOREIGN KEY (`client_ref_region`) REFERENCES `t_region` (`region_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=80 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Data exporting was unselected.

-- Dumping structure for table rms_migrated.t_division
CREATE TABLE IF NOT EXISTS `t_division` (
  `division_id` int NOT NULL AUTO_INCREMENT,
  `division_title` varchar(255) NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`division_id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Data exporting was unselected.

-- Dumping structure for table rms_migrated.t_document
CREATE TABLE IF NOT EXISTS `t_document` (
  `document_id` int NOT NULL AUTO_INCREMENT,
  `document_ref_po` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `document_title` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `document_file` varchar(2048) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `document_create_date` datetime NOT NULL,
  `document_modify_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`document_id`),
  KEY `idx_document_po` (`document_ref_po`),
  CONSTRAINT `fk_document_po` FOREIGN KEY (`document_ref_po`) REFERENCES `t_po` (`po_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=6700 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Data exporting was unselected.

-- Dumping structure for table rms_migrated.t_po
CREATE TABLE IF NOT EXISTS `t_po` (
  `po_id` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `po_order_num` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `po_invoice` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `po_ref_client` int NOT NULL,
  `po_ref_admin` varchar(144) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'admin pembuat PO',
  `po_ref_pic` varchar(144) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'admin PIC yang menangani PO (dulu: po_ref_admin_client)',
  `po_ref_pic_client` varchar(144) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'PIC yang menangani relasi ke klien, terpisah dari po_ref_pic (PIC project)',
  `po_ref_division` int DEFAULT NULL,
  `po_ref_region` int NOT NULL,
  `po_ref_ppn` int NOT NULL,
  `po_client_name` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `po_client_email` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `po_client_phone` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `po_client_address` varchar(4048) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `po_subclient` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `po_subtotal` decimal(15,2) NOT NULL DEFAULT '0.00',
  `po_ppn_rate` decimal(5,2) NOT NULL DEFAULT '0.00' COMMENT 'snapshot persentase ppn yang dipakai',
  `po_ppn_amount` decimal(15,2) NOT NULL DEFAULT '0.00',
  `po_total` decimal(15,2) NOT NULL DEFAULT '0.00',
  `po_item_total` int NOT NULL DEFAULT '0',
  `po_status` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `po_document` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'no',
  `po_paid` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT 'no',
  `po_notes` varchar(8143) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `po_date` date DEFAULT NULL,
  `po_exp_date` date DEFAULT NULL,
  `po_prepared_date` datetime DEFAULT NULL,
  `po_progress_date` datetime DEFAULT NULL,
  `po_complete_date` datetime DEFAULT NULL,
  `po_cancel_date` datetime DEFAULT NULL,
  `po_create_date` datetime NOT NULL,
  `po_modify_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`po_id`),
  UNIQUE KEY `uq_po_order_num` (`po_order_num`),
  KEY `idx_po_status` (`po_status`),
  KEY `idx_po_region` (`po_ref_region`),
  KEY `idx_po_pic` (`po_ref_pic`),
  KEY `idx_po_admin` (`po_ref_admin`),
  KEY `idx_po_client` (`po_ref_client`),
  KEY `idx_po_date` (`po_date`),
  KEY `idx_status_region_date` (`po_status`,`po_ref_region`,`po_date`),
  KEY `idx_status_pic_date` (`po_status`,`po_ref_pic`,`po_date`),
  KEY `idx_po_division` (`po_ref_division`),
  KEY `fk_po_ppn` (`po_ref_ppn`),
  KEY `idx_po_pic_client` (`po_ref_pic_client`),
  CONSTRAINT `fk_po_admin` FOREIGN KEY (`po_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_po_client` FOREIGN KEY (`po_ref_client`) REFERENCES `t_client` (`client_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_po_division` FOREIGN KEY (`po_ref_division`) REFERENCES `t_division` (`division_id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_po_pic` FOREIGN KEY (`po_ref_pic`) REFERENCES `t_admin` (`admin_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_po_pic_client` FOREIGN KEY (`po_ref_pic_client`) REFERENCES `t_admin` (`admin_id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_po_ppn` FOREIGN KEY (`po_ref_ppn`) REFERENCES `t_ppn` (`ppn_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_po_region` FOREIGN KEY (`po_ref_region`) REFERENCES `t_region` (`region_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Data exporting was unselected.

-- Dumping structure for table rms_migrated.t_po_item
CREATE TABLE IF NOT EXISTS `t_po_item` (
  `item_id` int NOT NULL AUTO_INCREMENT,
  `item_ref_po` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `item_ref_unit` int NOT NULL,
  `item_product` varchar(2048) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `item_desc` varchar(8048) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `item_qty` int NOT NULL,
  `item_price` decimal(15,2) NOT NULL,
  `item_create_date` datetime NOT NULL,
  `item_modify_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`item_id`),
  KEY `idx_item_po` (`item_ref_po`),
  KEY `idx_item_unit` (`item_ref_unit`),
  CONSTRAINT `fk_item_po` FOREIGN KEY (`item_ref_po`) REFERENCES `t_po` (`po_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_item_unit` FOREIGN KEY (`item_ref_unit`) REFERENCES `t_unit` (`unit_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=12709 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Data exporting was unselected.

-- Dumping structure for table rms_migrated.t_ppn
CREATE TABLE IF NOT EXISTS `t_ppn` (
  `ppn_id` int NOT NULL AUTO_INCREMENT,
  `ppn_value` decimal(5,2) NOT NULL COMMENT 'persentase, misal 11.00 = 11%',
  `ppn_create_date` datetime NOT NULL,
  `ppn_modify_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`ppn_id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Data exporting was unselected.

-- Dumping structure for table rms_migrated.t_region
CREATE TABLE IF NOT EXISTS `t_region` (
  `region_id` int NOT NULL AUTO_INCREMENT,
  `region_title` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `region_create_date` datetime NOT NULL,
  `region_modify_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`region_id`)
) ENGINE=InnoDB AUTO_INCREMENT=24 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Data exporting was unselected.

-- Dumping structure for table rms_migrated.t_role
CREATE TABLE IF NOT EXISTS `t_role` (
  `role_id` int NOT NULL AUTO_INCREMENT,
  `role_title` varchar(156) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `role_slug` varchar(156) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `role_create_date` datetime NOT NULL,
  `role_modify_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`role_id`),
  UNIQUE KEY `uq_role_slug` (`role_slug`)
) ENGINE=InnoDB AUTO_INCREMENT=14 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Data exporting was unselected.

-- Dumping structure for table rms_migrated.t_role_access
CREATE TABLE IF NOT EXISTS `t_role_access` (
  `role_access_id` int NOT NULL AUTO_INCREMENT,
  `role_access_ref_role` int NOT NULL,
  `role_access_ref_access` int NOT NULL,
  `role_access_create_date` datetime NOT NULL,
  `role_access_modify_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`role_access_id`),
  UNIQUE KEY `uq_role_access` (`role_access_ref_role`,`role_access_ref_access`),
  KEY `fk_ra_access` (`role_access_ref_access`),
  CONSTRAINT `fk_ra_access` FOREIGN KEY (`role_access_ref_access`) REFERENCES `t_access` (`access_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_ra_role` FOREIGN KEY (`role_access_ref_role`) REFERENCES `t_role` (`role_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=105 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Data exporting was unselected.

-- Dumping structure for table rms_migrated.t_unit
CREATE TABLE IF NOT EXISTS `t_unit` (
  `unit_id` int NOT NULL AUTO_INCREMENT,
  `unit_title` varchar(196) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `unit_create_date` datetime NOT NULL,
  `unit_modify_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`unit_id`)
) ENGINE=InnoDB AUTO_INCREMENT=36 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- Data exporting was unselected.

/*!40103 SET TIME_ZONE=IFNULL(@OLD_TIME_ZONE, 'system') */;
/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IFNULL(@OLD_FOREIGN_KEY_CHECKS, 1) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40111 SET SQL_NOTES=IFNULL(@OLD_SQL_NOTES, 1) */;
