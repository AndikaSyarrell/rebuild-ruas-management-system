-- ============================================================
-- 000002_seed_pr_reference_data (UP)
-- Hanya data referensi yang dibutuhkan modul PR. Data dummy
-- (SEED-*, id 9xxx, PR/payment contoh) SENGAJA tidak dibawa.
-- Idempotent: aman dijalankan ulang. Id dicari lewat slug,
-- bukan hardcode, karena id di DB produksi bisa berbeda.
-- ============================================================

-- 1) Hak akses baru (unique key: access_slug)
INSERT IGNORE INTO `t_access`
  (`access_title`, `access_slug`, `access_module`, `access_sort`, `access_create_date`)
VALUES
  ('Buat PR',                           'create_pr',                'pr', 1,  NOW()),
  ('Ubah PR',                           'edit_pr',                  'pr', 2,  NOW()),
  ('Submit PR',                         'submit_pr',                'pr', 3,  NOW()),
  ('Export PR ke Excel',                'export_pr',                'pr', 4,  NOW()),
  ('Ubah Nominal PR',                   'update_pr_amounts',        'pr', 8,  NOW()),
  ('Buat Pembayaran PR',                'create_pr_payment',        'pr', 9,  NOW()),
  ('Konfirmasi Pembayaran PR',          'confirm_pr_payment',       'pr', 10, NOW()),
  ('Batalkan PR',                       'cancel_pr',                'pr', 11, NOW()),
  ('Upload Dokumen PR',                 'upload_pr_document',       'pr', 12, NOW()),
  ('Checker PR',                        'checker',                  'pr', 13, NOW()),
  ('Director PR',                       'director',                 'pr', 14, NOW()),
  ('Finance PR',                        'finance',                  'pr', 15, NOW()),
  ('Export Excel for Report',           'export_pr_payment_report', 'pr', 16, NOW()),
  ('Buat Penanggung Jawab',             'create_responsible',       'responsible', 1, NOW()),
  ('Ubah Penanggung Jawab',             'edit_responsible',         'responsible', 2, NOW()),
  ('Hapus Penanggung Jawab',            'delete_responsible',       'responsible', 3, NOW()),
  ('Aktif/Nonaktifkan Penanggung Jawab','change_stat_responsible',  'responsible', 4, NOW()),
  ('Link Purchase Request ke PO',       'link_pr_po',               'po', 20, NOW()),
  ('Unlink Purchase Request dari PO',   'unlink_pr_po',             'po', 21, NOW());

-- 2) Pemberian akses ke role (dicocokkan lewat slug role & slug access).
--    Role yang slug-nya tidak ada di DB lama otomatis dilewati.
INSERT IGNORE INTO `t_role_access`
  (`role_access_ref_role`, `role_access_ref_access`, `role_access_create_date`)
SELECT r.role_id, a.access_id, NOW()
FROM (
  -- super_admin: semua akses PR baru
  SELECT 'super_admin' AS role_slug, s AS access_slug FROM (
    SELECT 'create_pr' s UNION ALL SELECT 'edit_pr' UNION ALL SELECT 'submit_pr'
    UNION ALL SELECT 'export_pr' UNION ALL SELECT 'update_pr_amounts'
    UNION ALL SELECT 'create_pr_payment' UNION ALL SELECT 'confirm_pr_payment'
    UNION ALL SELECT 'cancel_pr' UNION ALL SELECT 'upload_pr_document'
    UNION ALL SELECT 'checker' UNION ALL SELECT 'director' UNION ALL SELECT 'finance'
    UNION ALL SELECT 'export_pr_payment_report'
    UNION ALL SELECT 'create_responsible' UNION ALL SELECT 'edit_responsible'
    UNION ALL SELECT 'delete_responsible' UNION ALL SELECT 'change_stat_responsible'
    UNION ALL SELECT 'link_pr_po' UNION ALL SELECT 'unlink_pr_po'
  ) x
  UNION ALL SELECT 'admin_keuangan', 'export_pr'
  UNION ALL SELECT 'admin_keuangan', 'create_pr_payment'
  UNION ALL SELECT 'admin_keuangan', 'confirm_pr_payment'
  UNION ALL SELECT 'admin_keuangan', 'finance'
  UNION ALL SELECT 'admin_keuangan', 'link_pr_po'
  UNION ALL SELECT 'admin_keuangan', 'unlink_pr_po'
  UNION ALL SELECT 'approver_level_1', 'checker'
  UNION ALL SELECT 'approver_level_2', 'director'
  UNION ALL SELECT 'staff', 'create_pr'
  UNION ALL SELECT 'staff', 'edit_pr'
  UNION ALL SELECT 'staff', 'submit_pr'
  UNION ALL SELECT 'staff', 'cancel_pr'
  UNION ALL SELECT 'staff', 'upload_pr_document'
) m
JOIN `t_role`   r ON r.role_slug   COLLATE utf8mb4_general_ci = m.role_slug   COLLATE utf8mb4_general_ci
JOIN `t_access` a ON a.access_slug COLLATE utf8mb4_general_ci = m.access_slug COLLATE utf8mb4_general_ci;

-- 3) Penanggung jawab awal (unique key: responsible_coa_code).
--    PR wajib punya responsible, jadi minimal satu harus ada.
--    GANTI nama & kode COA sesuai kondisi perusahaan.
INSERT IGNORE INTO `t_responsible`
  (`responsible_name`, `responsible_coa_code`, `responsible_status`, `responsible_create_date`)
VALUES
  ('Kepala Divisi Operasional', 'COA-001', 'active', NOW()),
  ('Direktur Keuangan',         'COA-002', 'active', NOW());
