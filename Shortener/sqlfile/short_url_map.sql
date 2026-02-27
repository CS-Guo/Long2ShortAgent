CREATE TABLE `short_url_map` (
                                 `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
                                 `create_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
                                 `create_by` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '创建者',
                                 `is_del` tinyint UNSIGNED NOT NULL DEFAULT '0' COMMENT '是否删除：0正常1删除',
                                 `lurl` varchar(2048) DEFAULT NULL COMMENT '⻓长链接',
                                 `md5` char(32) DEFAULT NULL COMMENT '⻓长链接MD5',
                                 `surl` varchar(11) DEFAULT NULL COMMENT '短链接',
                                 `expire_at` DATETIME NULL COMMENT '过期时间，NULL 表示永不过期',
                                 PRIMARY KEY (`id`),
                                 INDEX(`is_del`),
                                 UNIQUE(`md5`),
                                 UNIQUE(`surl`),
                                 KEY `idx_expire_at` (`expire_at`)
)ENGINE=INNODB DEFAULT CHARSET=utf8mb4 COMMENT = '⻓长短链映射表';