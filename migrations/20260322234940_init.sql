-- +goose Up
-- +goose StatementBegin

-- users это пользователи, привязанные к github user, благодаря чему не требуется хранить пароль
CREATE TABLE users (
    -- уникальный ID пользователя
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- github ID для дедупликации пользователей
    github_id       BIGINT UNIQUE NOT NULL,
    -- ссылка на аватар пользователя из github, красиво для UI
    avatar_url      TEXT,
    -- зашифрованный access_token от GitHub
    github_token    BYTEA,
    -- дата создания записи
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE TABLE projects (
    -- уникальный ID проекта
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- foreign key на id пользователя из таблицы users, удаление пользователя означает удаление всех его проектов
    user_id         UUID REFERENCES users(id) ON DELETE CASCADE,
    -- полное имя репозитория на GitHub (owner/repo)
    repo_full_name  TEXT NOT NULL,
    -- название ветки, которая будет собираться и деплоиться джобой
    branch          TEXT NOT NULL,
    -- ссылка для git clone
    clone_url       TEXT NOT NULL,
    -- ID вебхука на GitHub, нужен для удаления при отвязке проекта
    webhook_id      BIGINT,
    -- зашифрованный секрет для валидации подписи входящих вебхуков
    webhook_secret  BYTEA,
    -- дата создания записи
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE TABLE jobs (
    -- уникальный ID джобы
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- foreign key на id проекта, удаление проекта означает удаление всех его джоб
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    -- текущий статус джобы: pending, running, success, failed
    status      TEXT NOT NULL DEFAULT 'pending',
    -- SHA коммита который триггернул джобу
    commit_sha  TEXT NOT NULL,
    -- сообщение коммита
    commit_msg  TEXT NOT NULL,
    -- ссылка на лог в S3, NULL пока воркер не завершил сборку
    log_url     TEXT,
    -- дата создания записи
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    -- дата завершения джобы, NULL пока джоба не завершена
    finished_at TIMESTAMP
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS jobs;

DROP TABLE IF EXISTS projects;

DROP TABLE IF EXISTS users;

-- +goose StatementEnd