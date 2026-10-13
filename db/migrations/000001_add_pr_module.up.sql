-- ============================================================
-- 000001_add_pr_module (UP)
-- Dari: rms_structure.sql  ->  Ke: rms_with_pr_with_seed_.sql
-- Catatan: DDL MySQL = implicit commit (tidak bisa di-rollback).
-- Koleksi tabel baru memakai utf8mb4_general_ci agar FK ke
-- t_admin / t_po (collation lama) tidak error 3780.
-- ============================================================

-- ---------- A. Perubahan tabel lama ----------

ALTER TABLE `t_division`
  ADD COLUMN `division_code` varchar(20) NOT NULL DEFAULT '' AFTER `division_title`;

ALTER TABLE `t_admin`
  ADD COLUMN `admin_ref_division` int DEFAULT NULL AFTER `admin_ref_role`,
  ADD KEY `fk_admin_division` (`admin_ref_division`),
  ADD CONSTRAINT `fk_admin_division` FOREIGN KEY (`admin_ref_division`)
    REFERENCES `t_division` (`division_id`) ON DELETE SET NULL ON UPDATE CASCADE;

ALTER TABLE `t_po`
  ADD COLUMN `po_quot_no` varchar(196) DEFAULT NULL
    COMMENT 'Hint pencarian quotation kandidat saat linking dari sisi PO - informasional saja'
    AFTER `po_invoice`;

-- ---------- B. Tabel baru (urut sesuai dependensi FK) ----------

