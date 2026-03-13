-- Remove seeded default conversations
DELETE FROM conversations WHERE id IN ('conv-find-store', 'conv-greeting', 'conv-goodbye');
