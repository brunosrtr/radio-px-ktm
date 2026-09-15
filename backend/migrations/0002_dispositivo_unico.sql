-- Permite registrar/atualizar o dispositivo do usuário a cada login
-- (upsert por usuário + plataforma), guardando o push_token usado para
-- avisos de remoção por geocerca (FR-017).
alter table dispositivo
    add constraint dispositivo_usuario_plataforma_unico unique (usuario_id, plataforma);
