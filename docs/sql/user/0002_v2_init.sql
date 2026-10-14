-- v2 基础账户与好友模型；时间统一 Unix 毫秒。
CREATE TABLE IF NOT EXISTS accounts (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 account VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 password_hash VARCHAR(255) NOT NULL,
 nickname VARCHAR(32) NOT NULL,
 avatar_url VARCHAR(512) NOT NULL DEFAULT '',
 bio VARCHAR(200) NOT NULL DEFAULT '',
 disabled BOOLEAN NOT NULL DEFAULT FALSE,
 ack_cursor BIGINT UNSIGNED NOT NULL DEFAULT 0,
 created_at BIGINT NOT NULL,
 updated_at BIGINT NOT NULL,
 UNIQUE KEY accounts_name (account)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS friendships (
 owner_id BIGINT UNSIGNED NOT NULL,
 target_id BIGINT UNSIGNED NOT NULL,
 active BOOLEAN NOT NULL DEFAULT TRUE,
 remark VARCHAR(32) NOT NULL DEFAULT '',
 conversation_id BIGINT UNSIGNED NOT NULL,
 updated_at BIGINT NOT NULL,
 PRIMARY KEY (owner_id, target_id),
 KEY friendships_target (target_id, owner_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS blocks (
 owner_id BIGINT UNSIGNED NOT NULL,
 target_id BIGINT UNSIGNED NOT NULL,
 created_at BIGINT NOT NULL,
 PRIMARY KEY (owner_id, target_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS friend_requests (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 sender_id BIGINT UNSIGNED NOT NULL,
 recipient_id BIGINT UNSIGNED NOT NULL,
 note VARCHAR(200) NOT NULL DEFAULT '',
 state INT NOT NULL DEFAULT 1,
 conversation_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
 created_at BIGINT NOT NULL,
 updated_at BIGINT NOT NULL,
 KEY friend_requests_sender (sender_id, id),
 KEY friend_requests_recipient (recipient_id, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS friend_events (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 user_id BIGINT UNSIGNED NOT NULL,
 data JSON NOT NULL,
 created_at BIGINT NOT NULL,
 KEY friend_events_user (user_id, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS outbox (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 event_key VARCHAR(100) NOT NULL,
 payload JSON NOT NULL,
 published BOOLEAN NOT NULL DEFAULT FALSE,
 attempts INT UNSIGNED NOT NULL DEFAULT 0,
 next_attempt_at BIGINT NOT NULL DEFAULT 0,
 created_at BIGINT NOT NULL,
 UNIQUE KEY outbox_event (event_key),
 KEY outbox_retry (published, next_attempt_at, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
