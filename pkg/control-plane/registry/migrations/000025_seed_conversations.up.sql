-- Seed default conversations for Speech Pipeline (find_store, greeting, goodbye).
-- Shared (tenant_id NULL) for all tenants. Supported languages: uz, en, ru, az, ar.
INSERT INTO conversations (id, intent, name, description, response_template, response_provider_url, supported_languages, tenant_id, created_at, updated_at)
VALUES
  ('conv-find-store', 'find_store', 'Find Store', 'Visitor asks for store location', '{{store_name}} store is here.', NULL, '["uz","en","ru","az","ar"]', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('conv-greeting', 'greeting', 'Greeting', 'Visitor greets the robot', 'Hello! How can I help you?', NULL, '["uz","en","ru","az","ar"]', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('conv-goodbye', 'goodbye', 'Goodbye', 'Visitor says goodbye', 'Goodbye! Have a nice day.', NULL, '["uz","en","ru","az","ar"]', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;
