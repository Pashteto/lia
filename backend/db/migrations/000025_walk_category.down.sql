DELETE FROM event_categories
WHERE category_id IN (SELECT id FROM categories WHERE slug = 'walk');
DELETE FROM categories WHERE slug = 'walk';
