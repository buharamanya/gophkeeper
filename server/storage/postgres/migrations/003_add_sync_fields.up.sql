-- Добавляем поля для синхронизации в таблицу data_entries
ALTER TABLE data_entries 
ADD COLUMN last_sync_time TIMESTAMP WITH TIME ZONE,
ADD COLUMN is_deleted BOOLEAN DEFAULT FALSE,
ADD COLUMN device_id VARCHAR(100);

-- Создаем индекс для оптимизации запросов синхронизации
CREATE INDEX IF NOT EXISTS idx_data_entries_sync ON data_entries(user_id, updated_at, last_sync_time);
CREATE INDEX IF NOT EXISTS idx_data_entries_deleted ON data_entries(user_id, is_deleted);