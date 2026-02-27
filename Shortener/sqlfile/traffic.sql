CREATE TABLE IF NOT EXISTS `short_url_click_event` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `event_id` VARCHAR(64) NOT NULL COMMENT '事件ID',
  `trace_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '链路ID',
  `request_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '请求ID',
  `short_code` VARCHAR(32) NOT NULL COMMENT '短链编码',
  `ip` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '请求IP',
  `ua` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT 'User-Agent',
  `referer` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '来源',
  `country` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '国家',
  `province` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '省份',
  `city` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '城市',
  `device_type` VARCHAR(32) NOT NULL DEFAULT 'unknown' COMMENT '设备类型',
  `occurred_at` DATETIME NOT NULL COMMENT '点击发生时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '入库时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_event_id` (`event_id`),
  KEY `idx_short_code_time` (`short_code`, `occurred_at`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='短链点击明细事件表';


CREATE TABLE IF NOT EXISTS `short_url_daily_stat` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `stat_date` DATE NOT NULL COMMENT '统计日期',
  `short_code` VARCHAR(32) NOT NULL COMMENT '短链编码',
  `pv` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '访问次数',
  `uv` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '独立访客数',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_date_short_code` (`stat_date`, `short_code`),
  KEY `idx_short_code_date` (`short_code`, `stat_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='短链按天聚合统计表';


CREATE TABLE IF NOT EXISTS `short_url_audit_log` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `task_id` VARCHAR(64) NOT NULL COMMENT 'AI任务ID',
  `request_id` VARCHAR(64) NOT NULL COMMENT '请求ID',
  `operator_id` VARCHAR(64) NOT NULL COMMENT '操作人ID',
  `tenant_id` VARCHAR(64) NOT NULL COMMENT '租户ID',
  `action` VARCHAR(64) NOT NULL COMMENT '动作类型:create/update/delete/query',
  `resource` VARCHAR(128) NOT NULL COMMENT '资源标识',
  `input_payload` JSON NULL COMMENT '输入参数',
  `result_payload` JSON NULL COMMENT '输出参数',
  `status` VARCHAR(32) NOT NULL COMMENT '执行状态',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_task_id` (`task_id`),
  KEY `idx_request_id` (`request_id`),
  KEY `idx_operator_id` (`operator_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI操作审计日志';

