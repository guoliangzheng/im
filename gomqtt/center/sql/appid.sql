
DROP TABLE if exists `gomqtt`.`appid`; 
CREATE TABLE `gomqtt`.`appid` (
  `appid` varchar(32) NOT NULL COMMENT 'app应用的唯一ID',

  `user`    varchar(40) NOT NULL COMMENT '管理员的用户名',
  `password` varchar(40) NOT NULL COMMENT '管理员密码',

  `topics` mediumtext COMMENT '要订阅的topics,json格式',

  `compress` int(11) COMMENT '数据使用的压缩格式',

  `payloadType` int(11) COMMENT 'mqtt payload使用的编码类型1.Text 2.Json 3.protobuf',

  `desc` varchar(255) COMMENT 'appid描述',


  `inputDate` DATETIME  NOT NULL COMMENT '插入时间',
  `updateDate` DATETIME DEFAULT NULL COMMENT '更新时间',

   PRIMARY KEY (`appid`),

  KEY `Index_Service_user` (`user`)
) DEFAULT CHARSET=utf8  COMMENT='gomqtt appid配置';