CREATE TABLE `t_responsible` (
  `responsible_id` int NOT NULL AUTO_INCREMENT,
  `responsible_name` varchar(196) NOT NULL,
  `responsible_coa_code` varchar(50) NOT NULL COMMENT 'kode Chart of Accounts terkait',
  `responsible_status` varchar(20) NOT NULL DEFAULT 'active' COMMENT 'active/inactive',
  `responsible_create_date` datetime NOT NULL,
  `responsible_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`responsible_id`),
  UNIQUE KEY `uq_responsible_coa_code` (`responsible_coa_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `t_admin_signature` (
  `signature_id` int NOT NULL AUTO_INCREMENT,
  `signature_ref_admin` varchar(144) NOT NULL,
  `signature_name_pic` varchar(196) NOT NULL,
  `signature_file` varchar(2048) NOT NULL,
  `signature_create_date` datetime NOT NULL,
  `signature_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`signature_id`),
  KEY `idx_signature_admin` (`signature_ref_admin`),
  CONSTRAINT `fk_signature_admin` FOREIGN KEY (`signature_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `t_pr_quotation` (
  `quotation_id` int NOT NULL AUTO_INCREMENT,
  `quotation_no` varchar(196) NOT NULL COMMENT 'no_quot ter-normalisasi: trim + uppercase',
  `quotation_ref_po` varchar(196) DEFAULT NULL COMMENT 'FK ke t_po - NULL = belum di-link',
  `quotation_link_ref_admin` varchar(144) DEFAULT NULL COMMENT 'admin yang melakukan link',
  `quotation_link_date` datetime DEFAULT NULL,
  `quotation_revoke_ref_admin` varchar(144) DEFAULT NULL COMMENT 'admin yang melakukan unlink (manual/cascade)',
  `quotation_revoke_date` datetime DEFAULT NULL,
  `quotation_create_date` datetime NOT NULL,
  `quotation_modify_date` timestamp NOT NULL DEFAULT (now()),
  PRIMARY KEY (`quotation_id`),
  UNIQUE KEY `uq_quotation_no` (`quotation_no`),
  KEY `idx_quotation_po` (`quotation_ref_po`),
  KEY `idx_quotation_link_admin` (`quotation_link_ref_admin`),
  KEY `idx_quotation_revoke_admin` (`quotation_revoke_ref_admin`),
  CONSTRAINT `fk_quotation_link_admin` FOREIGN KEY (`quotation_link_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_quotation_po` FOREIGN KEY (`quotation_ref_po`) REFERENCES `t_po` (`po_id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_quotation_revoke_admin` FOREIGN KEY (`quotation_revoke_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `t_purchase_request` (
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
  `pr_priority` varchar(20) DEFAULT NULL COMMENT 'low/medium/high - ditetapkan requester saat membuat PR',
  `pr_priority_ref_admin` varchar(144) DEFAULT NULL COMMENT 'admin approver level 1 yang menetapkan/mengubah priority',
  `pr_priority_date` datetime DEFAULT NULL COMMENT 'kapan priority terakhir ditetapkan/diubah',
  `pr_signature_ref` int DEFAULT NULL,
  `pr_status` varchar(50) NOT NULL DEFAULT 'draft' COMMENT 'draft/submitted/approved/rejected/revision/completed/cancelled',
  `pr_create_date` datetime NOT NULL,
  `pr_modify_date` timestamp NOT NULL DEFAULT (now()),
  `pr_ref_previous_pr` int DEFAULT NULL COMMENT 'referensi ke PR sebelumnya dalam siklus pembayaran bertahap (DP -> Progress -> Pelunasan), NULL jika PR pertama dalam siklus',
  `pr_ref_quotation` int DEFAULT NULL COMMENT 'FK ke t_pr_quotation - grup pengelompokan berbasis no_quot',
  `pr_po_no` varchar(196) DEFAULT NULL COMMENT 'Nomor PO/SPK aktual dari klien, diisi finance setelah PO release. NULL = PO belum release.',
  PRIMARY KEY (`pr_id`),
  UNIQUE KEY `uq_pr_rfp_no` (`pr_rfp_no`),
  KEY `idx_pr_admin` (`pr_ref_admin`),
  KEY `idx_pr_responsible` (`pr_ref_responsible`),
  KEY `idx_pr_status` (`pr_status`),
  KEY `idx_pr_priority_admin` (`pr_priority_ref_admin`),
  KEY `fk_pr_signature` (`pr_signature_ref`),
  KEY `idx_pr_ref_previous` (`pr_ref_previous_pr`),
  KEY `idx_pr_quotation` (`pr_ref_quotation`),
  CONSTRAINT `fk_pr_admin` FOREIGN KEY (`pr_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_pr_previous` FOREIGN KEY (`pr_ref_previous_pr`) REFERENCES `t_purchase_request` (`pr_id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_pr_priority_admin` FOREIGN KEY (`pr_priority_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE SET NULL ON UPDATE CASCADE,
  CONSTRAINT `fk_pr_quotation` FOREIGN KEY (`pr_ref_quotation`) REFERENCES `t_pr_quotation` (`quotation_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_pr_responsible` FOREIGN KEY (`pr_ref_responsible`) REFERENCES `t_responsible` (`responsible_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_pr_signature` FOREIGN KEY (`pr_signature_ref`) REFERENCES `t_admin_signature` (`signature_id`) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `t_pr_approval` (
  `approval_id` int NOT NULL AUTO_INCREMENT,
  `approval_ref_admin` varchar(144) DEFAULT NULL COMMENT 'NULL = level general (director/finance) yang belum diputuskan',
  `approval_ref_pr` int NOT NULL,
  `approval_level` int NOT NULL DEFAULT '1' COMMENT 'urutan approval berjenjang, 1 = level pertama',
  `approval_round` int NOT NULL DEFAULT '1' COMMENT 'bertambah tiap kali PR direvisi & disubmit ulang',
  `approval_type` varchar(100) NOT NULL COMMENT 'mis. manager/finance/director - disesuaikan struktur organisasi',
  `approval_status` varchar(20) NOT NULL DEFAULT 'pending' COMMENT 'pending/approved/rejected/revision_requested',
  `approval_signature_ref` int DEFAULT NULL COMMENT 'FK ke t_admin_signature - tanda tangan yang dipakai saat approve',
  `approval_notes` varchar(2048) DEFAULT NULL,
  `approval_decided_date` datetime DEFAULT NULL,
  `approval_create_date` datetime NOT NULL,
  PRIMARY KEY (`approval_id`),
  UNIQUE KEY `uq_pr_approval_level_round` (`approval_ref_pr`,`approval_level`,`approval_round`),
  KEY `idx_approval_pr` (`approval_ref_pr`),
  KEY `idx_approval_admin` (`approval_ref_admin`),
  KEY `fk_approval_signature` (`approval_signature_ref`),
  CONSTRAINT `fk_approval_admin` FOREIGN KEY (`approval_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_approval_pr` FOREIGN KEY (`approval_ref_pr`) REFERENCES `t_purchase_request` (`pr_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_approval_signature` FOREIGN KEY (`approval_signature_ref`) REFERENCES `t_admin_signature` (`signature_id`) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `t_pr_cancel_request` (
  `cancel_id` int NOT NULL AUTO_INCREMENT,
  `cancel_ref_pr` int NOT NULL,
  `cancel_ref_admin` varchar(144) NOT NULL COMMENT 'requester',
  `cancel_reason` varchar(4048) NOT NULL,
  `cancel_document_name` varchar(196) NOT NULL COMMENT 'berita acara',
  `cancel_document_path` varchar(2048) NOT NULL,
  `cancel_status` varchar(20) NOT NULL DEFAULT 'pending' COMMENT 'pending/approved/rejected',
  `cancel_ref_admin_review` varchar(144) DEFAULT NULL COMMENT 'finance yang mereview',
  `cancel_review_notes` varchar(2048) DEFAULT NULL,
  `cancel_review_date` datetime DEFAULT NULL,
  `cancel_create_date` datetime NOT NULL,
  `cancel_pending_flag` tinyint GENERATED ALWAYS AS (if((`cancel_status` = _utf8mb4'pending'),1,NULL)) STORED,
  PRIMARY KEY (`cancel_id`),
  UNIQUE KEY `uq_cancel_one_pending` (`cancel_ref_pr`,`cancel_pending_flag`),
  KEY `idx_cancel_pr` (`cancel_ref_pr`),
  KEY `idx_cancel_status` (`cancel_status`),
  KEY `fk_cancel_admin` (`cancel_ref_admin`),
  KEY `fk_cancel_review_admin` (`cancel_ref_admin_review`),
  CONSTRAINT `fk_cancel_admin` FOREIGN KEY (`cancel_ref_admin`) REFERENCES `t_admin` (`admin_id`) ON DELETE RESTRICT ON UPDATE CASCADE,
  CONSTRAINT `fk_cancel_pr` FOREIGN KEY (`cancel_ref_pr`) REFERENCES `t_purchase_request` (`pr_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_cancel_review_admin` FOREIGN KEY (`cancel_ref_admin_review`) REFERENCES `t_admin` (`admin_id`) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `t_pr_comment` (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `t_pr_document` (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `t_pr_payment` (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `t_pr_status_history` (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `t_pr_admin_counter` (
  `counter_key` varchar(144) NOT NULL,
  `counter_last_seq` int NOT NULL DEFAULT '0',
  PRIMARY KEY (`counter_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
