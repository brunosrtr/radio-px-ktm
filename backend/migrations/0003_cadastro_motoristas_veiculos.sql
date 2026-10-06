-- Mantém as contas existentes e acrescenta o cadastro administrativo.
create table if not exists veiculo (
 id uuid primary key default gen_random_uuid(),
 empresa_id uuid not null references empresa(id),
 descricao text not null,
 placa varchar(7) not null,
 criado_em timestamptz not null default now(),
 unique (empresa_id, placa),
 unique (empresa_id, id)
);
alter table usuario add column if not exists primeiro_nome text;
alter table usuario add column if not exists sobrenome text;
alter table usuario add column if not exists cpf varchar(11);
alter table usuario add column if not exists veiculo_id uuid;
create unique index if not exists usuario_cpf_unico on usuario(cpf) where cpf is not null;
do $$ begin
 if not exists (select 1 from pg_constraint where conname='usuario_veiculo_empresa_fk' and conrelid='usuario'::regclass) then
  alter table usuario add constraint usuario_veiculo_empresa_fk foreign key (empresa_id, veiculo_id) references veiculo(empresa_id, id);
 end if;
end $$;
