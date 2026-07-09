TRUNCATE TABLE `casbin_rule`;

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
SELECT DISTINCT
  'p' AS `ptype`,
  `base_tenant`.`code` AS `v0`,
  `base_role`.`code` AS `v1`,
  `base_api`.`operation` AS `v2`,
  `base_api`.`method` AS `v3`,
  '*' AS `v4`,
  '' AS `v5`
FROM `base_role`
INNER JOIN `base_tenant`
  ON `base_tenant`.`id` = `base_role`.`tenant_id`
  AND `base_tenant`.`deleted_at` IS NULL
INNER JOIN JSON_TABLE(
  IFNULL(`base_role`.`menus`, JSON_ARRAY()),
  '$[*]' COLUMNS (
    `menu_id` BIGINT PATH '$'
  )
) AS `role_menu`
INNER JOIN `base_menu`
  ON `base_menu`.`id` = `role_menu`.`menu_id`
  AND `base_menu`.`deleted_at` IS NULL
INNER JOIN JSON_TABLE(
  IFNULL(`base_menu`.`api`, JSON_ARRAY()),
  '$[*]' COLUMNS (
    `operation` VARCHAR(100) PATH '$'
  )
) AS `menu_api`
INNER JOIN `base_api`
  ON BINARY `base_api`.`operation` = BINARY `menu_api`.`operation`
  AND `base_api`.`deleted_at` IS NULL
WHERE `base_role`.`deleted_at` IS NULL;
