DELETE FROM event_categories
WHERE category_id IN (SELECT id FROM categories WHERE slug = 'reading-group');
DELETE FROM categories WHERE slug = 'reading-group';
