-- Etapa 1 da 003-rede-orbit: contas abertas ao público (FR-023 a FR-028,
-- FR-033). O vínculo com empresa passa a ser opcional; contas de empresa da
-- 001 continuam válidas sem nenhuma alteração de dados além do apelido.

create extension if not exists citext;

alter table usuario alter column empresa_id drop not null;
alter table usuario alter column login drop not null;

alter table usuario add column if not exists email citext;
alter table usuario add column if not exists apelido text;
alter table usuario add column if not exists excluido_em timestamptz;

create unique index if not exists usuario_email_unico
    on usuario (email) where email is not null;

-- Apelido de rádio das contas existentes: primeiro nome, limitado a 24
-- caracteres (FR-027).
update usuario
set apelido = left(coalesce(nullif(primeiro_nome, ''), split_part(nome, ' ', 1), nome), 24)
where apelido is null;

alter table usuario drop constraint if exists usuario_papel_check;

do $$ begin
    if not exists (select 1 from pg_constraint where conname = 'usuario_papel_valido' and conrelid = 'usuario'::regclass) then
        alter table usuario add constraint usuario_papel_valido
            check (papel in ('usuario', 'motorista', 'admin', 'admin_ktm', 'demo'));
    end if;
    -- Papéis de empresa exigem empresa; os demais nunca têm (FR-033).
    if not exists (select 1 from pg_constraint where conname = 'usuario_papel_empresa_coerentes' and conrelid = 'usuario'::regclass) then
        alter table usuario add constraint usuario_papel_empresa_coerentes
            check ((papel in ('motorista', 'admin')) = (empresa_id is not null));
    end if;
    if not exists (select 1 from pg_constraint where conname = 'usuario_tem_identificador' and conrelid = 'usuario'::regclass) then
        alter table usuario add constraint usuario_tem_identificador
            check (email is not null or login is not null);
    end if;
    if not exists (select 1 from pg_constraint where conname = 'usuario_apelido_tamanho' and conrelid = 'usuario'::regclass) then
        alter table usuario add constraint usuario_apelido_tamanho
            check (apelido is null or char_length(apelido) between 1 and 24);
    end if;
end $$;

-- Recuperação de senha por código de 6 dígitos enviado por e-mail (FR-025).
-- Só o hash do código é guardado.
create table if not exists recuperacao_senha (
    id uuid primary key default gen_random_uuid(),
    usuario_id uuid not null references usuario (id) on delete cascade,
    codigo_hash text not null,
    expira_em timestamptz not null,
    tentativas int not null default 0,
    usado_em timestamptz,
    criado_em timestamptz not null default now()
);

create index if not exists idx_recuperacao_senha_usuario
    on recuperacao_senha (usuario_id, criado_em desc);

-- Consentimentos versionados (FR-028, Princípio VII): aceite dos termos e
-- autorização para atuar como ponto de transmissão, revogável.
create table if not exists consentimento (
    usuario_id uuid not null references usuario (id) on delete cascade,
    tipo text not null check (tipo in ('termos', 'ponto_transmissao')),
    versao text not null,
    aceito_em timestamptz not null default now(),
    revogado_em timestamptz,
    primary key (usuario_id, tipo, versao)
);
