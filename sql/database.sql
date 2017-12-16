DROP database if EXISTS x;
CREATE database x;
use x;

DROP TABLE if EXISTS `visitor`;
CREATE TABLE visitor (ip varchar(64) NOT NULL, create_time datetime NOT NULL);
