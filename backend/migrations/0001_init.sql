-- Migração inicial: esquema completo do Rádio PX Digital v1.
-- Idempotente: seguro reaplicar sobre um banco onde já foi aplicada.
-- Princípio I da constituição: nenhuma tabela abaixo armazena áudio.

create extension if not exists pgcrypto;

create table if not exists empresa (
    id uuid primary key default gen_random_uuid(),
    razao_social text not null,
    cnpj varchar(14) not null unique,
    ativa boolean not null default true,
    criado_em timestamptz not null default now()
);

create table if not exists usuario (
    id uuid primary key default gen_random_uuid(),
    empresa_id uuid not null references empresa (id),
    nome text not null,
    login text not null unique,
    senha_hash text not null,
    papel text not null check (papel in ('admin', 'motorista')),
    ativo boolean not null default true,
    criado_em timestamptz not null default now()
);

create index if not exists idx_usuario_empresa_ativo
    on usuario (empresa_id)
    where ativo;

create table if not exists dispositivo (
    id uuid primary key default gen_random_uuid(),
    usuario_id uuid not null references usuario (id),
    plataforma text not null check (plataforma in ('android', 'ios')),
    push_token text,
    versao_app text,
    visto_em timestamptz not null default now()
);

create index if not exists idx_dispositivo_usuario
    on dispositivo (usuario_id);

create table if not exists canal (
    id uuid primary key default gen_random_uuid(),
    empresa_id uuid not null references empresa (id),
    nome text not null,
    descricao text,
    tipo_acesso text not null default 'privado'
        check (tipo_acesso in ('privado', 'compartilhado')),
    limite_participantes int not null default 10
        check (limite_participantes in (5, 10, 15, 20)),
    geocerca_ativa boolean not null default false,
    centro_latitude numeric(9, 6),
    centro_longitude numeric(9, 6),
    raio_metros int,
    ativo boolean not null default true,
    criado_em timestamptz not null default now(),
    constraint geocerca_completa check (
        not geocerca_ativa
        or (
            centro_latitude is not null
            and centro_longitude is not null
            and raio_metros is not null
        )
    )
);

create index if not exists idx_canal_empresa
    on canal (empresa_id);

create table if not exists canal_empresa (
    canal_id uuid not null references canal (id) on delete cascade,
    empresa_id uuid not null references empresa (id),
    liberado_em timestamptz not null default now(),
    primary key (canal_id, empresa_id)
);

create table if not exists participacao_canal (
    id uuid primary key default gen_random_uuid(),
    canal_id uuid not null references canal (id),
    usuario_id uuid not null references usuario (id),
    entrou_em timestamptz not null default now(),
    saiu_em timestamptz,
    motivo_saida text
        check (motivo_saida in ('manual', 'geocerca', 'desconexao', 'encerramento')),
    constraint saiu_em_motivo_coerentes check (
        (saiu_em is null and motivo_saida is null)
        or (saiu_em is not null and motivo_saida is not null)
    )
);

create unique index if not exists idx_participacao_ativa
    on participacao_canal (canal_id, usuario_id)
    where saiu_em is null;

create index if not exists idx_participacao_canal
    on participacao_canal (canal_id)
    where saiu_em is null;

create table if not exists preferencia_canal (
    usuario_id uuid not null references usuario (id),
    canal_id uuid not null references canal (id) on delete cascade,
    silenciado boolean not null default false,
    primary key (usuario_id, canal_id)
);

create table if not exists posicao (
    id bigserial primary key,
    usuario_id uuid not null references usuario (id),
    latitude numeric(9, 6) not null,
    longitude numeric(9, 6) not null,
    precisao_metros numeric(6, 2),
    velocidade_kmh numeric(5, 2),
    capturado_em timestamptz not null,
    recebido_em timestamptz not null default now()
);

create index if not exists idx_posicao_usuario_capturado
    on posicao (usuario_id, capturado_em desc);

create table if not exists posicao_atual (
    usuario_id uuid primary key references usuario (id),
    latitude numeric(9, 6) not null,
    longitude numeric(9, 6) not null,
    velocidade_kmh numeric(5, 2),
    capturado_em timestamptz not null,
    atualizado_em timestamptz not null default now()
);
