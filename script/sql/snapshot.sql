DROP TABLE IF EXISTS `snapshot_info`;

CREATE TABLE `snapshot_info` (
    `id` bigint(20) unsigned not null auto_increment comment '自增id',
    `module` varchar(128) not null default '' comment '模块名',
    `branch` varchar(128) not null default '' comment '分支名',
    `commit` varchar(128) not null default '' comment '当前版本',
    `base_commit` varchar(128) not null default '' comment '基准版本',
    `snapshot_id` varchar(128) not null default '' comment '快照id',
    `report_partition_key`  text not null comment '报告数据分区key',
    `_created_time` datetime not null default current_timestamp comment '创建时间',
    `_updated_time` datetime not null default current_timestamp on update current_timestamp comment '更新时间',
    `_deleted` tinyint(1) not null default '0' comment '是否删除',
    primary key (`id`),
    key `idx_snapshot_id` (`snapshot_id`),
    key `idx_module_branch_commit` (`module`,`branch`, `commit`)
) engine=InnoDB default charset=utf8mb4 comment='覆盖率快照信息表';
