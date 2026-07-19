-- 回滚上架包推送扩展。
ALTER TABLE push_campaign        DROP COLUMN kind;
ALTER TABLE push_record          DROP COLUMN listing_id;
ALTER TABLE push_campaign_target DROP INDEX idx_target_listing, DROP COLUMN listing_id;
ALTER TABLE push_device_token    DROP INDEX idx_token_listing_gate,
                                 DROP COLUMN last_gate_at,
                                 DROP COLUMN last_gate_mode,
                                 DROP COLUMN listing_id;
