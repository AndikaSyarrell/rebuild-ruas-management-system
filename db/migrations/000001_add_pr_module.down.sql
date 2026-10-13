-- 000001_add_pr_module (DOWN) - urutan kebalikan dari UP
DROP TABLE IF EXISTS `t_pr_admin_counter`;
DROP TABLE IF EXISTS `t_pr_status_history`;
DROP TABLE IF EXISTS `t_pr_payment`;
DROP TABLE IF EXISTS `t_pr_document`;
DROP TABLE IF EXISTS `t_pr_comment`;
DROP TABLE IF EXISTS `t_pr_cancel_request`;
DROP TABLE IF EXISTS `t_pr_approval`;
DROP TABLE IF EXISTS `t_purchase_request`;
DROP TABLE IF EXISTS `t_pr_quotation`;
DROP TABLE IF EXISTS `t_admin_signature`;
DROP TABLE IF EXISTS `t_responsible`;

ALTER TABLE `t_po` DROP COLUMN `po_quot_no`;

ALTER TABLE `t_admin`
  DROP FOREIGN KEY `fk_admin_division`,
  DROP KEY `fk_admin_division`,
  DROP COLUMN `admin_ref_division`;

ALTER TABLE `t_division` DROP COLUMN `division_code`;
