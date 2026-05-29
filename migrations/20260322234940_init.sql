-- +goose Up
-- +goose StatementBegin

-- users это пользователи, привязанные к github user, благодаря чему не требуется хранить пароль
CREATE TABLE users (
    -- уникальный ID пользователя
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- github ID для дедупликации пользователей
    github_id       BIGINT UNIQUE NOT NULL,
    -- github login (никнейм), используется в UI вместо id
    github_login    TEXT NOT NULL DEFAULT '',
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
    created_at      TIMESTAMP DEFAULT NOW(),
    -- хост целевого сервера для деплоя
    ssh_host        TEXT NOT NULL DEFAULT '',
    -- SSH-порт целевого сервера
    ssh_port        INT NOT NULL DEFAULT 22,
    -- имя пользователя для SSH-подключения
    ssh_user        TEXT NOT NULL DEFAULT '',
    -- зашифрованный SSH-ключ (PEM) для аутентификации на сервере
    ssh_key         BYTEA,
    -- команда, которую deployer выполнит после загрузки артефакта
    deploy_restart_cmd TEXT NOT NULL DEFAULT '',
    -- рабочая директория на целевом сервере, куда загружается артефакт
    deploy_workdir  TEXT NOT NULL DEFAULT ''
);

CREATE TABLE jobs (
    -- уникальный ID джобы
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- foreign key на id проекта, удаление проекта означает удаление всех его джоб
    project_id          UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    -- текущий статус джобы: pending, running, success, failed
    status              TEXT NOT NULL DEFAULT 'pending',
    -- SHA коммита который триггернул джобу
    commit_sha          TEXT NOT NULL,
    -- сообщение коммита
    commit_msg          TEXT NOT NULL,
    -- дата создания записи
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    -- ссылка на лог сборки в S3, NULL пока воркер не завершил сборку
    build_log_url       TEXT,
    -- дата начала фазы сборки, NULL пока воркер не взял задачу
    build_started_at    TIMESTAMP,
    -- дата завершения фазы сборки, NULL пока сборка не завершена
    build_finished_at   TIMESTAMP,
    -- ссылка на лог деплоя в S3, NULL пока deployer не завершил деплой
    deploy_log_url      TEXT,
    -- дата начала фазы деплоя, NULL пока deployer не взял задачу
    deploy_started_at   TIMESTAMP,
    -- дата завершения фазы деплоя, NULL пока деплой не завершён
    deploy_finished_at  TIMESTAMP
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS jobs;

DROP TABLE IF EXISTS projects;

DROP TABLE IF EXISTS users;

-- +goose StatementEnd