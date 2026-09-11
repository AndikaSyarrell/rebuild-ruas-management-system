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

-- Dumping structure for table rms_new_pr.t_access
CREATE TABLE IF NOT EXISTS `t_access` (
  `access_id` int NOT NULL AUTO_INCREMENT,
  `access_title` varchar(156) NOT NULL,
  `access_slug` varchar(156) NOT NULL,
  `access_module` varchar(100) NOT NULL,
  `access_sort` int NOT NULL DEFAULT '1',
  `access_create_date` datetime NOT NULL,
  `access_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`access_id`),
  UNIQUE KEY `uq_access_slug` (`access_slug`)
) ENGINE=InnoDB AUTO_INCREMENT=91 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_access: ~53 rows (approximately)
INSERT INTO `t_access` (`access_id`, `access_title`, `access_slug`, `access_module`, `access_sort`, `access_create_date`, `access_modify_date`) VALUES
	(38, 'Buat Region', 'create_region', 'region', 1, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(39, 'Ubah Region', 'edit_region', 'region', 2, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(40, 'Hapus Region', 'delete_region', 'region', 3, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(41, 'Buat Divisi', 'create_division', 'division', 1, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(42, 'Ubah Divisi', 'edit_division', 'division', 2, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(43, 'Hapus Divisi', 'delete_division', 'division', 3, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(44, 'Buat Unit', 'create_unit', 'unit', 1, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(45, 'Ubah Unit', 'edit_unit', 'unit', 2, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(46, 'Hapus Unit', 'delete_unit', 'unit', 3, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(47, 'Ubah PPN', 'edit_ppn', 'ppn', 1, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(48, 'Buat Role', 'create_role', 'rbac', 1, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(49, 'Ubah Role', 'edit_role', 'rbac', 2, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(50, 'Hapus Role', 'delete_role', 'rbac', 3, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(51, 'Buat Access', 'create_access', 'rbac', 4, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(52, 'Ubah Access', 'edit_access', 'rbac', 5, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(53, 'Hapus Access', 'delete_access', 'rbac', 6, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(54, 'Buat Admin Baru', 'create_new_admin', 'admin', 1, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(55, 'Ubah Admin Lain', 'edit_other_admin', 'admin', 2, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(56, 'Ubah Foto Profil Admin', 'edit_pic', 'admin', 3, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(57, 'Kirim Ulang Email Aktivasi', 'resend_admin_activation', 'admin', 4, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(58, 'Aktif/Nonaktifkan Admin', 'deactivate_other_admin', 'admin', 5, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(59, 'Hapus Admin', 'delete_admin', 'admin', 6, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(60, 'Buat Klien', 'create_client', 'client', 1, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(61, 'Ubah Klien', 'edit_client', 'client', 2, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(62, 'Hapus Klien', 'delete_client', 'client', 3, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(63, 'Aktif/Nonaktifkan Klien', 'change_stat_active', 'client', 4, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(64, 'Buat PO', 'create_po', 'po', 1, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(65, 'Ubah PO', 'edit_po', 'po', 2, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(66, 'Hapus PO', 'delete_po', 'po', 3, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(67, 'Ubah Status PO', 'change_stat_po', 'po', 4, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(68, 'Ubah Status Bayar PO', 'change_stat_paid', 'po', 5, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(69, 'Ubah Invoice PO', 'update_invoice', 'po', 6, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(70, 'Ubah Catatan PO', 'update_po_notes', 'po', 7, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(71, 'Tambah Catatan Aktivitas PO', 'create_notes', 'po', 8, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(72, 'Upload Dokumen PO', 'upload_document', 'po', 9, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(73, 'Export PO ke Excel', 'export_po', 'po', 10, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(74, 'Lihat Laporan', 'view_report_table', 'report', 1, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(75, 'Buat PR', 'create_pr', 'pr', 1, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(76, 'Ubah PR', 'edit_pr', 'pr', 2, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(77, 'Submit PR', 'submit_pr', 'pr', 3, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(78, 'Export PR ke Excel', 'export_pr', 'pr', 4, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(79, 'Atur Prioritas PR', 'set_pr_priority', 'pr', 5, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(80, 'Setujui PR', 'approve_pr', 'pr', 6, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(81, 'Minta Revisi PR', 'request_revision_pr', 'pr', 7, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(82, 'Ubah Nominal PR', 'update_pr_amounts', 'pr', 8, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(83, 'Buat Pembayaran PR', 'create_pr_payment', 'pr', 9, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(84, 'Konfirmasi Pembayaran PR', 'confirm_pr_payment', 'pr', 10, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(85, 'Batalkan PR', 'cancel_pr', 'pr', 11, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(86, 'Upload Dokumen PR', 'upload_pr_document', 'pr', 12, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(87, 'Buat Penanggung Jawab', 'create_responsible', 'responsible', 1, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(88, 'Ubah Penanggung Jawab', 'edit_resonsible', 'responsible', 2, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(89, 'Hapus Penanggung Jawab', 'delete_responsible', 'responsible', 3, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(90, 'Aktif/Nonaktifkan Penanggung Jawab', 'change_stat_responsible', 'responsible', 4, '2026-01-01 08:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_activity
CREATE TABLE IF NOT EXISTS `t_activity` (
  `activity_id` int NOT NULL AUTO_INCREMENT,
  `activity_ref_po` varchar(196) NOT NULL,
  `activity_ref_user` varchar(144) NOT NULL,
  `activity_type` varchar(100) NOT NULL COMMENT 'open/progress/prepared/complete/cancel/quarantine/edit',
  `activity_notes` varchar(4048) DEFAULT NULL,
  `activity_create_date` datetime NOT NULL,
  `activity_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`activity_id`),
  KEY `idx_activity_po` (`activity_ref_po`),
  KEY `idx_activity_user` (`activity_ref_user`),
  CONSTRAINT `fk_activity_po` FOREIGN KEY (`activity_ref_po`) REFERENCES `t_po` (`po_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_activity_user` FOREIGN KEY (`activity_ref_user`) REFERENCES `t_admin` (`admin_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=52813 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_activity: ~0 rows (approximately)
INSERT INTO `t_activity` (`activity_id`, `activity_ref_po`, `activity_ref_user`, `activity_type`, `activity_notes`, `activity_create_date`, `activity_modify_date`) VALUES
	(1, 'PO20260001', 'adm-super-01', 'open', 'PO dibuat dan menunggu diproses', '2026-01-06 09:05:00', '2026-09-11 03:01:13'),
	(2, 'PO20260001', 'adm-appr1-01', 'progress', 'Pekerjaan lapangan dimulai', '2026-01-07 09:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_admin
CREATE TABLE IF NOT EXISTS `t_admin` (
  `admin_id` varchar(144) NOT NULL,
  `admin_ref_region` int NOT NULL,
  `admin_ref_role` int DEFAULT NULL COMMENT 'FK ke T_Role, menggantikan admin_role bertipe teks bebas',
  `admin_ref_division` int DEFAULT NULL,
  `admin_token` varchar(256) DEFAULT NULL,
  `admin_reset_code` varchar(250) DEFAULT NULL,
  `admin_email` varchar(196) NOT NULL,
  `admin_name` varchar(196) NOT NULL,
  `admin_password` varchar(196) DEFAULT NULL,
  `admin_active` varchar(20) NOT NULL DEFAULT 'active',
  `admin_pic` varchar(14) NOT NULL DEFAULT 'yes',
  `admin_pic_client` varchar(196) NOT NULL DEFAULT 'no',
  `admin_img` varchar(196) DEFAULT NULL,
  `admin_img_thmb` varchar(196) DEFAULT NULL,
  `admin_create_date` datetime NOT NULL,
  `admin_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`admin_id`),
  UNIQUE KEY `uq_admin_email` (`admin_email`),
  KEY `idx_admin_region` (`admin_ref_region`),
  KEY `idx_admin_role` (`admin_ref_role`),
  KEY `fk_admin_division` (`admin_ref_division`),
  CONSTRAINT `fk_admin_division` FOREIGN KEY (`admin_ref_division`) REFERENCES `t_division` (`division_id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_admin_region` FOREIGN KEY (`admin_ref_region`) REFERENCES `t_region` (`region_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_admin_role` FOREIGN KEY (`admin_ref_role`) REFERENCES `t_role` (`role_id`) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_admin: ~5 rows (approximately)
INSERT INTO `t_admin` (`admin_id`, `admin_ref_region`, `admin_ref_role`, `admin_ref_division`, `admin_token`, `admin_reset_code`, `admin_email`, `admin_name`, `admin_password`, `admin_active`, `admin_pic`, `admin_pic_client`, `admin_img`, `admin_img_thmb`, `admin_create_date`, `admin_modify_date`) VALUES
	('adm-appr1-01', 1, 3, 1, NULL, NULL, 'manager.ops@ruasutamajaya.co.id', 'Agus Prabowo', '$2a$12$JMLTejdJx2vrz8vdvyOOaOqS//sI0AinLpAnQHxxNqVgDyaOunWUu', 'active', 'yes', 'no', NULL, NULL, '2026-01-03 08:00:00', '2026-09-11 03:01:13'),
	('adm-appr2-01', 1, 4, 2, NULL, NULL, 'direktur.keuangan@ruasutamajaya.co.id', 'Dewi Lestari', '$2a$12$JMLTejdJx2vrz8vdvyOOaOqS//sI0AinLpAnQHxxNqVgDyaOunWUu', 'active', 'yes', 'no', NULL, NULL, '2026-01-03 08:00:00', '2026-09-11 03:01:13'),
	('adm-fin-01', 1, 2, 2, NULL, NULL, 'finance@ruasutamajaya.co.id', 'Siti Rahmawati', '$2a$12$JMLTejdJx2vrz8vdvyOOaOqS//sI0AinLpAnQHxxNqVgDyaOunWUu', 'active', 'yes', 'no', NULL, NULL, '2026-01-03 08:00:00', '2026-09-11 03:01:13'),
	('adm-staff-01', 2, 5, 1, NULL, NULL, 'staff.ops@ruasutamajaya.co.id', 'Rian Hidayat', '$2a$12$JMLTejdJx2vrz8vdvyOOaOqS//sI0AinLpAnQHxxNqVgDyaOunWUu', 'active', 'yes', 'no', NULL, NULL, '2026-01-03 08:00:00', '2026-09-11 03:01:13'),
	('adm-super-01', 1, 1, 3, NULL, NULL, 'superadmin@ruasutamajaya.co.id', 'Budi Santoso', '$2a$12$JMLTejdJx2vrz8vdvyOOaOqS//sI0AinLpAnQHxxNqVgDyaOunWUu', 'active', 'yes', 'no', NULL, NULL, '2026-01-03 08:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_admin_signature
CREATE TABLE IF NOT EXISTS `t_admin_signature` (
  `signature_id` int NOT NULL AUTO_INCREMENT,
  `signature_ref_admin` varchar(144) NOT NULL,
  `signature_name_pic` varchar(196) NOT NULL,
  `signature_file` varchar(2048) NOT NULL,
  `signature_create_date` datetime NOT NULL,
  `signature_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`signature_id`),
  KEY `idx_signature_admin` (`signature_ref_admin`),
  CONSTRAINT `fk_signature_admin` FOREIGN KEY (`signature_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_admin_signature: ~2 rows (approximately)
INSERT INTO `t_admin_signature` (`signature_id`, `signature_ref_admin`, `signature_name_pic`, `signature_file`, `signature_create_date`, `signature_modify_date`) VALUES
	(1, 'adm-appr1-01', 'Agus Prabowo', '/uploads/signatures/agus-prabowo.png', '2026-01-04 08:00:00', '2026-09-11 03:01:13'),
	(2, 'adm-appr2-01', 'Dewi Lestari', '/uploads/signatures/dewi-lestari.png', '2026-01-04 08:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_client
CREATE TABLE IF NOT EXISTS `t_client` (
  `client_id` int NOT NULL AUTO_INCREMENT,
  `client_ref_region` int NOT NULL,
  `client_name` varchar(196) NOT NULL,
  `client_phone` varchar(196) NOT NULL,
  `client_email` varchar(196) NOT NULL,
  `client_password` varchar(296) NOT NULL,
  `client_reset_code` varchar(196) DEFAULT NULL,
  `client_token` varchar(196) DEFAULT NULL,
  `client_active` varchar(20) NOT NULL DEFAULT 'no',
  `client_address` varchar(4048) NOT NULL,
  `client_create_date` datetime NOT NULL,
  `client_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`client_id`),
  UNIQUE KEY `uq_client_email` (`client_email`),
  KEY `idx_client_region` (`client_ref_region`),
  CONSTRAINT `fk_client_region` FOREIGN KEY (`client_ref_region`) REFERENCES `t_region` (`region_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=80 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_client: ~2 rows (approximately)
INSERT INTO `t_client` (`client_id`, `client_ref_region`, `client_name`, `client_phone`, `client_email`, `client_password`, `client_reset_code`, `client_token`, `client_active`, `client_address`, `client_create_date`, `client_modify_date`) VALUES
	(1, 1, 'PT Sumber Jaya Konstruksi', '021-5550001', 'contact@sumberjaya.co.id', '$2y$10$examplehashedpassword000000000000000000000000000000', NULL, NULL, 'yes', 'Jl. Gatot Subroto No. 10, Jakarta Selatan', '2026-01-05 08:00:00', '2026-09-11 03:01:13'),
	(2, 2, 'CV Mitra Bangun Jaya', '031-5550002', 'admin@mitrabangunjaya.co.id', '$2y$10$examplehashedpassword000000000000000000000000000000', NULL, NULL, 'yes', 'Jl. Diponegoro No. 25, Surabaya', '2026-01-05 08:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_division
CREATE TABLE IF NOT EXISTS `t_division` (
  `division_id` int NOT NULL AUTO_INCREMENT,
  `division_title` varchar(255) NOT NULL,
  `division_code` varchar(20) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT (now()),
  `updated_at` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`division_id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_division: ~3 rows (approximately)
INSERT INTO `t_division` (`division_id`, `division_title`, `division_code`, `created_at`, `updated_at`) VALUES
	(1, 'Operasional', 'OPS', '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(2, 'Keuangan', 'FIN', '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(3, 'Information Technology', 'IT', '2026-01-01 08:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_document
CREATE TABLE IF NOT EXISTS `t_document` (
  `document_id` int NOT NULL AUTO_INCREMENT,
  `document_ref_po` varchar(196) NOT NULL,
  `document_title` varchar(196) NOT NULL,
  `document_file` varchar(2048) NOT NULL,
  `document_create_date` datetime NOT NULL,
  `document_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`document_id`),
  KEY `idx_document_po` (`document_ref_po`),
  CONSTRAINT `fk_document_po` FOREIGN KEY (`document_ref_po`) REFERENCES `t_po` (`po_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=6700 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_document: ~0 rows (approximately)
INSERT INTO `t_document` (`document_id`, `document_ref_po`, `document_title`, `document_file`, `document_create_date`, `document_modify_date`) VALUES
	(1, 'PO20260001', 'Surat Perintah Kerja', '/uploads/po/spk-po20260001.pdf', '2026-01-06 09:10:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_po
CREATE TABLE IF NOT EXISTS `t_po` (
  `po_id` varchar(196) NOT NULL,
  `po_order_num` varchar(196) NOT NULL,
  `po_invoice` varchar(196) DEFAULT NULL,
  `po_ref_client` int NOT NULL,
  `po_ref_admin` varchar(144) NOT NULL COMMENT 'admin pembuat PO',
  `po_ref_pic` varchar(144) NOT NULL COMMENT 'admin PIC yang menangani PO (dulu: po_ref_admin_client)',
  `po_ref_pic_client` varchar(144) DEFAULT NULL COMMENT 'PIC yang menangani relasi ke klien, terpisah dari po_ref_pic (PIC project)',
  `po_ref_division` int DEFAULT NULL,
  `po_ref_region` int NOT NULL,
  `po_ref_ppn` int NOT NULL,
  `po_client_name` varchar(196) NOT NULL,
  `po_client_email` varchar(196) NOT NULL,
  `po_client_phone` varchar(196) NOT NULL,
  `po_client_address` varchar(4048) NOT NULL,
  `po_subclient` varchar(196) DEFAULT NULL,
  `po_subtotal` decimal(15,2) NOT NULL DEFAULT '0.00',
  `po_ppn_rate` decimal(5,2) NOT NULL DEFAULT '0.00' COMMENT 'snapshot persentase ppn yang dipakai',
  `po_ppn_amount` decimal(15,2) NOT NULL DEFAULT '0.00',
  `po_total` decimal(15,2) NOT NULL DEFAULT '0.00',
  `po_item_total` int NOT NULL DEFAULT '0',
  `po_status` varchar(50) NOT NULL,
  `po_document` varchar(10) NOT NULL DEFAULT 'no',
  `po_paid` varchar(10) NOT NULL DEFAULT 'no',
  `po_notes` varchar(8143) DEFAULT NULL,
  `po_date` date DEFAULT NULL,
  `po_exp_date` date DEFAULT NULL,
  `po_prepared_date` datetime DEFAULT NULL,
  `po_progress_date` datetime DEFAULT NULL,
  `po_complete_date` datetime DEFAULT NULL,
  `po_cancel_date` datetime DEFAULT NULL,
  `po_create_date` datetime NOT NULL,
  `po_modify_date` timestamp NOT NULL DEFAULT (now()),
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_po: ~0 rows (approximately)
INSERT INTO `t_po` (`po_id`, `po_order_num`, `po_invoice`, `po_ref_client`, `po_ref_admin`, `po_ref_pic`, `po_ref_pic_client`, `po_ref_division`, `po_ref_region`, `po_ref_ppn`, `po_client_name`, `po_client_email`, `po_client_phone`, `po_client_address`, `po_subclient`, `po_subtotal`, `po_ppn_rate`, `po_ppn_amount`, `po_total`, `po_item_total`, `po_status`, `po_document`, `po_paid`, `po_notes`, `po_date`, `po_exp_date`, `po_prepared_date`, `po_progress_date`, `po_complete_date`, `po_cancel_date`, `po_create_date`, `po_modify_date`) VALUES
	('PO20260001', 'ORD/2026/0001', NULL, 1, 'adm-super-01', 'adm-appr1-01', NULL, 1, 1, 1, 'PT Sumber Jaya Konstruksi', 'contact@sumberjaya.co.id', '021-5550001', 'Jl. Gatot Subroto No. 10, Jakarta Selatan', NULL, 50000000.00, 11.00, 5500000.00, 55500000.00, 2, 'progress', 'no', 'no', 'PO pemeliharaan ruas jalan tol seksi 1', '2026-01-06', NULL, NULL, NULL, NULL, NULL, '2026-01-06 09:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_po_item
CREATE TABLE IF NOT EXISTS `t_po_item` (
  `item_id` int NOT NULL AUTO_INCREMENT,
  `item_ref_po` varchar(196) NOT NULL,
  `item_ref_unit` int NOT NULL,
  `item_product` varchar(2048) NOT NULL,
  `item_desc` varchar(8048) DEFAULT NULL,
  `item_qty` int NOT NULL,
  `item_price` decimal(15,2) NOT NULL,
  `item_create_date` datetime NOT NULL,
  `item_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`item_id`),
  KEY `idx_item_po` (`item_ref_po`),
  KEY `idx_item_unit` (`item_ref_unit`),
  CONSTRAINT `fk_item_po` FOREIGN KEY (`item_ref_po`) REFERENCES `t_po` (`po_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_item_unit` FOREIGN KEY (`item_ref_unit`) REFERENCES `t_unit` (`unit_id`) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=12709 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_po_item: ~2 rows (approximately)
INSERT INTO `t_po_item` (`item_id`, `item_ref_po`, `item_ref_unit`, `item_product`, `item_desc`, `item_qty`, `item_price`, `item_create_date`, `item_modify_date`) VALUES
	(1, 'PO20260001', 2, 'Aspal hotmix', 'Pengerasan permukaan jalan', 1000, 40000.00, '2026-01-06 09:00:00', '2026-09-11 03:01:13'),
	(2, 'PO20260001', 3, 'Rambu lalu lintas sementara', 'Pengaman area kerja', 10, 1000000.00, '2026-01-06 09:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_ppn
CREATE TABLE IF NOT EXISTS `t_ppn` (
  `ppn_id` int NOT NULL AUTO_INCREMENT,
  `ppn_value` decimal(5,2) NOT NULL COMMENT 'persentase, misal 11.00 = 11%',
  `ppn_create_date` datetime NOT NULL,
  `ppn_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`ppn_id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_ppn: ~2 rows (approximately)
INSERT INTO `t_ppn` (`ppn_id`, `ppn_value`, `ppn_create_date`, `ppn_modify_date`) VALUES
	(1, 11.00, '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(2, 0.00, '2026-01-01 08:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_pr_admin_counter
CREATE TABLE IF NOT EXISTS `t_pr_admin_counter` (
  `counter_ref_admin` varchar(144) NOT NULL,
  `counter_last_seq` int NOT NULL DEFAULT '0',
  PRIMARY KEY (`counter_ref_admin`),
  CONSTRAINT `fk_pr_counter_admin` FOREIGN KEY (`counter_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_pr_admin_counter: ~0 rows (approximately)
INSERT INTO `t_pr_admin_counter` (`counter_ref_admin`, `counter_last_seq`) VALUES
	('adm-staff-01', 2);

-- Dumping structure for table rms_new_pr.t_pr_approval
CREATE TABLE IF NOT EXISTS `t_pr_approval` (
  `approval_id` int NOT NULL AUTO_INCREMENT,
  `approval_ref_admin` varchar(144) NOT NULL COMMENT 'admin approver',
  `approval_ref_pr` int NOT NULL,
  `approval_level` int NOT NULL DEFAULT '1' COMMENT 'urutan approval berjenjang, 1 = level pertama',
  `approval_round` int NOT NULL DEFAULT '1' COMMENT 'bertambah tiap kali PR direvisi & disubmit ulang',
  `approval_type` varchar(100) NOT NULL COMMENT 'mis. manager/finance/director - disesuaikan struktur organisasi',
  `approval_status` varchar(20) NOT NULL DEFAULT 'pending' COMMENT 'pending/approved/rejected/revision_requested',
  `approval_signature_ref` int DEFAULT NULL COMMENT 'FK ke t_admin_signature - tanda tangan yang dipakai saat approve',
  `approval_notes` varchar(2048) DEFAULT NULL,
  `approval_create_date` datetime NOT NULL,
  PRIMARY KEY (`approval_id`),
  UNIQUE KEY `uq_pr_approval_level_round` (`approval_ref_pr`,`approval_level`,`approval_round`),
  KEY `idx_approval_pr` (`approval_ref_pr`),
  KEY `idx_approval_admin` (`approval_ref_admin`),
  KEY `fk_approval_signature` (`approval_signature_ref`),
  CONSTRAINT `fk_approval_admin` FOREIGN KEY (`approval_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_approval_pr` FOREIGN KEY (`approval_ref_pr`) REFERENCES `t_purchase_request` (`pr_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_approval_signature` FOREIGN KEY (`approval_signature_ref`) REFERENCES `t_admin_signature` (`signature_id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `chk_approval_status` CHECK ((`approval_status` in (_utf8mb4'pending',_utf8mb4'approved',_utf8mb4'rejected',_utf8mb4'revision_requested')))
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_pr_approval: ~0 rows (approximately)
INSERT INTO `t_pr_approval` (`approval_id`, `approval_ref_admin`, `approval_ref_pr`, `approval_level`, `approval_round`, `approval_type`, `approval_status`, `approval_signature_ref`, `approval_notes`, `approval_create_date`) VALUES
	(1, 'adm-appr1-01', 1, 1, 1, 'manager', 'approved', 1, 'Disetujui, sesuai kebutuhan lapangan', '2026-01-08 10:00:00'),
	(2, 'adm-appr2-01', 1, 2, 1, 'finance', 'approved', 2, 'Disetujui oleh keuangan', '2026-01-08 14:00:00');

-- Dumping structure for table rms_new_pr.t_pr_comment
CREATE TABLE IF NOT EXISTS `t_pr_comment` (
  `comment_id` int NOT NULL AUTO_INCREMENT,
  `comment_ref_admin` varchar(144) NOT NULL,
  `comment_ref_pr` int NOT NULL,
  `comment_text` varchar(4048) NOT NULL,
  `comment_type` varchar(50) NOT NULL DEFAULT 'general' COMMENT 'general/revision_request/internal_note, dst',
  `comment_create_date` datetime NOT NULL,
  PRIMARY KEY (`comment_id`),
  KEY `idx_pr_comment_pr` (`comment_ref_pr`),
  KEY `idx_pr_comment_admin` (`comment_ref_admin`),
  CONSTRAINT `fk_pr_comment_admin` FOREIGN KEY (`comment_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_pr_comment_pr` FOREIGN KEY (`comment_ref_pr`) REFERENCES `t_purchase_request` (`pr_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_pr_comment: ~0 rows (approximately)
INSERT INTO `t_pr_comment` (`comment_id`, `comment_ref_admin`, `comment_ref_pr`, `comment_text`, `comment_type`, `comment_create_date`) VALUES
	(1, 'adm-appr1-01', 1, 'Mohon lampirkan quotation vendor terbaru', 'internal_note', '2026-01-08 09:30:00');

-- Dumping structure for table rms_new_pr.t_pr_document
CREATE TABLE IF NOT EXISTS `t_pr_document` (
  `document_id` int NOT NULL AUTO_INCREMENT,
  `document_type` varchar(100) NOT NULL COMMENT 'mis. quotation/invoice/kontrak/lainnya',
  `document_file_name` varchar(196) NOT NULL,
  `document_file_path` varchar(2048) NOT NULL,
  `document_ref_admin` varchar(144) NOT NULL COMMENT 'admin yang mengunggah',
  `document_ref_pr` int NOT NULL,
  `document_create_date` datetime NOT NULL,
  PRIMARY KEY (`document_id`),
  KEY `idx_pr_document_pr` (`document_ref_pr`),
  KEY `idx_pr_document_admin` (`document_ref_admin`),
  CONSTRAINT `fk_pr_document_admin` FOREIGN KEY (`document_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_pr_document_pr` FOREIGN KEY (`document_ref_pr`) REFERENCES `t_purchase_request` (`pr_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_pr_document: ~0 rows (approximately)
INSERT INTO `t_pr_document` (`document_id`, `document_type`, `document_file_name`, `document_file_path`, `document_ref_admin`, `document_ref_pr`, `document_create_date`) VALUES
	(1, 'quotation', 'quotation-vendor-aspal.pdf', '/uploads/pr/quotation-vendor-aspal.pdf', 'adm-staff-01', 1, '2026-01-07 08:30:00');

-- Dumping structure for table rms_new_pr.t_pr_payment
CREATE TABLE IF NOT EXISTS `t_pr_payment` (
  `payment_id` int NOT NULL AUTO_INCREMENT,
  `payment_ref_pr` int NOT NULL,
  `payment_ref_admin_input` varchar(144) NOT NULL COMMENT 'admin yang mencatat/mengajukan pembayaran ini',
  `payment_stage` varchar(100) NOT NULL COMMENT 'mis. down_payment/full_payment/final_payment',
  `payment_amount` decimal(15,2) NOT NULL DEFAULT '0.00',
  `payment_type` varchar(50) NOT NULL COMMENT 'mis. transfer/cash/giro',
  `payment_bank` varchar(196) DEFAULT NULL,
  `payment_bank_account_no` varchar(100) DEFAULT NULL,
  `payment_bank_account_name` varchar(196) DEFAULT NULL,
  `payment_priority_date` date DEFAULT NULL,
  `payment_status` varchar(20) NOT NULL DEFAULT 'pending' COMMENT 'pending/paid/cancelled',
  `payment_paid_date` datetime DEFAULT NULL,
  `payment_ref_admin_paid` varchar(144) DEFAULT NULL COMMENT 'admin yang mengonfirmasi pembayaran sudah dilakukan',
  `payment_create_date` datetime NOT NULL,
  `payment_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`payment_id`),
  KEY `idx_pr_payment_pr` (`payment_ref_pr`),
  KEY `idx_pr_payment_admin_input` (`payment_ref_admin_input`),
  KEY `idx_pr_payment_admin_paid` (`payment_ref_admin_paid`),
  CONSTRAINT `fk_pr_payment_admin_input` FOREIGN KEY (`payment_ref_admin_input`) REFERENCES `t_admin` (`admin_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_pr_payment_admin_paid` FOREIGN KEY (`payment_ref_admin_paid`) REFERENCES `t_admin` (`admin_id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_pr_payment_pr` FOREIGN KEY (`payment_ref_pr`) REFERENCES `t_purchase_request` (`pr_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_pr_payment: ~0 rows (approximately)
INSERT INTO `t_pr_payment` (`payment_id`, `payment_ref_pr`, `payment_ref_admin_input`, `payment_stage`, `payment_amount`, `payment_type`, `payment_bank`, `payment_bank_account_no`, `payment_bank_account_name`, `payment_priority_date`, `payment_status`, `payment_paid_date`, `payment_ref_admin_paid`, `payment_create_date`, `payment_modify_date`) VALUES
	(1, 1, 'adm-fin-01', 'down_payment', 20000000.00, 'transfer', 'Bank Mandiri', '1234567890', 'PT Ruas Utama Jaya', '2026-01-10', 'pending', NULL, NULL, '2026-01-09 08:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_pr_status_history
CREATE TABLE IF NOT EXISTS `t_pr_status_history` (
  `history_id` int NOT NULL AUTO_INCREMENT,
  `history_ref_admin` varchar(144) NOT NULL COMMENT 'admin yang mengubah status',
  `history_ref_pr` int NOT NULL,
  `history_from_status` varchar(50) DEFAULT NULL,
  `history_to_status` varchar(50) NOT NULL,
  `history_notes` varchar(2048) DEFAULT NULL,
  `history_create_date` datetime NOT NULL,
  PRIMARY KEY (`history_id`),
  KEY `idx_history_pr` (`history_ref_pr`),
  KEY `idx_history_admin` (`history_ref_admin`),
  CONSTRAINT `fk_history_admin` FOREIGN KEY (`history_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_history_pr` FOREIGN KEY (`history_ref_pr`) REFERENCES `t_purchase_request` (`pr_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_pr_status_history: ~2 rows (approximately)
INSERT INTO `t_pr_status_history` (`history_id`, `history_ref_admin`, `history_ref_pr`, `history_from_status`, `history_to_status`, `history_notes`, `history_create_date`) VALUES
	(1, 'adm-staff-01', 1, 'draft', 'submitted', 'PR diajukan untuk approval', '2026-01-07 08:05:00'),
	(2, 'adm-appr1-01', 1, 'submitted', 'approved', 'Disetujui level 1 & 2', '2026-01-08 14:00:00');

-- Dumping structure for table rms_new_pr.t_purchase_request
CREATE TABLE IF NOT EXISTS `t_purchase_request` (
  `pr_id` int NOT NULL AUTO_INCREMENT,
  `pr_ref_admin` varchar(144) NOT NULL COMMENT 'admin/requester pembuat PR',
  `pr_ref_responsible` int NOT NULL,
  `pr_rfp_no` varchar(196) NOT NULL,
  `pr_description_item` varchar(4048) NOT NULL,
  `pr_subclient` varchar(196) DEFAULT NULL,
  `pr_requested_amount` decimal(15,2) NOT NULL DEFAULT '0.00',
  `pr_qout_no` varchar(196) DEFAULT NULL COMMENT 'nomor quotation vendor',
  `pr_po_amount` decimal(15,2) NOT NULL DEFAULT '0.00',
  `pr_hpp` decimal(15,2) NOT NULL DEFAULT '0.00' COMMENT 'harga pokok penjualan',
  `pr_target_invoice_date` date DEFAULT NULL,
  `pr_priority` varchar(20) DEFAULT NULL COMMENT 'low/medium/high/urgent - hanya boleh ditetapkan oleh approver level 1',
  `pr_priority_ref_admin` varchar(144) DEFAULT NULL COMMENT 'admin approver level 1 yang menetapkan/mengubah priority',
  `pr_priority_date` datetime DEFAULT NULL COMMENT 'kapan priority terakhir ditetapkan/diubah',
  `pr_signature_ref` int DEFAULT NULL,
  `pr_status` varchar(50) NOT NULL DEFAULT 'draft' COMMENT 'draft/submitted/approved/rejected/revision/completed/cancelled',
  `pr_create_date` datetime NOT NULL,
  `pr_modify_date` timestamp NOT NULL DEFAULT (now()),
  `pr_ref_previous_pr` int DEFAULT NULL COMMENT 'referensi ke PR sebelumnya dalam siklus pembayaran bertahap (DP -> Progress -> Pelunasan), NULL jika PR pertama dalam siklus',
  PRIMARY KEY (`pr_id`),
  UNIQUE KEY `uq_pr_rfp_no` (`pr_rfp_no`),
  KEY `idx_pr_admin` (`pr_ref_admin`),
  KEY `idx_pr_responsible` (`pr_ref_responsible`),
  KEY `idx_pr_status` (`pr_status`),
  KEY `idx_pr_priority_admin` (`pr_priority_ref_admin`),
  KEY `fk_pr_signature` (`pr_signature_ref`),
  KEY `idx_pr_ref_previous` (`pr_ref_previous_pr`),
  CONSTRAINT `fk_pr_admin` FOREIGN KEY (`pr_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_pr_previous` FOREIGN KEY (`pr_ref_previous_pr`) REFERENCES `t_purchase_request` (`pr_id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_pr_priority_admin` FOREIGN KEY (`pr_priority_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_pr_responsible` FOREIGN KEY (`pr_ref_responsible`) REFERENCES `t_responsible` (`responsible_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_pr_signature` FOREIGN KEY (`pr_signature_ref`) REFERENCES `t_admin_signature` (`signature_id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `chk_pr_priority` CHECK (((`pr_priority` in (_utf8mb4'low',_utf8mb4'medium',_utf8mb4'high',_utf8mb4'urgent')) or (`pr_priority` is null))),
  CONSTRAINT `chk_pr_status` CHECK ((`pr_status` in (_utf8mb4'draft',_utf8mb4'submitted',_utf8mb4'approved',_utf8mb4'rejected',_utf8mb4'revision',_utf8mb4'completed',_utf8mb4'cancelled')))
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_purchase_request: ~0 rows (approximately)
INSERT INTO `t_purchase_request` (`pr_id`, `pr_ref_admin`, `pr_ref_responsible`, `pr_rfp_no`, `pr_description_item`, `pr_subclient`, `pr_requested_amount`, `pr_qout_no`, `pr_po_amount`, `pr_hpp`, `pr_target_invoice_date`, `pr_priority`, `pr_priority_ref_admin`, `pr_priority_date`, `pr_signature_ref`, `pr_status`, `pr_create_date`, `pr_modify_date`, `pr_ref_previous_pr`) VALUES
	(1, 'adm-staff-01', 1, 'RFP-2026-0001', 'Pengadaan material aspal hotmix untuk pemeliharaan ruas tol seksi 1', NULL, 45000000.00, 'QUOT-2026-0001', 50000000.00, 42000000.00, '2026-02-15', 'high', 'adm-appr1-01', '2026-01-08 10:00:00', 2, 'approved', '2026-01-07 08:00:00', '2026-09-11 03:01:13', NULL),
	(2, 'adm-staff-01', 2, 'RFP-2026-0002', 'Pengadaan rambu lalu lintas tambahan untuk area kerja', NULL, 10000000.00, NULL, 0.00, 0.00, NULL, NULL, NULL, NULL, NULL, 'draft', '2026-01-09 08:00:00', '2026-09-11 03:01:13', NULL);

-- Dumping structure for table rms_new_pr.t_region
CREATE TABLE IF NOT EXISTS `t_region` (
  `region_id` int NOT NULL AUTO_INCREMENT,
  `region_title` varchar(196) NOT NULL,
  `region_create_date` datetime NOT NULL,
  `region_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`region_id`)
) ENGINE=InnoDB AUTO_INCREMENT=24 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_region: ~3 rows (approximately)
INSERT INTO `t_region` (`region_id`, `region_title`, `region_create_date`, `region_modify_date`) VALUES
	(1, 'Jakarta', '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(2, 'Surabaya', '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(3, 'Bandung', '2026-01-01 08:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_responsible
CREATE TABLE IF NOT EXISTS `t_responsible` (
  `responsible_id` int NOT NULL AUTO_INCREMENT,
  `responsible_name` varchar(196) NOT NULL,
  `responsible_coa_code` varchar(50) NOT NULL COMMENT 'kode Chart of Accounts terkait',
  `responsible_status` varchar(20) NOT NULL DEFAULT 'active' COMMENT 'active/inactive',
  `responsible_create_date` datetime NOT NULL,
  `responsible_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`responsible_id`),
  UNIQUE KEY `uq_responsible_coa_code` (`responsible_coa_code`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_responsible: ~2 rows (approximately)
INSERT INTO `t_responsible` (`responsible_id`, `responsible_name`, `responsible_coa_code`, `responsible_status`, `responsible_create_date`, `responsible_modify_date`) VALUES
	(1, 'Kepala Divisi Operasional', 'COA-001', 'active', '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(2, 'Direktur Keuangan', 'COA-002', 'active', '2026-01-02 08:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_role
CREATE TABLE IF NOT EXISTS `t_role` (
  `role_id` int NOT NULL AUTO_INCREMENT,
  `role_title` varchar(156) NOT NULL,
  `role_slug` varchar(156) NOT NULL,
  `role_create_date` datetime NOT NULL,
  `role_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`role_id`),
  UNIQUE KEY `uq_role_slug` (`role_slug`)
) ENGINE=InnoDB AUTO_INCREMENT=14 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_role: ~5 rows (approximately)
INSERT INTO `t_role` (`role_id`, `role_title`, `role_slug`, `role_create_date`, `role_modify_date`) VALUES
	(1, 'Super Admin', 'super_admin', '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(2, 'Admin Keuangan', 'admin_keuangan', '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(3, 'Approver Level 1', 'approver_level_1', '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(4, 'Approver Level 2', 'approver_level_2', '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(5, 'Staff', 'staff', '2026-01-01 08:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_role_access
CREATE TABLE IF NOT EXISTS `t_role_access` (
  `role_access_id` int NOT NULL AUTO_INCREMENT,
  `role_access_ref_access` int NOT NULL,
  `role_access_ref_role` int NOT NULL,
  `role_access_create_date` datetime NOT NULL,
  `role_access_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`role_access_id`),
  UNIQUE KEY `uq_role_access` (`role_access_ref_role`,`role_access_ref_access`),
  KEY `fk_ra_access` (`role_access_ref_access`),
  CONSTRAINT `fk_ra_access` FOREIGN KEY (`role_access_ref_access`) REFERENCES `t_access` (`access_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_ra_role` FOREIGN KEY (`role_access_ref_role`) REFERENCES `t_role` (`role_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=197 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_role_access: ~73 rows (approximately)
INSERT INTO `t_role_access` (`role_access_id`, `role_access_ref_access`, `role_access_ref_role`, `role_access_create_date`, `role_access_modify_date`) VALUES
	(105, 80, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(106, 85, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(107, 63, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(108, 68, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(109, 67, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(110, 90, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(111, 84, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(112, 51, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(113, 60, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(114, 41, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(115, 54, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(116, 71, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(117, 64, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(118, 75, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(119, 83, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(120, 38, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(121, 87, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(122, 48, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(123, 44, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(124, 58, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(125, 53, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(126, 59, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(127, 62, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(128, 43, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(129, 66, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(130, 40, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(131, 89, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(132, 50, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(133, 46, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(134, 52, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(135, 61, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(136, 42, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(137, 55, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(138, 56, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(139, 65, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(140, 47, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(141, 76, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(142, 39, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(143, 88, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(144, 49, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(145, 45, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(146, 73, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(147, 78, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(148, 81, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(149, 57, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(150, 79, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(151, 77, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(152, 69, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(153, 70, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(154, 82, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(155, 72, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(156, 86, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(157, 74, 1, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(168, 68, 2, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(169, 84, 2, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(170, 83, 2, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(171, 73, 2, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(172, 78, 2, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(173, 69, 2, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(174, 74, 2, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(175, 74, 3, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(176, 79, 3, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(177, 81, 3, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(178, 80, 3, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(179, 74, 4, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(180, 79, 4, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(181, 81, 4, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(182, 80, 4, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(190, 85, 5, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(191, 75, 5, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(192, 76, 5, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(193, 77, 5, '2026-01-02 08:00:00', '2026-09-11 03:01:13'),
	(194, 86, 5, '2026-01-02 08:00:00', '2026-09-11 03:01:13');

-- Dumping structure for table rms_new_pr.t_unit
CREATE TABLE IF NOT EXISTS `t_unit` (
  `unit_id` int NOT NULL AUTO_INCREMENT,
  `unit_title` varchar(196) NOT NULL,
  `unit_create_date` datetime NOT NULL,
  `unit_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`unit_id`)
) ENGINE=InnoDB AUTO_INCREMENT=36 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Dumping data for table rms_new_pr.t_unit: ~3 rows (approximately)
INSERT INTO `t_unit` (`unit_id`, `unit_title`, `unit_create_date`, `unit_modify_date`) VALUES
	(1, 'Unit', '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(2, 'Meter', '2026-01-01 08:00:00', '2026-09-11 03:01:13'),
	(3, 'Buah', '2026-01-01 08:00:00', '2026-09-11 03:01:13');

/*!40103 SET TIME_ZONE=IFNULL(@OLD_TIME_ZONE, 'system') */;
/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IFNULL(@OLD_FOREIGN_KEY_CHECKS, 1) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40111 SET SQL_NOTES=IFNULL(@OLD_SQL_NOTES, 1) */;
