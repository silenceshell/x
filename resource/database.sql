DROP database if EXISTS x;
CREATE database x;
use x;

DROP TABLE if EXISTS `visitor`;
CREATE TABLE visitor (ip varchar(64) NOT NULL, create_time datetime NOT NULL);

DROP TABLE if EXISTS `shorturl`;
CREATE TABLE `shorturl` (
  `id` bigint(20) UNSIGNED NOT NULL AUTO_INCREMENT,
  `long_url` varchar(255) NOT NULL,
  `short_url` varchar(255) DEFAULT NULL,
  `create_time` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
