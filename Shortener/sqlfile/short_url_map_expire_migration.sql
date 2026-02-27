SET @has_expire_col := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'short_url_map'
    AND COLUMN_NAME = 'expire_at'
);
SET @sql_add_col := IF(
  @has_expire_col = 0,
  'ALTER TABLE `short_url_map` ADD COLUMN `expire_at` DATETIME NULL COMMENT "过期时间，NULL 表示永不过期" AFTER `surl`',
  'SELECT 1'
);
PREPARE stmt_add_col FROM @sql_add_col;
EXECUTE stmt_add_col;
DEALLOCATE PREPARE stmt_add_col;

SET @has_expire_idx := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'short_url_map'
    AND INDEX_NAME = 'idx_expire_at'
);
SET @sql_add_idx := IF(
  @has_expire_idx = 0,
  'ALTER TABLE `short_url_map` ADD INDEX `idx_expire_at` (`expire_at`)',
  'SELECT 1'
);
PREPARE stmt_add_idx FROM @sql_add_idx;
EXECUTE stmt_add_idx;
DEALLOCATE PREPARE stmt_add_idx;
