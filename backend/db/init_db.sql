-- init_db.sql
-- Очистка таблиц
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS calendar_events CASCADE;
DROP TABLE IF EXISTS devices CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS files CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS families CASCADE;

-- Создание таблиц
CREATE TABLE families (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    family_id INTEGER REFERENCES families(id) ON DELETE CASCADE,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) CHECK (role IN ('child', 'adult', 'admin')) DEFAULT 'child',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE files (
    id SERIAL PRIMARY KEY,
    family_id INTEGER REFERENCES families(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    path VARCHAR(500) NOT NULL,
    access_type VARCHAR(20) DEFAULT 'private',
    share_token VARCHAR(100) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    family_id INTEGER REFERENCES families(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(10,2) NOT NULL,
    type VARCHAR(20) CHECK (type IN ('income', 'expense')),
    category VARCHAR(50),
    description TEXT,
    date DATE DEFAULT CURRENT_DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE messages (
    id SERIAL PRIMARY KEY,
    family_id INTEGER REFERENCES families(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    username VARCHAR(50),
    message TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE devices (
    id SERIAL PRIMARY KEY,
    family_id INTEGER REFERENCES families(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50),
    status VARCHAR(20) DEFAULT 'off',
    settings JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE calendar_events (
    id SERIAL PRIMARY KEY,
    family_id INTEGER REFERENCES families(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    event_date DATE NOT NULL,
    event_time TIME,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Вставка тестовых данных
INSERT INTO families (name) VALUES 
('Ивановы'),
('Петровы'),
('Сидоровы');

-- Хэш пароля "password123"
INSERT INTO users (family_id, username, email, password_hash, role) VALUES 
(1, 'admin_ivanov', 'admin@ivanov.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.Mr/.cYqMq3YdxgRqGVQwQyYFOHYdVcK', 'admin'),
(1, 'adult_ivanov', 'adult@ivanov.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.Mr/.cYqMq3YdxgRqGVQwQyYFOHYdVcK', 'adult'),
(1, 'child_ivanov', 'child@ivanov.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.Mr/.cYqMq3YdxgRqGVQwQyYFOHYdVcK', 'child'),
(1, 'mom_ivanov', 'mom@ivanov.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.Mr/.cYqMq3YdxgRqGVQwQyYFOHYdVcK', 'adult'),
(1, 'dad_ivanov', 'dad@ivanov.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.Mr/.cYqMq3YdxgRqGVQwQyYFOHYdVcK', 'adult'),
(2, 'admin_petrov', 'admin@petrov.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.Mr/.cYqMq3YdxgRqGVQwQyYFOHYdVcK', 'admin'),
(2, 'adult_petrov', 'adult@petrov.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.Mr/.cYqMq3YdxgRqGVQwQyYFOHYdVcK', 'adult'),
(2, 'child_petrov', 'child@petrov.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.Mr/.cYqMq3YdxgRqGVQwQyYFOHYdVcK', 'child'),
(3, 'admin_sidorov', 'admin@sidorov.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.Mr/.cYqMq3YdxgRqGVQwQyYFOHYdVcK', 'admin'),
(3, 'adult_sidorov', 'adult@sidorov.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.Mr/.cYqMq3YdxgRqGVQwQyYFOHYdVcK', 'adult');

-- Транзакции
INSERT INTO transactions (family_id, user_id, amount, type, category, description, date) VALUES
(1, 2, 50000, 'income', 'salary', 'Зарплата', CURRENT_DATE - INTERVAL '30 days'),
(1, 2, 15000, 'expense', 'food', 'Продукты', CURRENT_DATE - INTERVAL '25 days'),
(1, 2, 5000, 'expense', 'bills', 'Коммунальные платежи', CURRENT_DATE - INTERVAL '20 days'),
(1, 2, 3000, 'expense', 'transport', 'Бензин', CURRENT_DATE - INTERVAL '15 days'),
(1, 2, 55000, 'income', 'salary', 'Зарплата', CURRENT_DATE - INTERVAL '2 days'),
(1, 2, 18000, 'expense', 'food', 'Продукты', CURRENT_DATE - INTERVAL '3 days');

-- Сообщения
INSERT INTO messages (family_id, user_id, username, message) VALUES
(1, 1, 'admin_ivanov', 'Добро пожаловать в семейный чат!'),
(1, 2, 'adult_ivanov', 'Кто будет забирать детей из школы?'),
(1, 3, 'child_ivanov', 'Я пришел!'),
(1, 4, 'mom_ivanov', 'Не забудьте купить хлеб'),
(1, 5, 'dad_ivanov', 'Уже купил 👍');

-- Устройства
INSERT INTO devices (family_id, name, type, status, settings) VALUES
(1, 'Гостиная', 'light', 'off', '{"brightness": 80}'::jsonb),
(1, 'Кухня', 'light', 'on', '{"brightness": 100}'::jsonb),
(1, 'Термостат', 'thermostat', 'on', '{"temperature": 22}'::jsonb);

-- События
INSERT INTO calendar_events (family_id, user_id, title, description, event_date, event_time) VALUES
(1, 2, 'День рождения', 'Нужно купить торт', CURRENT_DATE + INTERVAL '10 days', '12:00'),
(1, 2, 'Родительское собрание', 'Школа', CURRENT_DATE + INTERVAL '5 days', '18:30');

-- Вывод статистики
SELECT 'Families: ' || COUNT(*) FROM families
UNION ALL
SELECT 'Users: ' || COUNT(*) FROM users
UNION ALL
SELECT 'Transactions: ' || COUNT(*) FROM transactions
UNION ALL
SELECT 'Messages: ' || COUNT(*) FROM messages
UNION ALL
SELECT 'Devices: ' || COUNT(*) FROM devices
UNION ALL
SELECT 'Events: ' || COUNT(*) FROM calendar_events;