
DROP TABLE IF EXISTS `systest_task`;

CREATE TABLE `systest_task` (
    `id` bigint(20) unsigned not null auto_increment comment '自增id',
    `language` varchar(8) not null default '' comment '语言',
    `module` varchar(128) not null default '' comment '模块名',
    `branch` varchar(128) not null default '' comment '分支名',
    `commit` varchar(128) not null default '' comment '当前版本',
    `commit_create_time` datetime not null default current_timestamp comment '当前版本创建时间',
    `base_commit` varchar(128) not null default '' comment '基准版本',
    `status` varchar(16) not null default '' comment '任务状态',
    `inherit_commit` varchar(128) not null default '' comment '继承版本',
    `inherit_status` varchar(16) not null default '' comment '继承任务状态',
    `inherit_log` text not null comment '继承日志',
    `normal_cover_data_partition_key` text not null comment '归一化数据分区key',
    `report_partition_key`  text not null comment '报告数据分区key',
    `_created_time` datetime not null default current_timestamp comment '创建时间',
    `_updated_time` datetime not null default current_timestamp on update current_timestamp comment '更新时间',
    `_deleted` tinyint(1) not null default '0' comment '是否删除',
    primary key (`id`),
    key `idx_module_branch_commit` (`module`,`branch`, `commit`)
) engine=InnoDB default charset=utf8mb4 comment='系统测试覆盖率任务表';