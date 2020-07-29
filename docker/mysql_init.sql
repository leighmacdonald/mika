CREATE USER 'mika'@'%' IDENTIFIED BY 'mika';
GRANT ALL PRIVILEGES ON mika.* TO 'mika'@'%';
FLUSH PRIVILEGES;
SELECT "Initialized mika database user"