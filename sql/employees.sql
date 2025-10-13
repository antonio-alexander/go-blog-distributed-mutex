-- DROP DATABASE IF EXISTS go_blog_distributed_mutex
CREATE DATABASE IF NOT EXISTS go_blog_distributed_mutex;

USE go_blog_distributed_mutex;

-- DROP TABLE IF EXISTS employee
CREATE TABLE IF NOT EXISTS employee (
    email_address TEXT NOT NULL,
    first_name TEXT,
    last_name TEXT,
    version INT NOT NULL DEFAULT 1,
    PRIMARY KEY (email_address(255))
) ENGINE = InnoDB;

