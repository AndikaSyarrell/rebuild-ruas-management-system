-- 000002 (DOWN): hapus hanya data yang ditambahkan UP.
-- role_access ikut terhapus otomatis via ON DELETE CASCADE.
DELETE FROM `t_access` WHERE `access_slug` IN (
  'create_pr','edit_pr','submit_pr','export_pr','update_pr_amounts',
  'create_pr_payment','confirm_pr_payment','cancel_pr','upload_pr_document',
  'checker','director','finance','export_pr_payment_report',
  'create_responsible','edit_responsible','delete_responsible','change_stat_responsible',
  'link_pr_po','unlink_pr_po'
);
DELETE FROM `t_responsible` WHERE `responsible_coa_code` IN ('COA-001','COA-002')
  AND NOT EXISTS (SELECT 1 FROM `t_purchase_request` p WHERE p.pr_ref_responsible = `t_responsible`.responsible_id);
