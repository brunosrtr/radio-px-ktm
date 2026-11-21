-- Etapa 2 da 003-rede-orbit: leituras Orbit feitas por pontos de
-- transmissão (celulares) e por pontos fixos (data-model.md §4).
-- Princípio VII: nenhuma tabela de leitura aponta para o usuário que leu.

create table if not exists orbit (
    id varchar(12) primary key check (id ~ '^ORBIT-[0-9A-F]{6}$'),
    virtual boolean not null default false,
    -- Placa ou apelido do caminhão (demonstração). Só o painel da KTM vê.
    rotulo text,
    ultima_latitude numeric(9, 6),
    ultima_longitude numeric(9, 6),
    ultima_leitura_em timestamptz,
    criado_em timestamptz not null default now()
);

create table if not exists ponto_fixo (
    id uuid primary key default gen_random_uuid(),
    nome text not null,
    latitude numeric(9, 6) not null,
    longitude numeric(9, 6) not null,
    raio_metros int not null check (raio_metros between 50 and 5000),
    simulado boolean not null default true,
    criado_em timestamptz not null default now()
);

create table if not exists leitura_orbit (
    id bigserial primary key,
    orbit_id varchar(12) not null references orbit (id),
    origem text not null check (origem in ('celular', 'fixo')),
    -- Celular: HMAC(segredo, usuario_id ‖ dia), opaco e renovado a cada 24 h.
    -- Fixo: id do ponto_fixo. Nunca o usuario_id.
    ponto_id varchar(36) not null,
    latitude numeric(9, 6) not null,
    longitude numeric(9, 6) not null,
    precisao_metros numeric(6, 2),
    rssi smallint,
    detectado_em timestamptz not null,
    recebido_em timestamptz not null default now(),
    -- Reenviar um lote depois de falha de rede não duplica leituras.
    constraint leitura_orbit_unica unique (orbit_id, ponto_id, detectado_em)
);

create index if not exists idx_leitura_orbit_historico
    on leitura_orbit (orbit_id, detectado_em desc);
create index if not exists idx_leitura_orbit_recebido
    on leitura_orbit (recebido_em);

-- Contadores de "Minha contribuição" (FR-040), separados de leitura_orbit
-- para que a leitura nunca aponte para quem a fez.
create table if not exists contribuicao_diaria (
    usuario_id uuid not null references usuario (id) on delete cascade,
    dia date not null,
    leituras int not null default 0 check (leituras >= 0),
    primary key (usuario_id, dia)
);
