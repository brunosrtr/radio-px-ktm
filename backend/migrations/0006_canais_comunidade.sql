-- Etapa 6 da 003-rede-orbit: canais criados pela comunidade (FR-029 a
-- FR-032). Canais de empresa da 001 continuam iguais; os da comunidade têm
-- empresa_id nulo e um criador. A exclusão continua lógica (ativo = false).

alter table canal alter column empresa_id drop not null;
alter table canal add column if not exists criador_id uuid references usuario (id);
alter table canal add column if not exists visibilidade text not null default 'privado';
-- Código de convite de canal privado: 8 caracteres sem ambiguidade,
-- mostrado como XXXX-XXXX.
alter table canal add column if not exists codigo_convite varchar(8);

create unique index if not exists canal_codigo_convite_unico
    on canal (codigo_convite) where codigo_convite is not null;

-- Canais de empresa antigos: o primeiro administrador da empresa vira o
-- criador.
update canal c set criador_id = (
    select u.id from usuario u
    where u.empresa_id = c.empresa_id and u.papel = 'admin'
    order by u.criado_em
    limit 1
)
where c.criador_id is null and c.empresa_id is not null;

do $$ begin
    if not exists (select 1 from pg_constraint where conname = 'canal_visibilidade_valida' and conrelid = 'canal'::regclass) then
        alter table canal add constraint canal_visibilidade_valida check (visibilidade in ('publico', 'privado'));
    end if;
    -- Todo canal é de uma empresa ou de alguém da comunidade.
    if not exists (select 1 from pg_constraint where conname = 'canal_tem_dono' and conrelid = 'canal'::regclass) then
        alter table canal add constraint canal_tem_dono check (empresa_id is not null or criador_id is not null);
    end if;
end $$;

-- Busca por nome na aba Explorar (FR-030).
create extension if not exists pg_trgm;
create index if not exists idx_canal_nome_trgm on canal using gin (lower(nome) gin_trgm_ops);

-- Quem entrou por convite num canal privado, ou foi removido pelo criador.
create table if not exists acesso_canal (
    canal_id uuid not null references canal (id) on delete cascade,
    usuario_id uuid not null references usuario (id) on delete cascade,
    situacao text not null check (situacao in ('liberado', 'banido')),
    atualizado_em timestamptz not null default now(),
    primary key (canal_id, usuario_id)
);

create table if not exists favorito_canal (
    usuario_id uuid not null references usuario (id) on delete cascade,
    canal_id uuid not null references canal (id) on delete cascade,
    criado_em timestamptz not null default now(),
    primary key (usuario_id, canal_id)
);

-- Saída por remoção do criador.
alter table participacao_canal drop constraint if exists participacao_canal_motivo_saida_check;
do $$ begin
    if not exists (select 1 from pg_constraint where conname = 'participacao_motivo_valido' and conrelid = 'participacao_canal'::regclass) then
        alter table participacao_canal add constraint participacao_motivo_valido
            check (motivo_saida in ('manual', 'geocerca', 'desconexao', 'encerramento', 'removido'));
    end if;
end $$;
