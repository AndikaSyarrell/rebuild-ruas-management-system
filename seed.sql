-- =========================================================
-- SEED DATA - untuk kebutuhan testing lokal (Postman/dev)
-- Jalankan SETELAH rms_normalized.sql.
--
-- Membuat:
--   1. 1 region ("Head Office")
--   2. 1 role ("Super Admin", slug super_admin) dengan SELURUH access_slug
--      yang dipakai middleware.RequireAccess di internal/routes/routes.go
--   3. 1 admin aktif siap dipakai login:
--        email    : admin@rms.local
--        password : Admin123!
--
-- Hash password di bawah dibuat dengan:
--   go run ./cmd/hashpw "Admin123!"
-- Ganti passwordnya sendiri untuk lingkungan selain lokal/testing.
-- =========================================================

INSERT INTO T_Region (region_title, region_create_date)
VALUES ('Head Office', NOW());
SET @region_id = LAST_INSERT_ID();

INSERT INTO T_Role (role_title, role_slug, role_create_date)
VALUES ('Super Admin', 'super_admin', NOW());
SET @role_id = LAST_INSERT_ID();

-- Seluruh access_slug yang direferensikan middleware.RequireAccess(...) di routes.go
INSERT INTO T_Access (access_title, access_module, access_slug, access_create_date) VALUES
  ('Create Region',        'region',  'create_region',            NOW()),
  ('Edit Region',          'region',  'edit_region',               NOW()),
  ('Delete Region',        'region',  'delete_region',             NOW()),
  ('Create Division',      'division','create_division',          NOW()),
  ('Edit Division',        'division','edit_division',            NOW()),
  ('Delete Division',      'division','delete_division',          NOW()),
  ('Create Unit',          'unit',    'create_unit',               NOW()),
  ('Edit Unit',            'unit',    'edit_unit',                 NOW()),
  ('Delete Unit',          'unit',    'delete_unit',               NOW()),
  ('Edit PPN',             'ppn',     'edit_ppn',                  NOW()),
  ('Create New Admin',     'admin',   'create_new_admin',          NOW()),
  ('Edit Other Admin',     'admin',   'edit_other_admin',          NOW()),
  ('Edit PIC',             'admin',   'edit_pic',                  NOW()),
  ('Deactivate Other Admin','admin',  'deactivate_other_admin',    NOW()),
  ('Create Client',        'client',  'create_client',             NOW()),
  ('Edit Client',          'client',  'edit_client',               NOW()),
  ('Delete Client',        'client',  'delete_client',             NOW()),
  ('Change Client Active', 'client',  'change_stat_active',        NOW()),
  ('Create PO',            'po',      'create_po',                 NOW()),
  ('Edit PO',               'po',     'edit_po',                   NOW()),
  ('Delete PO',            'po',      'delete_po',                 NOW()),
  ('Change PO Paid',       'po',      'change_stat_paid',          NOW()),
  ('Update Invoice',       'po',      'update_invoice',            NOW()),
  ('Create Notes',         'po',      'create_notes',              NOW()),
  ('Upload Document',      'po',      'upload_document',           NOW()),
  ('Export PO',            'po',      'export_po',                 NOW()),
  ('View Report Table',    'report',  'view_report_table',         NOW());

-- Berikan SEMUA access di atas ke role Super Admin
INSERT INTO T_Role_Access (role_access_ref_role, role_access_ref_access, role_access_create_date)
SELECT @role_id, access_id, NOW() FROM T_Access;

-- Admin pertama: admin@rms.local / Admin123!
INSERT INTO T_Admin
  (admin_id, admin_ref_region, admin_ref_role, admin_email, admin_name, admin_password, admin_active, admin_pic, admin_pic_client, admin_create_date)
VALUES
  ('ADM000001', @region_id, @role_id, 'admin@rms.local', 'Super Admin',
   '$2a$12$SaA36MZlNv.A8Q0gfHPBTO38RtXwaC0NU7EycAkB.VIvITUow5fcu',
   'active', 'yes', 'no', NOW());

-- Data pelengkap supaya endpoint list langsung ada isinya saat dites
INSERT INTO T_Unit (unit_title, unit_create_date) VALUES ('Pcs', NOW()), ('Box', NOW());
INSERT INTO T_Ppn (ppn_value, ppn_create_date) VALUES (11.00, NOW());
INSERT INTO T_Division (division_title) VALUES ('Sales'), ('Operations');
