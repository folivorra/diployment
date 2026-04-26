-- +goose Up
-- +goose StatementBegin

-- users это пользователи, привязанные к github user, благодаря чему не требуется хранить пароль
CREATE TABLE users (
    -- уникальный ID пользователя
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- github ID для дедупликации пользователей
    github_id BIGINT UNIQUE NOT NULL,
    -- ссылка на аватар пользователя из github, красиво для UI
    avatar_url TEXT,
    -- зашифрованный access_token от GitHub
    github_token BYTEA,
    -- дата создания записи
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE projects (
    -- уникальный ID проекта
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- foreign key на id пользователя из таблицы users, удаление пользователя означает удаление всех его проектов
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    -- полное имя репозитория на GitHub (owner/repo)
    repo_full_name TEXT NOT NULL,
    -- ссылка для git clone
    clone_url TEXT NOT NULL,
    -- ID вебхука на GitHub, нужен для удаления при отвязке проекта
    webhook_id BIGINT,
    -- зашифрованный секрет для валидации подписи входящих вебхуков
    webhook_secret BYTEA,
    -- дата создания записи
    created_at TIMESTAMP DEFAULT NOW()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS projects;

DROP TABLE IF EXISTS users;

-- +goose StatementEnd