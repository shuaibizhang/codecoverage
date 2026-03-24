drop table if exists `diff_cache`;

create table `diff_cache` (
    `id` bigint(20) unsigned not null auto_increment comment '自增id',
    `module` varchar(255) not null COMMENT '模块名',
    `commit_id` varchar(64) not null COMMENT '当前版本',
    `base_commit_id` varchar(64) not null COMMENT '基准版本',
    `diff_partition_key` text not null comment 'diff在oss上的数据分区key',
    `_created_time` datetime not null default current_timestamp comment '创建时间',
    `_updated_time` datetime not null default current_timestamp on update current_timestamp comment '更新时间',
    `_deleted` tinyint(1) not null default '0' comment '是否删除',
    primary key (`id`),
    key `idx_module_commit_base_commit` (`module`, `commit_id`, `base_commit_id`)
) engine=InnoDB default charset=utf8mb4 comment='diff数据缓存表';